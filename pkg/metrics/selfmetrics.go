// Package metrics provides standard methods for components to emit metrics about themselves.
package metrics

import (
	"fmt"
	"log"
	"time"
)

// MetricsEmitter provides a standardized way to emit self-metrics
type MetricsEmitter struct {
	Name        string
	Component   string
	CommonAttrs map[string]string
}

// NewMetricsEmitter creates a new MetricsEmitter for a component
func NewMetricsEmitter(componentType string, componentName string) *MetricsEmitter {
	return &MetricsEmitter{
		Component: componentType,
		Name:      componentName,
		CommonAttrs: map[string]string{
			"component.type": componentType,
			"component.name": componentName,
		},
	}
}

// RegisterCounter creates and returns a new counter metric
// This is a placeholder method since we've simplified the metrics system
func (e *MetricsEmitter) RegisterCounter(name string, description string) (interface{}, error) {
	log.Printf("Registered counter metric: %s_%s (%s)", e.Component, name, description)
	return nil, nil
}

// RegisterGauge creates and returns a new gauge metric
// This is a placeholder method since we've simplified the metrics system
func (e *MetricsEmitter) RegisterGauge(name string, description string) (interface{}, error) {
	log.Printf("Registered gauge metric: %s_%s (%s)", e.Component, name, description)
	return nil, nil
}

// SetMetricsCollector sets the metrics collector for testing
// This is a placeholder method since we've simplified the metrics system
func (e *MetricsEmitter) SetMetricsCollector(collector interface{}) {
	// No-op in simplified implementation
}

// AddMetrics adds metrics to the collector if available
// This is a placeholder method since we've simplified the metrics system
func (e *MetricsEmitter) AddMetrics(metrics interface{}) {
	// No-op in simplified implementation
}

// CreatePatchMetric creates a simple string representation of a configuration patch
// This is a simplified version that doesn't use OpenTelemetry types
func CreatePatchMetric(patch interface{}) string {
	// Simple string representation for now
	return fmt.Sprintf("Config patch created at %s", time.Now().Format(time.RFC3339))
}
