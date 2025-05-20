// Package pid contains end-to-end tests for PID controller behavior.
package pid

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/deepaucksharma/Phoenix/internal/control/pid"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
)

// TestWindupProtection verifies that PID controller integral windup protection
// works correctly when output is saturated.
// Scenario: PID-08
func TestWindupProtection(t *testing.T) {
	// Skip if running in short mode - this is an integration test
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a test metrics collector to capture emitted metrics
	metricsCollector := metrics.NewMetricsCollector()

	// Create a PID controller with specific settings for windup testing
	// We want a controller with:
	// - High integral gain to force windup more quickly
	// - Small output limits to cause saturation
	// - Relatively small integral limit for testing
	controller := pid.NewController(1.0, 2.0, 0.1, 10.0) // kp=1, ki=2 (high), kd=0.1, setpoint=10
	controller.SetName("windup_test")
	controller.SetOutputLimits(-5.0, 5.0) // Tight output limits for quick saturation
	controller.SetIntegralLimit(10.0)     // Limit integral accumulation
	controller.SetAntiWindupEnabled(true) // Enable anti-windup protection
	controller.SetAntiWindupGain(1.0)     // Default anti-windup gain

	// Simulate control of a system that stays at a fixed value (simulating a stuck system)
	// This would cause integral windup without protection
	const stuckValue = 8.0 // This value is below setpoint, causing positive error

	// Run controller for several iterations to allow windup to occur
	var lastOutput float64
	var outputs []float64

	// Run for several iterations with anti-windup enabled
	for i := 0; i < 10; i++ {
		output := controller.Compute(stuckValue)
		outputs = append(outputs, output)
		lastOutput = output

		// Introduce a small delay to simulate control loop timing
		time.Sleep(10 * time.Millisecond)
	}

	// Verify output is saturated at max limit
	assert.Equal(t, 5.0, lastOutput, "Output should be saturated at max limit")

	// Get current integral term for later comparison
	_, integralWithProtection, _ := controller.GetState()

	// Now disable anti-windup protection and run again with a fresh controller
	controllerNoProtection := pid.NewController(1.0, 2.0, 0.1, 10.0)
	controllerNoProtection.SetName("no_windup_protection_test")
	controllerNoProtection.SetOutputLimits(-5.0, 5.0)
	controllerNoProtection.SetIntegralLimit(10.0)
	controllerNoProtection.SetAntiWindupEnabled(false) // Disable anti-windup protection

	// Run for the same number of iterations
	var outputsNoProtection []float64

	for i := 0; i < 10; i++ {
		output := controllerNoProtection.Compute(stuckValue)
		outputsNoProtection = append(outputsNoProtection, output)

		// Introduce a small delay to simulate control loop timing
		time.Sleep(10 * time.Millisecond)
	}

	// Get integral term without protection
	_, integralNoProtection, _ := controllerNoProtection.GetState()

	// Verify integral term with protection is smaller than without protection
	// This is the key test for anti-windup - it should prevent integral growth
	assert.Less(t, integralWithProtection, integralNoProtection,
		"Integral term with anti-windup should be smaller than without protection")

	// Now test recovery from saturation
	// When setpoint is achieved, controller with anti-windup should recover faster

	// First, reset both controllers
	controller.ResetIntegral()
	controllerNoProtection.ResetIntegral()

	// Run both to saturation
	for i := 0; i < 5; i++ {
		controller.Compute(stuckValue)
		controllerNoProtection.Compute(stuckValue)
	}

	// Now change to a value that allows recovery and measure response
	const recoveryValue = 9.5 // Much closer to setpoint

	// Capture recovery times
	var recoveryWithProtection []float64
	var recoveryNoProtection []float64

	// Run recovery phase
	for i := 0; i < 10; i++ {
		outputWithProtection := controller.Compute(recoveryValue)
		outputNoProtection := controllerNoProtection.Compute(recoveryValue)

		recoveryWithProtection = append(recoveryWithProtection, outputWithProtection)
		recoveryNoProtection = append(recoveryNoProtection, outputNoProtection)
	}

	// Calculate how quickly output returns to proportional range
	// In a real system, we'd want to measure overshoot and settling time
	// For this test, we'll simply compare the last values

	_, integralWithProtectionAfterRecovery, _ := controller.GetState()
	_, integralNoProtectionAfterRecovery, _ := controllerNoProtection.GetState()

	// The integral term with protection should recover faster (be smaller)
	// This indicates less overshoot and faster settling
	assert.Less(t, integralWithProtectionAfterRecovery, integralNoProtectionAfterRecovery,
		"Integral term with anti-windup should recover faster from saturation")

	// Additional test: Verify that integral limit works properly
	// Create a controller with a low integral limit
	controllerWithLimit := pid.NewController(1.0, 2.0, 0.1, 10.0)
	controllerWithLimit.SetIntegralLimit(2.0) // Very small integral limit

	// Run controller with a large error for several iterations
	for i := 0; i < 10; i++ {
		controllerWithLimit.Compute(0.0) // Zero value creates maximum error (10)
	}

	// Get state and verify integral term is limited to specified value
	_, integralValue, _ := controllerWithLimit.GetState()
	assert.LessOrEqual(t, integralValue, 2.0, "Integral term should be limited to 2.0")
}
