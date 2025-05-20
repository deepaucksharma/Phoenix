// Package pic_control_ext implements the pic_control extension for the SA-OMF.
package pic_control_ext

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

	// Host may implement an additional interface exposing processors
	type processorGetter interface {
		GetProcessors() map[component.ID]component.Component
	}

	getter, ok := e.host.(processorGetter)
	if !ok {
		return fmt.Errorf("host does not expose processors")
	}

	processors := getter.GetProcessors()
	if len(processors) == 0 {
		e.logger.Warn("no processors found on host")
	}

	for id, comp := range processors {
		proc, ok := comp.(interfaces.UpdateableProcessor)
		if !ok {
			e.logger.Debug("processor does not implement UpdateableProcessor", zap.String("id", id.String()))
			continue
		}

		e.processors[id] = proc
		e.logger.Info("Registered updateable processor", zap.String("id", id.String()))
	}

	if len(e.processors) == 0 {
		return fmt.Errorf("no UpdateableProcessor components discovered")
	}

	return nil
}

// SubmitConfigPatch processes a ConfigPatch
func (e *Extension) SubmitConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	// Check if safe mode is active
	if e.safeMode {
		e.logger.Warn("Patch rejected: Safe mode active",
			zap.String("patch_id", patch.PatchID),
			zap.String("target", patch.TargetProcessorName.String()))
		return errSafeModeActive
	}

	// Check patch TTL
	if patch.Timestamp > 0 && patch.TTLSeconds > 0 {
		expirationTime := time.Unix(patch.Timestamp, 0).Add(time.Duration(patch.TTLSeconds) * time.Second)
		if time.Now().After(expirationTime) {
			e.logger.Warn("Patch rejected: Expired",
				zap.String("patch_id", patch.PatchID),
				zap.Time("expiration", expirationTime))
			return errPatchExpired
		}
	}

	// Check rate limiting
	if !e.checkRateLimit() {
		e.logger.Warn("Patch rejected: Rate limited",
			zap.String("patch_id", patch.PatchID),
			zap.String("target", patch.TargetProcessorName.String()))
		return errPatchRateLimited
	}

	// Check cooldown
	if !e.lastPatchTime.IsZero() &&
		time.Since(e.lastPatchTime) < time.Duration(e.config.PatchCooldownSeconds)*time.Second {
		e.logger.Warn("Patch rejected: Cooldown active",
			zap.String("patch_id", patch.PatchID),
			zap.Duration("remaining",
				time.Duration(e.config.PatchCooldownSeconds)*time.Second-time.Since(e.lastPatchTime)))
		return errPatchRateLimited
	}

	// Get the target processor
	processor, exists := e.processors[patch.TargetProcessorName]
	if !exists {
		e.logger.Warn("Patch rejected: Target processor not found",
			zap.String("patch_id", patch.PatchID),
			zap.String("target", patch.TargetProcessorName.String()))
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

// checkRateLimit checks if the patch should be rate limited
func (e *Extension) checkRateLimit() bool {
	// Reset counter if minute boundary passed
	if time.Since(e.patchCountReset) > time.Minute {
		e.patchCount = 0
		e.patchCountReset = time.Now()
	}

	return e.patchCount < e.config.MaxPatchesPerMinute
}

// loadPolicy loads the policy from a file
func (e *Extension) loadPolicy(filename string) error {
	newPolicy, err := policy.LoadPolicy(filename)
	if err != nil {
		return err
	}

	e.policy = newPolicy

	// Apply initial processor configurations from policy
	return e.applyPolicyConfig()
}

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

// startWatcher starts the policy file watcher
func (e *Extension) startWatcher() error {
	var err error
	e.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// Watch the directory, not the file
	dir := filepath.Dir(e.config.PolicyFilePath)
	err = e.watcher.Add(dir)
	if err != nil {
		return err
	}

	// Start watcher goroutine
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelWatcher = cancel

	go func() {
		defer e.watcher.Close()

		for {
			select {
			case event, ok := <-e.watcher.Events:
				if !ok {
					return
				}

				// Check if this is for our policy file
				if event.Name == e.config.PolicyFilePath {
					if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
						e.logger.Info("Policy file changed, reloading")

						// Wait a brief moment for the file to be completely written
						time.Sleep(100 * time.Millisecond)

						if err := e.loadPolicy(e.config.PolicyFilePath); err != nil {
							e.logger.Error("Failed to reload policy file", zap.Error(err))
						}
					}
				}

			case err, ok := <-e.watcher.Errors:
				if !ok {
					return
				}
				e.logger.Error("Policy watcher error", zap.Error(err))

			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// startOpAMPClient starts the OpAMP client
func (e *Extension) startOpAMPClient(ctx context.Context) error {
	cfg := e.config.OpAMPConfig
	if cfg == nil {
		return nil
	}

	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: cfg.InsecureSkipVerify}

	if cfg.CACertFile != "" {
		caData, err := os.ReadFile(cfg.CACertFile)
		if err != nil {
			return fmt.Errorf("read CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caData)
		tlsCfg.RootCAs = pool
	}

	if cfg.ClientCertFile != "" && cfg.ClientKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.ClientCertFile, cfg.ClientKeyFile)
		if err != nil {
			return fmt.Errorf("load client cert: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	client := &http.Client{Transport: &http.Transport{TLSClientConfig: tlsCfg}}

	go func() {
		ticker := time.NewTicker(time.Duration(cfg.PollIntervalSeconds) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				e.pollOpAMPServer(ctx, client)
			}
		}
	}()

	return nil
}

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
