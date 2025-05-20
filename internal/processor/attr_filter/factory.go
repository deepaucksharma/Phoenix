package attr_filter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"

	"github.com/deepaucksharma/Phoenix/internal/processor/base"
)

// Type is the processor type string.
const Type = "attr_filter"

func createDefaultConfig() component.Config {
	return &Config{BaseConfig: *base.WithEnabled(true)}
}

// NewFactory returns a new Factory for the attr_filter processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		Type,
		createDefaultConfig,
		processor.WithMetrics(createMetricsProcessor, component.StabilityLevelAlpha),
	)
}

func createMetricsProcessor(ctx context.Context, set processor.CreateSettings, cfg component.Config, next consumer.Metrics) (processor.Metrics, error) {
	pCfg := cfg.(*Config)
	process := func(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
		if !pCfg.Enabled {
			return md, nil
		}
		rms := md.ResourceMetrics()
		for i := 0; i < rms.Len(); i++ {
			attrs := rms.At(i).Resource().Attributes()
			for _, key := range pCfg.Attributes {
				attrs.Remove(key)
			}
		}
		return md, nil
	}
	return processorhelper.NewMetricsProcessor(ctx, set, cfg, next, process)
}
