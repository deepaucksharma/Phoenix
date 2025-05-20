// Package metricpipeline contains tests for the metric_pipeline processor
package metricpipeline

import (
	"context"
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

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/metric_pipeline"
	"github.com/deepaucksharma/Phoenix/internal/processor/resource_filter"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

// testNow is a fixed timestamp for testing
var testNow = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

func TestMetricPipelineProcessor_CreateDefaultConfig(t *testing.T) {
	factory := metric_pipeline.NewFactory()
	assert.NotNil(t, factory)

	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg)
	assert.IsType(t, &metric_pipeline.Config{}, cfg)
}

func TestMetricPipelineProcessor_ValidConfig(t *testing.T) {
	factory := metric_pipeline.NewFactory()
	assert.NotNil(t, factory)

	cfg := createTestConfig()
	assert.NoError(t, cfg.Validate())
}

func TestMetricPipelineProcessor_InvalidConfig(t *testing.T) {
	// Test with invalid filter strategy
	cfg := createTestConfig()
	cfg.ResourceFilter.FilterStrategy = "invalid_strategy"
	assert.Error(t, cfg.Validate())

	// Test with no priority rules for strategy that needs them
	cfg = createTestConfig()
	cfg.ResourceFilter.FilterStrategy = resource_filter.StrategyPriority
	cfg.ResourceFilter.PriorityRules = nil
	assert.Error(t, cfg.Validate())

	// Test with invalid k_value
	cfg = createTestConfig()
	cfg.ResourceFilter.FilterStrategy = resource_filter.StrategyTopK
	cfg.ResourceFilter.TopK.KValue = 0
	assert.Error(t, cfg.Validate())
}

func TestMetricPipelineProcessor_CreateWithDefaultConfig(t *testing.T) {
	factory := metric_pipeline.NewFactory()
	require.NotNil(t, factory)

	next := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(
		context.Background(),
		processor.Settings{
			TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		},
		factory.CreateDefaultConfig(),
		next,
	)
	assert.NoError(t, err)
	assert.NotNil(t, proc)
}

