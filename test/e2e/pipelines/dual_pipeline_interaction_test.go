// Package pipelines contains tests for the dual-pipeline architecture.
package pipelines

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/internal/processor/adaptive_pid"
	"github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

// TestDualPipelineInteraction verifies that changes in the data pipeline are reflected in the 
// control pipeline, and that the control pipeline can correctly influence the data pipeline.
// Scenario: PIPE-06
func TestDualPipelineInteraction(t *testing.T) {
	// Skip if running in short mode - this is an integration test
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a test metrics collector to capture emitted metrics
	metricsCollector := metrics.NewMetricsCollector()

	// Set up the control pipeline components
	// 1. Create and configure the pic_control extension
	picControlConfig := pic_control_ext.NewFactory().CreateDefaultConfig().(*pic_control_ext.Config)
	picControlConfig.PolicyFile = "../policy/testdata/valid_policy.yaml"
	picControlConfig.WatchPolicy = false // Disable watching for the test
	picControlConfig.MetricsEmitter = metricsCollector
	
	picControlExt, err := pic_control_ext.NewPICControlExtension(picControlConfig, component.TelemetrySettings{})
	require.NoError(t, err, "Failed to create pic_control extension")
	
	// 2. Start the extension with a mock host
	mockHost := testutils.NewMockHost()
	err = picControlExt.Start(ctx, mockHost)
	require.NoError(t, err, "Failed to start pic_control extension")
	defer picControlExt.Shutdown(ctx)
	
	// 3. Create the adaptive_pid processor (part of the control pipeline)
	pidConfig := &adaptive_pid.Config{
		Controllers: []adaptive_pid.ControllerConfig{
			{
				Name:           "test_controller",
				Enabled:        true,
				KPIMetricName:  "coverage_score",
				KPITargetValue: 0.9,
				KP:             10,
				KI:             2,
				KD:             0,
				OutputConfigPatches: []adaptive_pid.OutputConfigPatch{
					{
						TargetProcessorName: "adaptive_topk",
						ParameterPath:       "k_value",
						ChangeScaleFactor:   1.0,
						MinValue:            5,
						MaxValue:            100,
					},
				},
			},
		},
	}
	
	pidProcessor, err := adaptive_pid.NewProcessor(
		pidConfig, 
		component.TelemetrySettings{}, 
		picControlExt, // Connect to pic_control_ext for patch submission
		component.NewIDWithName(component.MustNewType("processor"), "adaptive_pid"),
	)
	require.NoError(t, err, "Failed to create adaptive_pid processor")
	
	// Set up the data pipeline components
	// 1. Create a mock UpdateableProcessor representing adaptive_topk
	mockTopK := testutils.NewMockUpdateableProcessor("adaptive_topk")
	mockTopK.SetParameter("k_value", 10) // Initial k-value
	
	// 2. Register it with pic_control_ext
	err = picControlExt.RegisterUpdateableProcessor(mockTopK)
	require.NoError(t, err, "Failed to register mockTopK processor")
	
	// Now simulate a complete dual-pipeline interaction:
	
	// STEP 1: Generate low coverage score metrics for the control pipeline
	// This simulates data pipeline metrics showing poor coverage
	lowCoverageMetrics := createTestMetricsWithKPI(t, "coverage_score", 0.5) // Well below target of 0.9
	
	// Process these metrics through the control pipeline
	// The control pipeline should generate patches to increase k_value
	patches, err := pidProcessor.ProcessMetricsForTest(ctx, lowCoverageMetrics)
	require.NoError(t, err, "Failed to process metrics")
	require.NotEmpty(t, patches, "Should generate patches")
	
	// Verify the patch
	var kValuePatch *interfaces.ConfigPatch
	for i, p := range patches {
		if p.ParameterPath == "k_value" {
			kValuePatch = &patches[i]
			break
		}
	}
	require.NotNil(t, kValuePatch, "Should generate a patch for k_value")
	
	// Verify the patch correctly targets adaptive_topk
	assert.Equal(t, component.NewIDWithName(component.MustNewType("processor"), "adaptive_topk"), kValuePatch.TargetProcessorName)
	
	// Verify the patch increases k_value (to improve coverage)
	newKValue, ok := kValuePatch.NewValue.(float64)
	require.True(t, ok, "New value should be float64")
	assert.Greater(t, newKValue, 10.0, "New k_value should be greater than initial value")
	
	// STEP 2: Verify the data pipeline component was updated
	// Wait a moment for the patch to be applied
	time.Sleep(50 * time.Millisecond)
	
	// Check if the processor's parameter was updated
	updatedKValue, exists := mockTopK.GetParameter("k_value")
	assert.True(t, exists, "k_value parameter should exist")
	assert.Equal(t, newKValue, updatedKValue, "Processor's k_value should be updated")
	
	// STEP 3: Now simulate improved metrics after the adjustment
	// This simulates data pipeline responding to the control pipeline changes
	improvedCoverageMetrics := createTestMetricsWithKPI(t, "coverage_score", 0.85) // Closer to target
	
	// Process these metrics through the control pipeline
	patches, err = pidProcessor.ProcessMetricsForTest(ctx, improvedCoverageMetrics)
	require.NoError(t, err, "Failed to process metrics")
	
	// Verify we still get patches, but with smaller adjustments
	require.NotEmpty(t, patches, "Should generate patches")
	
	// Find k_value patch
	kValuePatch = nil
	for i, p := range patches {
		if p.ParameterPath == "k_value" {
			kValuePatch = &patches[i]
			break
		}
	}
	require.NotNil(t, kValuePatch, "Should generate a patch for k_value")
	
	// Verify the new change is smaller (since we're closer to target)
	newerKValue, ok := kValuePatch.NewValue.(float64)
	require.True(t, ok, "New value should be float64")
	
	// Calculate adjustment size for each change
	firstAdjustment := newKValue - 10.0
	secondAdjustment := newerKValue - newKValue
	
	// The absolute second adjustment should be smaller than the first
	assert.Less(t, abs(secondAdjustment), abs(firstAdjustment), 
		"Second adjustment should be smaller than first (closer to target)")
	
	// STEP 4: Finally, simulate reaching the target
	// This simulates the data pipeline fully responding to control pipeline changes
	targetMetrics := createTestMetricsWithKPI(t, "coverage_score", 0.9) // At target
	
	// Process these metrics
	patches, err = pidProcessor.ProcessMetricsForTest(ctx, targetMetrics)
	require.NoError(t, err, "Failed to process metrics")
	
	// We should still get patches, but with very small or zero adjustments
	// Find k_value patch
	kValuePatch = nil
	for i, p := range patches {
		if p.ParameterPath == "k_value" {
			kValuePatch = &patches[i]
			break
		}
	}
	
	// If we got a patch, the adjustment should be very small
	if kValuePatch != nil {
		finalKValue, ok := kValuePatch.NewValue.(float64)
		require.True(t, ok, "New value should be float64")
		
		finalAdjustment := finalKValue - newerKValue
		assert.Less(t, abs(finalAdjustment), abs(secondAdjustment),
			"Final adjustment should be smaller than second (at target)")
	}
	
	// Verify we emitted the appropriate metrics during this interaction
	kpiProcessedMetrics := metricsCollector.GetMetricsByName("aemf_adaptive_pid_kpi_processed_total")
	assert.NotEmpty(t, kpiProcessedMetrics, "Should emit KPI processed metrics")
	
	patchGeneratedMetrics := metricsCollector.GetMetricsByName("aemf_adaptive_pid_patch_generated_total")
	assert.NotEmpty(t, patchGeneratedMetrics, "Should emit patch generated metrics")
	
	// Finally, verify that the pic_control extension properly recorded the patches
	assert.GreaterOrEqual(t, len(metricsCollector.GetMetricsByName("aemf_ctrl_patch_applied_total")), 1, 
		"Should emit patch applied metrics")
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}