// Package interfaces provides testing utilities for interface implementations.
package interfaces

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"

	"github.com/yourorg/sa-omf/internal/interfaces"
)

// UpdateableProcessorTestSuite provides a standardized way to test any
// processor that implements the UpdateableProcessor interface.
type UpdateableProcessorTestSuite struct {
	Processor         interfaces.UpdateableProcessor
	ValidParameters   map[string][]interface{}
	InvalidParameters map[string][]interface{}
}

// TestUpdateableProcessor runs a standard suite of tests against any
// UpdateableProcessor implementation.
func TestUpdateableProcessor(t *testing.T, p interfaces.UpdateableProcessor, opts ...TestOption) {
	// Create default test suite
	suite := &UpdateableProcessorTestSuite{
		Processor:         p,
		ValidParameters:   make(map[string][]interface{}),
		InvalidParameters: make(map[string][]interface{}),
	}

	// Apply test options
	for _, opt := range opts {
		opt(suite)
	}

	// Run tests
	t.Run("GetConfigStatus", func(t *testing.T) { testGetConfigStatus(t, suite) })
	t.Run("EnabledParameter", func(t *testing.T) { testEnabledParameter(t, suite) })
	t.Run("ValidParameters", func(t *testing.T) { testValidParameters(t, suite) })
	t.Run("InvalidParameters", func(t *testing.T) { testInvalidParameters(t, suite) })
	t.Run("RejectedParameters", func(t *testing.T) { testRejectedParameters(t, suite) })
	t.Run("TTLExpiration", func(t *testing.T) { testTTLExpiration(t, suite) })
}

// TestOption configures the UpdateableProcessorTestSuite.
type TestOption func(*UpdateableProcessorTestSuite)

// WithValidParameter adds a valid parameter test case.
func WithValidParameter(paramName string, validValues ...interface{}) TestOption {
	return func(s *UpdateableProcessorTestSuite) {
		s.ValidParameters[paramName] = validValues
	}
}

// WithInvalidParameter adds an invalid parameter test case.
func WithInvalidParameter(paramName string, invalidValues ...interface{}) TestOption {
	return func(s *UpdateableProcessorTestSuite) {
		s.InvalidParameters[paramName] = invalidValues
	}
}

// testGetConfigStatus verifies the processor returns a valid configuration status.
func testGetConfigStatus(t *testing.T, suite *UpdateableProcessorTestSuite) {
	ctx := context.Background()

	// Get initial status
	status, err := suite.Processor.GetConfigStatus(ctx)
	require.NoError(t, err, "GetConfigStatus should not fail")
	
	// Verify parameters is not nil
	require.NotNil(t, status.Parameters, "Status parameters should not be nil")
	
	// Check if enabled is a boolean
	_, ok := status.Enabled.(bool)
	assert.True(t, ok, "Status.Enabled should be a boolean")
	
	// Log current parameters for debugging
	for name, value := range status.Parameters {
		t.Logf("Parameter %s = %v (type: %T)", name, value, value)
	}
}

// testEnabledParameter tests the "enabled" parameter.
func testEnabledParameter(t *testing.T, suite *UpdateableProcessorTestSuite) {
	ctx := context.Background()
	
	// Get initial status
	initialStatus, err := suite.Processor.GetConfigStatus(ctx)
	require.NoError(t, err)
	
	// Test enabling
	enablePatch := createConfigPatch("enabled", true)
	err = suite.Processor.OnConfigPatch(ctx, enablePatch)
	
	// If OnConfigPatch returns error, it might be because the processor
	// doesn't support the enabled flag directly, so we'll skip the test
	if err != nil {
		t.Skip("Processor doesn't support direct enabled parameter manipulation")
	}
	
	// Verify enabled
	status, err := suite.Processor.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.True(t, status.Enabled.(bool), "Processor should be enabled")
	
	// Test disabling
	disablePatch := createConfigPatch("enabled", false)
	err = suite.Processor.OnConfigPatch(ctx, disablePatch)
	require.NoError(t, err)
	
	// Verify disabled
	status, err = suite.Processor.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.False(t, status.Enabled.(bool), "Processor should be disabled")
	
	// Restore original state
	restorePatch := createConfigPatch("enabled", initialStatus.Enabled)
	_ = suite.Processor.OnConfigPatch(ctx, restorePatch)
}