func TestMetricPipelineProcessor_PriorityTagging(t *testing.T) {
	// Create test processor with priority strategy
	factory := metric_pipeline.NewFactory()
	require.NotNil(t, factory)

	cfg := createTestConfig()
	cfg.ResourceFilter.FilterStrategy = resource_filter.StrategyPriority
	cfg.ResourceFilter.PriorityAttribute = "test.priority"

	// Set priority rules
	cfg.ResourceFilter.PriorityRules = []resource_filter.PriorityRule{
		{
			Match:    "process.executable.name=~/java|javaw/",
			Priority: resource_filter.PriorityHigh,
		},
		{
			Match:    "process.executable.name=~/mysql|postgres/",
			Priority: resource_filter.PriorityCritical,
		},
		{
			Match:    ".*",
			Priority: resource_filter.PriorityLow,
		},
	}

	next := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(
		context.Background(),
		processor.Settings{
			TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		},
		cfg,
		next,
	)
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Start the processor
	err = proc.Start(context.Background(), nil)
	require.NoError(t, err)

	// Create test metrics
	md := pmetric.NewMetrics()
	
	// Java process should be high priority
	rm1 := md.ResourceMetrics().AppendEmpty()
	rm1.Resource().Attributes().PutStr("process.executable.name", "java")
	sm1 := rm1.ScopeMetrics().AppendEmpty()
	sm1.Scope().SetName("test-scope")
	metric1 := sm1.Metrics().AppendEmpty()
	metric1.SetName("cpu.time")
	metric1.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(10.0)

	// MySQL process should be critical priority
	rm2 := md.ResourceMetrics().AppendEmpty()
	rm2.Resource().Attributes().PutStr("process.executable.name", "mysql")
	sm2 := rm2.ScopeMetrics().AppendEmpty()
	sm2.Scope().SetName("test-scope")
	metric2 := sm2.Metrics().AppendEmpty()
	metric2.SetName("cpu.time")
	metric2.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(20.0)

	// Some other process should be low priority
	rm3 := md.ResourceMetrics().AppendEmpty()
	rm3.Resource().Attributes().PutStr("process.executable.name", "other")
	sm3 := rm3.ScopeMetrics().AppendEmpty()
	sm3.Scope().SetName("test-scope")
	metric3 := sm3.Metrics().AppendEmpty()
	metric3.SetName("cpu.time")
	metric3.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(5.0)

	// Process the metrics
	err = proc.ConsumeMetrics(context.Background(), md)
	require.NoError(t, err)

	// Verify results
	result := next.AllMetrics()[0]
	
	// Only high and critical priority resources should be included
	assert.Equal(t, 2, result.ResourceMetrics().Len())
	
	// Find each resource by process name
	var foundJava, foundMySQL bool
	for i := 0; i < result.ResourceMetrics().Len(); i++ {
		rm := result.ResourceMetrics().At(i)
		procName, exists := rm.Resource().Attributes().Get("process.executable.name")
		assert.True(t, exists)
		
		if procName.Str() == "java" {
			foundJava = true
			priority, exists := rm.Resource().Attributes().Get("test.priority")
			assert.True(t, exists)
			assert.Equal(t, string(resource_filter.PriorityHigh), priority.Str())
		} else if procName.Str() == "mysql" {
			foundMySQL = true
			priority, exists := rm.Resource().Attributes().Get("test.priority")
			assert.True(t, exists)
			assert.Equal(t, string(resource_filter.PriorityCritical), priority.Str())
		}
	}
	
	assert.True(t, foundJava)
	assert.True(t, foundMySQL)

	// Shutdown the processor
	err = proc.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestMetricPipelineProcessor_TopK(t *testing.T) {
	// Create test processor with top-k strategy
	factory := metric_pipeline.NewFactory()
	require.NotNil(t, factory)

	cfg := createTestConfig()
	cfg.ResourceFilter.FilterStrategy = resource_filter.StrategyTopK
	cfg.ResourceFilter.TopK = resource_filter.TopKConfig{
		KValue:         2, // Only keep top 2 processes
		KMin:           1,
		KMax:           10,
		ResourceField:  "process.executable.name",
		CounterField:   "cpu.time",
		CoverageTarget: 0.95,
	}

	next := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(
		context.Background(),
		processor.Settings{
			TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		},
		cfg,
		next,
	)
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Start the processor
	err = proc.Start(context.Background(), nil)
	require.NoError(t, err)

	// Create test metrics
	md := pmetric.NewMetrics()
	
	// Process 1 with CPU 50
	rm1 := md.ResourceMetrics().AppendEmpty()
	rm1.Resource().Attributes().PutStr("process.executable.name", "process1")
	sm1 := rm1.ScopeMetrics().AppendEmpty()
	sm1.Scope().SetName("test-scope")
	metric1 := sm1.Metrics().AppendEmpty()
	metric1.SetName("cpu.time")
	metric1.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(50.0)

	// Process 2 with CPU 30
	rm2 := md.ResourceMetrics().AppendEmpty()
	rm2.Resource().Attributes().PutStr("process.executable.name", "process2")
	sm2 := rm2.ScopeMetrics().AppendEmpty()
	sm2.Scope().SetName("test-scope")
	metric2 := sm2.Metrics().AppendEmpty()
	metric2.SetName("cpu.time")
	metric2.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(30.0)

	// Process 3 with CPU 10
	rm3 := md.ResourceMetrics().AppendEmpty()
	rm3.Resource().Attributes().PutStr("process.executable.name", "process3")
	sm3 := rm3.ScopeMetrics().AppendEmpty()
	sm3.Scope().SetName("test-scope")
	metric3 := sm3.Metrics().AppendEmpty()
	metric3.SetName("cpu.time")
	metric3.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(10.0)

	// Process the metrics
	err = proc.ConsumeMetrics(context.Background(), md)
	require.NoError(t, err)

	// Verify results
	result := next.AllMetrics()[0]
	
	// Should only include the top 2 processes
	assert.Equal(t, 2, result.ResourceMetrics().Len())
	
	// Find each resource by process name
	var foundProcess1, foundProcess2 bool
	for i := 0; i < result.ResourceMetrics().Len(); i++ {
		rm := result.ResourceMetrics().At(i)
		procName, exists := rm.Resource().Attributes().Get("process.executable.name")
		assert.True(t, exists)
		
		if procName.Str() == "process1" {
			foundProcess1 = true
		} else if procName.Str() == "process2" {
			foundProcess2 = true
		}
	}
	
	assert.True(t, foundProcess1)
	assert.True(t, foundProcess2)

	// Shutdown the processor
	err = proc.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestMetricPipelineProcessor_Rollup(t *testing.T) {
	// Create test processor with priority strategy and rollup
	factory := metric_pipeline.NewFactory()
	require.NotNil(t, factory)

	cfg := createTestConfig()
	cfg.ResourceFilter.FilterStrategy = resource_filter.StrategyPriority
	cfg.ResourceFilter.PriorityAttribute = "test.priority"

	// Set priority rules
	cfg.ResourceFilter.PriorityRules = []resource_filter.PriorityRule{
		{
			Match:    "process.executable.name=~/highpri/",
			Priority: resource_filter.PriorityHigh,
		},
		{
			Match:    "process.executable.name=~/medpri/",
			Priority: resource_filter.PriorityMedium,
		},
		{
			Match:    ".*",
			Priority: resource_filter.PriorityLow,
		},
	}

	// Enable rollup for low priority
	cfg.ResourceFilter.Rollup = resource_filter.RollupConfig{
		Enabled:           true,
		PriorityThreshold: resource_filter.PriorityLow,
		Strategy:          resource_filter.AggregationSum,
		NamePrefix:        "others",
	}

	next := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(
		context.Background(),
		processor.Settings{
			TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		},
		cfg,
		next,
	)
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Start the processor
	err = proc.Start(context.Background(), nil)
	require.NoError(t, err)

	// Create test metrics
	md := pmetric.NewMetrics()
	
	// High priority process
	rm1 := md.ResourceMetrics().AppendEmpty()
	rm1.Resource().Attributes().PutStr("process.executable.name", "highpri-app")
	sm1 := rm1.ScopeMetrics().AppendEmpty()
	sm1.Scope().SetName("test-scope")
	metric1 := sm1.Metrics().AppendEmpty()
	metric1.SetName("cpu.time")
	metric1.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(100.0)

	// Medium priority process
	rm2 := md.ResourceMetrics().AppendEmpty()
	rm2.Resource().Attributes().PutStr("process.executable.name", "medpri-app")
	sm2 := rm2.ScopeMetrics().AppendEmpty()
	sm2.Scope().SetName("test-scope")
	metric2 := sm2.Metrics().AppendEmpty()
	metric2.SetName("cpu.time")
	metric2.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(50.0)

	// Low priority process 1
	rm3 := md.ResourceMetrics().AppendEmpty()
	rm3.Resource().Attributes().PutStr("process.executable.name", "lowpri-app1")
	sm3 := rm3.ScopeMetrics().AppendEmpty()
	sm3.Scope().SetName("test-scope")
	metric3 := sm3.Metrics().AppendEmpty()
	metric3.SetName("cpu.time")
	metric3.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(10.0)

	// Low priority process 2
	rm4 := md.ResourceMetrics().AppendEmpty()
	rm4.Resource().Attributes().PutStr("process.executable.name", "lowpri-app2")
	sm4 := rm4.ScopeMetrics().AppendEmpty()
	sm4.Scope().SetName("test-scope")
	metric4 := sm4.Metrics().AppendEmpty()
	metric4.SetName("cpu.time")
	metric4.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(5.0)

	// Process the metrics
	err = proc.ConsumeMetrics(context.Background(), md)
	require.NoError(t, err)

	// Verify results
	result := next.AllMetrics()[0]
	
	// Should include high and medium priority resources, plus one rollup resource
	assert.Equal(t, 3, result.ResourceMetrics().Len())
	
	// Find each resource type
	var foundHigh, foundMedium, foundRollup bool
	var rollupValue float64
	
	for i := 0; i < result.ResourceMetrics().Len(); i++ {
		rm := result.ResourceMetrics().At(i)
		procName, exists := rm.Resource().Attributes().Get("process.executable.name")
		assert.True(t, exists)
		
		if procName.Str() == "highpri-app" {
			foundHigh = true
			priority, exists := rm.Resource().Attributes().Get("test.priority")
			assert.True(t, exists)
			assert.Equal(t, string(resource_filter.PriorityHigh), priority.Str())
		} else if procName.Str() == "medpri-app" {
			foundMedium = true
			priority, exists := rm.Resource().Attributes().Get("test.priority")
			assert.True(t, exists)
			assert.Equal(t, string(resource_filter.PriorityMedium), priority.Str())
		} else if procName.Str() == "others" {
			foundRollup = true
			priority, exists := rm.Resource().Attributes().Get("test.priority")
			assert.True(t, exists)
			assert.Equal(t, string(resource_filter.PriorityLow), priority.Str())
			
			// Check rollup count attribute
			_, exists = rm.Resource().Attributes().Get("aemf.rollup.count")
			assert.True(t, exists)
			
			// Verify rolled up metrics
			assert.Equal(t, 1, rm.ScopeMetrics().Len())
			sm := rm.ScopeMetrics().At(0)
			assert.Equal(t, 1, sm.Metrics().Len())
			
			rollupMetric := sm.Metrics().At(0)
			assert.Equal(t, "others.cpu.time", rollupMetric.Name())
			
			// Verify value (should be sum of rolled up values: 10.0 + 5.0 = 15.0)
			assert.Equal(t, pmetric.MetricTypeGauge, rollupMetric.Type())
			assert.Equal(t, 1, rollupMetric.Gauge().DataPoints().Len())
			rollupValue = rollupMetric.Gauge().DataPoints().At(0).DoubleValue()
		}
	}
	
	assert.True(t, foundHigh)
	assert.True(t, foundMedium)
	assert.True(t, foundRollup)
	assert.Equal(t, 15.0, rollupValue)

	// Shutdown the processor
	err = proc.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestMetricPipelineProcessor_Histograms(t *testing.T) {
	// Create test processor with histogram transformation
	factory := metric_pipeline.NewFactory()
	require.NotNil(t, factory)

	cfg := createTestConfig()
	// Disable resource filtering for this test
	cfg.ResourceFilter.Enabled = false
	
	// Enable histogram generation
	cfg.Transformation.Histograms.Enabled = true
	cfg.Transformation.Histograms.MaxBuckets = 5
	cfg.Transformation.Histograms.Metrics = map[string]metric_pipeline.HistogramMetric{
		"cpu.time": {
			Boundaries: []float64{10.0, 20.0, 50.0, 100.0},
		},
	}

	next := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(
		context.Background(),
		processor.Settings{
			TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		},
		cfg,
		next,
	)
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Start the processor
	err = proc.Start(context.Background(), nil)
	require.NoError(t, err)

	// Create test metrics
	md := pmetric.NewMetrics()
	
	// Process with CPU time 25.0 (should go in the 20-50 bucket)
	rm := md.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("process.executable.name", "test-app")
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("test-scope")
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("cpu.time")
	metric.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(25.0)

	// Process the metrics
	err = proc.ConsumeMetrics(context.Background(), md)
	require.NoError(t, err)

	// Verify results
	result := next.AllMetrics()[0]
	
	// Should still have one resource
	assert.Equal(t, 1, result.ResourceMetrics().Len())
	
	// Now there should be both the original metric and a histogram
	rm = result.ResourceMetrics().At(0)
	assert.Equal(t, 1, rm.ScopeMetrics().Len())
	sm = rm.ScopeMetrics().At(0)
	assert.Equal(t, 2, sm.Metrics().Len())
	
	// Find the histogram metric
	var foundHistogram bool
	for i := 0; i < sm.Metrics().Len(); i++ {
		m := sm.Metrics().At(i)
		if m.Name() == "cpu.time_histogram" {
			foundHistogram = true
			assert.Equal(t, pmetric.MetricTypeHistogram, m.Type())
			
			// Check histogram properties
			histogram := m.Histogram()
			assert.Equal(t, 1, histogram.DataPoints().Len())
			dp := histogram.DataPoints().At(0)
			
			// Boundaries should match config
			assert.Equal(t, 4, dp.ExplicitBounds().Len())
			assert.Equal(t, 10.0, dp.ExplicitBounds().At(0))
			assert.Equal(t, 20.0, dp.ExplicitBounds().At(1))
			assert.Equal(t, 50.0, dp.ExplicitBounds().At(2))
			assert.Equal(t, 100.0, dp.ExplicitBounds().At(3))
			
			// Should have 5 buckets (4 boundaries + 1)
			assert.Equal(t, 5, dp.BucketCounts().Len())
			
			// Value 25.0 should be in the 3rd bucket (index 2)
			assert.Equal(t, uint64(0), dp.BucketCounts().At(0)) // ≤ 10
			assert.Equal(t, uint64(0), dp.BucketCounts().At(1)) // ≤ 20
			assert.Equal(t, uint64(1), dp.BucketCounts().At(2)) // ≤ 50
			assert.Equal(t, uint64(0), dp.BucketCounts().At(3)) // ≤ 100
			assert.Equal(t, uint64(0), dp.BucketCounts().At(4)) // > 100
			
			// Count should be 1 datapoint
			assert.Equal(t, uint64(1), dp.Count())
			
			// Sum should be original value
			assert.Equal(t, 25.0, dp.Sum())
		}
	}
	
	assert.True(t, foundHistogram)

	// Shutdown the processor
	err = proc.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestMetricPipelineProcessor_AttributeActions(t *testing.T) {
	// Create test processor with attribute actions
	factory := metric_pipeline.NewFactory()
	require.NotNil(t, factory)

	cfg := createTestConfig()
	// Disable resource filtering for this test
	cfg.ResourceFilter.Enabled = false
	
	// Configure attribute actions
	cfg.Transformation.Attributes.Actions = []metric_pipeline.AttributeAction{
		{
			Key:    "process.command_line",
			Action: "delete",
		},
		{
			Key:    "collector.name",
			Action: "insert",
			Value:  "test-collector",
		},
		{
			Key:    "process.executable.name",
			Action: "update",
			Value:  "renamed-process",
		},
	}

	next := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(
		context.Background(),
		processor.Settings{
			TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		},
		cfg,
		next,
	)
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Start the processor
	err = proc.Start(context.Background(), nil)
	require.NoError(t, err)

	// Create test metrics
	md := pmetric.NewMetrics()
	
	// Process with attributes to be modified
	rm := md.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("process.executable.name", "test-app")
	rm.Resource().Attributes().PutStr("process.command_line", "/bin/test-app --flag")
	rm.Resource().Attributes().PutStr("process.pid", "12345")
	
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("test-scope")
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("test.metric")
	metric.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(42.0)

	// Process the metrics
	err = proc.ConsumeMetrics(context.Background(), md)
	require.NoError(t, err)

	// Verify results
	result := next.AllMetrics()[0]
	
	// Should still have one resource
	assert.Equal(t, 1, result.ResourceMetrics().Len())
	
	// Check the attribute modifications
	rm = result.ResourceMetrics().At(0)
	attrs := rm.Resource().Attributes()
	
	// Verify delete action
	_, exists := attrs.Get("process.command_line")
	assert.False(t, exists)
	
	// Verify insert action
	val, exists := attrs.Get("collector.name")
	assert.True(t, exists)
	assert.Equal(t, "test-collector", val.Str())
	
	// Verify update action
	val, exists = attrs.Get("process.executable.name")
	assert.True(t, exists)
	assert.Equal(t, "renamed-process", val.Str())
	
	// Verify untouched attribute
	val, exists = attrs.Get("process.pid")
	assert.True(t, exists)
	assert.Equal(t, "12345", val.Str())

	// Shutdown the processor
	err = proc.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestMetricPipelineProcessor_ConfigPatching(t *testing.T) {
	// Create test processor
	factory := metric_pipeline.NewFactory()
	require.NotNil(t, factory)

	cfg := createTestConfig()
	cfg.ResourceFilter.FilterStrategy = resource_filter.StrategyTopK
	cfg.ResourceFilter.TopK.KValue = 5

	next := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(
		context.Background(),
		processor.Settings{
			TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
		},
		cfg,
		next,
	)
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Ensure it implements the UpdateableProcessor interface
	updateableProc, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok, "Processor does not implement UpdateableProcessor")

	// Start the processor
	err = proc.Start(context.Background(), nil)
	require.NoError(t, err)

	// Get initial config status
	status, err := updateableProc.GetConfigStatus(context.Background())
	require.NoError(t, err)
	assert.True(t, status.Enabled)

	// Apply config patch to change k_value
	patch := interfaces.ConfigPatch{
		PatchID:             "test-patch",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("metric_pipeline"), ""),
		ParameterPath:       "resource_filter.topk.k_value",
		NewValue:            10,
		Reason:              "testing",
	}
	
	err = updateableProc.OnConfigPatch(context.Background(), patch)
	require.NoError(t, err)

	// Get updated config status
	newStatus, err := updateableProc.GetConfigStatus(context.Background())
	require.NoError(t, err)
	assert.True(t, newStatus.Enabled)

	// Create test metrics to verify the change took effect
	metrics := generateTopKTestMetrics(15) // Create 15 processes
	err = proc.ConsumeMetrics(context.Background(), metrics)
	require.NoError(t, err)

	// Should have filtered to top 10 (after patch) instead of top 5 (initial config)
	result := next.AllMetrics()[0]
	assert.Equal(t, 10, result.ResourceMetrics().Len())

	// Shutdown the processor
	err = proc.Shutdown(context.Background())
	require.NoError(t, err)
}

// Helper functions

// createTestConfig creates a test configuration
func createTestConfig() *metric_pipeline.Config {
	config := &metric_pipeline.Config{
		ResourceFilter: metric_pipeline.ResourceFilterConfig{
			Enabled:           true,
			FilterStrategy:    resource_filter.StrategyHybrid,
			PriorityAttribute: "aemf.process.priority",
			PriorityRules: []resource_filter.PriorityRule{
				{
					Match:    "process.executable.name=~/java|javaw/",
					Priority: resource_filter.PriorityHigh,
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
				Enabled:           true,
				PriorityThreshold: resource_filter.PriorityLow,
				Strategy:          resource_filter.AggregationSum,
				NamePrefix:        "others",
			},
		},
		Transformation: metric_pipeline.TransformationConfig{
			Histograms: metric_pipeline.HistogramConfig{
				Enabled:    true,
				MaxBuckets: 10,
				Metrics: map[string]metric_pipeline.HistogramMetric{
					"process.cpu.time": {
						Boundaries: []float64{0.1, 0.5, 1.0, 5.0, 10.0},
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
						Value:  "SA-OMF",
					},
				},
			},
		},
	}
	return config
}

// generateTopKTestMetrics creates metrics for testing the top-k functionality
func generateTopKTestMetrics(processCount int) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	
	// Create processes with decreasing CPU values
	for i := 0; i < processCount; i++ {
		rm := metrics.ResourceMetrics().AppendEmpty()
		
		// Set resource attributes
		rm.Resource().Attributes().PutStr("process.executable.name", fmt.Sprintf("process-%d", i))
		
		// Add CPU metric
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("test-scope")
		
		metric := sm.Metrics().AppendEmpty()
		metric.SetName("process.cpu.time")
		metric.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(float64(processCount - i))
	}
	
	return metrics
}