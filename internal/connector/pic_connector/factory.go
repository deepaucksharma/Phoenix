package pic_connector

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

const Type = "pic_connector"

type Config struct{}

func createDefaultConfig() component.Config { return &Config{} }

func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		Type,
		createDefaultConfig,
		exporter.WithMetrics(createMetricsExporter, component.StabilityLevelAlpha),
	)
}

func createMetricsExporter(ctx context.Context, set exporter.CreateSettings, cfg component.Config) (exporter.Metrics, error) {
	consume := func(ctx context.Context, md pmetric.Metrics) error { return nil }
	return exporterhelper.NewMetricsExporter(ctx, set, cfg, consume)
}
