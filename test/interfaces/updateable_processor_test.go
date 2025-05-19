// Package interfaces provides test utilities for validating interface implementations.
package interfaces

import (
	"context"
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	
	"github.com/yourorg/sa-omf/internal/interfaces"
)

// TestUpdateableProcessor provides a reusable test suite for any
// processor implementing the UpdateableProcessor interface
func TestUpdateableProcessor(t *testing.T, processor interfaces.UpdateableProcessor) {
	ctx := context.Background()
	
	// Test GetConfigStatus
	status, err := processor.GetConfigStatus(ctx)
	require.NoError(t, err)
	require.NotNil(t, status.Parameters)
	
	// Get initial enabled state
	initialEnabled := status.Enabled
	
	// Test enabling/disabling
	enablePatch := interfaces.ConfigPatch{
		PatchID:             "test-enable-patch",
		TargetProcessorName: component.NewID("test"),
		ParameterPath:       "enabled",
		NewValue:            !initialEnabled,
		Reason:              "test enabling/disabling",
		Severity:            "normal",
		Source:              "test",
	}
	err = processor.OnConfigPatch(ctx, enablePatch)
	require.NoError(t, err, "Applying enable/disable patch should not fail")
	
	// Verify the enabled state changed
	status, err = processor.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, !initialEnabled, status.Enabled, "Enabled state should have toggled")
	
	// Restore original state
	restorePatch := interfaces.ConfigPatch{
		PatchID:             "test-restore-patch",
		TargetProcessorName: component.NewID("test"),
		ParameterPath:       "enabled",
		NewValue:            initialEnabled,
		Reason:              "restore original state",
		Severity:            "normal",
		Source:              "test",
	}
	err = processor.OnConfigPatch(ctx, restorePatch)
	require.NoError(t, err)
	
	// Verify state was restored
	status, err = processor.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, initialEnabled, status.Enabled, "Original enabled state should be restored")
}

// ValidateRequiredParams ensures that specific parameters are present in the processor's config
func ValidateRequiredParams(t *testing.T, processor interfaces.UpdateableProcessor, requiredParams []string) {
	ctx := context.Background()
	
	status, err := processor.GetConfigStatus(ctx)
	require.NoError(t, err)
	
	for _, param := range requiredParams {
		_, exists := status.Parameters[param]
		assert.True(t, exists, "Required parameter %s missing from config", param)
	}
}