package histogram_aggregator

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"

	"github.com/deepaucksharma/Phoenix/internal/processor/base"
)

const (
	// The value of "type" key in configuration.
	typeStr = "histogram_aggregator"
)

// NewFactory creates a new factory for the histogram aggregation processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		typeStr,
		createDefaultConfig,
		processor.WithMetrics(createMetricsProcessor, component.StabilityLevelBeta),
	)
}

// createDefaultConfig creates the default configuration for the processor.
func createDefaultConfig() component.Config {
	return &Config{
		BaseConfig: base.NewBaseConfig(),
		MaxBuckets: 10, // Default to 10 buckets max
	}
}

// createMetricsProcessor creates a metrics processor based on the config.
func createMetricsProcessor(
	ctx context.Context,
	params processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	pCfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("could not cast configuration to %s", typeStr)
	}

	histProcessor, err := newProcessor(pCfg, params, nextConsumer)
	if err != nil {
		return nil, err
	}

	return histProcessor, nil
}