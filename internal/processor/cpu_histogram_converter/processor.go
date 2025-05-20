package cpu_histogram_converter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"

	"github.com/deepaucksharma/Phoenix/internal/processor/base"
)

type processorImpl struct {
	*base.BaseProcessor
	config *Config
}

var _ processor.Metrics = (*processorImpl)(nil)

func newProcessor(cfg *Config, set processor.CreateSettings, next consumer.Metrics) (*processorImpl, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &processorImpl{
		BaseProcessor: base.NewBaseProcessor(set.TelemetrySettings.Logger, next, Type, set.ID),
		config:        cfg,
	}, nil
}

func (p *processorImpl) Start(ctx context.Context, host component.Host) error {
	return p.BaseProcessor.Start(ctx, host)
}

func (p *processorImpl) Shutdown(ctx context.Context) error {
	return p.BaseProcessor.Shutdown(ctx)
}

func (p *processorImpl) Capabilities() consumer.Capabilities {
	return p.BaseProcessor.Capabilities()
}

func (p *processorImpl) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	p.Lock()
	defer p.Unlock()

	if !p.config.Enabled {
		return p.GetNext().ConsumeMetrics(ctx, md)
	}

	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			metrics := sm.Metrics()
			for k := 0; k < metrics.Len(); k++ {
				m := metrics.At(k)
				if !p.shouldConvert(m.Name()) {
					continue
				}

				// Prepare histogram metric
				histMetric := sm.Metrics().AppendEmpty()
				histMetric.SetName(m.Name())
				h := histMetric.SetEmptyHistogram()
				h.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
				h.ExplicitBounds().FromRaw(p.config.Boundaries)

				switch m.Type() {
				case pmetric.MetricTypeGauge:
					dps := m.Gauge().DataPoints()
					for l := 0; l < dps.Len(); l++ {
						src := dps.At(l)
						dp := h.DataPoints().AppendEmpty()
						p.convertDataPoint(src.DoubleValue(), src.Timestamp(), src.Attributes(), dp)
					}
				case pmetric.MetricTypeSum:
					dps := m.Sum().DataPoints()
					for l := 0; l < dps.Len(); l++ {
						src := dps.At(l)
						dp := h.DataPoints().AppendEmpty()
						p.convertDataPoint(src.DoubleValue(), src.Timestamp(), src.Attributes(), dp)
					}
				default:
					continue
				}

				// remove original metric
				metrics.RemoveIf(func(x pmetric.Metric) bool { return x.Name() == m.Name() })
				// restart scanning since slice changed
				k = -1
			}
		}
	}

	return p.GetNext().ConsumeMetrics(ctx, md)
}

func (p *processorImpl) convertDataPoint(val float64, ts pcommon.Timestamp, attrs pcommon.Map, dp pmetric.HistogramDataPoint) {
	dp.SetTimestamp(ts)
	attrs.CopyTo(dp.Attributes())
	dp.SetCount(1)
	dp.SetSum(val)
	counts := make([]uint64, len(p.config.Boundaries)+1)
	idx := p.bucketIndex(val)
	counts[idx] = 1
	dp.BucketCounts().FromRaw(counts)
}

func (p *processorImpl) shouldConvert(name string) bool {
	for _, n := range p.config.MetricNames {
		if n == name {
			return true
		}
	}
	return false
}

func (p *processorImpl) bucketIndex(v float64) int {
	for i, b := range p.config.Boundaries {
		if v <= b {
			return i
		}
	}
	return len(p.config.Boundaries)
}
