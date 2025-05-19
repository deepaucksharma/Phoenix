// Package main provides the entry point for the SA-OMF OpenTelemetry Collector.
package main

import (
	"fmt"
	"log"
	"os"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"

	// Import your components here
	"github.com/yourorg/sa-omf/internal/extension/piccontrolext"
	// "github.com/yourorg/sa-omf/internal/connector/picconnector"
	"github.com/yourorg/sa-omf/internal/processor/prioritytagger"
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
	var err error
	factories := otelcol.Factories{}

	// Extensions
	extensions := []extension.Factory{
		piccontrolext.NewFactory(),
		// Add more extensions as needed
	}
	factories.Extensions, err = extension.MakeFactoryMap(extensions...)
	if err != nil {
		return otelcol.Factories{}, fmt.Errorf("failed to create extension factories: %w", err)
	}

	// Receivers
	// Use standard receivers from contrib packages
	factories.Receivers, err = receiver.MakeFactoryMap(
		// Add receivers as needed
	)
	if err != nil {
		return otelcol.Factories{}, fmt.Errorf("failed to create receiver factories: %w", err)
	}

	// Processors
	processors := []processor.Factory{
		// Add custom processors as they are implemented:
		prioritytagger.NewFactory(),
		// adaptivepid.NewFactory(),
		// etc.
	}
	factories.Processors, err = processor.MakeFactoryMap(processors...)
	if err != nil {
		return otelcol.Factories{}, fmt.Errorf("failed to create processor factories: %w", err)
	}

	// Exporters
	exporters := []exporter.Factory{
		// Add custom exporters as they are implemented:
		// picconnector.NewFactory(),
		// etc.
	}
	factories.Exporters, err = exporter.MakeFactoryMap(exporters...)
	if err != nil {
		return otelcol.Factories{}, fmt.Errorf("failed to create exporter factories: %w", err)
	}

	return factories, nil
}

func run(factories otelcol.Factories, info component.BuildInfo) error {
	params := otelcol.CollectorSettings{
		BuildInfo:  info,
		Factories:  factories,
		ConfigFile: getConfigFile(),
	}
	
	col, err := otelcol.NewCollector(params)
	if err != nil {
		return fmt.Errorf("failed to create collector: %w", err)
	}

	return col.Run()
}

func getConfigFile() string {
	// Check if config file is provided as a command-line argument
	if len(os.Args) > 1 && os.Args[1] == "--config" && len(os.Args) > 2 {
		return os.Args[2]
	}
	
	// Default config file location
	return "config/config.yaml"
}
