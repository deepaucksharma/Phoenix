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
	iftest "github.com/deepaucksharma/Phoenix/test/interfaces"
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
	settings := processor.CreateSettings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
		ID: component.NewID("pid_decider"),
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
				Name: "ModifyControllerTarget",
				Patch: interfaces.ConfigPatch{
					PatchID:             "test-modify-target",
					TargetProcessorName: component.NewID("pid_decider"),
					ParameterPath:       "controllers[0].kpi_target_value",
					NewValue:            0.90,
				},
				ExpectedValue: 0.90,
			},
			{
				Name: "ModifyControllerPID",
				Patch: interfaces.ConfigPatch{
					PatchID:             "test-modify-pid",
					TargetProcessorName: component.NewID("pid_decider"),
					ParameterPath:       "controllers[0].kp",
					NewValue:            20.0,
				},
				ExpectedValue: 20.0,
			},
		},
		InvalidPatches: []iftest.TestPatch{
			{
				Name: "InvalidController",
				Patch: interfaces.ConfigPatch{
					PatchID:             "test-invalid-controller",
					TargetProcessorName: component.NewID("pid_decider"),
					ParameterPath:       "controllers[999].kp",
					NewValue:            5.0,
				},
			},
		},
	}
	iftest.RunUpdateableProcessorTests(t, suite)

	// Test actual metric processing functionality
	t.Run("ProcessMetrics", func(t *testing.T) {
		// Create test metrics with KPI values
		metrics := generateTestKPIMetrics()
		
		// Process metrics
		err = proc.ConsumeMetrics(ctx, metrics)
		require.NoError(t, err)
		
		// Verify output - check that PID controller generates config patches as metrics
		processedMetrics := sink.AllMetrics()
		require.NotEmpty(t, processedMetrics)
		
		// In a real test, we would verify that the right metrics were emitted with proper values
		// For this stub test, just check that some metrics were produced
		assert.Greater(t, processedMetrics[0].MetricCount(), uint(0), "No metrics were produced")
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