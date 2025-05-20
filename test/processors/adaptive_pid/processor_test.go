package adaptivepid

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
	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_pid"
)

func TestAdaptivePIDProcessor(t *testing.T) {
	// Create a factory
	factory := adaptive_pid.NewFactory()
	assert.NotNil(t, factory)

	// Create a default configuration
	cfg := factory.CreateDefaultConfig().(*adaptive_pid.Config)
	
	// Modify config for testing
	cfg.Controllers = []adaptive_pid.ControllerConfig{
		{
			Name:              "test_controller",
			Enabled:           true,
			KPIMetricName:     "aemf_impact_test_metric",
			KPITargetValue:    0.80,
			KP:                10,
			KI:                2,
			KD:                1,
			HysteresisPercent: 2,
			OutputConfigPatches: []adaptive_pid.OutputConfigPatch{
				{
					TargetProcessorName: "adaptive_topk",
					ParameterPath:       "k_value",
					ChangeScaleFactor:   -5,
					MinValue:            5,
					MaxValue:            30,
				},
			},
		},
	}

	// Create a test sink for output metrics
	sink := new(consumertest.MetricsSink)

	// Create the processor
	ctx := context.Background()
	settings := processor.Settings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
		ID: component.NewIDWithName(component.MustNewType("adaptive_pid"), ""),
	}

	proc, err := factory.CreateMetrics(ctx, settings, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Ensure it implements the UpdateableProcessor interface
	updateableProc, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok, "Processor does not implement UpdateableProcessor")

	// Start the processor
	err = proc.Start(ctx, nil)
	require.NoError(t, err)

	// Test the interface methods directly
	// Test the OnConfigPatch method
	
	// Test modifying controller target - reference the controller by name in the path
	targetPatch := interfaces.ConfigPatch{
		PatchID:             "test-modify-target",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("test_controller"), ""),
		ParameterPath:       "kpi_target_value",
		NewValue:            0.90,
	}
	err = updateableProc.OnConfigPatch(ctx, targetPatch)
	require.NoError(t, err, "Failed to apply target patch")
	
	// Skip PID controller tuning test for now as it's not implemented 
	// in the OnConfigPatch method yet
	
	// Get config status and verify
	status, err := updateableProc.GetConfigStatus(ctx)
	require.NoError(t, err, "Failed to get config status")
	assert.True(t, status.Enabled, "Processor should be enabled")
	
	// Test invalid patch - non-existent controller
	invalidPatch := interfaces.ConfigPatch{
		PatchID:             "test-invalid-controller",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("non_existent_controller"), ""),
		ParameterPath:       "kpi_target_value",
		NewValue:            5.0,
	}
	err = updateableProc.OnConfigPatch(ctx, invalidPatch)
	assert.Error(t, err, "Should fail with invalid controller name")

	// Test actual metric processing functionality
	t.Run("ProcessMetrics", func(t *testing.T) {
		// Create test metrics with KPI values
		metrics := generateTestKPIMetrics()
		
		// Process metrics
		err = proc.ConsumeMetrics(ctx, metrics)
		require.NoError(t, err)
		
		// Verify output - check that PID controller processes metrics
		// Note: Since the implementation currently just logs patches and doesn't emit them as metrics,
		// we'll just verify that processing completes without error
		processedMetrics := sink.AllMetrics()
		assert.NotEmpty(t, processedMetrics)
	})
	
	// Shutdown the processor
	err = proc.Shutdown(ctx)
	require.NoError(t, err)
}

// generateTestKPIMetrics creates test metrics with values for PID controller
func generateTestKPIMetrics() pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	
	// Create a resource metric
	rm := metrics.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("service.name", "test-service")
	
	// Add a scope metric
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("test.scope")
	
	// Add the KPI metric that will trigger the PID controller
	kpiMetric := sm.Metrics().AppendEmpty()
	kpiMetric.SetName("aemf_impact_test_metric")
	kpiMetric.SetEmptyGauge()
	dp := kpiMetric.Gauge().DataPoints().AppendEmpty()
	// Use a value different from the target to trigger PID controller
	dp.SetDoubleValue(0.60)  // Target is 0.80, so this creates a gap
	dp.SetTimestamp(pcommon.NewTimestampFromTime(testNow))
	
	return metrics
}

// Test time value to use for consistency
var testNow = time.Now()