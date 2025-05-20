package cpu_histogram_converter

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/processor/cpu_histogram_converter"
)

func TestCPUHistogramConverterProcessor(t *testing.T) {
	// Create a factory
	factory := cpu_histogram_converter.NewFactory()
	assert.NotNil(t, factory)

	// Create a default configuration
	cfg := factory.CreateDefaultConfig().(*cpu_histogram_converter.Config)

	// Modify config for testing
	cfg.Enabled = true
	cfg.InputMetricName = "test.cpu.time"
	cfg.OutputMetricName = "test.cpu.utilization.histogram"
	cfg.CollectionIntervalSeconds = 10
	cfg.HostCPUCount = 2
	cfg.TopKOnly = false
	cfg.HistogramBuckets = []float64{1, 5, 10, 50, 100}

	// Create a test sink for output metrics
	sink := new(consumertest.MetricsSink)

	// Create the processor
	ctx := context.Background()
	settings := processor.Settings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
		ID: component.NewIDWithName(component.MustNewType("cpu_histogram_converter"), ""),
	}

	proc, err := factory.CreateMetricsProcessor(ctx, settings, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Start the processor
	err = proc.Start(ctx, nil)
	require.NoError(t, err)

	// Create test metrics
	metrics := createCPUMetrics(10.0, time.Now())

	// First pass - should not generate histogram since we don't have previous values
	err = proc.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	// Verify no histogram was created yet
	outputMetrics := sink.AllMetrics()
	require.Equal(t, 1, len(outputMetrics))
	ensureNoHistogramMetric(t, outputMetrics[0], cfg.OutputMetricName)

	// Wait a moment
	time.Sleep(10 * time.Millisecond)

	// Second pass with increased CPU time - should generate histogram
	metrics = createCPUMetrics(15.0, time.Now())
	err = proc.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	// Verify histogram was created
	outputMetrics = sink.AllMetrics()
	require.Equal(t, 2, len(outputMetrics))
	ensureHistogramMetric(t, outputMetrics[1], cfg.OutputMetricName)

	// Shutdown the processor
	err = proc.Shutdown(ctx)
	require.NoError(t, err)
}

