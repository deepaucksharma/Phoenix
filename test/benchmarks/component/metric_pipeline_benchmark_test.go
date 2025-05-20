// Package component contains benchmark tests for Phoenix components
package component

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/internal/processor/metric_pipeline"
	"github.com/deepaucksharma/Phoenix/internal/processor/resource_filter"
)

// BenchmarkMetricPipeline_ResourceFiltering benchmarks the resource filtering functionality
func BenchmarkMetricPipeline_ResourceFiltering(b *testing.B) {
	// Create test metrics with varying number of processes
	benchCases := []struct {
		name          string
		processCount  int
		metricPerProc int
	}{
		{"Small_10Proc_5Metrics", 10, 5},
		{"Medium_100Proc_5Metrics", 100, 5},
		{"Large_1000Proc_5Metrics", 1000, 5},
		{"ExtraLarge_5000Proc_5Metrics", 5000, 5},
		{"ManyMetrics_100Proc_50Metrics", 100, 50},
	}

	for _, bc := range benchCases {
		b.Run(bc.name, func(b *testing.B) {
			// Generate test metrics once
			metrics := generateBenchmarkMetrics(bc.processCount, bc.metricPerProc)
			
			// Create a new processor for each benchmark run
			factory := metric_pipeline.NewFactory()
			cfg := createTestConfig()
			
			// Fresh consumer for each test
			sink := new(consumertest.MetricsSink)
			
			settings := processor.Settings{
				TelemetrySettings: component.TelemetrySettings{
					Logger: zap.NewNop(),
				},
				ID: component.NewIDWithName(component.MustNewType("metric_pipeline"), "benchmark"),
			}
			
			proc, err := factory.CreateMetrics(context.Background(), settings, cfg, sink)
			require.NoError(b, err)
			
			err = proc.Start(context.Background(), nil)
			require.NoError(b, err)
			
			// Reset benchmark timer
			b.ResetTimer()
			
			// Run the benchmark
			for i := 0; i < b.N; i++ {
				err := proc.ConsumeMetrics(context.Background(), metrics)
				if err != nil {
					b.Fatal(err)
				}
				
				// Clear sink between iterations
				sink.Reset()
			}
			
			// Stop the timer before cleanup
			b.StopTimer()
			
			// Cleanup
			err = proc.Shutdown(context.Background())
			require.NoError(b, err)
		})
	}
}

// BenchmarkMetricPipeline_HistogramGeneration benchmarks the histogram generation functionality
func BenchmarkMetricPipeline_HistogramGeneration(b *testing.B) {
	// Create test metrics with varying number of metrics to convert to histograms
	benchCases := []struct {
		name              string
		processCount      int
		histogramMetrics  int
		histogramBuckets  int
	}{
		{"Small_10Proc_1Hist_5Buckets", 10, 1, 5},
		{"Medium_100Proc_1Hist_5Buckets", 100, 1, 5},
		{"Large_1000Proc_1Hist_5Buckets", 1000, 1, 5},
		{"Small_10Proc_5Hist_5Buckets", 10, 5, 5},
		{"Medium_100Proc_5Hist_5Buckets", 100, 5, 5},
		{"Small_10Proc_1Hist_20Buckets", 10, 1, 20},
		{"Medium_100Proc_1Hist_20Buckets", 100, 1, 20},
	}

	for _, bc := range benchCases {
		b.Run(bc.name, func(b *testing.B) {
			// Generate test metrics once
			metrics := generateHistogramBenchmarkMetrics(bc.processCount, bc.histogramMetrics)
			
			// Create a new processor for each benchmark run
			factory := metric_pipeline.NewFactory()
			cfg := createHistogramTestConfig(bc.histogramMetrics, bc.histogramBuckets)
			
			// Fresh consumer for each test
			sink := new(consumertest.MetricsSink)
			
			settings := processor.Settings{
				TelemetrySettings: component.TelemetrySettings{
					Logger: zap.NewNop(),
				},
				ID: component.NewIDWithName(component.MustNewType("metric_pipeline"), "benchmark"),
			}
			
			proc, err := factory.CreateMetrics(context.Background(), settings, cfg, sink)
			require.NoError(b, err)
			
			err = proc.Start(context.Background(), nil)
			require.NoError(b, err)
			
			// Reset benchmark timer
			b.ResetTimer()
			
			// Run the benchmark
			for i := 0; i < b.N; i++ {
				err := proc.ConsumeMetrics(context.Background(), metrics)
				if err != nil {
					b.Fatal(err)
				}
				
				// Clear sink between iterations
				sink.Reset()
			}
			
			// Stop the timer before cleanup
			b.StopTimer()
			
			// Cleanup
			err = proc.Shutdown(context.Background())
			require.NoError(b, err)
		})
	}
}

