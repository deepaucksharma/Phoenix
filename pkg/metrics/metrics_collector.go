package metrics

import (
	"context"
	"sync"
	"time"
)

// Metric represents a simplified metric for testing.
type Metric struct {
	Name  string
	Value float64
	Time  time.Time
	// Add more fields as needed (labels, etc.)
}

// MetricsCollector collects metrics for testing.
type MetricsCollector struct {
	metrics []Metric
	lock    sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make([]Metric, 0),
	}
}

// AddMetric adds a metric to the collector.
func (c *MetricsCollector) AddMetric(name string, value float64) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.metrics = append(c.metrics, Metric{
		Name:  name,
		Value: value,
		Time:  time.Now(),
	})
}

// AddMetrics adds multiple metrics to the collector.
// This is a simplified version that accepts an interface for flexibility
func (c *MetricsCollector) AddMetrics(metrics interface{}) {
	// In a real implementation, we would process the metrics
	// For now, just add a placeholder metric
	c.AddMetric("metrics_added", 1.0)
}

// GetMetrics returns all collected metrics.
func (c *MetricsCollector) GetMetrics() []Metric {
	c.lock.RLock()
	defer c.lock.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]Metric, len(c.metrics))
	copy(result, c.metrics)
	return result
}

// GetMetricsByName returns metrics with the given name.
func (c *MetricsCollector) GetMetricsByName(name string) []Metric {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var result []Metric
	for _, m := range c.metrics {
		if m.Name == name {
			result = append(result, m)
		}
	}
	return result
}

// Clear clears all collected metrics.
func (c *MetricsCollector) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.metrics = make([]Metric, 0)
}

// EmitMetrics is a compatibility function for MetricsEmitter.
func (c *MetricsCollector) EmitMetrics(ctx context.Context) interface{} {
	// Do nothing, metrics are collected directly
	return nil
}
