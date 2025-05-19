// Package picconnector implements an exporter that forwards configuration 
// patches from pid_decider to pic_control.
package picconnector

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
)

const (
	// typeStr is the unique identifier for the pic_connector exporter.
	typeStr = "pic_connector"
)

// NewFactory creates a factory for the pic_connector exporter.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		typeStr,
		createDefaultConfig,
		exporter.WithMetrics(createMetricsExporter, component.StabilityLevelDevelopment),
	)
}

// createDefaultConfig creates the default configuration for the exporter.
func createDefaultConfig() component.Config {
	return &Config{}
}

// createMetricsExporter creates a metrics exporter based on the config.
func createMetricsExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
) (exporter.Metrics, error) {
	return newExporter(set)
}