// BenchmarkMetricPipeline_CompleteProcessing benchmarks the complete processing pipeline
func BenchmarkMetricPipeline_CompleteProcessing(b *testing.B) {
	// Create test cases with varying process counts and strategies
	benchCases := []struct {
		name            string
		processCount    int
		filterStrategy  resource_filter.FilterStrategy
		enableHistogram bool
		enableRollup    bool
	}{
		{"Small_10Proc_Priority", 10, resource_filter.StrategyPriority, false, false},
		{"Medium_100Proc_Priority", 100, resource_filter.StrategyPriority, false, false},
		{"Large_1000Proc_Priority", 1000, resource_filter.StrategyPriority, false, false},
		
		{"Small_10Proc_TopK", 10, resource_filter.StrategyTopK, false, false},
		{"Medium_100Proc_TopK", 100, resource_filter.StrategyTopK, false, false},
		{"Large_1000Proc_TopK", 1000, resource_filter.StrategyTopK, false, false},
		
		{"Small_10Proc_Hybrid", 10, resource_filter.StrategyHybrid, false, false},
		{"Medium_100Proc_Hybrid", 100, resource_filter.StrategyHybrid, false, false},
		{"Large_1000Proc_Hybrid", 1000, resource_filter.StrategyHybrid, false, false},
		
		{"Medium_100Proc_Hybrid_Histogram", 100, resource_filter.StrategyHybrid, true, false},
		{"Medium_100Proc_Hybrid_Rollup", 100, resource_filter.StrategyHybrid, false, true},
		{"Medium_100Proc_Hybrid_Complete", 100, resource_filter.StrategyHybrid, true, true},
		
		{"Large_1000Proc_Hybrid_Complete", 1000, resource_filter.StrategyHybrid, true, true},
	}

	for _, bc := range benchCases {
		b.Run(bc.name, func(b *testing.B) {
			// Generate test metrics once
			metrics := generateCompleteBenchmarkMetrics(bc.processCount)
			
			// Create a new processor for each benchmark run
			factory := metric_pipeline.NewFactory()
			cfg := createCompleteTestConfig(bc.filterStrategy, bc.enableHistogram, bc.enableRollup)
			
			// Fresh consumer for each test
			sink := new(consumertest.MetricsSink)
			
			settings := processor.Settings{
				TelemetrySettings: component.TelemetrySettings{
					Logger: zap.NewNop(),
				},
				ID: component.NewIDWithName(component.MustNewType("metric_pipeline"), "benchmark"),
			}
			
			proc, err := factory.CreateMetrics(context.Background(), settings, cfg, sink)
			require.NoError(b, err)
			
			err = proc.Start(context.Background(), nil)
			require.NoError(b, err)
			
			// Reset benchmark timer
			b.ResetTimer()
			
			// Run the benchmark
			for i := 0; i < b.N; i++ {
				err := proc.ConsumeMetrics(context.Background(), metrics)
				if err != nil {
					b.Fatal(err)
				}
				
				// Clear sink between iterations
				sink.Reset()
			}
			
			// Stop the timer before cleanup
			b.StopTimer()
			
			// Cleanup
			err = proc.Shutdown(context.Background())
			require.NoError(b, err)
		})
	}
}

