package multi_temporal_adaptive_engine

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

const typeStr = "multi_temporal_adaptive_engine"

// NewFactory creates the processor factory.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		typeStr,
		createDefaultConfig,
		processor.WithMetrics(createProcessor, component.StabilityLevelDevelopment),
	)
}

func createDefaultConfig() component.Config {
	return &Config{Enabled: true, Threshold: 3}
}

func createProcessor(_ context.Context, set processor.CreateSettings, cfg component.Config, next consumer.Metrics) (processor.Metrics, error) {
	return newProcessor(set, cfg.(*Config), next)
}
