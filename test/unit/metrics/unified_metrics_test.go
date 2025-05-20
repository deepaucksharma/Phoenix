package metrics_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/deepaucksharma/Phoenix/pkg/metrics"
)

func TestUnifiedMetricsCollector_Creation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	collector := metrics.NewUnifiedMetricsCollector(logger)
	assert.NotNil(t, collector)
}

func TestUnifiedMetricsCollector_DefaultAttributes(t *testing.T) {
	logger := zaptest.NewLogger(t)
	collector := metrics.NewUnifiedMetricsCollector(logger)
	
	// Set default attributes
	collector.SetDefaultAttributes(map[string]string{
		"service.name": "test-service",
		"host.name":    "test-host",
	})
	
	// Add a default attribute
	collector.AddDefaultAttribute("collector.version", "1.0.0")
	
	// Create a metric
	collector.AddGauge("test.metric", "A test metric", "count").
		WithValue(42.0)
	
	// Emit metrics (in a real scenario, this would be captured by the
	// metrics emitter, but for testing we're not capturing the actual metrics)
	err := collector.Emit(context.Background())
	assert.NoError(t, err)
	
	// Get the metric back to verify it exists (note: we can't directly verify
	// the default attributes were applied as we don't have access to the emitted
	// metrics in this test)
	metric := collector.GetMetric("test.metric")
	assert.NotNil(t, metric)
	assert.Equal(t, "test.metric", metric.Name)
	assert.Equal(t, "A test metric", metric.Description)
	assert.Equal(t, "count", metric.Unit)
}

func TestUnifiedMetricsCollector_GaugeMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	collector := metrics.NewUnifiedMetricsCollector(logger)
	
	// Add a gauge metric with a value
	collector.AddGauge("test.gauge", "A test gauge", "count").
		WithValue(42.0).
		WithAttributes(map[string]string{
			"dim1": "value1",
			"dim2": "value2",
		})
	
	// Verify the gauge metric was created correctly
	metric := collector.GetMetric("test.gauge")
	assert.NotNil(t, metric)
	assert.Equal(t, metrics.MetricTypeGauge, metric.Type)
	assert.Equal(t, "test.gauge", metric.Name)
	assert.Equal(t, "A test gauge", metric.Description)
	assert.Equal(t, "count", metric.Unit)
	
	// Verify data points
	assert.Equal(t, 1, len(metric.DataPoints))
	dp := metric.DataPoints[0]
	assert.Equal(t, 42.0, dp.Value)
	
	// Verify attributes
	assert.Equal(t, 2, len(dp.Attributes))
	assert.Equal(t, "value1", dp.Attributes["dim1"])
	assert.Equal(t, "value2", dp.Attributes["dim2"])
}

func TestUnifiedMetricsCollector_CounterMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	collector := metrics.NewUnifiedMetricsCollector(logger)
	
	// Add a counter metric with a value
	collector.AddCounter("test.counter", "A test counter", "operations").
		WithValue(10.0).
		WithAttributes(map[string]string{
			"operation": "read",
		})
	
	// Verify the counter metric was created correctly
	metric := collector.GetMetric("test.counter")
	assert.NotNil(t, metric)
	assert.Equal(t, metrics.MetricTypeCounter, metric.Type)
	assert.Equal(t, "test.counter", metric.Name)
	assert.Equal(t, "A test counter", metric.Description)
	assert.Equal(t, "operations", metric.Unit)
	
	// Verify data points
	assert.Equal(t, 1, len(metric.DataPoints))
	dp := metric.DataPoints[0]
	assert.Equal(t, 10.0, dp.Value)
	
	// Verify attributes
	assert.Equal(t, 1, len(dp.Attributes))
	assert.Equal(t, "read", dp.Attributes["operation"])
}

func TestUnifiedMetricsCollector_HistogramMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	collector := metrics.NewUnifiedMetricsCollector(logger)
	
	// Create histogram buckets
	buckets := []metrics.HistogramBucket{
		{Boundary: 10.0, Count: 5},
		{Boundary: 20.0, Count: 10},
		{Boundary: 50.0, Count: 3},
		{Boundary: 100.0, Count: 1},
	}
	
	// Add a histogram metric
	collector.AddHistogram("test.histogram", "A test histogram", "ms").
		WithValue(0). // Value doesn't matter for histograms
		WithHistogramBuckets(buckets).
		WithAttributes(map[string]string{
			"component": "processor",
		})
	
	// Verify the histogram metric was created correctly
	metric := collector.GetMetric("test.histogram")
	assert.NotNil(t, metric)
	assert.Equal(t, metrics.MetricTypeHistogram, metric.Type)
	assert.Equal(t, "test.histogram", metric.Name)
	assert.Equal(t, "A test histogram", metric.Description)
	assert.Equal(t, "ms", metric.Unit)
	
	// Verify data points
	assert.Equal(t, 1, len(metric.DataPoints))
	dp := metric.DataPoints[0]
	
	// Verify histogram buckets
	assert.Equal(t, 4, len(dp.HistogramBuckets))
	assert.Equal(t, 10.0, dp.HistogramBuckets[0].Boundary)
	assert.Equal(t, uint64(5), dp.HistogramBuckets[0].Count)
	assert.Equal(t, 20.0, dp.HistogramBuckets[1].Boundary)
	assert.Equal(t, uint64(10), dp.HistogramBuckets[1].Count)
	
	// Verify attributes
	assert.Equal(t, 1, len(dp.Attributes))
	assert.Equal(t, "processor", dp.Attributes["component"])
}

