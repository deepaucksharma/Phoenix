// Package base provides the base implementation for SA-OMF processors.
package base

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/config"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
)

// UpdateableProcessor provides a standardized implementation of the UpdateableProcessor interface.
// It extends BaseProcessor with configuration management capabilities.
type UpdateableProcessor struct {
	*BaseProcessor
	configManager *config.Manager
	config        config.EnabledConfig
	name          string
	mutex         sync.RWMutex
}

// NewUpdateableProcessor creates a new UpdateableProcessor with standardized configuration management.
func NewUpdateableProcessor(
	logger *zap.Logger,
	next consumer.Metrics,
	name string,
	id component.ID,
	config config.EnabledConfig,
) *UpdateableProcessor {
	base := NewBaseProcessor(logger, next, name, id)
	
	up := &UpdateableProcessor{
		BaseProcessor: base,
		config:        config,
		name:          name,
	}
	
	// Create the configuration manager
	up.configManager = config.NewManager(logger, up, config)
	
	return up
}

// Start initializes the processor.
func (p *UpdateableProcessor) Start(ctx context.Context, host component.Host) error {
	// Initialize metrics emitter if available
	metricsEmitter := metrics.NewMetricsEmitter()
	
	// Set metrics emitter field in base processor
	p.metricsEmitter = metricsEmitter
	
	return nil
}

// GetName returns the processor name.
func (p *UpdateableProcessor) GetName() string {
	return p.name
}

// OnConfigPatch implements the UpdateableProcessor interface.
func (p *UpdateableProcessor) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	// Use the standardized configuration manager
	return p.configManager.HandleConfigPatch(ctx, patch)
}

// GetConfigStatus implements the UpdateableProcessor interface.
func (p *UpdateableProcessor) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	// Use the standardized configuration manager
	return p.configManager.GetConfigStatus(ctx)
}

// Lock acquires a write lock on the processor.
func (p *UpdateableProcessor) Lock() {
	p.mutex.Lock()
}

// Unlock releases a write lock on the processor.
func (p *UpdateableProcessor) Unlock() {
	p.mutex.Unlock()
}

// RLock acquires a read lock on the processor.
func (p *UpdateableProcessor) RLock() {
	p.mutex.RLock()
}

// RUnlock releases a read lock on the processor.
func (p *UpdateableProcessor) RUnlock() {
	p.mutex.RUnlock()
}

// IsEnabled returns whether the processor is enabled.
func (p *UpdateableProcessor) IsEnabled() bool {
	return p.config.IsEnabled()
}

// GetConfig returns the processor's configuration.
func (p *UpdateableProcessor) GetConfig() interface{} {
	return p.config
}