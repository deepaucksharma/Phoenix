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

// TestPatchRateLimiting verifies that pic_control correctly applies and enforces
// rate limiting for configuration patches.
// Scenario: POL-07
func TestPatchRateLimiting(t *testing.T) {
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
	
	// Configure with test settings and low rate limit for testing
	defaultConfig.PolicyFile = "testdata/valid_policy.yaml"
	defaultConfig.WatchPolicy = false // Disable watching to avoid file system dependencies
	defaultConfig.MetricsEmitter = metricsCollector
	defaultConfig.MaxPatchesPerMinute = 2 // Set a low limit for testing
	defaultConfig.PatchCooldownSeconds = 2 // Set a short cooldown for testing
	
	// Create the extension
	extension, err := pic_control_ext.NewPICControlExtension(defaultConfig, component.TelemetrySettings{})
	require.NoError(t, err, "Failed to create pic_control extension")
	
	// Start the extension
	mockHost := testutils.NewMockHost()
	err = extension.Start(ctx, mockHost)
	require.NoError(t, err, "Failed to start pic_control extension")
	defer extension.Shutdown(ctx)

	// Register an updateable processor for testing
	mockTopK := testutils.NewMockUpdateableProcessor("adaptive_topk")
	mockTopK.SetParameter("k_value", 10)
	
	err = extension.RegisterUpdateableProcessor(mockTopK)
	require.NoError(t, err, "Failed to register mock processor")

	// Create a valid patch to apply
	validPatch := func(id string, value int) interfaces.ConfigPatch {
		return interfaces.ConfigPatch{
			PatchID:             id,
			TargetProcessorName: component.NewIDWithName(component.MustNewType("processor"), "adaptive_topk"),
			ParameterPath:       "k_value",
			NewValue:            value,
			Reason:              "Testing rate limiting",
			Severity:            "normal",
			Source:              "test",
			Timestamp:           time.Now().Unix(),
			TTLSeconds:          300,
		}
	}

	// Apply first patch - should succeed
	err = extension.ApplyConfigPatch(ctx, validPatch("patch-1", 20))
	assert.NoError(t, err, "First patch should be accepted")
	value, exists := mockTopK.GetParameter("k_value")
	assert.True(t, exists, "k_value parameter should exist")
	assert.Equal(t, 20, value, "First patch should be applied")
	
	// Apply second patch - should succeed
	err = extension.ApplyConfigPatch(ctx, validPatch("patch-2", 30))
	assert.NoError(t, err, "Second patch should be accepted")
	value, exists = mockTopK.GetParameter("k_value")
	assert.True(t, exists, "k_value parameter should exist")
	assert.Equal(t, 30, value, "Second patch should be applied")
	
	// Apply third patch - should be rate limited
	err = extension.ApplyConfigPatch(ctx, validPatch("patch-3", 40))
	assert.Error(t, err, "Third patch should be rate limited")
	value, exists = mockTopK.GetParameter("k_value")
	assert.True(t, exists, "k_value parameter should exist")
	assert.Equal(t, 30, value, "Third patch should not be applied")
	
	// Verify the rate limit metric was emitted
	metrics := metricsCollector.GetMetricsByName("aemf_patch_rate_limited_total")
	assert.GreaterOrEqual(t, len(metrics), 1, "Rate limit metric should be emitted")

	// Wait for cooldown to expire
	time.Sleep(3 * time.Second)
	
	// Apply a new patch after cooldown - should succeed
	err = extension.ApplyConfigPatch(ctx, validPatch("patch-4", 50))
	assert.NoError(t, err, "Patch after cooldown should be accepted")
	value, exists = mockTopK.GetParameter("k_value")
	assert.True(t, exists, "k_value parameter should exist")
	assert.Equal(t, 50, value, "Patch after cooldown should be applied")
	
	// Test priority handling - urgent patches should bypass rate limiting
	urgentPatch := validPatch("urgent-patch", 100)
	urgentPatch.Severity = "urgent"
	
	// Apply urgent patch - should bypass rate limiting
	err = extension.ApplyConfigPatch(ctx, urgentPatch)
	assert.NoError(t, err, "Urgent patch should bypass rate limiting")
	value, exists = mockTopK.GetParameter("k_value")
	assert.True(t, exists, "k_value parameter should exist")
	assert.Equal(t, 100, value, "Urgent patch should be applied")
}