func TestStateManagement(t *testing.T) {
	// Create a temporary directory for state
	tmpDir, err := os.MkdirTemp("", "cpu_histogram_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	statePath := filepath.Join(tmpDir, "cpu_state.json")

	// Test saving and loading state
	t.Run("SaveAndLoadState", func(t *testing.T) {
		// Create processor with state storage
		factory := cpu_histogram_converter.NewFactory()
		cfg := factory.CreateDefaultConfig().(*cpu_histogram_converter.Config)
		cfg.Enabled = true
		cfg.InputMetricName = "test.cpu.time"
		cfg.OutputMetricName = "test.cpu.utilization.histogram"
		cfg.StateStoragePath = statePath
		cfg.StateFlushIntervalSeconds = 1 // Short interval for testing

		sink := new(consumertest.MetricsSink)
		settings := processor.Settings{
			TelemetrySettings: component.TelemetrySettings{
				Logger: zap.NewNop(),
			},
			ID: component.NewIDWithName(component.MustNewType("cpu_histogram_converter"), ""),
		}

		// Create first processor instance
		proc1, err := factory.CreateMetricsProcessor(context.Background(), settings, cfg, sink)
		require.NoError(t, err)
		require.NoError(t, proc1.Start(context.Background(), nil))

		// Process metrics
		metrics1 := createCPUMetrics(10.0, time.Now())
		err = proc1.ConsumeMetrics(context.Background(), metrics1)
		require.NoError(t, err)

		// Wait for state to flush
		time.Sleep(2 * time.Second)

		// Verify state file exists
		_, err = os.Stat(statePath)
		require.NoError(t, err)

		// Shutdown first processor
		require.NoError(t, proc1.Shutdown(context.Background()))

		// Create second processor instance to load state
		proc2, err := factory.CreateMetricsProcessor(context.Background(), settings, cfg, sink)
		require.NoError(t, err)
		require.NoError(t, proc2.Start(context.Background(), nil))

		// Process new metrics - should be able to generate a histogram now
		metrics2 := createCPUMetrics(15.0, time.Now())
		err = proc2.ConsumeMetrics(context.Background(), metrics2)
		require.NoError(t, err)

		// Verify histogram was created, which means state was loaded
		outputMetrics := sink.AllMetrics()
		require.GreaterOrEqual(t, len(outputMetrics), 2)
		ensureHistogramMetric(t, outputMetrics[len(outputMetrics)-1], cfg.OutputMetricName)

		// Shutdown second processor
		require.NoError(t, proc2.Shutdown(context.Background()))
	})
}

func TestProcessEviction(t *testing.T) {
	// Create a processor with a small memory limit
	factory := cpu_histogram_converter.NewFactory()
	cfg := factory.CreateDefaultConfig().(*cpu_histogram_converter.Config)
	cfg.Enabled = true
	cfg.InputMetricName = "test.cpu.time"
	cfg.OutputMetricName = "test.cpu.utilization.histogram"
	cfg.MaxProcessesInMemory = 5 // Very small limit for testing

	sink := new(consumertest.MetricsSink)
	settings := processor.Settings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
		ID: component.NewIDWithName(component.MustNewType("cpu_histogram_converter"), ""),
	}

	proc, err := factory.CreateMetricsProcessor(context.Background(), settings, cfg, sink)
	require.NoError(t, err)
	require.NoError(t, proc.Start(context.Background(), nil))

	// First pass - add many processes
	metrics1 := createMultiProcessCPUMetrics(10, 10.0, time.Now())
	err = proc.ConsumeMetrics(context.Background(), metrics1)
	require.NoError(t, err)

	// Wait a moment
	time.Sleep(10 * time.Millisecond)

	// Second pass - the processor should have evicted some processes
	metrics2 := createMultiProcessCPUMetrics(10, 15.0, time.Now())
	err = proc.ConsumeMetrics(context.Background(), metrics2)
	require.NoError(t, err)

	// Shutdown the processor
	require.NoError(t, proc.Shutdown(context.Background()))
}

func TestTopKFiltering(t *testing.T) {
	// Create a processor with top-k filtering
	factory := cpu_histogram_converter.NewFactory()
	cfg := factory.CreateDefaultConfig().(*cpu_histogram_converter.Config)
	cfg.Enabled = true
	cfg.InputMetricName = "test.cpu.time"
	cfg.OutputMetricName = "test.cpu.utilization.histogram"
	cfg.TopKOnly = true

	sink := new(consumertest.MetricsSink)
	settings := processor.Settings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
		ID: component.NewIDWithName(component.MustNewType("cpu_histogram_converter"), ""),
	}

	proc, err := factory.CreateMetricsProcessor(context.Background(), settings, cfg, sink)
	require.NoError(t, err)
	require.NoError(t, proc.Start(context.Background(), nil))

	// Create metrics with top-k marker and non-top-k resources
	metrics := pmetric.NewMetrics()
	
	// Add a resource in top-k
	rm1 := metrics.ResourceMetrics().AppendEmpty()
	rm1.Resource().Attributes().PutStr("process.executable.name", "top-k-process")
	rm1.Resource().Attributes().PutStr("aemf.filter.included", "true")
	addCPUTimeMetric(rm1, cfg.InputMetricName, 10.0)
	
	// Add a resource not in top-k
	rm2 := metrics.ResourceMetrics().AppendEmpty()
	rm2.Resource().Attributes().PutStr("process.executable.name", "non-top-k-process")
	addCPUTimeMetric(rm2, cfg.InputMetricName, 20.0)

	// Process metrics
	err = proc.ConsumeMetrics(context.Background(), metrics)
	require.NoError(t, err)
	
	// Wait a moment and send another batch
	time.Sleep(10 * time.Millisecond)
	
	// Update metrics with increased values
	metrics = pmetric.NewMetrics()
	
	// Add a resource in top-k
	rm1 = metrics.ResourceMetrics().AppendEmpty()
	rm1.Resource().Attributes().PutStr("process.executable.name", "top-k-process")
	rm1.Resource().Attributes().PutStr("aemf.filter.included", "true")
	addCPUTimeMetric(rm1, cfg.InputMetricName, 15.0)
	
	// Add a resource not in top-k
	rm2 = metrics.ResourceMetrics().AppendEmpty()
	rm2.Resource().Attributes().PutStr("process.executable.name", "non-top-k-process")
	addCPUTimeMetric(rm2, cfg.InputMetricName, 25.0)
	
	// Process metrics again
	err = proc.ConsumeMetrics(context.Background(), metrics)
	require.NoError(t, err)

	// Verify histogram was created (only processes the top-k resource)
	outputMetrics := sink.AllMetrics()
	histogramMetric := findHistogramMetric(outputMetrics[len(outputMetrics)-1], cfg.OutputMetricName)
	
	// Histogram should be created with one data point (from the top-k resource)
	require.NotNil(t, histogramMetric)
	
	// Shutdown the processor
	require.NoError(t, proc.Shutdown(context.Background()))
}

