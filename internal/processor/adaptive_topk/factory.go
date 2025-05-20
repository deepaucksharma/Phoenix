package adaptive_topk

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

const Type = "adaptive_topk"

type Config struct {
	KValue         int     `mapstructure:"k_value"`
	KMin           int     `mapstructure:"k_min"`
	KMax           int     `mapstructure:"k_max"`
	ResourceField  string  `mapstructure:"resource_field"`
	CounterField   string  `mapstructure:"counter_field"`
	CoverageTarget float64 `mapstructure:"coverage_target"`
}

func createDefaultConfig() component.Config { return &Config{} }

func NewFactory() processor.Factory {
	return processor.NewFactory(
		Type,
		createDefaultConfig,
		processor.WithMetrics(createMetricsProcessor, component.StabilityLevelAlpha),
	)
}

func createMetricsProcessor(ctx context.Context, set processor.CreateSettings, cfg component.Config, next consumer.Metrics) (processor.Metrics, error) {
	process := func(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) { return md, nil }
	return processorhelper.NewMetricsProcessor(ctx, set, cfg, next, process)
}
