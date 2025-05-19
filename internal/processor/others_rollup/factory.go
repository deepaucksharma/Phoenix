package others_rollup

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

const typeStr = "others_rollup"

// NewFactory creates a factory for the others_rollup processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		component.MustNewType(typeStr),
		createDefaultConfig,
		processor.WithMetrics(createMetricsProcessor, component.StabilityLevelDevelopment),
	)
}

// createDefaultConfig creates default configuration for the processor.
func createDefaultConfig() component.Config {
	return &Config{
		Strategy: "sum",
		Enabled:  false,
	}
}

// createMetricsProcessor creates the processor instance.
func createMetricsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	next consumer.Metrics,
) (processor.Metrics, error) {
	pCfg := cfg.(*Config)
	return newProcessor(pCfg, set, next)
}
