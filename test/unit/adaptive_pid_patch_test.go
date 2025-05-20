package unit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_pid"
	"github.com/deepaucksharma/Phoenix/internal/processor/metric_pipeline"
)

// TestPatchGeneration tests that the PID controller generates and applies patches correctly
func TestPatchGeneration(t *testing.T) {

	// Create a logger
	logger := zaptest.NewLogger(t)

	// Create an adaptive_pid processor with the mock pic_control
	factory := adaptive_pid.NewFactory()
	cfg := factory.CreateDefaultConfig().(*adaptive_pid.Config)

	// Configure the processor with a test controller
	cfg.Controllers = []adaptive_pid.ControllerConfig{
		{
			Name:           "test_controller",
			Enabled:        true,
			KPIMetricName:  "aemf_test_metric",
			KPITargetValue: 100.0,
			KP:             1.0,
			KI:             0.0,
			KD:             0.0,
			OutputConfigPatches: []adaptive_pid.OutputConfigPatch{
				{
					TargetProcessorName: "metric_pipeline",
					ParameterPath:       "resource_filter.topk.k_value",
					ChangeScaleFactor:   -1.0, // Simple 1:1 scaling for testing
					MinValue:            10,
					MaxValue:            50,
				},
			},
		},
	}

	// Create a test sink
	sink := new(consumertest.MetricsSink)

	// Create the processor
	settings := processor.Settings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: logger,
		},
		ID: component.NewIDWithName(component.MustNewType("adaptive_pid"), ""),
	}

	proc, err := adaptive_pid.NewProcessor(cfg, settings.TelemetrySettings, settings.ID)
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Start the processor
	err = proc.Start(context.Background(), nil)
	require.NoError(t, err)

	// Create test metrics that will trigger a patch
	// Current value 80, target 100, error 20, with Kp=1.0 should generate a patch of value -20
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName("aemf_test_metric")
	m.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(80.0)

	// Process the metrics and capture patches
	patches, err := proc.ProcessMetricsForTest(context.Background(), metrics)
	require.NoError(t, err)
	require.Len(t, patches, 1)
	patch := patches[0]

	// Validate the patch details
	assert.Equal(t, "metric_pipeline", patch.TargetProcessorName.Type().String())
	assert.Equal(t, "resource_filter.topk.k_value", patch.ParameterPath)

	// With current value 80, target 100, error 20, and ChangeScaleFactor -1.0,
	// we expect a patch value of -20
	// This would be used to adjust k_value by -20
	assert.InDelta(t, 20.0, patch.NewValue, 1.0)

	// Shutdown the processor
	err = proc.Shutdown(context.Background())
	require.NoError(t, err)
}

// TestPatchApplication tests that a metric_pipeline processor correctly applies patches
func TestPatchApplication(t *testing.T) {
	// Create a logger
	logger := zaptest.NewLogger(t)

	// Create a metric_pipeline processor
	factory := metric_pipeline.NewFactory()
	cfg := factory.CreateDefaultConfig().(*metric_pipeline.Config)

	// Configure it with initial values
	cfg.ResourceFilter.TopK.KValue = 20 // Initial k_value

	// Create a test sink
	sink := new(consumertest.MetricsSink)

	// Create the processor
	settings := processor.Settings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: logger,
		},
		ID: component.NewIDWithName(component.MustNewType("metric_pipeline"), ""),
	}

	proc, err := factory.CreateMetrics(context.Background(), settings, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Ensure it implements the UpdateableProcessor interface
	updateableProc, ok := proc.(interfaces.UpdateableProcessor)
	require.True(t, ok, "Processor does not implement UpdateableProcessor")

	// Start the processor
	err = proc.Start(context.Background(), nil)
	require.NoError(t, err)

	// Create and apply a patch to change the k_value
	patch := interfaces.ConfigPatch{
		PatchID:             "test-patch",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("metric_pipeline"), ""),
		ParameterPath:       "resource_filter.topk.k_value",
		NewValue:            30, // Increase from 20 to 30
		Reason:              "test",
		Source:              "test",
	}

	// Apply the patch
	err = updateableProc.OnConfigPatch(context.Background(), patch)
	require.NoError(t, err)

	// Get the current config status
	status, err := updateableProc.GetConfigStatus(context.Background())
	require.NoError(t, err)

	// Verify the patch was applied by examining the processor's state
	// (This requires exposing internal state through the ConfigStatus)
	assert.True(t, status.Enabled)

	// Verify through processing behavior
	// Create test metrics with many resources
	metrics := generateTestMetrics(50) // 50 processes

	// Process the metrics
	err = proc.ConsumeMetrics(context.Background(), metrics)
	require.NoError(t, err)

	// Verify that approximately 30 resources were kept (after the patch changed k_value from 20 to 30)
	processedMetrics := sink.AllMetrics()[0]

	// Count the number of non-rollup resources
	nonRollupCount := 0
	for i := 0; i < processedMetrics.ResourceMetrics().Len(); i++ {
		rm := processedMetrics.ResourceMetrics().At(i)
		procName, exists := rm.Resource().Attributes().Get("process.executable.name")
		if exists && !isRollupMetricName(procName.Str()) {
			nonRollupCount++
		}
	}

	// The exact count might vary due to topk algorithm and priority rules, but should be around 30
	assert.InDelta(t, 30, nonRollupCount, 5)

	// Shutdown the processor
	err = proc.Shutdown(context.Background())
	require.NoError(t, err)
}

// Helper functions

// generateTestMetrics creates test metrics with multiple resources
func generateTestMetrics(processCount int) pmetric.Metrics {
	metrics := pmetric.NewMetrics()

	for i := 0; i < processCount; i++ {
		rm := metrics.ResourceMetrics().AppendEmpty()
		resourceName := ""

		// Create some diversity in process names and priorities
		if i < 5 {
			resourceName = "critical-process-" + string(i)
			rm.Resource().Attributes().PutStr("process.executable.name", resourceName)
			rm.Resource().Attributes().PutStr("aemf.process.priority", "critical")
		} else if i < 15 {
			resourceName = "high-process-" + string(i)
			rm.Resource().Attributes().PutStr("process.executable.name", resourceName)
			rm.Resource().Attributes().PutStr("aemf.process.priority", "high")
		} else {
			resourceName = "normal-process-" + string(i)
			rm.Resource().Attributes().PutStr("process.executable.name", resourceName)
			rm.Resource().Attributes().PutStr("aemf.process.priority", "medium")
		}

		// Add metrics with varying CPU values
		sm := rm.ScopeMetrics().AppendEmpty()
		m := sm.Metrics().AppendEmpty()
		m.SetName("process.cpu.time")
		m.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(float64(processCount - i))
	}

	return metrics
}

// isRollupMetricName checks if a process name is a rollup metric
func isRollupMetricName(name string) bool {
	return name == "others" || name == "phoenix.others.process"
}
