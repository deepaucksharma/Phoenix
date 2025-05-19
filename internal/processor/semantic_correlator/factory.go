package semantic_correlator

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

const typeStr = "semantic_correlator"

// NewFactory returns a new processor factory.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		typeStr,
		createDefaultConfig,
		processor.WithMetrics(createProcessor, component.StabilityLevelDevelopment),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		Enabled: true,
		Method:  "granger",
		Lag:     1,
		Bins:    5,
	}
}

func createProcessor(_ context.Context, set processor.CreateSettings, cfg component.Config, next consumer.Metrics) (processor.Metrics, error) {
	return newProcessor(set, cfg.(*Config), next)
}
