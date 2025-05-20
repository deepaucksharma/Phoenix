package timeseries_estimator

import (
	"context"
	"runtime"
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
	"github.com/deepaucksharma/Phoenix/internal/processor/timeseries_estimator"
)

func TestTimeseriesEstimatorProcessor(t *testing.T) {
	// Create a factory
	factory := timeseries_estimator.NewFactory()
	assert.NotNil(t, factory)

	// Create a default configuration
	cfg := factory.CreateDefaultConfig().(*timeseries_estimator.Config)

	// Modify config for testing
	cfg.Enabled = true
	cfg.OutputMetricName = "test_timeseries_count"
	cfg.EstimatorType = "exact" // Test with exact counting first
	cfg.MemoryLimitMB = 100
	cfg.RefreshInterval = time.Minute

	// Create a test sink for output metrics
	sink := new(consumertest.MetricsSink)

	// Create the processor
	ctx := context.Background()
	settings := processor.Settings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
		ID: component.NewIDWithName(component.MustNewType("timeseries_estimator"), ""),
	}

	proc, err := factory.CreateMetricsProcessor(ctx, settings, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Ensure it implements the UpdateableProcessor interface
	updateableProc, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok, "Processor does not implement UpdateableProcessor")

	// Start the processor
	err = proc.Start(ctx, nil)
	require.NoError(t, err)

	// Test cases
	t.Run("EmptyMetrics", func(t *testing.T) {
		// Process empty metrics
		err = proc.ConsumeMetrics(ctx, pmetric.NewMetrics())
		require.NoError(t, err)

		// Verify output
		metrics := sink.AllMetrics()
		require.NotEmpty(t, metrics)
		
		lastMetrics := metrics[len(metrics)-1]
		assert.Equal(t, 1, lastMetrics.ResourceMetrics().Len())
		
		// The estimate should be 0 since we sent empty metrics
		foundEstimateMetric := false
		rm := lastMetrics.ResourceMetrics().At(0)
		for i := 0; i < rm.ScopeMetrics().Len(); i++ {
			sm := rm.ScopeMetrics().At(i)
			for j := 0; j < sm.Metrics().Len(); j++ {
				m := sm.Metrics().At(j)
				if m.Name() == "test_timeseries_count" {
					foundEstimateMetric = true
					assert.Equal(t, pmetric.MetricTypeGauge, m.Type())
					assert.Equal(t, int64(0), m.Gauge().DataPoints().At(0).IntValue())
				}
			}
		}
		assert.True(t, foundEstimateMetric, "Estimate metric not found")
	})

	// Clear metrics sink
	sink.Reset()

	t.Run("UniqueTimeSeries", func(t *testing.T) {
		// Create test metrics with known unique time series
		metrics := createTestMetrics()
		
		// Process metrics
		err = proc.ConsumeMetrics(ctx, metrics)
		require.NoError(t, err)
		
		// Verify output
		outputMetrics := sink.AllMetrics()
		require.NotEmpty(t, outputMetrics)
		
		lastMetrics := outputMetrics[len(outputMetrics)-1]
		
		// The estimate should match our expected count (3 unique time series)
		foundEstimateMetric := false
		var estimateValue int64
		
		rm := lastMetrics.ResourceMetrics().At(0)
		for i := 0; i < rm.ScopeMetrics().Len(); i++ {
			sm := rm.ScopeMetrics().At(i)
			for j := 0; j < sm.Metrics().Len(); j++ {
				m := sm.Metrics().At(j)
				if m.Name() == "test_timeseries_count" {
					foundEstimateMetric = true
					estimateValue = m.Gauge().DataPoints().At(0).IntValue()
				}
			}
		}
		
		assert.True(t, foundEstimateMetric, "Estimate metric not found")
		assert.Equal(t, int64(3), estimateValue, "Estimate should be 3 unique time series")
	})

	// Clear metrics sink
	sink.Reset()

	t.Run("DuplicateTimeSeries", func(t *testing.T) {
		// Create test metrics with duplicate time series
		metrics := createDuplicateTestMetrics()
		
		// Process metrics
		err = proc.ConsumeMetrics(ctx, metrics)
		require.NoError(t, err)
		
		// Verify output
		outputMetrics := sink.AllMetrics()
		require.NotEmpty(t, outputMetrics)
		
		lastMetrics := outputMetrics[len(outputMetrics)-1]
		
		// The estimate should be 1 since we have duplicate time series
		foundEstimateMetric := false
		var estimateValue int64
		
		rm := lastMetrics.ResourceMetrics().At(0)
		for i := 0; i < rm.ScopeMetrics().Len(); i++ {
			sm := rm.ScopeMetrics().At(i)
			for j := 0; j < sm.Metrics().Len(); j++ {
				m := sm.Metrics().At(j)
				if m.Name() == "test_timeseries_count" {
					foundEstimateMetric = true
					estimateValue = m.Gauge().DataPoints().At(0).IntValue()
				}
			}
		}
		
		assert.True(t, foundEstimateMetric, "Estimate metric not found")
		assert.Equal(t, int64(1), estimateValue, "Estimate should be 1 unique time series")
	})

	// Test configuration patching
	t.Run("ConfigPatching", func(t *testing.T) {
		// Test enabling/disabling
		enablePatch := interfaces.ConfigPatch{
			PatchID:             "test-enable",
			TargetProcessorName: component.NewIDWithName(component.MustNewType("timeseries_estimator"), ""),
			ParameterPath:       "enabled",
			Parameter:           "enabled",
			NewValue:            false,
		}
		
		err = updateableProc.OnConfigPatch(ctx, enablePatch)
		require.NoError(t, err)
		
		// Verify config status
		status, err := updateableProc.GetConfigStatus(ctx)
		require.NoError(t, err)
		assert.False(t, status.Enabled)
		
		// Test changing estimator type
		typePatch := interfaces.ConfigPatch{
			PatchID:             "test-type",
			TargetProcessorName: component.NewIDWithName(component.MustNewType("timeseries_estimator"), ""),
			ParameterPath:       "estimator_type",
			Parameter:           "estimator_type",
			NewValue:            "hll",
		}
		
		err = updateableProc.OnConfigPatch(ctx, typePatch)
		require.NoError(t, err)
		
		// Verify config status
		status, err = updateableProc.GetConfigStatus(ctx)
		require.NoError(t, err)
		assert.Equal(t, "hll", status.Parameters["estimator_type"])
		
		// Test invalid parameter
		invalidPatch := interfaces.ConfigPatch{
			PatchID:             "test-invalid",
			TargetProcessorName: component.NewIDWithName(component.MustNewType("timeseries_estimator"), ""),
			ParameterPath:       "invalid_param",
			Parameter:           "invalid_param",
			NewValue:            "value",
		}
		
		err = updateableProc.OnConfigPatch(ctx, invalidPatch)
		assert.Error(t, err, "Should fail with invalid parameter")
	})

	// Test memory limit and HLL fallback
	t.Run("MemoryLimitFallback", func(t *testing.T) {
		// Reset to exact counting
		typePatch := interfaces.ConfigPatch{
			PatchID:             "reset-type",
			TargetProcessorName: component.NewIDWithName(component.MustNewType("timeseries_estimator"), ""),
			ParameterPath:       "estimator_type",
			Parameter:           "estimator_type",
			NewValue:            "exact",
		}
		
		err = updateableProc.OnConfigPatch(ctx, typePatch)
		require.NoError(t, err)
		
		// Set a very low memory limit to trigger fallback
		memoryPatch := interfaces.ConfigPatch{
			PatchID:             "test-memory-limit",
			TargetProcessorName: component.NewIDWithName(component.MustNewType("timeseries_estimator"), ""),
			ParameterPath:       "memory_limit_mb",
			Parameter:           "memory_limit_mb",
			NewValue:            1, // 1MB is very low, should trigger fallback
		}
		
		err = updateableProc.OnConfigPatch(ctx, memoryPatch)
		require.NoError(t, err)
		
		// Get current memory usage to verify we're above the limit
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		memUsageMB := float64(memStats.Alloc) / 1024.0 / 1024.0
		
		t.Logf("Current memory usage: %.2f MB", memUsageMB)
		
		// Create test metrics with many time series to increase memory pressure
		metrics := createLargeTestMetrics()
		
		// Process metrics
		err = proc.ConsumeMetrics(ctx, metrics)
		require.NoError(t, err)
		
		// Verify config status to check if memory_constrained is true
		status, err := updateableProc.GetConfigStatus(ctx)
		require.NoError(t, err)
		assert.True(t, status.Parameters["memory_constrained"].(bool), "Memory should be constrained with 1MB limit")
	})

	// Shutdown the processor
	err = proc.Shutdown(ctx)
	require.NoError(t, err)
}