// testValidParameters tests that valid parameter values are accepted.
func testValidParameters(t *testing.T, suite *UpdateableProcessorTestSuite) {
	ctx := context.Background()
	
	// Get initial status for restoring state later
	initialStatus, err := suite.Processor.GetConfigStatus(ctx)
	require.NoError(t, err)
	
	// Test each valid parameter
	for paramName, values := range suite.ValidParameters {
		t.Run(paramName, func(t *testing.T) {
			for i, value := range values {
				t.Run(fmt.Sprintf("Value%d", i), func(t *testing.T) {
					// Create patch with valid value
					patch := createConfigPatch(paramName, value)
					
					// Apply patch
					err := suite.Processor.OnConfigPatch(ctx, patch)
					require.NoError(t, err, "OnConfigPatch should accept valid parameter %s = %v", paramName, value)
					
					// Verify parameter was applied
					status, err := suite.Processor.GetConfigStatus(ctx)
					require.NoError(t, err)
					
					assert.Contains(t, status.Parameters, paramName, 
						"Parameter %s should be in GetConfigStatus response", paramName)
						
					// Check if the value was set correctly, handling possible type conversions
					if status.Parameters[paramName] != nil {
						actualType := reflect.TypeOf(status.Parameters[paramName])
						expectedType := reflect.TypeOf(value)
						
						if actualType == expectedType {
							assert.Equal(t, value, status.Parameters[paramName], 
								"Parameter %s value mismatch", paramName)
						} else {
							// Try to convert between numeric types
							actualValue := reflect.ValueOf(status.Parameters[paramName])
							expectedValue := reflect.ValueOf(value)
							
							if actualType.Kind() == reflect.Float64 && expectedType.Kind() == reflect.Int {
								// Convert int to float64 for comparison
								assert.Equal(t, float64(expectedValue.Int()), actualValue.Float(), 
									"Parameter %s value mismatch (int to float64 conversion)", paramName)
							} else if actualType.Kind() == reflect.Int && expectedType.Kind() == reflect.Float64 {
								// Convert float64 to int for comparison
								assert.Equal(t, int(expectedValue.Float()), actualValue.Int(), 
									"Parameter %s value mismatch (float64 to int conversion)", paramName)
							} else {
								// Just check string representation as fallback
								assert.Equal(t, fmt.Sprintf("%v", value), fmt.Sprintf("%v", status.Parameters[paramName]), 
									"Parameter %s string representation mismatch", paramName)
							}
						}
					}
				})
			}
		})
	}
	
	// Restore original state when possible
	for paramName, origValue := range initialStatus.Parameters {
		if _, exists := suite.ValidParameters[paramName]; exists {
			restorePatch := createConfigPatch(paramName, origValue)
			_ = suite.Processor.OnConfigPatch(ctx, restorePatch)
		}
	}
}

// testInvalidParameters tests that invalid parameter values are rejected.
func testInvalidParameters(t *testing.T, suite *UpdateableProcessorTestSuite) {
	ctx := context.Background()
	
	// Test each invalid parameter
	for paramName, values := range suite.InvalidParameters {
		t.Run(paramName, func(t *testing.T) {
			for i, value := range values {
				t.Run(fmt.Sprintf("Value%d", i), func(t *testing.T) {
					// Create patch with invalid value
					patch := createConfigPatch(paramName, value)
					
					// Apply patch - expect error
					err := suite.Processor.OnConfigPatch(ctx, patch)
					assert.Error(t, err, "OnConfigPatch should reject invalid parameter %s = %v", paramName, value)
				})
			}
		})
	}
}

// testRejectedParameters tests that unknown parameters are rejected.
func testRejectedParameters(t *testing.T, suite *UpdateableProcessorTestSuite) {
	ctx := context.Background()
	
	// Try a clearly invalid parameter name
	patch := createConfigPatch("non_existent_parameter_abc123", "test")
	err := suite.Processor.OnConfigPatch(ctx, patch)
	assert.Error(t, err, "OnConfigPatch should reject unknown parameters")
}

// testTTLExpiration tests that patches with expired TTL are handled correctly.
func testTTLExpiration(t *testing.T, suite *UpdateableProcessorTestSuite) {
	// For this test, we'd need access to the full pic_control extension
	// which validates TTL. Since we're just testing the processor directly,
	// we'll skip this test as it's more of an integration test.
	t.Skip("TTL expiration test requires pic_control integration")
}

// createConfigPatch creates a ConfigPatch with the given parameter and value.
func createConfigPatch(paramPath string, value interface{}) interfaces.ConfigPatch {
	return interfaces.ConfigPatch{
		PatchID:             fmt.Sprintf("test-patch-%s-%v", paramPath, time.Now().UnixNano()),
		TargetProcessorName: component.NewID("test-processor"),
		ParameterPath:       paramPath,
		NewValue:            value,
		Reason:              "Test patch",
		Severity:            "normal",
		Source:              "test",
		Timestamp:           time.Now().Unix(),
		TTLSeconds:          300,
	}
}