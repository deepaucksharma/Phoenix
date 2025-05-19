// Package adaptive_topk implements a processor that dynamically selects top-k resources 
// based on self-tuning mechanisms.
package adaptive_topk

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

const (
	// typeStr is the unique identifier for the adaptive_topk processor.
	typeStr = "adaptive_topk"
)

// NewFactory creates a factory for the adaptive_topk processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		typeStr,
		createDefaultConfig,
		processor.WithMetrics(createMetricsProcessor, component.StabilityLevelDevelopment),
	)
}

// createDefaultConfig creates the default configuration for the processor.
func createDefaultConfig() component.Config {
	return &Config{
		KValue:         30,
		KMin:           10,
		KMax:           60,
		ResourceField:  "process.name",
		CounterField:   "process.cpu_seconds_total",
		Enabled:        true,
	}
}

// createMetricsProcessor creates a metrics processor based on the config.
func createMetricsProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	pCfg := cfg.(*Config)
	return newProcessor(pCfg, set, nextConsumer)
}