func TestUnifiedMetricsCollector_MultipleDataPoints(t *testing.T) {
	logger := zaptest.NewLogger(t)
	collector := metrics.NewUnifiedMetricsCollector(logger)
	
	// Add a gauge metric with multiple data points
	gauge := collector.AddGauge("test.multi_gauge", "A gauge with multiple points", "count")
	
	// Add first data point
	gauge.WithValue(10.0).WithAttributes(map[string]string{"instance": "instance1"})
	
	// Add second data point
	gauge.WithValue(20.0).WithAttributes(map[string]string{"instance": "instance2"})
	
	// Add third data point
	gauge.WithValue(30.0).WithAttributes(map[string]string{"instance": "instance3"})
	
	// Verify the gauge metric was created correctly
	metric := collector.GetMetric("test.multi_gauge")
	assert.NotNil(t, metric)
	assert.Equal(t, metrics.MetricTypeGauge, metric.Type)
	
	// Verify data points
	assert.Equal(t, 3, len(metric.DataPoints))
	
	// Verify first data point
	assert.Equal(t, 10.0, metric.DataPoints[0].Value)
	assert.Equal(t, "instance1", metric.DataPoints[0].Attributes["instance"])
	
	// Verify second data point
	assert.Equal(t, 20.0, metric.DataPoints[1].Value)
	assert.Equal(t, "instance2", metric.DataPoints[1].Attributes["instance"])
	
	// Verify third data point
	assert.Equal(t, 30.0, metric.DataPoints[2].Value)
	assert.Equal(t, "instance3", metric.DataPoints[2].Attributes["instance"])
}

func TestUnifiedMetricsCollector_WithTimestamp(t *testing.T) {
	logger := zaptest.NewLogger(t)
	collector := metrics.NewUnifiedMetricsCollector(logger)
	
	// Create a specific timestamp
	ts := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	
	// Add a gauge metric with a specific timestamp
	collector.AddGauge("test.ts_gauge", "A gauge with timestamp", "count").
		WithValue(42.0).
		WithTimestamp(ts)
	
	// Verify the gauge metric was created correctly
	metric := collector.GetMetric("test.ts_gauge")
	assert.NotNil(t, metric)
	
	// Verify data point
	assert.Equal(t, 1, len(metric.DataPoints))
	dp := metric.DataPoints[0]
	assert.Equal(t, 42.0, dp.Value)
	
	// Verify timestamp
	assert.Equal(t, ts, dp.Timestamp)
}

func TestUnifiedMetricsCollector_ResetAll(t *testing.T) {
	logger := zaptest.NewLogger(t)
	collector := metrics.NewUnifiedMetricsCollector(logger)
	
	// Add several metrics
	collector.AddGauge("test.gauge", "A test gauge", "count").WithValue(42.0)
	collector.AddCounter("test.counter", "A test counter", "ops").WithValue(10.0)
	
	// Verify metrics were created
	assert.NotNil(t, collector.GetMetric("test.gauge"))
	assert.NotNil(t, collector.GetMetric("test.counter"))
	
	// Reset all metrics
	collector.ResetAll()
	
	// Verify metrics were reset
	assert.Nil(t, collector.GetMetric("test.gauge"))
	assert.Nil(t, collector.GetMetric("test.counter"))
}

func TestUnifiedMetricsCollector_FluentAPI(t *testing.T) {
	logger := zaptest.NewLogger(t)
	collector := metrics.NewUnifiedMetricsCollector(logger)
	
	// Test the fluent API chaining
	now := time.Now()
	collector.AddGauge("test.fluent", "Testing fluent API", "ms").
		WithValue(123.45).
		WithAttributes(map[string]string{"test": "fluent"}).
		WithTimestamp(now)
	
	// Verify the metric was created correctly
	metric := collector.GetMetric("test.fluent")
	assert.NotNil(t, metric)
	assert.Equal(t, 1, len(metric.DataPoints))
	
	dp := metric.DataPoints[0]
	assert.Equal(t, 123.45, dp.Value)
	assert.Equal(t, "fluent", dp.Attributes["test"])
	assert.Equal(t, now, dp.Timestamp)
}

func TestUnifiedMetricsCollector_WithLogger(t *testing.T) {
	logger := zaptest.NewLogger(t)
	collector := metrics.NewUnifiedMetricsCollector(nil) // Start with no logger
	
	// Should not cause issues when no logger is present
	collector.AddGauge("test.gauge", "Test gauge", "count").WithValue(1.0)
	
	// Update with logger
	newLogger := zaptest.NewLogger(t, zaptest.Level(zap.InfoLevel))
	collector.WithLogger(newLogger)
	
	// Continue using the collector with the new logger
	collector.AddGauge("test.gauge2", "Another test gauge", "count").WithValue(2.0)
	
	// Verify metrics were created
	assert.NotNil(t, collector.GetMetric("test.gauge"))
	assert.NotNil(t, collector.GetMetric("test.gauge2"))
}

func BenchmarkUnifiedMetricsCollector_AddGauge(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	collector := metrics.NewUnifiedMetricsCollector(logger)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.AddGauge("test.gauge", "A test gauge", "count").
			WithValue(float64(i)).
			WithAttributes(map[string]string{
				"dim1": "value1",
				"dim2": "value2",
			})
	}
}

func BenchmarkUnifiedMetricsCollector_Emit(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	collector := metrics.NewUnifiedMetricsCollector(logger)
	
	// Add some test metrics
	for i := 0; i < 100; i++ {
		collector.AddGauge(
			"test.gauge", 
			"A test gauge", 
			"count",
		).WithValue(float64(i)).WithAttributes(map[string]string{
			"instance": "instance-1",
		})
	}
	
	// Benchmark emission
	b.ResetTimer()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		_ = collector.Emit(ctx)
	}
}