// BenchmarkMetricPipeline_Comparison benchmarks the consolidated metric_pipeline 
// and compares it to simulating the separate processor approach
func BenchmarkMetricPipeline_Comparison(b *testing.B) {
	// For this test, we'll simulate the separate processors vs. the unified approach
	benchCases := []struct {
		name         string
		processCount int
	}{
		{"Small_10Proc", 10},
		{"Medium_100Proc", 100},
		{"Large_1000Proc", 1000},
	}

	for _, bc := range benchCases {
		// Create test metrics once
		metrics := generateCompleteBenchmarkMetrics(bc.processCount)
		
		// Benchmark the unified metric_pipeline
		b.Run(fmt.Sprintf("%s_Unified", bc.name), func(b *testing.B) {
			// Create the unified processor
			factory := metric_pipeline.NewFactory()
			cfg := createCompleteTestConfig(resource_filter.StrategyHybrid, true, true)
			
			sink := new(consumertest.MetricsSink)
			settings := processor.Settings{
				TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
				ID: component.NewIDWithName(component.MustNewType("metric_pipeline"), "benchmark"),
			}
			
			proc, err := factory.CreateMetrics(context.Background(), settings, cfg, sink)
			require.NoError(b, err)
			
			err = proc.Start(context.Background(), nil)
			require.NoError(b, err)
			
			// Benchmark
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := proc.ConsumeMetrics(context.Background(), metrics)
				if err != nil {
					b.Fatal(err)
				}
				sink.Reset()
			}
			
			b.StopTimer()
			err = proc.Shutdown(context.Background())
			require.NoError(b, err)
		})
		
		// The separate processors approach is simulated by making three passes through the
		// unified processor, each with a single function enabled.
		// This is a simplification but gives an approximation of the performance difference.
		b.Run(fmt.Sprintf("%s_Separate", bc.name), func(b *testing.B) {
			// Create three separate processors simulating the three-processor chain
			// Processor 1: Priority tagging only
			priorityFactory := metric_pipeline.NewFactory()
			priorityCfg := createCompleteTestConfig(resource_filter.StrategyPriority, false, false)
			prioritySink := new(consumertest.MetricsSink)
			prioritySettings := processor.Settings{
				TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
				ID: component.NewIDWithName(component.MustNewType("priority_tagger"), "benchmark"),
			}
			priorityProc, err := priorityFactory.CreateMetrics(context.Background(), prioritySettings, priorityCfg, prioritySink)
			require.NoError(b, err)
			err = priorityProc.Start(context.Background(), nil)
			require.NoError(b, err)
			
			// Processor 2: TopK only
			topkFactory := metric_pipeline.NewFactory()
			topkCfg := createCompleteTestConfig(resource_filter.StrategyTopK, false, false)
			topkSink := new(consumertest.MetricsSink)
			topkSettings := processor.Settings{
				TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
				ID: component.NewIDWithName(component.MustNewType("adaptive_topk"), "benchmark"),
			}
			topkProc, err := topkFactory.CreateMetrics(context.Background(), topkSettings, topkCfg, topkSink)
			require.NoError(b, err)
			err = topkProc.Start(context.Background(), nil)
			require.NoError(b, err)
			
			// Processor 3: Rollup, histograms, and attributes
			rollupFactory := metric_pipeline.NewFactory()
			rollupCfg := createCompleteTestConfig(resource_filter.StrategyPriority, true, true)
			rollupCfg.ResourceFilter.Enabled = false // Disable filtering for this stage
			rollupSink := new(consumertest.MetricsSink)
			rollupSettings := processor.Settings{
				TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
				ID: component.NewIDWithName(component.MustNewType("others_rollup"), "benchmark"),
			}
			rollupProc, err := rollupFactory.CreateMetrics(context.Background(), rollupSettings, rollupCfg, rollupSink)
			require.NoError(b, err)
			err = rollupProc.Start(context.Background(), nil)
			require.NoError(b, err)
			
			// Benchmark - simulating a sequence of three processors
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// 1. Priority tagging
				err := priorityProc.ConsumeMetrics(context.Background(), metrics)
				if err != nil {
					b.Fatal(err)
				}
				
				// Get results from first processor
				priorityResults := pmetric.NewMetrics()
				if len(prioritySink.AllMetrics()) > 0 {
					priorityResults = prioritySink.AllMetrics()[0]
				}
				prioritySink.Reset()
				
				// 2. Top-K filtering
				err = topkProc.ConsumeMetrics(context.Background(), priorityResults)
				if err != nil {
					b.Fatal(err)
				}
				
				// Get results from second processor
				topkResults := pmetric.NewMetrics()
				if len(topkSink.AllMetrics()) > 0 {
					topkResults = topkSink.AllMetrics()[0]
				}
				topkSink.Reset()
				
				// 3. Rollup, histograms, and attributes
				err = rollupProc.ConsumeMetrics(context.Background(), topkResults)
				if err != nil {
					b.Fatal(err)
				}
				
				rollupSink.Reset()
			}
			
			b.StopTimer()
			
			// Cleanup
			err = priorityProc.Shutdown(context.Background())
			require.NoError(b, err)
			err = topkProc.Shutdown(context.Background())
			require.NoError(b, err)
			err = rollupProc.Shutdown(context.Background())
			require.NoError(b, err)
		})
	}
}

