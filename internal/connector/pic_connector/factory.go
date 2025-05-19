// Package pic_connector implements an exporter that connects the pid_decider processor to the pic_control extension.
package pic_connector

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
)

// NewFactory creates a factory for the pic_connector exporter
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		component.MustNewType(typeStr),
		createDefaultConfig,
		exporter.WithMetrics(createExporter, component.StabilityLevelDevelopment),
	)
}

// createDefaultConfig creates the default configuration for the exporter
func createDefaultConfig() component.Config {
	return &Config{}
}

// createExporter creates a metrics exporter based on the config
func createExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Metrics, error) {
	return newExporter(cfg, set)
}