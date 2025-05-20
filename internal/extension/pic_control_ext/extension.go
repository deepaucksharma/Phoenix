// Package pic_control_ext implements the pic_control extension for the SA-OMF.
package pic_control_ext

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
	"github.com/deepaucksharma/Phoenix/pkg/policy"
)

const (
	typeStr = "pic_control"
)

var (
	errPolicyLoadFailed  = errors.New("failed to load policy")
	errProcessorNotFound = errors.New("target processor not found")
	errPatchRateLimited  = errors.New("patch rate limited")
	errSafeModeActive    = errors.New("safe mode active")
	errPatchExpired      = errors.New("patch has expired")
)

// Config defines configuration for the pic_control extension
type Config struct {
	PolicyFilePath       string                 `mapstructure:"policy_file_path"`
	MaxPatchesPerMinute  int                    `mapstructure:"max_patches_per_minute"`
	PatchCooldownSeconds int                    `mapstructure:"patch_cooldown_seconds"`
	SafeModeConfigs      map[string]interface{} `mapstructure:"safe_mode_processor_configs"`
	OpAMPConfig          *OpAMPClientConfig     `mapstructure:"opamp_client_config"`
	PolicyFile           string                 `mapstructure:"-"` // For testing, file path to use directly
	WatchPolicy          bool                   `mapstructure:"-"` // For testing, whether to watch policy file
	MetricsEmitter       interface{}            `mapstructure:"-"` // For testing, custom metrics emitter
}

// OpAMPClientConfig defines configuration for the OpAMP client
type OpAMPClientConfig struct {
	ServerURL           string `mapstructure:"server_url"`
	InsecureSkipVerify  bool   `mapstructure:"insecure_skip_verify"`
	ClientCertFile      string `mapstructure:"client_cert_file"`
	ClientKeyFile       string `mapstructure:"client_key_file"`
	CACertFile          string `mapstructure:"ca_cert_file"`
	PollIntervalSeconds int    `mapstructure:"poll_interval_seconds"`
}

// NewFactory creates a factory for the pic_control extension
func NewFactory() extension.Factory {
	return extension.NewFactory(
		component.MustNewType(typeStr),
		createDefaultConfig,
		createExtension,
		component.StabilityLevelDevelopment,
	)
}

// createDefaultConfig creates the default configuration
func createDefaultConfig() component.Config {
	return &Config{
		PolicyFilePath:       "/etc/sa-omf/policy.yaml",
		MaxPatchesPerMinute:  3,
		PatchCooldownSeconds: 10,
		SafeModeConfigs:      make(map[string]interface{}),
		OpAMPConfig: &OpAMPClientConfig{
			PollIntervalSeconds: 5,
		},
	}
}

// createExtension creates the extension
func createExtension(
	ctx context.Context,
	set extension.Settings,
	cfg component.Config,
) (extension.Extension, error) {
	config := cfg.(*Config)

	// Create extension
	return newExtension(config, set.TelemetrySettings.Logger)
}

// Interface for the safety monitor
type safetyMonitor interface {
	IsInSafeMode() bool
	TemporarilyOverrideThresholds(seconds int)
}

// Extension implements the pic_control extension
type Extension struct {
	config          *Config
	logger          *zap.Logger
	host            component.Host
	processors      map[component.ID]interfaces.UpdateableProcessor
	policy          *policy.Policy
	watcher         *fsnotify.Watcher
	cancelWatcher   context.CancelFunc
	patchHistory    []interfaces.ConfigPatch
	lastPatchTime   time.Time
	patchCount      int
	patchCountReset time.Time
	safeMode        bool
	lock            sync.RWMutex
	metrics         *metrics.MetricsEmitter
	emitMetric      func(name string, value float64) // Used for testing
	safetyMonitor   safetyMonitor                    // Interface to safety monitor
}

// PicControl exports the main API for pic_control
type PicControl interface {
	SubmitConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error
}

// Ensure Extension implements PicControl
var _ PicControl = (*Extension)(nil)

