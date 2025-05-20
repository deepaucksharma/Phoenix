// Package metric_pipeline implements a combined processor that handles resource filtering
// and metric transformation in a single processing step for efficiency.
package metric_pipeline

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"

	"github.com/deepaucksharma/Phoenix/internal/processor/resource_filter"
)

// Type is the registered type for this processor
const Type = "metric_pipeline"

// NewFactory creates a factory for the metric_pipeline processor
func NewFactory() processor.Factory {
	return processor.NewFactory(
		Type,
		createDefaultConfig,
		processor.WithMetrics(createMetricsProcessor, component.StabilityLevelBeta),
	)
}

// createDefaultConfig creates the default configuration for the processor
func createDefaultConfig() component.Config {
	return &Config{
		ResourceFilter: ResourceFilterConfig{
			Enabled:           true,
			FilterStrategy:    resource_filter.StrategyHybrid,
			PriorityAttribute: "aemf.process.priority",
			PriorityRules:     []resource_filter.PriorityRule{},
			TopK: resource_filter.TopKConfig{
				KValue:         20,
				KMin:           10,
				KMax:           40,
				ResourceField:  "process.executable.name",
				CounterField:   "process.cpu.utilization",
				CoverageTarget: 0.95,
			},
			Rollup: resource_filter.RollupConfig{
				Enabled:           true,
				PriorityThreshold: resource_filter.PriorityLow,
				Strategy:          resource_filter.AggregationSum,
				NamePrefix:        "others",
			},
		},
		Transformation: TransformationConfig{
			Histograms: HistogramConfig{
				Enabled:    true,
				MaxBuckets: 10,
				Metrics: map[string]HistogramMetric{
					"process.cpu.time": {
						Boundaries: []float64{0.1, 0.5, 1.0, 5.0, 10.0},
					},
				},
			},
			Attributes: AttributeConfig{
				Actions: []AttributeAction{},
			},
		},
	}
}

// createMetricsProcessor creates a metrics processor based on the config
func createMetricsProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	pCfg := cfg.(*Config)
	proc, err := newProcessor(pCfg, set, nextConsumer)
	if err != nil {
		return nil, err
	}

	return processorhelper.NewMetricsProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		proc.ConsumeMetrics,
		processorhelper.WithStart(proc.Start),
		processorhelper.WithShutdown(proc.Shutdown),
	)
}
