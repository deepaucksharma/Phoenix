package adaptivetopk

import (
	"context"
	"fmt"
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
	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_topk"
	iftest "github.com/deepaucksharma/Phoenix/test/interfaces"
)

func TestAdaptiveTopKProcessor(t *testing.T) {
	// Create a factory
	factory := adaptive_topk.NewFactory()
	assert.NotNil(t, factory)

	// Create a default configuration
	cfg := factory.CreateDefaultConfig().(*adaptive_topk.Config)
	
	// Modify config for testing
	cfg.KValue = 10
	cfg.KMin = 5
	cfg.KMax = 30
	cfg.ResourceField = "process.name"
	cfg.CounterField = "cpu.usage"
	cfg.Enabled = true

	// Create a test sink for output metrics
	sink := new(consumertest.MetricsSink)

	// Create the processor
	ctx := context.Background()
	settings := processor.CreateSettings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
		ID: component.NewID("adaptive_topk"),
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

	// Run the standard UpdateableProcessor tests
	suite := iftest.UpdateableProcessorSuite{
		Processor: updateableProc,
		ValidPatches: []iftest.TestPatch{
			{
				Name: "UpdateKValue",
				Patch: interfaces.ConfigPatch{
					PatchID:             "test-update-k",
					TargetProcessorName: component.NewID("adaptive_topk"),
					ParameterPath:       "k_value",
					NewValue:            15,
				},
				ExpectedValue: 15,
			},
			{
				Name: "ChangeEnabled",
				Patch: interfaces.ConfigPatch{
					PatchID:             "test-enable",
					TargetProcessorName: component.NewID("adaptive_topk"),
					ParameterPath:       "enabled",
					NewValue:            false,
				},
				ExpectedValue: false,
			},
		},
		InvalidPatches: []iftest.TestPatch{
			{
				Name: "InvalidKValue",
				Patch: interfaces.ConfigPatch{
					PatchID:             "test-invalid-k",
					TargetProcessorName: component.NewID("adaptive_topk"),
					ParameterPath:       "k_value",
					NewValue:            50, // Greater than k_max
				},
			},
		},
	}
	iftest.RunUpdateableProcessorTests(t, suite)

	// Test actual metric processing functionality
	t.Run("ProcessMetrics", func(t *testing.T) {
		// Create test metrics with process usage data
		metrics := generateTestProcessMetrics()
		
		// Re-enable the processor for testing
		enablePatch := interfaces.ConfigPatch{
			PatchID:             "test-enable-for-processing",
			TargetProcessorName: component.NewID("adaptive_topk"),
			ParameterPath:       "enabled",
			NewValue:            true,
		}
		err = updateableProc.OnConfigPatch(ctx, enablePatch)
		require.NoError(t, err)
		
		// Process metrics
		err = proc.ConsumeMetrics(ctx, metrics)
		require.NoError(t, err)
		
		// Verify output - check that metrics were appropriately processed
		processedMetrics := sink.AllMetrics()
		require.NotEmpty(t, processedMetrics)
		
		// Calculate how many resources should have been kept (top-k)
		expectedTopK := cfg.KValue
		if expectedTopK > 20 { // We generate 20 processes in our test
			expectedTopK = 20
		}
		
		// Count how many resources passed through the processor
		var resourcesEmitted int
		for i := 0; i < processedMetrics[0].ResourceMetrics().Len(); i++ {
			rm := processedMetrics[0].ResourceMetrics().At(i)
			resourcesEmitted++
			
			// Verify that the resource has the topk attribute set
			_, ok := rm.Resource().Attributes().Get("aemf.topk.included")
			assert.True(t, ok, "Resource is missing the topk inclusion attribute")
		}
		
		// Check that the right number of resources were emitted
		assert.Equal(t, expectedTopK, resourcesEmitted, "Expected %d resources to be emitted (k value), but got %d", expectedTopK, resourcesEmitted)
	})
	
	// Shutdown the processor
	err = proc.Shutdown(ctx)
	require.NoError(t, err)
}

// generateTestProcessMetrics creates test metrics for 20 processes with varying CPU usage
func generateTestProcessMetrics() pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	
	// Create 20 processes with different CPU/memory values in descending order
	// This allows us to predictably test the top-k behavior
	for i := 0; i < 20; i++ {
		processName := fmt.Sprintf("process-%02d", i)
		cpuValue := 100.0 - float64(i*5) // Processes have values from 100% down to 5%
		
		rm := metrics.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutStr("process.name", processName)
		rm.Resource().Attributes().PutStr("process.pid", fmt.Sprintf("%d", 1000+i))
		
		// Add a CPU metric
		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName("test.scope")
		
		cpuMetric := sm.Metrics().AppendEmpty()
		cpuMetric.SetName("cpu.usage")
		cpuMetric.SetEmptyGauge()
		dp := cpuMetric.Gauge().DataPoints().AppendEmpty()
		dp.SetDoubleValue(cpuValue)
		dp.SetTimestamp(pcommon.NewTimestampFromTime(testNow))
	}
	
	return metrics
}

// Test time value to use for consistency
var testNow = time.Now()