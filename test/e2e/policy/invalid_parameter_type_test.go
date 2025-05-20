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

// TestInvalidParameterType verifies that pic_control correctly validates and rejects
// patches with incorrect parameter types.
// Scenario: POL-05
func TestInvalidParameterType(t *testing.T) {
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
	err = extension.Start(ctx, testutils.NewMockHost())
	require.NoError(t, err, "Failed to start pic_control extension")
	defer extension.Shutdown(ctx)

	// Register mock updateable processors
	mockTopK := testutils.NewMockUpdateableProcessor("adaptive_topk")
	mockTopK.SetParameter("k_value", 10) // k_value is an integer parameter
	
	err = extension.RegisterUpdateableProcessor(mockTopK)
	require.NoError(t, err, "Failed to register mock processor")

	// Create an invalid patch with wrong type (string instead of int)
	invalidPatch := interfaces.ConfigPatch{
		PatchID:             "test-invalid-type-patch",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("processor"), "adaptive_topk"),
		ParameterPath:       "k_value",
		NewValue:            "20", // String value for an integer parameter
		Reason:              "Testing invalid type handling",
		Severity:            "normal",
		Source:              "test",
		Timestamp:           time.Now().Unix(),
		TTLSeconds:          300,
	}

	// Apply the invalid patch
	err = extension.ApplyConfigPatch(ctx, invalidPatch)
	
	// Verify it was rejected due to type mismatch
	assert.Error(t, err, "Invalid type patch should be rejected")
	assert.Contains(t, err.Error(), "type", "Error should mention type mismatch")

	// Verify the validation metric was incremented
	metrics := metricsCollector.GetMetrics()
	foundValidationFailure := false
	
	for _, metric := range metrics {
		if metric.Name == "aemf_patch_validation_failed_total" {
			foundValidationFailure = true
			break
		}
	}
	
	assert.True(t, foundValidationFailure, "patch_validation_failed_total metric should be emitted")

	// Verify the processor's value didn't change
	value, exists := mockTopK.GetParameter("k_value")
	assert.True(t, exists, "k_value parameter should exist")
	assert.Equal(t, 10, value, "Processor parameter should remain unchanged")
	
	// Now try with a valid patch to verify normal operation
	validPatch := interfaces.ConfigPatch{
		PatchID:             "test-valid-type-patch",
		TargetProcessorName: component.NewIDWithName(component.MustNewType("processor"), "adaptive_topk"),
		ParameterPath:       "k_value",
		NewValue:            30, // Correct integer type
		Reason:              "Testing valid type handling",
		Severity:            "normal",
		Source:              "test",
		Timestamp:           time.Now().Unix(),
		TTLSeconds:          300,
	}
	
	// Apply the valid patch
	err = extension.ApplyConfigPatch(ctx, validPatch)
	assert.NoError(t, err, "Valid patch should be accepted")
	
	// Verify the parameter was updated
	value, exists = mockTopK.GetParameter("k_value")
	assert.True(t, exists, "k_value parameter should exist")
	assert.Equal(t, 30, value, "Processor parameter should be updated")
}