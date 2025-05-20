// Package metrics provides utilities for metrics emission and collection
package metrics

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// MetricType represents the type of metric
type MetricType int

const (
	// MetricTypeGauge represents a gauge metric
	MetricTypeGauge MetricType = iota
	// MetricTypeCounter represents a counter metric
	MetricTypeCounter
	// MetricTypeHistogram represents a histogram metric
	MetricTypeHistogram
)

// MetricDataPoint represents a single data point for a metric
type MetricDataPoint struct {
	Value            float64
	Timestamp        time.Time
	Attributes       map[string]string
	HistogramBuckets []HistogramBucket
}

// HistogramBucket represents a single bucket in a histogram
type HistogramBucket struct {
	Boundary float64
	Count    uint64
}

// UnifiedMetricsCollector provides a centralized way to collect and emit metrics
// across the Phoenix system. It reduces duplication of metric collection logic
// and provides a standard interface for all components.
type UnifiedMetricsCollector struct {
	metrics      map[string]*MetricDefinition
	emitter      *MetricsEmitter
	logger       *zap.Logger
	mu           sync.RWMutex
	defaultAttrs map[string]string
}

// MetricDefinition represents the definition of a metric
type MetricDefinition struct {
	Name        string
	Description string
	Unit        string
	Type        MetricType
	DataPoints  []MetricDataPoint
}

// MetricsBuilder is a builder for creating metrics
type MetricsBuilder struct {
	metric *MetricDefinition
}

// NewUnifiedMetricsCollector creates a new UnifiedMetricsCollector
func NewUnifiedMetricsCollector(logger *zap.Logger) *UnifiedMetricsCollector {
	return &UnifiedMetricsCollector{
		metrics:      make(map[string]*MetricDefinition),
		emitter:      NewMetricsEmitter("unified_metrics_collector", "collector"),
		logger:       logger,
		defaultAttrs: make(map[string]string),
	}
}

// SetDefaultAttributes sets default attributes to be applied to all metrics
func (c *UnifiedMetricsCollector) SetDefaultAttributes(attrs map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.defaultAttrs = attrs
}

// AddDefaultAttribute adds a default attribute to be applied to all metrics
func (c *UnifiedMetricsCollector) AddDefaultAttribute(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.defaultAttrs[key] = value
}

// WithLogger sets the logger for the metrics collector
func (c *UnifiedMetricsCollector) WithLogger(logger *zap.Logger) *UnifiedMetricsCollector {
	c.logger = logger
	return c
}

// ResetAll clears all metrics
func (c *UnifiedMetricsCollector) ResetAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics = make(map[string]*MetricDefinition)
}

// AddGauge adds or updates a gauge metric
func (c *UnifiedMetricsCollector) AddGauge(name, description, unit string) *MetricsBuilder {
	c.mu.Lock()
	defer c.mu.Unlock()

	metric, exists := c.metrics[name]
	if !exists {
		metric = &MetricDefinition{
			Name:        name,
			Description: description,
			Unit:        unit,
			Type:        MetricTypeGauge,
			DataPoints:  make([]MetricDataPoint, 0),
		}
		c.metrics[name] = metric
	}

	return &MetricsBuilder{metric: metric}
}

// AddCounter adds or updates a counter metric
func (c *UnifiedMetricsCollector) AddCounter(name, description, unit string) *MetricsBuilder {
	c.mu.Lock()
	defer c.mu.Unlock()

	metric, exists := c.metrics[name]
	if !exists {
		metric = &MetricDefinition{
			Name:        name,
			Description: description,
			Unit:        unit,
			Type:        MetricTypeCounter,
			DataPoints:  make([]MetricDataPoint, 0),
		}
		c.metrics[name] = metric
	}

	return &MetricsBuilder{metric: metric}
}

// AddHistogram adds or updates a histogram metric
func (c *UnifiedMetricsCollector) AddHistogram(name, description, unit string) *MetricsBuilder {
	c.mu.Lock()
	defer c.mu.Unlock()

	metric, exists := c.metrics[name]
	if !exists {
		metric = &MetricDefinition{
			Name:        name,
			Description: description,
			Unit:        unit,
			Type:        MetricTypeHistogram,
			DataPoints:  make([]MetricDataPoint, 0),
		}
		c.metrics[name] = metric
	}

	return &MetricsBuilder{metric: metric}
}

// WithValue adds a value data point to the metric
func (b *MetricsBuilder) WithValue(value float64) *MetricsBuilder {
	dataPoint := MetricDataPoint{
		Value:      value,
		Timestamp:  time.Now(),
		Attributes: make(map[string]string),
	}

	b.metric.DataPoints = append(b.metric.DataPoints, dataPoint)
	return b
}

// WithAttributes adds attributes to the most recent data point
func (b *MetricsBuilder) WithAttributes(attrs map[string]string) *MetricsBuilder {
	if len(b.metric.DataPoints) == 0 {
		// Create a data point if none exists
		dataPoint := MetricDataPoint{
			Value:      0,
			Timestamp:  time.Now(),
			Attributes: attrs,
		}
		b.metric.DataPoints = append(b.metric.DataPoints, dataPoint)
		return b
	}

	// Add attributes to the most recent data point
	dp := &b.metric.DataPoints[len(b.metric.DataPoints)-1]
	for k, v := range attrs {
		dp.Attributes[k] = v
	}

	return b
}

