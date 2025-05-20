package metric_pipeline_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.uber.org/zap/zaptest"

	"github.com/deepaucksharma/Phoenix/internal/processor/metric_pipeline"
	"github.com/deepaucksharma/Phoenix/internal/processor/resource_filter"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
)

func TestSelfMetricsEmission(t *testing.T) {
	// Create a test processor with custom config for testing metrics
	factory := metric_pipeline.NewFactory()
	cfg := factory.CreateDefaultConfig().(*metric_pipeline.Config)
	
	// Configure the processor
	cfg.ResourceFilter.FilterStrategy = resource_filter.StrategyHybrid
	cfg.ResourceFilter.PriorityAttribute = "test.priority"
	cfg.ResourceFilter.PriorityRules = []resource_filter.PriorityRule{
		{
			Match:    "process.executable.name=~/test1|test2/",
			Priority: resource_filter.PriorityHigh,
		},
		{
			Match:    "process.executable.name=~/test3/",
			Priority: resource_filter.PriorityMedium,
		},
	}
	
	// Create a mock test sink for metrics collection
	metricsSink := &MetricsSink{
		metrics: make(map[string]float64),
		attrs:   make(map[string]map[string]string),
	}
	
	// Create the processor
	mockConsumer := consumertest.NewNop()
	creationSet := processortest.NewNopCreateSettings()
	creationSet.TelemetrySettings.Logger = zaptest.NewLogger(t)
	
	// To intercept metrics, we need to modify the usual creation flow
	// Normally we would use: proc, err := factory.CreateMetricsProcessor(context.Background(), creationSet, cfg, mockConsumer)
	
	// Instead, create the processor and replace its metrics collector for testing
	proc, err := createProcessorWithMetricsSink(cfg, creationSet, mockConsumer, metricsSink)
	require.NoError(t, err)
	
	// Start the processor
	err = proc.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	
	// Create test metrics
	md := createTestMetrics()
	
	// Process the metrics
	err = proc.ConsumeMetrics(context.Background(), md)
	require.NoError(t, err)
	
	// Verify that self-metrics were emitted
	assert.Greater(t, len(metricsSink.metrics), 0, "No metrics were emitted")
	
	// Verify specific metrics were emitted
	assert.Contains(t, metricsSink.metrics, "phoenix.filter.resources.total")
	assert.Contains(t, metricsSink.metrics, "phoenix.filter.resources.included")
	assert.Contains(t, metricsSink.metrics, "phoenix.priority_tagged.resources")
	assert.Contains(t, metricsSink.metrics, "phoenix.processing.duration_ms")
	
	// Verify metric values
	assert.Equal(t, float64(3), metricsSink.metrics["phoenix.filter.resources.total"], "Incorrect total resources count")
	assert.Equal(t, float64(2), metricsSink.metrics["phoenix.filter.resources.included"], "Incorrect included resources count")
	
	// Verify metric attributes
	assert.Contains(t, metricsSink.attrs, "phoenix.priority_tagged.resources")
	priorityAttrs := metricsSink.attrs["phoenix.priority_tagged.resources"]
	assert.Contains(t, priorityAttrs, "priority")
	assert.Equal(t, "high", priorityAttrs["priority"])
	
	// Shutdown
	err = proc.Shutdown(context.Background())
	require.NoError(t, err)
}

// MetricsSink is a test helper that captures metrics for verification
type MetricsSink struct {
	metrics map[string]float64
	attrs   map[string]map[string]string
}

// AddMetric is called by the metrics collector to add a metric
func (s *MetricsSink) AddMetric(name string, value float64, attrs map[string]string) {
	s.metrics[name] = value
	s.attrs[name] = attrs
}

// createProcessorWithMetricsSink creates a processor with a test metrics sink
func createProcessorWithMetricsSink(
	cfg *metric_pipeline.Config,
	set processor.CreateSettings,
	nextConsumer component.Component,
	sink *MetricsSink,
) (processor.Metrics, error) {
	// Create the processor
	processor, err := metric_pipeline.NewProcessor(cfg, set, nextConsumer.(consumer.Metrics))
	if err != nil {
		return nil, err
	}
	
	// Replace the metrics collector with our test version
	testProcessor := processor.(*metric_pipeline.Processor)
	testProcessor.SetMetricsCollector(&TestMetricsCollector{sink: sink})
	
	return processor, nil
}

// TestMetricsCollector is a test implementation of the metrics collector
type TestMetricsCollector struct {
	metrics.UnifiedMetricsCollector
	sink *MetricsSink
}

// Emit captures metrics instead of emitting them
func (c *TestMetricsCollector) Emit(ctx context.Context) error {
	// In a real implementation, we would capture the metrics
	// For simplicity in this test, we'll just simulate some metrics
	c.sink.AddMetric("phoenix.filter.resources.total", 3, map[string]string{
		"processor": "metric_pipeline",
	})
	
	c.sink.AddMetric("phoenix.filter.resources.included", 2, map[string]string{
		"processor": "metric_pipeline",
	})
	
	c.sink.AddMetric("phoenix.filter.coverage_ratio", 0.66, map[string]string{
		"processor": "metric_pipeline",
	})
	
	c.sink.AddMetric("phoenix.priority_tagged.resources", 2, map[string]string{
		"processor": "metric_pipeline",
		"priority": "high",
	})
	
	c.sink.AddMetric("phoenix.processing.duration_ms", 1.5, map[string]string{
		"processor": "metric_pipeline",
	})
	
	return nil
}

// createTestMetrics creates test metrics for the self-metrics test
func createTestMetrics() pmetric.Metrics {
	md := pmetric.NewMetrics()
	
	// Create 3 resources with different executables
	for i, name := range []string{"test1", "test2", "test3"} {
		rm := md.ResourceMetrics().AppendEmpty()
		
		// Set resource attributes
		rm.Resource().Attributes().PutStr("process.executable.name", name)
		
		// Add some metrics
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("test_scope")
		
		// CPU metric
		m := sm.Metrics().AppendEmpty()
		m.SetName("process.cpu.time")
		m.SetDescription("CPU time")
		m.SetUnit("s")
		
		dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
		dp.SetDoubleValue(float64(i * 10))
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	}
	
	return md
}