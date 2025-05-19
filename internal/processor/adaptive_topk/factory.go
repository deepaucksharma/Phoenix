// Package adaptive_topk implements a processor that dynamically selects top-k resources 
// based on self-tuning mechanisms.
package adaptive_topk

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	
	"github.com/deepaucksharma/Phoenix/internal/processor/base"
)

// typeStr is the unique identifier for the adaptive_topk processor.
const typeStr = "adaptive_topk"

// NewFactory creates a factory for the adaptive_topk processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		component.MustNewType(typeStr),
		createDefaultConfig,
		processor.WithMetrics(createMetricsProcessor, component.StabilityLevelDevelopment),
	)
}

// createDefaultConfig creates the default configuration for the processor.
func createDefaultConfig() component.Config {
	return &Config{
		BaseConfig:     base.WithEnabled(true),
		KValue:         30,
		KMin:           10,
		KMax:           60,
		ResourceField:  "process.name",
		CounterField:   "process.cpu_seconds_total",
	}
}

// createMetricsProcessor creates a metrics processor based on the config.
func createMetricsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	pCfg := cfg.(*Config)
	return newProcessor(pCfg, set.TelemetrySettings, nextConsumer, set.ID)
}