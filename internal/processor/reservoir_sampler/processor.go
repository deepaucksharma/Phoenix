package reservoir_sampler

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/control/pid"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/pkg/util/reservoir"
)

const (
	typeStr = "reservoir_sampler"
)

// processorImpl implements the reservoir_sampler processor.
type processorImpl struct {
	config         *Config
	logger         *zap.Logger
	next           consumer.Metrics
	lock           sync.RWMutex
	sampler        *reservoir.StratifiedReservoirSampler
	reservoirSize  int
	pid            *pid.Controller
	targetCoverage float64
	minSize        int
	maxSize        int
}

var _ processor.Metrics = (*processorImpl)(nil)
var _ interfaces.UpdateableProcessor = (*processorImpl)(nil)

// newProcessor creates a new reservoir_sampler processor.
func newProcessor(cfg *Config, settings processor.Settings, next consumer.Metrics) (*processorImpl, error) {
	p := &processorImpl{
		config:         cfg,
		logger:         settings.TelemetrySettings.Logger,
		next:           next,
		sampler:        reservoir.NewStratifiedReservoirSampler(),
		reservoirSize:  cfg.ReservoirSize,
		targetCoverage: 0.8,
		minSize:        10,
		maxSize:        1000,
	}

	// PID controller to adjust reservoir size around target coverage.
	var err error
	p.pid, err = pid.NewController(50, 1, 0, p.targetCoverage)
	if err != nil {
		return nil, fmt.Errorf("create PID controller: %w", err)
	}
	return p, nil
}

// Start implements the Component interface.
func (p *processorImpl) Start(context.Context, component.Host) error { return nil }

// Shutdown implements the Component interface.
func (p *processorImpl) Shutdown(context.Context) error { return nil }

// Capabilities implements the processor.Metrics interface.
func (p *processorImpl) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// ConsumeMetrics processes incoming metrics and outputs a sampled subset.
func (p *processorImpl) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if !p.config.Enabled {
		return p.next.ConsumeMetrics(ctx, md)
	}

	total := md.ResourceMetrics().Len()
	for i := 0; i < total; i++ {
		rm := md.ResourceMetrics().At(i)
		stratum := "default"
		if val, ok := rm.Resource().Attributes().Get("aemf.process.priority"); ok {
			stratum = val.AsString()
		}
		copyRM := pmetric.NewResourceMetrics()
		rm.CopyTo(copyRM)
		p.sampler.Add(stratum, copyRM, p.reservoirSize)
	}

	samples := p.sampler.GetSamples()
	out := pmetric.NewMetrics()
	count := 0
	for _, rms := range samples {
		for _, item := range rms {
			if rm, ok := item.(pmetric.ResourceMetrics); ok {
				newRM := out.ResourceMetrics().AppendEmpty()
				rm.CopyTo(newRM)
				count++
			}
		}
	}

	coverage := 1.0
	if total > 0 {
		coverage = float64(count) / float64(total)
	}

	delta := p.pid.Compute(coverage)
	if delta != 0 {
		newSize := p.reservoirSize + int(delta)
		if newSize < p.minSize {
			newSize = p.minSize
		}
		if newSize > p.maxSize {
			newSize = p.maxSize
		}
		if newSize != p.reservoirSize {
			p.reservoirSize = newSize
			p.config.ReservoirSize = newSize
			for _, s := range p.sampler.Strata() {
				p.sampler.SetCapacity(s, newSize)
			}
			p.logger.Debug("Adjusted reservoir size", zap.Int("reservoir_size", newSize))
		}
	}

	return p.next.ConsumeMetrics(ctx, out)
}

// OnConfigPatch implements UpdateableProcessor.
func (p *processorImpl) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	switch patch.ParameterPath {
	case "reservoir_size":
		v, ok := patch.NewValue.(int)
		if !ok {
			return fmt.Errorf("invalid value type for reservoir_size: %T", patch.NewValue)
		}
		if v <= 0 {
			return fmt.Errorf("reservoir_size must be > 0")
		}
		p.reservoirSize = v
		p.config.ReservoirSize = v
		for _, s := range p.sampler.Strata() {
			p.sampler.SetCapacity(s, v)
		}
		return nil
	case "enabled":
		b, ok := patch.NewValue.(bool)
		if !ok {
			return fmt.Errorf("invalid value type for enabled: %T", patch.NewValue)
		}
		p.config.Enabled = b
		return nil
	default:
		return fmt.Errorf("unknown parameter: %s", patch.ParameterPath)
	}
}

// GetConfigStatus implements UpdateableProcessor.
func (p *processorImpl) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return interfaces.ConfigStatus{
		Parameters: map[string]any{
			"reservoir_size": p.reservoirSize,
		},
		Enabled: p.config.Enabled,
	}, nil
}