// WithHistogramBuckets adds histogram buckets to the most recent data point
func (b *MetricsBuilder) WithHistogramBuckets(buckets []HistogramBucket) *MetricsBuilder {
	if b.metric.Type != MetricTypeHistogram {
		return b
	}

	if len(b.metric.DataPoints) == 0 {
		// Create a data point if none exists
		dataPoint := MetricDataPoint{
			Value:            0,
			Timestamp:        time.Now(),
			Attributes:       make(map[string]string),
			HistogramBuckets: buckets,
		}
		b.metric.DataPoints = append(b.metric.DataPoints, dataPoint)
		return b
	}

	// Add buckets to the most recent data point
	dp := &b.metric.DataPoints[len(b.metric.DataPoints)-1]
	dp.HistogramBuckets = buckets

	return b
}

// WithTimestamp adds a timestamp to the most recent data point
func (b *MetricsBuilder) WithTimestamp(ts time.Time) *MetricsBuilder {
	if len(b.metric.DataPoints) == 0 {
		// Create a data point if none exists
		dataPoint := MetricDataPoint{
			Value:      0,
			Timestamp:  ts,
			Attributes: make(map[string]string),
		}
		b.metric.DataPoints = append(b.metric.DataPoints, dataPoint)
		return b
	}

	// Update timestamp for the most recent data point
	dp := &b.metric.DataPoints[len(b.metric.DataPoints)-1]
	dp.Timestamp = ts

	return b
}

// Emit emits all collected metrics
func (c *UnifiedMetricsCollector) Emit(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Create metrics data
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()

	// Add default resource attributes
	for k, v := range c.defaultAttrs {
		rm.Resource().Attributes().PutStr(k, v)
	}

	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("sa-omf")

	// Add all metrics
	for _, metric := range c.metrics {
		m := sm.Metrics().AppendEmpty()
		m.SetName(metric.Name)
		m.SetDescription(metric.Description)
		m.SetUnit(metric.Unit)

		switch metric.Type {
		case MetricTypeGauge:
			gauge := m.SetEmptyGauge()
			addGaugeDataPoints(gauge, metric.DataPoints)

		case MetricTypeCounter:
			sum := m.SetEmptySum()
			sum.SetIsMonotonic(true)
			sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
			addSumDataPoints(sum, metric.DataPoints)

		case MetricTypeHistogram:
			histogram := m.SetEmptyHistogram()
			histogram.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
			addHistogramDataPoints(histogram, metric.DataPoints)
		}
	}

	// Call the emitter to actually emit the metrics
	if c.emitter != nil {
		return c.emitter.EmitMetrics(ctx, md)
	}

	return nil
}

// addGaugeDataPoints adds data points to a gauge metric
func addGaugeDataPoints(gauge pmetric.Gauge, dataPoints []MetricDataPoint) {
	for _, dp := range dataPoints {
		pdp := gauge.DataPoints().AppendEmpty()
		pdp.SetDoubleValue(dp.Value)
		pdp.SetTimestamp(pcommon.NewTimestampFromTime(dp.Timestamp))

		// Add attributes
		for k, v := range dp.Attributes {
			pdp.Attributes().PutStr(k, v)
		}
	}
}

// addSumDataPoints adds data points to a sum metric
func addSumDataPoints(sum pmetric.Sum, dataPoints []MetricDataPoint) {
	for _, dp := range dataPoints {
		pdp := sum.DataPoints().AppendEmpty()
		pdp.SetDoubleValue(dp.Value)
		pdp.SetTimestamp(pcommon.NewTimestampFromTime(dp.Timestamp))

		// Add attributes
		for k, v := range dp.Attributes {
			pdp.Attributes().PutStr(k, v)
		}
	}
}

// addHistogramDataPoints adds data points to a histogram metric
func addHistogramDataPoints(histogram pmetric.Histogram, dataPoints []MetricDataPoint) {
	for _, dp := range dataPoints {
		pdp := histogram.DataPoints().AppendEmpty()
		pdp.SetCount(uint64(len(dp.HistogramBuckets)))
		pdp.SetTimestamp(pcommon.NewTimestampFromTime(dp.Timestamp))

		// Calculate sum from buckets
		sum := 0.0
		for _, bucket := range dp.HistogramBuckets {
			sum += float64(bucket.Count) * bucket.Boundary
		}
		pdp.SetSum(sum)

		// Add bucket boundaries and counts
		if len(dp.HistogramBuckets) > 0 {
			pdp.ExplicitBounds().EnsureCapacity(len(dp.HistogramBuckets))
			pdp.BucketCounts().EnsureCapacity(len(dp.HistogramBuckets))

			for _, bucket := range dp.HistogramBuckets {
				pdp.ExplicitBounds().Append(bucket.Boundary)
				pdp.BucketCounts().Append(bucket.Count)
			}
		}

		// Add attributes
		for k, v := range dp.Attributes {
			pdp.Attributes().PutStr(k, v)
		}
	}
}

// GetMetric returns a specific metric by name
func (c *UnifiedMetricsCollector) GetMetric(name string) *MetricDefinition {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.metrics[name]
}
