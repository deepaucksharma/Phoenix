// Package main provides the entry point for the SA-OMF OpenTelemetry Collector.
package main

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"

	// Import your components here
	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_pid"
	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_topk"
	"github.com/deepaucksharma/Phoenix/internal/processor/priority_tagger"
	"github.com/deepaucksharma/Phoenix/internal/processor/reservoir_sampler"
	// Add more component imports as they are implemented
)

const (
	serviceName = "sa-omf-otelcol"
	version     = "0.1.0-dev"
)

func main() {
	factories, err := components()
	if err != nil {
		log.Fatalf("Failed to build components: %v", err)
	}

	info := component.BuildInfo{
		Command:     serviceName,
		Description: "Self-Aware OpenTelemetry Metrics Fabric Collector",
		Version:     version,
	}

	if err = run(factories, info); err != nil {
		log.Fatal(err)
	}
}

func components() (otelcol.Factories, error) {
	factories := otelcol.Factories{}

	// Extensions
	extensions := []extension.Factory{
		// Add more extensions as needed
	}
	factories.Extensions = make(map[component.Type]extension.Factory)
	for _, ext := range extensions {
		factories.Extensions[ext.Type()] = ext
	}

	// Receivers
	// Use standard receivers from contrib packages
	factories.Receivers = make(map[component.Type]receiver.Factory)
	// Add receivers as needed

	// Processors
	processors := []processor.Factory{
		// Add custom processors as they are implemented:
		priority_tagger.NewFactory(),
		adaptive_pid.NewFactory(),
		adaptive_topk.NewFactory(),
		reservoir_sampler.NewFactory(),
		// etc.
	}
	factories.Processors = make(map[component.Type]processor.Factory)
	for _, proc := range processors {
		factories.Processors[proc.Type()] = proc
	}

	// Exporters
	exporters := []exporter.Factory{
		// Add custom exporters as they are implemented:
		// etc.
	}
	factories.Exporters = make(map[component.Type]exporter.Factory)
	for _, exp := range exporters {
		factories.Exporters[exp.Type()] = exp
	}

	return factories, nil
}

func run(factories otelcol.Factories, info component.BuildInfo) error {
	params := otelcol.CollectorSettings{
		BuildInfo: info,
		Factories: func() (otelcol.Factories, error) {
			return factories, nil
		},
		// Config provider settings will use defaults and command-line flags
	}

	col, err := otelcol.NewCollector(params)
	if err != nil {
		return fmt.Errorf("failed to create collector: %w", err)
	}

	return col.Run(context.Background())
}

// Note: Config file is now handled by Collector flags
// The default config location is still configs/default/config.yaml
