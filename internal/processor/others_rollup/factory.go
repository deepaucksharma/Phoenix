package others_rollup

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

const Type = "others_rollup"

type Config struct {
	Enabled           bool   `mapstructure:"enabled"`
	PriorityThreshold string `mapstructure:"priority_threshold"`
	Strategy          string `mapstructure:"strategy"`
	NamePrefix        string `mapstructure:"name_prefix"`
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