// createTestMetrics creates test metrics with 3 unique time series
func createTestMetrics() pmetric.Metrics {
	md := pmetric.NewMetrics()
	
	// Resource 1
	rm1 := md.ResourceMetrics().AppendEmpty()
	rm1.Resource().Attributes().PutStr("service.name", "test-service")
	rm1.Resource().Attributes().PutStr("host.name", "host-1")
	
	sm1 := rm1.ScopeMetrics().AppendEmpty()
	sm1.Scope().SetName("test-scope")
	
	// Metric 1
	m1 := sm1.Metrics().AppendEmpty()
	m1.SetName("cpu.usage")
	gauge1 := m1.SetEmptyGauge()
	dp1 := gauge1.DataPoints().AppendEmpty()
	dp1.SetDoubleValue(0.5)
	dp1.Attributes().PutStr("cpu", "0")
	
	// Metric 2 (different name)
	m2 := sm1.Metrics().AppendEmpty()
	m2.SetName("memory.usage")
	gauge2 := m2.SetEmptyGauge()
	dp2 := gauge2.DataPoints().AppendEmpty()
	dp2.SetDoubleValue(1024.0)
	dp2.Attributes().PutStr("state", "used")
	
	// Resource 2 (different resource)
	rm2 := md.ResourceMetrics().AppendEmpty()
	rm2.Resource().Attributes().PutStr("service.name", "test-service")
	rm2.Resource().Attributes().PutStr("host.name", "host-2")
	
	sm2 := rm2.ScopeMetrics().AppendEmpty()
	sm2.Scope().SetName("test-scope")
	
	// Metric 3 (same name but different resource)
	m3 := sm2.Metrics().AppendEmpty()
	m3.SetName("cpu.usage") 
	gauge3 := m3.SetEmptyGauge()
	dp3 := gauge3.DataPoints().AppendEmpty()
	dp3.SetDoubleValue(0.7)
	dp3.Attributes().PutStr("cpu", "0")
	
	return md
}