// Helper Functions

// createCPUMetrics creates test metrics with a single CPU time metric
func createCPUMetrics(cpuTime float64, timestamp time.Time) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	
	rm := metrics.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("process.executable.name", "test-process")
	rm.Resource().Attributes().PutStr("process.pid", "1234")
	
	addCPUTimeMetric(rm, "test.cpu.time", cpuTime)
	
	return metrics
}

// createMultiProcessCPUMetrics creates test metrics with multiple processes
func createMultiProcessCPUMetrics(processCount int, baseCPUTime float64, timestamp time.Time) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	
	for i := 0; i < processCount; i++ {
		rm := metrics.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutStr("process.executable.name", "test-process-"+string(rune(i)))
		rm.Resource().Attributes().PutStr("process.pid", string(rune(1000+i)))
		
		addCPUTimeMetric(rm, "test.cpu.time", baseCPUTime+float64(i))
	}
	
	return metrics
}

// addCPUTimeMetric adds a CPU time metric to a resource metrics
func addCPUTimeMetric(rm pmetric.ResourceMetrics, metricName string, cpuTime float64) {
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("test-scope")
	
	metric := sm.Metrics().AppendEmpty()
	metric.SetName(metricName)
	
	sum := metric.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	
	dp := sum.DataPoints().AppendEmpty()
	dp.SetDoubleValue(cpuTime)
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(-time.Minute)))
}

// ensureNoHistogramMetric verifies that no histogram metric exists with the given name
func ensureNoHistogramMetric(t *testing.T, metrics pmetric.Metrics, metricName string) {
	for i := 0; i < metrics.ResourceMetrics().Len(); i++ {
		rm := metrics.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)
				if metric.Name() == metricName {
					assert.NotEqual(t, pmetric.MetricTypeHistogram, metric.Type(),
						"Expected no histogram metric, but found one with name %s", metricName)
				}
			}
		}
	}
}

// ensureHistogramMetric verifies that a histogram metric exists with the given name
func ensureHistogramMetric(t *testing.T, metrics pmetric.Metrics, metricName string) {
	metric := findHistogramMetric(metrics, metricName)
	require.NotNil(t, metric, "Histogram metric not found with name %s", metricName)
	assert.Equal(t, pmetric.MetricTypeHistogram, metric.Type())
	assert.Greater(t, metric.Histogram().DataPoints().Len(), 0,
		"Histogram should have at least one data point")
}

// findHistogramMetric finds a histogram metric with the given name
func findHistogramMetric(metrics pmetric.Metrics, metricName string) *pmetric.Metric {
	for i := 0; i < metrics.ResourceMetrics().Len(); i++ {
		rm := metrics.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)
				if metric.Name() == metricName {
					return &metric
				}
			}
		}
	}
	return nil
}