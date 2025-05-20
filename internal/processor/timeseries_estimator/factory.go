package timeseries_estimator

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

// The value of "type" key in configuration.
const typeStr = "timeseries_estimator"

// Factory is the factory for the timeseries_estimator processor.
type Factory struct {
}

// NewFactory creates a new Factory.
func NewFactory() processor.Factory {
	return &Factory{}
}

// Type returns the type of the processor.
func (f *Factory) Type() component.Type {
	return component.MustNewType(typeStr)
}

// CreateDefaultConfig creates the default configuration for the processor.
func (f *Factory) CreateDefaultConfig() component.Config {
	return createDefaultConfig()
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *Factory) CreateMetricsProcessor(
	ctx context.Context,
	settings processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	processorConfig := cfg.(*Config)
	
	if !processorConfig.Enabled {
		return processorhelper.NewMetricsProcessor(
			ctx,
			settings,
			cfg,
			nextConsumer,
			func(context.Context, consumer.Metrics) (processor.Metrics, error) {
				return &nopProcessor{nextConsumer: nextConsumer}, nil
			},
		)
	}

	proc, err := newProcessor(processorConfig, settings, nextConsumer)
	if err != nil {
		return nil, err
	}

	return proc, nil
}

// nopProcessor is a no-op processor that just forwards metrics.
type nopProcessor struct {
	nextConsumer consumer.Metrics
}

// ConsumeMetrics implements the consumer.Metrics interface.
func (p *nopProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	return p.nextConsumer.ConsumeMetrics(ctx, md)
}

// Capabilities implements the processor.Metrics interface.
func (p *nopProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// Start implements the component.Component interface.
func (p *nopProcessor) Start(ctx context.Context, host component.Host) error {
	return nil
}

// Shutdown implements the component.Component interface.
func (p *nopProcessor) Shutdown(ctx context.Context) error {
	return nil
}