// newExtension creates a new pic_control extension
func newExtension(config *Config, logger *zap.Logger) (*Extension, error) {
	return &Extension{
		config:          config,
		logger:          logger,
		processors:      make(map[component.ID]interfaces.UpdateableProcessor),
		patchHistory:    make([]interfaces.ConfigPatch, 0, 100),
		lastPatchTime:   time.Time{},
		patchCount:      0,
		patchCountReset: time.Now(),
		safeMode:        false,
	}, nil
}

// NewExtension creates a new pic_control extension - exported for testing
func NewExtension(config *Config, logger *zap.Logger) (*Extension, error) {
	return newExtension(config, logger)
}

// NewPICControlExtension creates a new PIC control extension with telemetry settings - exported for testing
func NewPICControlExtension(config *Config, telemetrySettings component.TelemetrySettings) (*Extension, error) {
	ext, err := newExtension(config, telemetrySettings.Logger)
	if err != nil {
		return nil, err
	}
	
	// If we have a metrics emitter configured in the test config, use it
	if testEmitter, ok := config.MetricsEmitter.(*metrics.MetricsCollector); ok {
		// Set up a custom handler for metrics
		ext.emitMetric = func(name string, value float64) {
			testEmitter.AddMetric(name, value)
		}
	}
	
	return ext, nil
}

// Start starts the extension
func (e *Extension) Start(ctx context.Context, host component.Host) error {
	e.host = host

	// Set up metrics
	// This needs a concrete implementation of metric.MeterProvider
	// We'll leave this commented for now
	/*
		metricProvider := host.GetExtensions()[component.MustNewID("prometheus")]
		if metricProvider != nil {
			e.metrics = metrics.NewMetricsEmitter(metricProvider.(metric.MeterProvider).Meter("pic_control"),
			                                     "pic_control", component.MustNewID(typeStr))
		}
	*/

	// Register processors
	if err := e.registerProcessors(); err != nil {
		return fmt.Errorf("registering processors: %w", err)
	}

	// Load initial policy
	if err := e.loadPolicy(e.config.PolicyFilePath); err != nil {
		return fmt.Errorf("loading initial policy: %w", err)
	}

	// Start policy file watcher
	if err := e.startWatcher(); err != nil {
		return fmt.Errorf("starting policy watcher: %w", err)
	}

	// Start OpAMP client if configured
	if e.config.OpAMPConfig != nil {
		if err := e.startOpAMPClient(ctx); err != nil {
			e.logger.Warn("Failed to start OpAMP client", zap.Error(err))
		}
	}

	return nil
}

// Shutdown stops the extension
func (e *Extension) Shutdown(ctx context.Context) error {
	// Stop the watcher
	if e.cancelWatcher != nil {
		e.cancelWatcher()
	}

	if e.watcher != nil {
		return e.watcher.Close()
	}

	return nil
}

// registerProcessors finds and registers all UpdateableProcessor instances
func (e *Extension) registerProcessors() error {
	if e.host == nil {
		return fmt.Errorf("host not initialized")
	}

	// Note: The current implementation doesn't provide direct access to processors
	// We'll use a placeholder method to simulate processor discovery
	// This would be replaced with actual processor discovery in a production environment

	// Find processors from the host - this simulated code must be updated when a real solution
	// for processor discovery is implemented
	testProcessors := map[component.ID]interfaces.UpdateableProcessor{}

	// Simulated processors for testing
	for id, proc := range testProcessors {
		e.processors[id] = proc
		e.logger.Info("Registered updateable processor", zap.String("id", id.String()))
	}

	return nil
}

// RegisterUpdateableProcessor registers a processor for dynamic configuration
func (e *Extension) RegisterUpdateableProcessor(processor interfaces.UpdateableProcessor) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	
	// For testing purposes, we'll construct an ID from the processor name
	// In a real system, we'd need to handle this more robustly
	// In a real system, we'd get the component ID directly. For testing we'll construct it.
	id := component.NewIDWithName(component.MustNewType("processor"), processor.(interfaces.UpdateableProcessor).GetName())
	e.processors[id] = processor
	e.logger.Info("Registered updateable processor", zap.String("id", id.String()))
	
	return nil
}

