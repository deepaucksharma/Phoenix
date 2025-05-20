package others_rollup

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/base"
)

const (
	// Type constant is defined in factory.go
	lowPriorityValue = "low"
	priorityAttr     = "aemf.process.priority"
)

// processorImpl aggregates metrics for low priority processes.
type processorImpl struct {
	*base.BaseProcessor
	config *Config
}

var _ processor.Metrics = (*processorImpl)(nil)
var _ interfaces.UpdateableProcessor = (*processorImpl)(nil)

// newProcessor creates a new others_rollup processor.
func newProcessor(cfg *Config, settings processor.Settings, next consumer.Metrics) (*processorImpl, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &processorImpl{
		BaseProcessor: base.NewBaseProcessor(settings.TelemetrySettings.Logger, next, typeStr, settings.ID),
		config:        cfg,
	}, nil
}

// Start implements the component.Component interface.
func (p *processorImpl) Start(ctx context.Context, host component.Host) error {
	return p.BaseProcessor.Start(ctx, host)
}

// Shutdown implements the component.Component interface.
func (p *processorImpl) Shutdown(ctx context.Context) error {
	return p.BaseProcessor.Shutdown(ctx)
}

// Capabilities implements the processor.Metrics interface.
func (p *processorImpl) Capabilities() consumer.Capabilities {
	return p.BaseProcessor.Capabilities()
}

// ConsumeMetrics aggregates low priority process metrics.
func (p *processorImpl) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	p.Lock()
	defer p.Unlock()

	if !p.config.Enabled {
		return p.GetNext().ConsumeMetrics(ctx, md)
	}

	out := pmetric.NewMetrics()
	out.ResourceMetrics().EnsureCapacity(md.ResourceMetrics().Len())

	// Aggregation map
	type agg struct {
		sum   float64
		count int
		typ   pmetric.MetricType
	}
	metricsAgg := map[string]*agg{}

	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		attrs := rm.Resource().Attributes()
		if val, ok := attrs.Get(priorityAttr); ok && val.Str() == lowPriorityValue {
			// Aggregate metrics
			for j := 0; j < rm.ScopeMetrics().Len(); j++ {
				sm := rm.ScopeMetrics().At(j)
				for k := 0; k < sm.Metrics().Len(); k++ {
					m := sm.Metrics().At(k)
					var value float64
					switch m.Type() {
					case pmetric.MetricTypeGauge:
						if m.Gauge().DataPoints().Len() == 0 {
							continue
						}
						value = m.Gauge().DataPoints().At(0).DoubleValue()
					case pmetric.MetricTypeSum:
						if m.Sum().DataPoints().Len() == 0 {
							continue
						}
						value = m.Sum().DataPoints().At(0).DoubleValue()
					default:
						continue
					}
					a := metricsAgg[m.Name()]
					if a == nil {
						a = &agg{typ: m.Type()}
						metricsAgg[m.Name()] = a
					}
					a.sum += value
					a.count++
				}
			}
		} else {
			// Keep as is
			newRM := out.ResourceMetrics().AppendEmpty()
			rm.CopyTo(newRM)
		}
	}

	// Build aggregated resource if any metrics aggregated
	if len(metricsAgg) > 0 {
		aggRM := out.ResourceMetrics().AppendEmpty()
		res := aggRM.Resource()
		res.Attributes().PutStr("process.name", "others")
		res.Attributes().PutStr(priorityAttr, lowPriorityValue)

		sm := aggRM.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName(typeStr)

		now := pcommon.NewTimestampFromTime(time.Now())
		for name, a := range metricsAgg {
			m := sm.Metrics().AppendEmpty()
			m.SetName(name)
			switch a.typ {
			case pmetric.MetricTypeSum:
				sum := m.SetEmptySum()
				sum.SetIsMonotonic(true)
				sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
				dp := sum.DataPoints().AppendEmpty()
				val := a.sum
				if p.config.Strategy == "avg" && a.count > 0 {
					val = a.sum / float64(a.count)
				}
				dp.SetDoubleValue(val)
				dp.SetTimestamp(now)
			default:
				gauge := m.SetEmptyGauge()
				dp := gauge.DataPoints().AppendEmpty()
				val := a.sum
				if p.config.Strategy == "avg" && a.count > 0 {
					val = a.sum / float64(a.count)
				}
				dp.SetDoubleValue(val)
				dp.SetTimestamp(now)
			}
		}
	}

	return p.GetNext().ConsumeMetrics(ctx, out)
}

// OnConfigPatch implements UpdateableProcessor.
func (p *processorImpl) OnConfigPatch(ctx context.Context, patch interfaces.ConfigPatch) error {
	p.Lock()
	defer p.Unlock()

	switch patch.ParameterPath {
	case "enabled":
		if v, ok := patch.NewValue.(bool); ok {
			p.config.Enabled = v
		} else {
			return fmt.Errorf("invalid type for enabled")
		}
	case "strategy":
		if s, ok := patch.NewValue.(string); ok {
			p.config.Strategy = s
			if err := p.config.Validate(); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("invalid type for strategy")
		}
	default:
		return fmt.Errorf("unknown parameter %s", patch.ParameterPath)
	}

	return nil
}

// GetName returns the processor name for identification
func (p *processorImpl) GetName() string {
	return "others_rollup"
}

// GetConfigStatus implements UpdateableProcessor.
func (p *processorImpl) GetConfigStatus(ctx context.Context) (interfaces.ConfigStatus, error) {
	p.RLock()
	defer p.RUnlock()

	return interfaces.ConfigStatus{
		Parameters: map[string]any{
			"strategy": p.config.Strategy,
		},
		Enabled: p.config.Enabled,
	}, nil
}
