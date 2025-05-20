// Package safety contains tests for the safety mechanisms in SA-OMF.
package safety

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"

	"github.com/deepaucksharma/Phoenix/internal/control/safety"
	"github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

// TestOverrideThresholds verifies that safety thresholds can be temporarily overridden
// in specific scenarios, and that the system correctly reverts to normal thresholds
// after override expiry.
// Scenario: SAFETY-03
func TestOverrideThresholds(t *testing.T) {
	// Skip if running in short mode - this is an integration test
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a test metrics collector to capture emitted metrics
	metricsCollector := metrics.NewMetricsCollector()

	// Set up the safety monitor with standard thresholds
	safetyConfig := &safety.Config{
		CPUUsageThresholdMCores: 500, // 0.5 cores
		MemoryThresholdMiB:      200, // 200 MiB
		SafeModeCooldownSeconds: 5,   // Short for testing
		OverrideExpirySeconds:   10,  // Override expiry after 10 seconds
		MetricsCheckIntervalMs:  100, // Check frequently for testing
	}

	safetyMonitor := safety.NewMonitor(safetyConfig, component.TelemetrySettings{})
	require.NotNil(t, safetyMonitor, "Failed to create safety monitor")

	// Mock the safety monitor's metrics readings for testing
	mockMetricsProvider := &testutils.MockMetricsProvider{
		CPUUsageMCores: 300, // Below threshold
		MemoryUsageMiB: 150, // Below threshold
	}
	safetyMonitor.SetMetricsProvider(mockMetricsProvider)

	// Start the safety monitor
	err := safetyMonitor.Start(ctx, nil)
	require.NoError(t, err, "Failed to start safety monitor")
	defer safetyMonitor.Shutdown(ctx)

	// Set up the pic_control extension
	picControlConfig := pic_control_ext.NewFactory().CreateDefaultConfig().(*pic_control_ext.Config)
	picControlConfig.PolicyFile = "../policy/testdata/valid_policy.yaml"
	picControlConfig.WatchPolicy = false // Disable watching for the test
	picControlConfig.MetricsEmitter = metricsCollector

	picControlExt, err := pic_control_ext.NewPICControlExtension(picControlConfig, component.TelemetrySettings{})
	require.NoError(t, err, "Failed to create pic_control extension")

	// Start the pic_control extension
	mockHost := testutils.NewMockHost()
	err = picControlExt.Start(ctx, mockHost)
	require.NoError(t, err, "Failed to start pic_control extension")
	defer picControlExt.Shutdown(ctx)

	// Register the safety monitor with pic_control
	picControlExt.RegisterSafetyMonitor(safetyMonitor)

	// Create a processor to test with
	mockProcessor := testutils.NewMockUpdateableProcessor("test_processor")
	mockProcessor.SetParameter("value", 10)

	err = picControlExt.RegisterUpdateableProcessor(mockProcessor)
	require.NoError(t, err, "Failed to register mock processor")

	// Create a standard patch to apply
	validPatch := interfaces.ConfigPatch{
		PatchID:             "standard-patch",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("processor"), "test_processor"),
		ParameterPath:       "value",
		NewValue:            20,
		Reason:              "Standard patch",
		Severity:            "normal",
		Source:              "test",
		Timestamp:           time.Now().Unix(),
		TTLSeconds:          300,
	}

	// Verify the patch works normally
	err = picControlExt.ApplyConfigPatch(ctx, validPatch)
	assert.NoError(t, err, "Patch should be accepted under normal conditions")
	value, exists := mockProcessor.GetParameter("value")
	assert.True(t, exists, "value parameter should exist")
	assert.Equal(t, 20, value, "Parameter should be updated")

	// Now simulate resource exhaustion
	mockMetricsProvider.CPUUsageMCores = 600 // Above threshold
	mockMetricsProvider.MemoryUsageMiB = 150 // Still below memory threshold

	// Wait for safety monitor to detect high usage
	time.Sleep(500 * time.Millisecond)

	// Verify safe mode is activated
	assert.True(t, safetyMonitor.IsInSafeMode(), "Safe mode should be active")

	// Manually set safe mode in the pic_control extension for testing
	// In a real system, this would happen via the safety monitor's callback
	picControlExt.SetSafeMode(true)

	// Try to apply a normal patch - should be rejected
	validPatch.PatchID = "rejected-patch"
	validPatch.NewValue = 30
	err = picControlExt.ApplyConfigPatch(ctx, validPatch)
	assert.Error(t, err, "Patch should be rejected in safe mode")
	value, exists = mockProcessor.GetParameter("value")
	assert.True(t, exists, "value parameter should exist")
	assert.Equal(t, 20, value, "Parameter should not be updated in safe mode")

	// Now create an urgent patch with override safety flag
	urgentPatch := interfaces.ConfigPatch{
		PatchID:             "urgent-override-patch",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("processor"), "test_processor"),
		ParameterPath:       "value",
		NewValue:            40,
		Reason:              "Urgent override patch",
		Severity:            "urgent",
		Source:              "test",
		Timestamp:           time.Now().Unix(),
		TTLSeconds:          300,
		SafetyOverride:      true, // Critical flag to override safety
	}

	// Apply the urgent patch with safety override
	err = picControlExt.ApplyConfigPatch(ctx, urgentPatch)
	assert.NoError(t, err, "Urgent patch with safety override should be accepted")
	value, exists = mockProcessor.GetParameter("value")
	assert.True(t, exists, "value parameter should exist")
	assert.Equal(t, 40, value, "Parameter should be updated")

	// Verify threshold values were temporarily increased
	assert.Greater(t, safetyMonitor.GetCurrentCPUThreshold(), safetyConfig.CPUUsageThresholdMCores,
		"CPU threshold should be temporarily increased")

	// Set usage back to normal levels and wait for override to expire
	mockMetricsProvider.CPUUsageMCores = 300

	// Wait for override to expire
	time.Sleep(12 * time.Second)

	// Verify thresholds returned to normal
	assert.Equal(t, safetyConfig.CPUUsageThresholdMCores, safetyMonitor.GetCurrentCPUThreshold(),
		"CPU threshold should return to normal after expiry")

	// Verify safe mode is deactivated after cooldown
	assert.False(t, safetyMonitor.IsInSafeMode(), "Safe mode should be inactive")

	// Verify normal patches work again
	validPatch.PatchID = "normal-again-patch"
	validPatch.NewValue = 50
	err = picControlExt.ApplyConfigPatch(ctx, validPatch)
	assert.NoError(t, err, "Patch should be accepted after safe mode deactivation")
	value, exists = mockProcessor.GetParameter("value")
	assert.True(t, exists, "value parameter should exist")
	assert.Equal(t, 50, value, "Parameter should be updated")
}