// RegisterProcessor registers a processor with a specific ID for dynamic configuration
func (e *Extension) RegisterProcessor(id component.ID, processor interfaces.UpdateableProcessor) {
	e.lock.Lock()
	defer e.lock.Unlock()
	
	e.processors[id] = processor
	e.logger.Info("Registered processor with explicit ID", zap.String("id", id.String()))
}

// ApplyConfigPatch applies a configuration patch to a processor
// This is an external API for testing
func (e *Extension) ApplyConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	// This is a wrapper around SubmitConfigPatch for testing
	return e.SubmitConfigPatch(ctx, patch)
}

// SubmitConfigPatch processes a ConfigPatch
func (e *Extension) SubmitConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	// Check if safe mode is active
	if e.safeMode {
		// If this is an urgent patch with safety override, temporarily bypass safe mode
		if patch.Severity == "urgent" && patch.SafetyOverride {
			e.logger.Info("Processing urgent patch with safety override",
				zap.String("patch_id", patch.PatchID),
				zap.String("target", patch.TargetProcessorName.String()))
			
			// Call back into the safety monitor to temporarily increase thresholds
			// This would require access to the safety monitor instance
			if e.safetyMonitor != nil {
				e.safetyMonitor.TemporarilyOverrideThresholds(0) // Use default expiry time
				
				if e.emitMetric != nil {
					e.emitMetric("aemf_safety_thresholds_overridden_total", 1.0)
				}
			}
		} else {
			e.logger.Warn("Patch rejected: Safe mode active",
				zap.String("patch_id", patch.PatchID),
				zap.String("target", patch.TargetProcessorName.String()))
			
			if e.emitMetric != nil {
				e.emitMetric("aemf_patch_safe_mode_rejected_total", 1.0)
			}
			return errSafeModeActive
		}
	}

	// Check patch TTL
	if patch.Timestamp > 0 && patch.TTLSeconds > 0 {
		expirationTime := time.Unix(patch.Timestamp, 0).Add(time.Duration(patch.TTLSeconds) * time.Second)
		if time.Now().After(expirationTime) {
			e.logger.Warn("Patch rejected: Expired",
				zap.String("patch_id", patch.PatchID),
				zap.Time("expiration", expirationTime))
			
			if e.emitMetric != nil {
				e.emitMetric("aemf_patch_expired_total", 1.0)
			}
			return errPatchExpired
		}
	}

	// Check rate limiting - urgent patches bypass rate limiting
	if patch.Severity != "urgent" && !e.checkRateLimit() {
		e.logger.Warn("Patch rejected: Rate limited",
			zap.String("patch_id", patch.PatchID),
			zap.String("target", patch.TargetProcessorName.String()))
		
		if e.emitMetric != nil {
			e.emitMetric("aemf_patch_rate_limited_total", 1.0)
		}
		return errPatchRateLimited
	}

	// Check cooldown - urgent patches bypass cooldown
	if patch.Severity != "urgent" && !e.lastPatchTime.IsZero() &&
		time.Since(e.lastPatchTime) < time.Duration(e.config.PatchCooldownSeconds)*time.Second {
		e.logger.Warn("Patch rejected: Cooldown active",
			zap.String("patch_id", patch.PatchID),
			zap.Duration("remaining",
				time.Duration(e.config.PatchCooldownSeconds)*time.Second-time.Since(e.lastPatchTime)))
		
		if e.emitMetric != nil {
			e.emitMetric("aemf_patch_cooldown_rejected_total", 1.0)
		}
		return errPatchRateLimited
	}

	// Get the target processor
	processor, exists := e.processors[patch.TargetProcessorName]
	if !exists {
		e.logger.Warn("Patch rejected: Target processor not found",
			zap.String("patch_id", patch.PatchID),
			zap.String("target", patch.TargetProcessorName.String()))
		
		if e.emitMetric != nil {
			e.emitMetric("aemf_patch_target_not_found_total", 1.0)
		}
		return errProcessorNotFound
	}

	// Get current status to set PrevValue
	status, err := processor.GetConfigStatus(ctx)
	if err != nil {
		e.logger.Warn("Failed to get processor status",
			zap.String("processor", patch.TargetProcessorName.String()),
			zap.Error(err))
	} else if status.Parameters != nil {
		// Try to find the parameter in the current status
		// This is simplified; in a real implementation, we would traverse the parameter path
		if val, exists := status.Parameters[patch.ParameterPath]; exists {
			patch.PrevValue = val
		}
	}

	// Apply the patch
	err = processor.OnConfigPatch(ctx, patch)
	if err != nil {
		e.logger.Warn("Failed to apply patch",
			zap.String("patch_id", patch.PatchID),
			zap.String("target", patch.TargetProcessorName.String()),
			zap.Error(err))
		
		if e.emitMetric != nil {
			e.emitMetric("aemf_patch_validation_failed_total", 1.0)
		}
		return err
	}

	// Update rate limiting state
	e.lastPatchTime = time.Now()
	e.patchCount++

	// Record patch in history
	e.patchHistory = append(e.patchHistory, patch)
	if len(e.patchHistory) > 100 {
		// Keep last 100 patches
		e.patchHistory = e.patchHistory[len(e.patchHistory)-100:]
	}

	e.logger.Info("Applied patch",
		zap.String("patch_id", patch.PatchID),
		zap.String("target", patch.TargetProcessorName.String()),
		zap.String("parameter", patch.ParameterPath),
		zap.Any("new_value", patch.NewValue))

	return nil
}

