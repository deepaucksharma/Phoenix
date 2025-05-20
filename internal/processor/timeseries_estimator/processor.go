package timeseries_estimator

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
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

	var count int64
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				switch m.Type() {
				case pmetric.MetricTypeGauge:
					count += int64(m.Gauge().DataPoints().Len())
				case pmetric.MetricTypeSum:
					count += int64(m.Sum().DataPoints().Len())
				case pmetric.MetricTypeHistogram:
					count += int64(m.Histogram().DataPoints().Len())
				case pmetric.MetricTypeSummary:
					count += int64(m.Summary().DataPoints().Len())
				}
			}
		}
	}

	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("phoenix.timeseries.estimate")
	dp := metric.SetEmptyGauge().DataPoints().AppendEmpty()
	dp.SetIntValue(count)

	return p.GetNext().ConsumeMetrics(ctx, md)
}
