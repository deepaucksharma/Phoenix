package metrics

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/pmetric"
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
func (c *MetricsCollector) AddMetrics(metrics pmetric.Metrics) {
	// Extract metrics from the pmetric.Metrics object
	for i := 0; i < metrics.ResourceMetrics().Len(); i++ {
		rm := metrics.ResourceMetrics().At(i)
		
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)
				
				// Process based on metric type
				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					for l := 0; l < metric.Gauge().DataPoints().Len(); l++ {
						dp := metric.Gauge().DataPoints().At(l)
						c.addDataPoint(metric.Name(), dp)
					}
				case pmetric.MetricTypeSum:
					for l := 0; l < metric.Sum().DataPoints().Len(); l++ {
						dp := metric.Sum().DataPoints().At(l)
						c.addDataPoint(metric.Name(), dp)
					}
				}
			}
		}
	}
}

// addDataPoint processes a data point and adds it to the collector.
func (c *MetricsCollector) addDataPoint(name string, dp pmetric.NumberDataPoint) {
	var value float64
	
	switch dp.ValueType() {
	case pmetric.NumberDataPointValueTypeDouble:
		value = dp.DoubleValue()
	case pmetric.NumberDataPointValueTypeInt:
		value = float64(dp.IntValue())
	default:
		// Skip other types
		return
	}
	
	c.lock.Lock()
	defer c.lock.Unlock()
	
	c.metrics = append(c.metrics, Metric{
		Name:  name,
		Value: value,
		Time:  dp.Timestamp().AsTime(),
	})
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
func (c *MetricsCollector) EmitMetrics(ctx context.Context) pmetric.Metrics {
	// Do nothing, metrics are collected directly
	return pmetric.NewMetrics()
}