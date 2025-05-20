// Package pic_control_ext implements the Policy-In-Code Control Extension
// which serves as the central governance layer for configuration changes.
package pic_control_ext

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
)

// picControlExtension implements the Policy-In-Code control extension
type picControlExtension struct {
	processors     map[string]interfaces.UpdateableProcessor
	config         *Config
	lock           sync.RWMutex
	logger         *zap.Logger
	metricsEmitter *metrics.MetricsEmitter
	safeMode       bool
	safetyMonitors []interfaces.SafetyMonitor
	patchHistory   []patchRecord
	rateLimiter    *rateLimiter
}

// patchRecord represents a record of an applied configuration patch
type patchRecord struct {
	Patch     interfaces.ConfigPatch
	Timestamp time.Time
	Success   bool
	Error     string
}

// rateLimiter implements a simple rate limiter for patch applications
type rateLimiter struct {
	windowSize       time.Duration
	maxChanges       int
	processorWindows map[string][]time.Time
	lock             sync.Mutex
}

// newRateLimiter creates a new rate limiter with the specified window size and max changes
func newRateLimiter(windowSize time.Duration, maxChanges int) *rateLimiter {
	return &rateLimiter{
		windowSize:       windowSize,
		maxChanges:       maxChanges,
		processorWindows: make(map[string][]time.Time),
	}
}

// checkLimit returns true if a change is allowed for the given processor
func (r *rateLimiter) checkLimit(processorID string) bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.windowSize)

	// Get processor's history
	history, exists := r.processorWindows[processorID]
	if !exists {
		// First change for this processor
		r.processorWindows[processorID] = []time.Time{now}
		return true
	}

	// Filter out old entries
	filtered := make([]time.Time, 0, len(history))
	for _, t := range history {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}

	// Check against limit
	if len(filtered) < r.maxChanges {
		// Under limit, allow change
		r.processorWindows[processorID] = append(filtered, now)
		return true
	}

	// At limit, don't allow change
	r.processorWindows[processorID] = filtered
	return false
}

// Ensure our extension implements the required interfaces
var _ extension.Extension = (*picControlExtension)(nil)
var _ interfaces.PicControl = (*picControlExtension)(nil)

// newPicControlExtension creates a new pic_control_ext extension
func newPicControlExtension(config *Config, settings component.TelemetrySettings) (*picControlExtension, error) {
	ext := &picControlExtension{
		processors:     make(map[string]interfaces.UpdateableProcessor),
		config:         config,
		logger:         settings.Logger,
		metricsEmitter: metrics.NewMetricsEmitter("pic_control_ext", nil),
		safeMode:       false,
		safetyMonitors: make([]interfaces.SafetyMonitor, 0),
		patchHistory:   make([]patchRecord, 0, config.HistorySize),
		rateLimiter:    newRateLimiter(time.Duration(config.RateLimitWindowSeconds)*time.Second, config.MaxChangesPerWindow),
	}

	return ext, nil
}

// Start implements the component.Component interface
func (p *picControlExtension) Start(ctx context.Context, host component.Host) error {
	p.logger.Info("Starting pic_control_ext extension")
	return nil
}

// Shutdown implements the component.Component interface
func (p *picControlExtension) Shutdown(ctx context.Context) error {
	p.logger.Info("Shutting down pic_control_ext extension")
	return nil
}

// RegisterUpdateableProcessor implements the interfaces.PicControl interface
func (p *picControlExtension) RegisterUpdateableProcessor(processor interfaces.UpdateableProcessor) error {
	if processor == nil {
		return fmt.Errorf("cannot register nil processor")
	}

	name := processor.GetName()
	if name == "" {
		return fmt.Errorf("processor must have a name")
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	p.processors[name] = processor
	p.logger.Info("Registered updateable processor", zap.String("name", name))
	return nil
}

// SubmitConfigPatch receives configuration patches from adaptive controllers
func (p *picControlExtension) SubmitConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	return p.ApplyConfigPatch(ctx, patch)
}

// ApplyConfigPatch implements the interfaces.PicControl interface
func (p *picControlExtension) ApplyConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	// Check if we're in safe mode
	if p.IsInSafeMode() && !patch.SafetyOverride {
		return fmt.Errorf("system is in safe mode, patch rejected (use safety override to force)")
	}

	// Get target processor name as string
	targetName := patch.TargetProcessorName.String()

	// Check rate limits
	if !p.rateLimiter.checkLimit(targetName) {
		return fmt.Errorf("rate limit exceeded for processor %s", targetName)
	}

	// Find the target processor
	p.lock.RLock()
	processor, exists := p.processors[targetName]
	p.lock.RUnlock()

	if !exists {
		return fmt.Errorf("target processor not found: %s", targetName)
	}

	// Create a record of this patch
	record := patchRecord{
		Patch:     patch,
		Timestamp: time.Now(),
		Success:   false,
	}

	// Apply the patch
	err := processor.OnConfigPatch(ctx, patch)
	if err != nil {
		record.Error = err.Error()
		p.logPatchRecord(record)
		return fmt.Errorf("failed to apply patch to processor %s: %w", targetName, err)
	}

	// Mark as successful
	record.Success = true
	p.logPatchRecord(record)

	return nil
}

// IsInSafeMode implements the interfaces.PicControl interface
func (p *picControlExtension) IsInSafeMode() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	// If explicitly in safe mode, return true
	if p.safeMode {
		return true
	}

	// Check safety monitors
	for _, monitor := range p.safetyMonitors {
		if monitor.IsInSafeMode() {
			return true
		}
	}

	return false
}

// SetSafeMode implements the interfaces.PicControl interface
func (p *picControlExtension) SetSafeMode(safeMode bool) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.safeMode = safeMode
	if safeMode {
		p.logger.Warn("System entering safe mode")
	} else {
		p.logger.Info("System exiting safe mode")
	}
}

// RegisterSafetyMonitor implements the interfaces.PicControl interface
func (p *picControlExtension) RegisterSafetyMonitor(monitor interfaces.SafetyMonitor) {
	if monitor == nil {
		return
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	p.safetyMonitors = append(p.safetyMonitors, monitor)
}

// logPatchRecord adds a patch record to the history
func (p *picControlExtension) logPatchRecord(record patchRecord) {
	p.lock.Lock()
	defer p.lock.Unlock()

	// Add to history
	p.patchHistory = append(p.patchHistory, record)

	// Trim history if needed
	if len(p.patchHistory) > p.config.HistorySize {
		p.patchHistory = p.patchHistory[len(p.patchHistory)-p.config.HistorySize:]
	}

	// Log the patch
	if record.Success {
		p.logger.Info("Applied configuration patch",
			zap.String("target", record.Patch.TargetProcessorName.String()),
			zap.String("parameter", record.Patch.ParameterPath),
			zap.Any("value", record.Patch.NewValue),
			zap.String("reason", record.Patch.Reason))
	} else {
		p.logger.Warn("Failed to apply configuration patch",
			zap.String("target", record.Patch.TargetProcessorName.String()),
			zap.String("parameter", record.Patch.ParameterPath),
			zap.Any("value", record.Patch.NewValue),
			zap.String("reason", record.Patch.Reason),
			zap.String("error", record.Error))
	}
}

// GetPatchHistory returns the recent patch history
func (p *picControlExtension) GetPatchHistory() []patchRecord {
	p.lock.RLock()
	defer p.lock.RUnlock()

	// Return a copy of the history
	history := make([]patchRecord, len(p.patchHistory))
	copy(history, p.patchHistory)

	return history
}