// Helper functions

// createTestConfig creates a basic configuration for resource filtering benchmark
func createTestConfig() *metric_pipeline.Config {
	config := &metric_pipeline.Config{
		ResourceFilter: metric_pipeline.ResourceFilterConfig{
			Enabled:           true,
			FilterStrategy:    resource_filter.StrategyHybrid,
			PriorityAttribute: "aemf.process.priority",
			PriorityRules: []resource_filter.PriorityRule{
				{
					Match:    "process.executable.name=~/java|javaw|kafka/",
					Priority: resource_filter.PriorityHigh,
				},
				{
					Match:    "process.executable.name=~/mysql|postgres|redis/",
					Priority: resource_filter.PriorityCritical,
				},
				{
					Match:    ".*",
					Priority: resource_filter.PriorityLow,
				},
			},
			TopK: resource_filter.TopKConfig{
				KValue:         20,
				KMin:           10,
				KMax:           50,
				ResourceField:  "process.executable.name",
				CounterField:   "process.cpu.time",
				CoverageTarget: 0.95,
			},
			Rollup: resource_filter.RollupConfig{
				Enabled:           false, // Disabled for this benchmark
				PriorityThreshold: resource_filter.PriorityLow,
				Strategy:          resource_filter.AggregationSum,
				NamePrefix:        "others",
			},
		},
		Transformation: metric_pipeline.TransformationConfig{
			Histograms: metric_pipeline.HistogramConfig{
				Enabled:    false, // Disabled for this benchmark
				MaxBuckets: 10,
				Metrics:    make(map[string]metric_pipeline.HistogramMetric),
			},
			Attributes: metric_pipeline.AttributeConfig{
				Actions: []metric_pipeline.AttributeAction{},
			},
		},
	}
	return config
}

// createHistogramTestConfig creates a configuration focused on histogram generation
func createHistogramTestConfig(metricCount, bucketCount int) *metric_pipeline.Config {
	config := &metric_pipeline.Config{
		ResourceFilter: metric_pipeline.ResourceFilterConfig{
			Enabled: false, // Disabled for histogram benchmark
		},
		Transformation: metric_pipeline.TransformationConfig{
			Histograms: metric_pipeline.HistogramConfig{
				Enabled:    true,
				MaxBuckets: bucketCount,
				Metrics:    make(map[string]metric_pipeline.HistogramMetric),
			},
			Attributes: metric_pipeline.AttributeConfig{
				Actions: []metric_pipeline.AttributeAction{},
			},
		},
	}
	
	// Create boundaries for histogram metrics
	boundaries := make([]float64, bucketCount)
	for i := 0; i < bucketCount; i++ {
		boundaries[i] = float64((i + 1) * 10) // 10, 20, 30, ...
	}
	
	// Add metrics for histogram conversion
	for i := 0; i < metricCount; i++ {
		metricName := fmt.Sprintf("test.metric.%d", i)
		config.Transformation.Histograms.Metrics[metricName] = metric_pipeline.HistogramMetric{
			Boundaries: boundaries,
		}
	}
	
	return config
}