// External implementations:
// checkRateLimit from ratelimit.go
// loadPolicy from policy.go

// loadPolicyBytes loads policy YAML from memory and applies it.
func (e *Extension) loadPolicyBytes(data []byte) error {
	newPolicy, err := policy.ParsePolicy(data)
	if err != nil {
		return err
	}

	e.policy = newPolicy
	return e.applyPolicyConfig()
}

// applyPolicyConfig applies the processor configurations from the policy
func (e *Extension) applyPolicyConfig() error {
	ctx := context.Background()

	// Apply configurations to each processor
	for id, processor := range e.processors {
		// Extract the processor type from the ID
		parts := strings.Split(id.String(), "/")
		if len(parts) == 0 {
			continue
		}
		procType := parts[len(parts)-1]

		// Check if this processor type has a configuration in the policy
		if procConfig, exists := e.policy.ProcessorsConfig[procType]; exists {
			// Create a ConfigPatch for each parameter
			for paramName, paramValue := range procConfig {
				// Skip "enabled" parameter for now
				if paramName == "enabled" {
					continue
				}

				patch := interfaces.ConfigPatch{
					PatchID:             fmt.Sprintf("policy-init-%s-%s", procType, paramName),
					TargetProcessorName: id,
					ParameterPath:       paramName,
					NewValue:            paramValue,
					Reason:              "Initial policy configuration",
					Severity:            "normal",
					Source:              "policy_file",
					Timestamp:           time.Now().Unix(),
					TTLSeconds:          0, // No expiration for initial config
				}

				// Apply the patch
				err := processor.OnConfigPatch(ctx, patch)
				if err != nil {
					e.logger.Warn("Failed to apply initial policy configuration",
						zap.String("processor", id.String()),
						zap.String("parameter", paramName),
						zap.Error(err))
				} else {
					e.logger.Info("Applied initial policy configuration",
						zap.String("processor", id.String()),
						zap.String("parameter", paramName),
						zap.Any("value", paramValue))
				}
			}
		}
	}

	return nil
}

// External implementations:
// startWatcher from policy.go
// startOpAMPClient from opamp.go

