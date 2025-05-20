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

	"github.com/deepaucksharma/Phoenix/internal/interfaces"
)

// TestPatch defines a test case for a configuration patch.
type TestPatch struct {
	Name          string
	Patch         interfaces.ConfigPatch
	ExpectedValue interface{}
}

// UpdateableProcessorSuite provides a standardized way to test any
// processor that implements the UpdateableProcessor interface.
type UpdateableProcessorSuite struct {
	Processor      interfaces.UpdateableProcessor
	ValidPatches   []TestPatch
	InvalidPatches []TestPatch
}

// UpdateableProcessorTestSuite provides a standardized way to test any
// processor that implements the UpdateableProcessor interface.
type UpdateableProcessorTestSuite struct {
	Processor         interfaces.UpdateableProcessor
	ValidParameters   map[string][]interface{}
	InvalidParameters map[string][]interface{}
}

// RunUpdateableProcessorTests runs tests for a processor using the new suite.
func RunUpdateableProcessorTests(t *testing.T, suite UpdateableProcessorSuite) {
	ctx := context.Background()
	
	// Test valid patches
	for _, testCase := range suite.ValidPatches {
		t.Run(testCase.Name, func(t *testing.T) {
			// Apply the patch
			err := suite.Processor.OnConfigPatch(ctx, testCase.Patch)
			require.NoError(t, err, "Valid patch should be accepted: %v", testCase.Patch)
			
			// Verify the result
			status, err := suite.Processor.GetConfigStatus(ctx)
			require.NoError(t, err, "GetConfigStatus should not fail after patch")
			
			// Check if the parameter was updated
			if testCase.Patch.ParameterPath == "enabled" {
				assert.Equal(t, testCase.ExpectedValue, status.Enabled, 
					"Enabled state should match expected value")
			} else {
				paramValue, found := status.Parameters[testCase.Patch.ParameterPath]
				assert.True(t, found, "Parameter %s should be in status", testCase.Patch.ParameterPath)
				assert.Equal(t, testCase.ExpectedValue, paramValue, 
					"Parameter %s should match expected value", testCase.Patch.ParameterPath)
			}
		})
	}
	
	// Test invalid patches
	for _, testCase := range suite.InvalidPatches {
		t.Run(testCase.Name, func(t *testing.T) {
			// Apply the patch - should fail
			err := suite.Processor.OnConfigPatch(ctx, testCase.Patch)
			assert.Error(t, err, "Invalid patch should be rejected: %v", testCase.Patch)
		})
	}
}

// Original RunUpdateableProcessorTests runs a standard suite of tests against any
// UpdateableProcessor implementation.
func RunUpdateableProcessorTestsOriginal(t *testing.T, p interfaces.UpdateableProcessor, opts ...TestOption) {
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
	
	// Enabled should be a boolean
	assert.IsType(t, bool(false), status.Enabled, "Status.Enabled should be a boolean")
	
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
	assert.True(t, status.Enabled, "Processor should be enabled")
	
	// Test disabling
	disablePatch := createConfigPatch("enabled", false)
	err = suite.Processor.OnConfigPatch(ctx, disablePatch)
	require.NoError(t, err)
	
	// Verify disabled
	status, err = suite.Processor.GetConfigStatus(ctx)
	require.NoError(t, err)
	assert.False(t, status.Enabled, "Processor should be disabled")
	
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
						compareValues(t, value, status.Parameters[paramName], paramName)
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

// compareValues compares expected and actual values with proper type conversion when needed
func compareValues(t *testing.T, expected, actual interface{}, paramName string) bool {
	if expected == nil || actual == nil {
		return assert.Equal(t, expected, actual, "Parameter %s value mismatch", paramName)
	}

	expectedType := reflect.TypeOf(expected)
	actualType := reflect.TypeOf(actual)
	
	// If types match directly, use direct comparison
	if expectedType == actualType {
		return assert.Equal(t, expected, actual, "Parameter %s value mismatch", paramName)
	}
	
	// Handle numeric conversions
	expectedValue := reflect.ValueOf(expected)
	actualValue := reflect.ValueOf(actual)
	
	// Check if both are numeric types
	if isNumeric(expectedType.Kind()) && isNumeric(actualType.Kind()) {
		// Convert both to float64 for comparison
		expectedFloat := convertToFloat64(expectedValue)
		actualFloat := convertToFloat64(actualValue)
		
		// For integer types, ensure we don't lose precision by comparing as integers if both are whole numbers
		if isInteger(expectedType.Kind()) && isInteger(actualType.Kind()) {
			return assert.Equal(t, int64(expectedFloat), int64(actualFloat), 
				"Parameter %s value mismatch (integer conversion)", paramName)
		}
		
		// Allow small floating point differences
		return assert.InDelta(t, expectedFloat, actualFloat, 1e-6, 
			"Parameter %s value mismatch (float conversion)", paramName)
	}
	
	// Handle strings and string convertible types
	if (expectedType.Kind() == reflect.String || actualType.Kind() == reflect.String) {
		expectedStr := fmt.Sprintf("%v", expected)
		actualStr := fmt.Sprintf("%v", actual)
		return assert.Equal(t, expectedStr, actualStr, 
			"Parameter %s string representation mismatch", paramName)
	}
	
	// If we get here, types are incompatible - log details and fail
	t.Logf("Type mismatch for parameter %s: expected %T but got %T", 
		paramName, expected, actual)
	return assert.Fail(t, "Parameter types are incompatible")
}

// isNumeric checks if a reflect.Kind is a numeric type
func isNumeric(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// isInteger checks if a reflect.Kind is an integer type
func isInteger(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

// convertToFloat64 converts any numeric value to float64
func convertToFloat64(value reflect.Value) float64 {
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(value.Uint())
	case reflect.Float32, reflect.Float64:
		return value.Float()
	default:
		return 0.0
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
		TargetProcessorName: component.MustNewID("test-processor"),
		ParameterPath:       paramPath,
		NewValue:            value,
		Reason:              "Test patch",
		Severity:            "normal",
		Source:              "test",
		Timestamp:           time.Now().Unix(),
		TTLSeconds:          300,
	}
}