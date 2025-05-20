// Package policy contains end-to-end tests for SA-OMF policy and configuration patch handling.
package policy

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"

	"github.com/deepaucksharma/Phoenix/internal/extension/pic_control_ext"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
	"github.com/deepaucksharma/Phoenix/test/testutils"
)

// TestNonUpdateableTarget verifies that pic_control correctly handles and rejects
// configuration patches targeting components that don't implement the UpdateableProcessor interface.
// Scenario: POL-06
func TestNonUpdateableTarget(t *testing.T) {
	// Skip if running in short mode - this is an integration test
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a test metrics collector to capture emitted metrics
	metricsCollector := metrics.NewMetricsCollector()

	// Create the pic_control extension with test configuration
	extensionFactory := pic_control_ext.NewFactory()
	defaultConfig := extensionFactory.CreateDefaultConfig().(*pic_control_ext.Config)
	
	// Configure with test settings
	defaultConfig.PolicyFile = "testdata/valid_policy.yaml"
	defaultConfig.WatchPolicy = false // Disable watching to avoid file system dependencies
	defaultConfig.MetricsEmitter = metricsCollector
	
	// Create the extension
	extension, err := pic_control_ext.NewPICControlExtension(defaultConfig, component.TelemetrySettings{})
	require.NoError(t, err, "Failed to create pic_control extension")
	
	// Start the extension
	mockHost := testutils.NewMockHost()
	err = extension.Start(ctx, mockHost)
	require.NoError(t, err, "Failed to start pic_control extension")
	defer extension.Shutdown(ctx)

	// Register a standard updateable processor (for comparison)
	mockTopK := testutils.NewMockUpdateableProcessor("adaptive_topk")
	mockTopK.SetParameter("k_value", 10)
	
	err = extension.RegisterUpdateableProcessor(mockTopK)
	require.NoError(t, err, "Failed to register mock processor")

	// Create a patch targeting a non-existent component (batch processor from standard_components)
	nonUpdateablePatch := interfaces.ConfigPatch{
		PatchID:             "test-non-updateable-patch",
		TargetProcessorName: component.NewIDWithName("processor", "batch"),
		ParameterPath:       "batch_size",
		NewValue:            1000,
		Reason:              "Testing non-updateable target handling",
		Severity:            "normal",
		Source:              "test",
		Timestamp:           time.Now().Unix(),
		TTLSeconds:          300,
	}

	// Apply the patch targeting non-updateable component
	err = extension.ApplyConfigPatch(ctx, nonUpdateablePatch)
	
	// Verify it was rejected with appropriate error
	assert.Error(t, err, "Patch targeting non-updateable component should be rejected")
	assert.Contains(t, err.Error(), "not registered", "Error should mention component not being registered")

	// Verify the proper metric was emitted
	metrics := metricsCollector.GetMetrics()
	foundTargetError := false
	
	for _, metric := range metrics {
		if metric.Name == "aemf_patch_target_not_found_total" {
			foundTargetError = true
			break
		}
	}
	
	assert.True(t, foundTargetError, "patch_target_not_found_total metric should be emitted")

	// Verify the updateable processor is still properly registered
	validPatch := interfaces.ConfigPatch{
		PatchID:             "test-valid-target-patch",
		TargetProcessorName: component.NewIDWithName("processor", "adaptive_topk"),
		ParameterPath:       "k_value",
		NewValue:            20,
		Reason:              "Testing valid target handling",
		Severity:            "normal",
		Source:              "test",
		Timestamp:           time.Now().Unix(),
		TTLSeconds:          300,
	}
	
	// Apply the valid patch
	err = extension.ApplyConfigPatch(ctx, validPatch)
	assert.NoError(t, err, "Valid patch should be accepted")
	
	// Verify the parameter was updated on the valid target
	assert.Equal(t, 20, mockTopK.GetParameter("k_value"), "Valid processor parameter should be updated")

	// Test targeting a processor that exists in the policy but hasn't been registered
	unregisteredPatch := interfaces.ConfigPatch{
		PatchID:             "test-unregistered-processor-patch",
		TargetProcessorName: component.NewIDWithName("processor", "priority_tagger"),
		ParameterPath:       "priority_threshold",
		NewValue:            0.8,
		Reason:              "Testing unregistered processor handling",
		Severity:            "normal",
		Source:              "test",
		Timestamp:           time.Now().Unix(),
		TTLSeconds:          300,
	}

	// Apply the patch targeting unregistered processor
	err = extension.ApplyConfigPatch(ctx, unregisteredPatch)
	
	// Verify it was rejected with appropriate error
	assert.Error(t, err, "Patch targeting unregistered processor should be rejected")
	assert.Contains(t, err.Error(), "not registered", "Error should mention processor not being registered")
}