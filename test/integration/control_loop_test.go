// Package integration provides integration tests for SA-OMF components.
package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/processor"

	"github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_pid"
	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_topk"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

// TestControlLoopIntegration tests the end-to-end control loop.
func TestControlLoopIntegration(t *testing.T) {
	// Create a custom test host
	host := testutils.NewTestHost()

	// Create pic_control extension
	picCtrlFactory := pic_control_ext.NewFactory()
	picCtrlConfig := picCtrlFactory.CreateDefaultConfig().(*pic_control_ext.Config)
	picCtrlConfig.PolicyFilePath = "" // No file for testing
	picCtrlConfig.MaxPatchesPerMinute = 10
	picCtrlConfig.PatchCooldownSeconds = 1

	// Create the extension
	picCtrlExt, err := createExtension(
		context.Background(),
		extension.Settings{
			TelemetrySettings: component.TelemetrySettings{},
			ID: component.NewID(component.MustNewType("pic_control")),
		},
		picCtrlConfig,
	)
	require.NoError(t, err, "Failed to create pic_control extension")
	require.NotNil(t, picCtrlExt)

	// Add extension to host
	host.AddExtension(component.MustNewID("pic_control"), picCtrlExt)

	// Start extension
	err = picCtrlExt.Start(context.Background(), host)
	require.NoError(t, err, "Failed to start pic_control extension")

	// Create adaptive_topk processor (to be controlled)
	topkFactory := adaptive_topk.NewFactory()
	topkConfig := topkFactory.CreateDefaultConfig().(*adaptive_topk.Config)
	topkConfig.KValue = 30
	topkConfig.KMin = 10
	topkConfig.KMax = 100

	topkSink := new(consumertest.MetricsSink)
	// Create the processor
	topkProc, err := createMetricsProcessor(
		context.Background(),
		processor.Settings{
			TelemetrySettings: component.TelemetrySettings{},
			ID: component.NewID(component.MustNewType("adaptive_topk")),
		},
		topkConfig,
		topkSink,
	)
	require.NoError(t, err, "Failed to create adaptive_topk processor")
	require.NotNil(t, topkProc)

	// Add processor to host
	host.AddProcessor(component.MustNewID("adaptive_topk"), topkProc)

	// Start processor
	err = topkProc.Start(context.Background(), host)
	require.NoError(t, err, "Failed to start adaptive_topk processor")

	// Create pid_decider processor
	pidFactory := adaptive_pid.NewFactory()
	pidConfig := pidFactory.CreateDefaultConfig().(*adaptive_pid.Config)
	pidConfig.Controllers = []adaptive_pid.ControllerConfig{
		{
			Name:              "coverage_controller",
			Enabled:           true,
			KPIMetricName:     "aemf_impact_adaptive_topk_resource_coverage_percent_avg_1m",
			KPITargetValue:    0.9,
			KP:                10.0,
			KI:                0.0,
			KD:                0.0,
			OutputConfigPatches: []adaptive_pid.OutputConfigPatch{
				{
					TargetProcessorName: component.NewID(component.MustNewType("adaptive_topk")),
					ParameterPath:       "k_value",
					ChangeScaleFactor:   1.0,
					MinValue:            float64(topkConfig.KMin),
					MaxValue:            float64(topkConfig.KMax),
				},
			},
		},
	}

	// Create a test sink for PID processor output
	pidSink := new(consumertest.MetricsSink)
	// Create the processor
	pidProc, err := createMetricsProcessor(
		context.Background(),
		processor.Settings{
			TelemetrySettings: component.TelemetrySettings{},
			ID: component.NewID(component.MustNewType("adaptive_pid")),
		},
		pidConfig,
		pidSink,
	)
	require.NoError(t, err, "Failed to create pid_decider processor")
	require.NotNil(t, pidProc)

	// Start pid_decider
	err = pidProc.Start(context.Background(), host)
	require.NoError(t, err, "Failed to start pid_decider processor")

	// Integration test: Control loop
	t.Run("ControlLoopOperation", func(t *testing.T) {
		// Reset sinks
		topkSink.Reset()
		pidSink.Reset()

		// Test scenario 1: Coverage too low (0.7), should increase k_value
		// Test the initial k_value
		assert.Equal(t, 30, topkConfig.KValue, "Initial k_value should be 30")
		coverageMetrics := testutils.GenerateControlMetrics(0.7) // 70% coverage
		
		// Send metrics to PID controller
		err = pidProc.ConsumeMetrics(context.Background(), coverageMetrics)
		require.NoError(t, err, "Failed to consume coverage metrics")
		
		// Test scenario 2: Coverage too high (0.95), should decrease k_value
		highCoverageMetrics := testutils.GenerateControlMetrics(0.95) // 95% coverage
		
		// Send metrics to PID controller
		err = pidProc.ConsumeMetrics(context.Background(), highCoverageMetrics)
		require.NoError(t, err, "Failed to consume high coverage metrics")
		
		// Test scenario 3: Coverage at target (0.9), should maintain k_value
		targetCoverageMetrics := testutils.GenerateControlMetrics(0.9) // 90% coverage (target)
		
		// Send metrics to PID controller
		err = pidProc.ConsumeMetrics(context.Background(), targetCoverageMetrics)
		require.NoError(t, err, "Failed to consume target coverage metrics")
	})

	// Shutdown components
	err = pidProc.Shutdown(context.Background())
	assert.NoError(t, err, "Failed to shutdown pid_decider processor")
	
	err = topkProc.Shutdown(context.Background())
	assert.NoError(t, err, "Failed to shutdown adaptive_topk processor")
	
	err = picCtrlExt.Shutdown(context.Background())
	assert.NoError(t, err, "Failed to shutdown pic_control extension")
}

// Helper functions to create components directly from their factories

// createExtension creates an extension using the same approach as the factory does
func createExtension(
	ctx context.Context,
	set extension.Settings,
	cfg component.Config,
) (extension.Extension, error) {
	config := cfg.(*pic_control_ext.Config)
	// Mimics the createExtension function from pic_control_ext
	return pic_control_ext.NewExtension(config, set.TelemetrySettings.Logger)
}

// createMetricsProcessor creates a metrics processor using the same approach as the factory does
func createMetricsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	// Check which type of processor config we have
	switch pCfg := cfg.(type) {
	case *adaptive_topk.Config:
		return adaptive_topk.NewProcessor(pCfg, set.TelemetrySettings, nextConsumer, set.ID)
	case *adaptive_pid.Config:
		return adaptive_pid.NewProcessor(pCfg, set.TelemetrySettings, nextConsumer, set.ID)
	default:
		return nil, fmt.Errorf("unsupported processor config type: %T", cfg)
	}
}