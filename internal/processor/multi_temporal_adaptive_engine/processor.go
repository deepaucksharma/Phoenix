package multi_temporal_adaptive_engine

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"

	"github.com/deepaucksharma/Phoenix/internal/processor/base"
	"github.com/deepaucksharma/Phoenix/pkg/util/timeseries"
)

type engineProcessor struct {
	*base.BaseProcessor
	cfg *Config
}

func newProcessor(set processor.Settings, cfg *Config, next consumer.Metrics) (*engineProcessor, error) {
	processorType := component.MustNewType(typeStr)
	return &engineProcessor{
		BaseProcessor: base.NewBaseProcessor(set.TelemetrySettings.Logger, next, typeStr, component.NewID(processorType)),
		cfg:           cfg,
	}, nil
}

func (p *engineProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	if !p.cfg.Enabled {
		return p.GetNext().ConsumeMetrics(ctx, md)
	}
	data := []float64{1, 2, 3, 4, 10, 5, 6}
	_ = timeseries.DetectZScore(data, p.cfg.Threshold)
	_ = timeseries.ForecastEMA(data, 0.5, 1)
	return p.GetNext().ConsumeMetrics(ctx, md)
}

var _ processor.Metrics = (*engineProcessor)(nil)
var _ component.Component = (*engineProcessor)(nil)