func (e *Extension) pollOpAMPServer(ctx context.Context, client *http.Client) {
	e.sendStatus(ctx, client)

	// Fetch policy
	resp, err := client.Get(e.config.OpAMPConfig.ServerURL + "/policy")
	if err == nil {
		if resp.StatusCode == http.StatusOK {
			data, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err == nil {
				if err := e.loadPolicyBytes(data); err != nil {
					e.logger.Warn("Failed to apply remote policy", zap.Error(err))
				} else {
					e.logger.Info("Applied remote policy")
				}
			}
		} else {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	} else {
		e.logger.Warn("Failed to fetch policy", zap.Error(err))
	}

	// Fetch patch
	resp, err = client.Get(e.config.OpAMPConfig.ServerURL + "/patch")
	if err == nil {
		if resp.StatusCode == http.StatusOK {
			var patch interfaces.ConfigPatch
			if err := json.NewDecoder(resp.Body).Decode(&patch); err == nil {
				resp.Body.Close()
				if err := e.SubmitConfigPatch(ctx, patch); err != nil {
					e.logger.Warn("Failed to apply remote patch", zap.Error(err))
				} else {
					e.logger.Info("Applied remote patch", zap.String("patch_id", patch.PatchID))
				}
			} else {
				resp.Body.Close()
				e.logger.Warn("Failed to decode patch", zap.Error(err))
			}
		} else {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	} else {
		e.logger.Warn("Failed to fetch patch", zap.Error(err))
	}
}

func (e *Extension) sendStatus(ctx context.Context, client *http.Client) {
	status := map[string]any{"safe_mode": e.safeMode}
	body, _ := json.Marshal(status)
	resp, err := client.Post(e.config.OpAMPConfig.ServerURL+"/status", "application/json", bytes.NewReader(body))
	if err != nil {
		e.logger.Warn("Failed to send status", zap.Error(err))
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

// enterSafeMode puts the system into safe mode
func (e *Extension) enterSafeMode() error {
	e.lock.Lock()
	defer e.lock.Unlock()

	if e.safeMode {
		return nil // Already in safe mode
	}

	e.safeMode = true
	e.logger.Warn("Entering safe mode")

	// Apply safe mode configurations to all processors
	ctx := context.Background()
	for id, processor := range e.processors {
		// Extract processor type from ID
		parts := strings.Split(id.String(), "/")
		if len(parts) == 0 {
			continue
		}
		procType := parts[len(parts)-1]

		// Check if this processor has a safe mode config
		if safeConfig, exists := e.config.SafeModeConfigs[procType]; exists {
			if configMap, ok := safeConfig.(map[string]interface{}); ok {
				for paramName, paramValue := range configMap {
					patch := interfaces.ConfigPatch{
						PatchID:             fmt.Sprintf("safe-mode-%s-%s", procType, paramName),
						TargetProcessorName: id,
						ParameterPath:       paramName,
						NewValue:            paramValue,
						Reason:              "Safe mode activated",
						Severity:            "safety",
						Source:              "pic_control",
						Timestamp:           time.Now().Unix(),
						TTLSeconds:          0, // No expiration for safe mode
					}

					// Apply the patch
					err := processor.OnConfigPatch(ctx, patch)
					if err != nil {
						e.logger.Warn("Failed to apply safe mode configuration",
							zap.String("processor", id.String()),
							zap.String("parameter", paramName),
							zap.Error(err))
					} else {
						e.logger.Info("Applied safe mode configuration",
							zap.String("processor", id.String()),
							zap.String("parameter", paramName),
							zap.Any("value", paramValue))
					}
				}
			}
		}
	}

	return nil
}

// exitSafeMode returns the system to normal operation
func (e *Extension) exitSafeMode() error {
	e.lock.Lock()
	defer e.lock.Unlock()

	if !e.safeMode {
		return nil // Not in safe mode
	}

	e.safeMode = false
	e.logger.Info("Exiting safe mode")

	// Reapply policy configuration
	return e.applyPolicyConfig()
}

// buildTLSConfig creates a TLS configuration for the OpAMP client
func (e *Extension) buildTLSConfig() *tls.Config {
	if e.config.OpAMPConfig == nil {
		return nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: e.config.OpAMPConfig.InsecureSkipVerify,
	}

	// For now, we'll provide a simplistic implementation without mutual TLS
	// In a real system, we would:
	// 1. Load client certificates if provided
	// 2. Load CA certificates if provided
	// 3. Set up proper verification functions
	
	// This is a placeholder implementation for testing purposes
	return tlsConfig
}