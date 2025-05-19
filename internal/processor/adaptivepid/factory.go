// Package adaptivepid implements the pid_decider processor which generates configuration
// patches using PID control loops to maintain KPI targets.
package adaptivepid

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

const (
	// typeStr is the unique identifier for the adaptivepid processor.
	typeStr = "pid_decider"
)

// NewFactory creates a factory for the pid_decider processor.
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
		Controllers: []ControllerConfig{
			{
				Name:               "default",
				Enabled:            false,
				KPIMetricName:      "aemf_impact_adaptive_topk_resource_coverage_percent_avg_1m",
				KPITargetValue:     0.90,
				KP:                 30.0,
				KI:                 5.0,
				KD:                 0.0,
				IntegralWindupLimit: 60.0,
				HysteresisPercent:  3.0,
				OutputConfigPatches: []OutputConfigPatch{},
			},
		},
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