// Package piccontrolext implements the pic_control extension for the SA-OMF.
package piccontrolext

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.uber.org/zap"

	"github.com/yourorg/sa-omf/internal/interfaces"
	"github.com/yourorg/sa-omf/pkg/metrics"
	"github.com/yourorg/sa-omf/pkg/policy"
)

const (
	typeStr = "pic_control"
)

var (
	errPolicyLoadFailed = errors.New("failed to load policy")
	errProcessorNotFound = errors.New("target processor not found")
	errPatchRateLimited = errors.New("patch rate limited")
	errSafeModeActive = errors.New("safe mode active")
	errPatchExpired = errors.New("patch has expired")
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
	ServerURL          string `mapstructure:"server_url"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
}

// NewFactory creates a factory for the pic_control extension
func NewFactory() extension.Factory {
	return extension.NewFactory(
		component.MustNewType(typeStr),
		createDefaultConfig,
		createExtension,
	)
}

// createDefaultConfig creates the default configuration
func createDefaultConfig() component.Config {
	return &Config{
		PolicyFilePath:      "/etc/sa-omf/policy.yaml",
		MaxPatchesPerMinute: 3,
		PatchCooldownSeconds: 10,
		SafeModeConfigs:     make(map[string]interface{}),
	}
}

// createExtension creates the extension
func createExtension(
	ctx context.Context,
	set extension.CreateSettings,
	cfg component.Config,
) (extension.Extension, error) {
	config := cfg.(*Config)
	
	// Create extension
	return newExtension(config, set.Logger)
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
	
	// Access all processors from the host's GetExtensions method
	pipelines := e.host.GetExtensions()
	for id, proc := range pipelines {
		// Try to cast to UpdateableProcessor
		if updateable, ok := proc.(interfaces.UpdateableProcessor); ok {
			e.processors[id] = updateable
			e.logger.Info("Registered updateable processor", zap.String("id", id.String()))
		}
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
		              time.Duration(e.config.PatchCooldownSeconds)*time.Second - time.Since(e.lastPatchTime)))
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

// applyPolicyConfig applies the processor configurations from the policy
func (e *Extension) applyPolicyConfig() error {
	ctx := context.Background()
	
	// Apply configurations to each processor
	for id, processor := range e.processors {
		// Extract the processor type from the ID
		parts := filepath.SplitList(id.String())
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
	// Placeholder for OpAMP client implementation
	return nil
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
		parts := filepath.SplitList(id.String())
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