// createCompleteTestConfig creates a complete configuration with all features
func createCompleteTestConfig(
	strategy resource_filter.FilterStrategy,
	enableHistogram, 
	enableRollup bool,
) *metric_pipeline.Config {
	config := &metric_pipeline.Config{
		ResourceFilter: metric_pipeline.ResourceFilterConfig{
			Enabled:           true,
			FilterStrategy:    strategy,
			PriorityAttribute: "aemf.process.priority",
			PriorityRules: []resource_filter.PriorityRule{
				{
					Match:    "process.executable.name=~/java|javaw|kafka/",
					Priority: resource_filter.PriorityHigh,
				},
				{
					Match:    "process.executable.name=~/mysql|postgres|redis/",
					Priority: resource_filter.PriorityCritical,
				},
				{
					Match:    ".*",
					Priority: resource_filter.PriorityLow,
				},
			},
			TopK: resource_filter.TopKConfig{
				KValue:         20,
				KMin:           10,
				KMax:           50,
				ResourceField:  "process.executable.name",
				CounterField:   "process.cpu.time",
				CoverageTarget: 0.95,
			},
			Rollup: resource_filter.RollupConfig{
				Enabled:           enableRollup,
				PriorityThreshold: resource_filter.PriorityLow,
				Strategy:          resource_filter.AggregationSum,
				NamePrefix:        "others",
			},
		},
		Transformation: metric_pipeline.TransformationConfig{
			Histograms: metric_pipeline.HistogramConfig{
				Enabled:    enableHistogram,
				MaxBuckets: 10,
				Metrics: map[string]metric_pipeline.HistogramMetric{
					"process.cpu.time": {
						Boundaries: []float64{0.1, 0.5, 1.0, 5.0, 10.0},
					},
					"process.memory.usage": {
						Boundaries: []float64{1024 * 1024, 10 * 1024 * 1024, 100 * 1024 * 1024, 1000 * 1024 * 1024},
					},
				},
			},
			Attributes: metric_pipeline.AttributeConfig{
				Actions: []metric_pipeline.AttributeAction{
					{
						Key:    "process.command_line",
						Action: "delete",
					},
					{
						Key:    "collector.name",
						Action: "insert",
						Value:  "benchmark",
					},
				},
			},
		},
	}
	return config
}

// generateBenchmarkMetrics creates test metrics with the specified number of processes
// and metrics per process
func generateBenchmarkMetrics(processCount, metricsPerProcess int) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	
	for i := 0; i < processCount; i++ {
		rm := metrics.ResourceMetrics().AppendEmpty()
		resourceName := fmt.Sprintf("process-%d", i)
		
		// Set resource attributes based on index to simulate different types of processes
		if i % 10 == 0 {
			resourceName = fmt.Sprintf("java-%d", i)
		} else if i % 7 == 0 {
			resourceName = fmt.Sprintf("mysql-%d", i)
		}
		
		rm.Resource().Attributes().PutStr("process.executable.name", resourceName)
		rm.Resource().Attributes().PutStr("process.command_line", fmt.Sprintf("/usr/bin/%s --arg1 --arg2", resourceName))
		rm.Resource().Attributes().PutStr("process.pid", fmt.Sprintf("%d", 1000 + i))
		
		// Add scope metrics
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("test.scope")
		
		// Add the specified number of metrics
		for j := 0; j < metricsPerProcess; j++ {
			m := sm.Metrics().AppendEmpty()
			m.SetName(fmt.Sprintf("process.metric.%d", j))
			m.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(float64(i * j))
		}
	}
	
	return metrics
}