// createDuplicateTestMetrics creates test metrics with duplicate time series
func createDuplicateTestMetrics() pmetric.Metrics {
	md := pmetric.NewMetrics()
	
	// Create 5 identical time series (should count as 1 unique)
	for i := 0; i < 5; i++ {
		rm := md.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutStr("service.name", "dup-service")
		rm.Resource().Attributes().PutStr("host.name", "dup-host")
		
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("dup-scope")
		
		m := sm.Metrics().AppendEmpty()
		m.SetName("dup.metric")
		gauge := m.SetEmptyGauge()
		dp := gauge.DataPoints().AppendEmpty()
		dp.SetDoubleValue(1.0)
		dp.Attributes().PutStr("key", "value")
	}
	
	return md
}

// createLargeTestMetrics creates test metrics with many time series to test memory pressure
func createLargeTestMetrics() pmetric.Metrics {
	md := pmetric.NewMetrics()
	
	// Create 1000 unique time series to increase memory usage
	for i := 0; i < 1000; i++ {
		rm := md.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutStr("service.name", "service-"+string(rune(i%10)))
		rm.Resource().Attributes().PutStr("host.name", "host-"+string(rune(i%20)))
		rm.Resource().Attributes().PutStr("host.id", string(rune(i)))
		
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("scope-"+string(rune(i%5)))
		
		m := sm.Metrics().AppendEmpty()
		m.SetName("metric-"+string(rune(i%15)))
		gauge := m.SetEmptyGauge()
		dp := gauge.DataPoints().AppendEmpty()
		dp.SetDoubleValue(float64(i))
		dp.Attributes().PutStr("key-1", "value-"+string(rune(i%30)))
		dp.Attributes().PutStr("key-2", "value-"+string(rune(i%40)))
		dp.Attributes().PutStr("key-3", "value-"+string(rune(i%50)))
	}
	
	return md
}