// Package adaptive_pid implements a processor that uses PID control for adaptive configuration.
package adaptive_pid

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

// processorType is the unique identifier for the adaptive_pid processor.
var processorType = component.MustNewType(typeStr)

// NewFactory creates a factory for the adaptive_pid processor
func NewFactory() processor.Factory {
	return processor.NewFactory(
		processorType,
		createDefaultConfig,
		processor.WithMetrics(createMetricsProcessor, component.StabilityLevelDevelopment),
	)
}

// createDefaultConfig creates the default configuration for the processor
func createDefaultConfig() component.Config {
	return &Config{
		Controllers: []ControllerConfig{
			{
				Name:              "coverage_controller",
				Enabled:           true,
				KPIMetricName:     "aemf_impact_adaptive_topk_resource_coverage_percent_avg_1m",
				KPITargetValue:    0.90,
				KP:                30,
				KI:                5,
				KD:                0,
				HysteresisPercent: 3,
				UseBayesian:       true,
				StallThreshold:    3,
				OutputConfigPatches: []OutputConfigPatch{
					{
						TargetProcessorName: "adaptive_topk",
						ParameterPath:       "k_value",
						ChangeScaleFactor:   -20,
						MinValue:            10,
						MaxValue:            60,
					},
				},
			},
			{
				Name:              "cardinality_controller",
				Enabled:           true,
				KPIMetricName:     "aemf_impact_cardinality_reduction_ratio",
				KPITargetValue:    0.80,
				KP:                10,
				KI:                2,
				KD:                0,
				HysteresisPercent: 2,
				OutputConfigPatches: []OutputConfigPatch{
					{
						TargetProcessorName: "cardinality_guardian",
						ParameterPath:       "max_unique",
						ChangeScaleFactor:   -100,
						MinValue:            100,
						MaxValue:            10000,
					},
				},
			},
		},
	}
}

// createMetricsProcessor creates a processor based on the config
func createMetricsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	pCfg := cfg.(*Config)
	return newProcessor(pCfg, set.TelemetrySettings, set.ID)
}