// generateHistogramBenchmarkMetrics creates test metrics specifically for histogram benchmark
func generateHistogramBenchmarkMetrics(processCount, histogramMetricCount int) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	
	for i := 0; i < processCount; i++ {
		rm := metrics.ResourceMetrics().AppendEmpty()
		resourceName := fmt.Sprintf("process-%d", i)
		rm.Resource().Attributes().PutStr("process.executable.name", resourceName)
		
		// Add scope metrics
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("test.scope")
		
		// Add metrics to be converted to histograms
		for j := 0; j < histogramMetricCount; j++ {
			m := sm.Metrics().AppendEmpty()
			m.SetName(fmt.Sprintf("test.metric.%d", j))
			m.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(float64(i * 10))
		}
		
		// Add some non-histogram metrics
		m := sm.Metrics().AppendEmpty()
		m.SetName("test.other_metric")
		m.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(float64(i))
	}
	
	return metrics
}

// generateCompleteBenchmarkMetrics creates comprehensive test metrics
// with attributes, process variety, and different metric types
func generateCompleteBenchmarkMetrics(processCount int) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	now := pcommon.NewTimestampFromTime(pcommon.Timestamp(0).AsTime())
	
	processTypes := []string{
		"java", "python", "node", "nginx", "httpd", "mysql", "postgres",
		"redis", "mongodb", "prometheus", "grafana", "fluentd", "kibana",
		"kafka", "zookeeper", "etcd", "consul", "nomad", "vault",
	}
	
	for i := 0; i < processCount; i++ {
		rm := metrics.ResourceMetrics().AppendEmpty()
		
		// Choose process type based on index
		processTypeIndex := i % len(processTypes)
		processName := processTypes[processTypeIndex]
		
		// Set resource attributes
		rm.Resource().Attributes().PutStr("process.executable.name", processName)
		rm.Resource().Attributes().PutStr("process.command_line", fmt.Sprintf("/usr/bin/%s --arg1 --arg2", processName))
		rm.Resource().Attributes().PutStr("process.pid", fmt.Sprintf("%d", 1000 + i))
		rm.Resource().Attributes().PutStr("host.name", fmt.Sprintf("host-%d", i / 10))
		
		// Add scope metrics
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("test.scope")
		
		// CPU metrics
		cpuMetric := sm.Metrics().AppendEmpty()
		cpuMetric.SetName("process.cpu.time")
		cpuMetric.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(float64(processCount - i) * 0.5)
		cpuMetric.SetDescription("CPU time")
		cpuMetric.SetUnit("s")
		cpuMetric.Gauge().DataPoints().At(0).SetTimestamp(now)
		
		// Memory metrics
		memMetric := sm.Metrics().AppendEmpty()
		memMetric.SetName("process.memory.usage")
		memMetric.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(float64(1024 * 1024 * (i % 100 + 10)))
		memMetric.SetDescription("Memory usage")
		memMetric.SetUnit("bytes")
		memMetric.Gauge().DataPoints().At(0).SetTimestamp(now)
		
		// Disk metrics
		diskMetric := sm.Metrics().AppendEmpty()
		diskMetric.SetName("process.disk.io")
		diskMetric.SetEmptySum().DataPoints().AppendEmpty().SetDoubleValue(float64(i * 1000))
		diskMetric.SetDescription("Disk I/O")
		diskMetric.SetUnit("bytes")
		diskMetric.Sum().DataPoints().At(0).SetTimestamp(now)
		
		// Thread metrics
		threadMetric := sm.Metrics().AppendEmpty()
		threadMetric.SetName("process.threads")
		threadMetric.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(float64(i % 20 + 1))
		threadMetric.SetDescription("Thread count")
		threadMetric.SetUnit("1")
		threadMetric.Gauge().DataPoints().At(0).SetTimestamp(now)
		
		// Open files metrics
		fileMetric := sm.Metrics().AppendEmpty()
		fileMetric.SetName("process.open_file_descriptors")
		fileMetric.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(float64(i % 50 + 5))
		fileMetric.SetDescription("Open file descriptors")
		fileMetric.SetUnit("1")
		fileMetric.Gauge().DataPoints().At(0).SetTimestamp(now)
	}
	
	return metrics
}