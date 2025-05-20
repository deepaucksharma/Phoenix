package timeseries_estimator

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"

	"github.com/deepaucksharma/Phoenix/internal/processor/base"
)

const (
	// Type is the configuration type key for this processor.
	Type = "timeseries_estimator"
)

// NewFactory returns a new processor factory.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		Type,
		createDefaultConfig,
		processor.WithMetrics(createProcessor, component.StabilityLevelAlpha),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		BaseConfig:          base.NewBaseConfig(),
		EstimatorType:       "exact",
		MaxUniqueTimeSeries: 5000,
	}
}

func createProcessor(ctx context.Context, settings processor.Settings, cfg component.Config, next consumer.Metrics) (processor.Metrics, error) {
	pcfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type for %s", Type)
	}
	return newProcessor(pcfg, settings, next)
}
