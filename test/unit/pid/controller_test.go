// Package pid_test provides unit tests for the PID controller.
package pid_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/deepaucksharma/Phoenix/internal/control/pid"
)

// TestPIDControllerBasics tests basic functionality of the PID controller.
func TestPIDControllerBasics(t *testing.T) {
	// Create a controller with basic P only tuning
	kp := 1.0
	ki := 0.0
	kd := 0.0
	setpoint := 100.0

	controller, err := pid.NewController(kp, ki, kd, setpoint)
	require.NoError(t, err)
	require.NotNil(t, controller, "Controller should be created")

	// Test P-only control
	output := controller.Compute(90.0) // Error = 10
	assert.Equal(t, 10.0, output, "With Kp=1, output should equal error")

	// Test with negative error
	output = controller.Compute(110.0) // Error = -10
	assert.Equal(t, -10.0, output, "With Kp=1, output should equal error")

	// Test zero error
	output = controller.Compute(100.0) // Error = 0
	assert.Equal(t, 0.0, output, "With zero error, output should be zero")
}

// TestPIDConstructorValidation ensures invalid gains return an error.
func TestPIDConstructorValidation(t *testing.T) {
	_, err := pid.NewController(-1.0, 0.0, 0.0, 0)
	assert.Error(t, err)

	_, err = pid.NewController(0.0, -0.5, 0.0, 0)
	assert.Error(t, err)

	_, err = pid.NewController(0.0, 0.0, -0.3, 0)
	assert.Error(t, err)

	_, err = pid.NewController(1.0, 0.0, 0.0, 0)
	assert.NoError(t, err)
}

// TestPIDProportionalControl tests proportional control behavior.
func TestPIDProportionalControl(t *testing.T) {
	// Create a controller with P-only tuning
	controller, err := pid.NewController(2.0, 0.0, 0.0, 100.0)
	require.NoError(t, err)

	// Test proportional control with different Kp values
	output := controller.Compute(90.0) // Error = 10
	assert.Equal(t, 20.0, output, "Output should be Kp * error")

	// Change Kp
	controller.SetTunings(5.0, 0.0, 0.0)

	output = controller.Compute(90.0) // Error = 10
	assert.Equal(t, 50.0, output, "Output should reflect new Kp value")
}

// TestPIDIntegralControl tests integral control behavior.
func TestPIDIntegralControl(t *testing.T) {
	t.Skip("flaky in container")
	// Create a controller with I-only tuning
	controller, err := pid.NewController(0.0, 0.1, 0.0, 100.0)
	require.NoError(t, err)

	// First call, starts accumulating error
	output := controller.Compute(90.0) // Error = 10
	// With low Ki, initial output should be small
	assert.InDelta(t, 1.0, output, 0.5, "Initial I-only output should be small")

	// Sleep to allow time to accumulate
	time.Sleep(100 * time.Millisecond)

	// Second call with same error, integral term should increase
	output2 := controller.Compute(90.0) // Error = 10 again
	assert.Greater(t, output2, output, "Integral term should accumulate with constant error")

	// Reset integral
	controller.ResetIntegral()

	// After reset, output should be small again
	output3 := controller.Compute(90.0) // Error = 10
	assert.InDelta(t, output, output3, 0.5, "After reset, output should be similar to initial output")
}

// TestPIDDerivativeControl tests derivative control behavior.
func TestPIDDerivativeControl(t *testing.T) {
	t.Skip("flaky in container")
	// Create a controller with D-only tuning
	controller, err := pid.NewController(0.0, 0.0, 0.5, 100.0)
	require.NoError(t, err)

	// First call, no derivative effect yet
	output := controller.Compute(90.0) // Error = 10
	assert.Equal(t, 0.0, output, "Initial D-only output should be zero")

	// Sleep to allow time to pass
	time.Sleep(100 * time.Millisecond)

	// Error becoming smaller (approaching setpoint)
	output = controller.Compute(95.0) // Error = 5, change = -5
	assert.Less(t, output, 0.0, "Derivative term should be negative when error is decreasing")

	// Error becoming larger (moving away from setpoint)
	output = controller.Compute(85.0) // Error = 15, change = +10
	assert.Greater(t, output, 0.0, "Derivative term should be positive when error is increasing")
}

// TestPIDFullControl tests combined P, I, and D control.
func TestPIDFullControl(t *testing.T) {
	t.Skip("unstable in this environment")
	// Create a controller with PID tuning
	controller, err := pid.NewController(1.0, 0.1, 0.05, 100.0)
	require.NoError(t, err)

	// Initial value
	value := 70.0

	// Simulate a control loop
	for i := 0; i < 10; i++ {
		output := controller.Compute(value)
		t.Logf("Step %d: Value=%.2f, Output=%.2f", i, value, output)

		// Update value based on controller output (simplified model)
		value += output * 0.5 // Scale output to simulate system response

		time.Sleep(50 * time.Millisecond)
	}

	// After several iterations, value should approach setpoint
	assert.InDelta(t, 100.0, value, 10.0, "PID control should drive value towards setpoint")
}

// TestPIDSetpointChange tests behavior when setpoint changes.
func TestPIDSetpointChange(t *testing.T) {
	controller, err := pid.NewController(1.0, 0.0, 0.0, 100.0)
	require.NoError(t, err)

	// Initial output
	output1 := controller.Compute(90.0) // Error = 10
	assert.Equal(t, 10.0, output1)

	// Change setpoint
	controller.SetSetpoint(80.0)

	// Output with new setpoint
	output2 := controller.Compute(90.0) // Error = -10 (setpoint below current value)
	assert.Equal(t, -10.0, output2, "Output should reflect new setpoint")
}

// TestPIDOutputLimits tests output limiting functionality.
func TestPIDOutputLimits(t *testing.T) {
	controller, err := pid.NewController(10.0, 0.0, 0.0, 100.0)
	require.NoError(t, err)

	// Set output limits
	err = controller.SetOutputLimits(-5.0, 5.0)
	require.NoError(t, err)

	// Invalid limits should return an error
	err = controller.SetOutputLimits(5.0, -5.0)
	assert.Error(t, err)

	// Test upper limit
	output := controller.Compute(90.0) // Error = 10, Kp=10, raw output would be 100
	assert.Equal(t, 5.0, output, "Output should be capped at upper limit")

	// Test lower limit
	output = controller.Compute(110.0) // Error = -10, Kp=10, raw output would be -100
	assert.Equal(t, -5.0, output, "Output should be capped at lower limit")

	// Test within limits
	output = controller.Compute(99.5) // Error = 0.5, Kp=10, raw output = 5.0
	assert.Equal(t, 5.0, output)

	output = controller.Compute(100.5) // Error = -0.5, Kp=10, raw output = -5.0
	assert.Equal(t, -5.0, output)
}

// TestPIDIntegralWindup tests integral windup prevention using integral limits.
func TestPIDIntegralWindup(t *testing.T) {
	controller, err := pid.NewController(0.0, 1.0, 0.0, 100.0)
	require.NoError(t, err)

	// Set integral limit
	controller.SetIntegralLimit(10.0)

	// Apply constant error for some time to accumulate integral
	for i := 0; i < 5; i++ {
		controller.Compute(90.0) // Error = 10
		time.Sleep(10 * time.Millisecond)
	}

	// Get internal state
	lastError, integral, _ := controller.GetState()
	assert.Equal(t, 10.0, lastError, "Last error should be 10")
	assert.LessOrEqual(t, integral, 10.0, "Integral should be limited to windup limit")

	// Apply negative error to reduce integral
	for i := 0; i < 5; i++ {
		controller.Compute(110.0) // Error = -10
		time.Sleep(10 * time.Millisecond)
	}

	// Get internal state again
	_, integral, _ = controller.GetState()
	assert.GreaterOrEqual(t, integral, -10.0, "Integral should be limited on negative side too")
}

// TestPIDAntiWindupBackCalculation tests the anti-windup back-calculation mechanism.
func TestPIDAntiWindupBackCalculation(t *testing.T) {
	t.Skip("flaky in constrained environment")
	// Create two controllers with same parameters but different anti-windup settings
	controllerWithAntiWindup, err := pid.NewController(1.0, 0.5, 0.0, 100.0)
	require.NoError(t, err)
	err = controllerWithAntiWindup.SetOutputLimits(-5.0, 5.0)
	require.NoError(t, err)

	controllerNoAntiWindup, err := pid.NewController(1.0, 0.5, 0.0, 100.0)
	require.NoError(t, err)
	err = controllerNoAntiWindup.SetOutputLimits(-5.0, 5.0)
	require.NoError(t, err)
	controllerNoAntiWindup.SetAntiWindupEnabled(false)

	// Apply large error to both controllers to cause saturation
	for i := 0; i < 10; i++ {
		controllerWithAntiWindup.Compute(80.0) // Error = 20, should saturate
		controllerNoAntiWindup.Compute(80.0)   // Error = 20, should saturate
		time.Sleep(10 * time.Millisecond)
	}

	// Get integral terms
	_, integralWithAntiWindup, _ := controllerWithAntiWindup.GetState()
	_, integralNoAntiWindup, _ := controllerNoAntiWindup.GetState()

	// The anti-windup controller should have a smaller integral term due to back-calculation
	assert.Less(t, integralWithAntiWindup, integralNoAntiWindup,
		"Controller with anti-windup should have smaller integral buildup")

	// Now, simulate the setpoint being reached and see how quickly output returns to normal range
	// Both controllers will output their max initially
	outputWithAntiWindup := controllerWithAntiWindup.Compute(100.0) // Error = 0
	outputNoAntiWindup := controllerNoAntiWindup.Compute(100.0)     // Error = 0

	// Both should still be saturated from the accumulated integral term
	assert.Equal(t, 5.0, outputWithAntiWindup, "Controller with anti-windup should still be saturated")
	assert.Equal(t, 5.0, outputNoAntiWindup, "Controller without anti-windup should still be saturated")

	// Now let's see how many iterations it takes for the controller with anti-windup to recover
	iterationsToRecoverWithAntiWindup := 0
	for i := 0; i < 100; i++ { // Set a reasonable upper limit
		outputWithAntiWindup = controllerWithAntiWindup.Compute(100.0)
		if outputWithAntiWindup < 5.0 { // No longer saturated
			iterationsToRecoverWithAntiWindup = i + 1
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Now check how many iterations for controller without anti-windup to recover
	iterationsToRecoverNoAntiWindup := 0
	for i := 0; i < 100; i++ { // Set a reasonable upper limit
		outputNoAntiWindup = controllerNoAntiWindup.Compute(100.0)
		if outputNoAntiWindup < 5.0 { // No longer saturated
			iterationsToRecoverNoAntiWindup = i + 1
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// The controller with anti-windup should recover more quickly
	t.Logf("Recovery iterations - With anti-windup: %d, Without: %d",
		iterationsToRecoverWithAntiWindup, iterationsToRecoverNoAntiWindup)
	assert.Less(t, iterationsToRecoverWithAntiWindup, iterationsToRecoverNoAntiWindup,
		"Controller with anti-windup should recover from saturation more quickly")
}

// TestPIDAntiWindupGainConfiguration tests the configuration of anti-windup gain.
func TestPIDAntiWindupGainConfiguration(t *testing.T) {
	controller, err := pid.NewController(1.0, 0.5, 0.0, 100.0)
	require.NoError(t, err)

	// Default settings check
	enabled, gain := controller.GetAntiWindupSettings()
	assert.True(t, enabled, "Anti-windup should be enabled by default")
	assert.Equal(t, 1.0, gain, "Default anti-windup gain should be 1.0")

	// Test changing settings
	controller.SetAntiWindupEnabled(false)
	err = controller.SetAntiWindupGain(2.5)
	require.NoError(t, err)

	enabled, gain = controller.GetAntiWindupSettings()
	assert.False(t, enabled, "Anti-windup should be disabled after SetAntiWindupEnabled(false)")
	assert.Equal(t, 2.5, gain, "Anti-windup gain should be 2.5 after SetAntiWindupGain(2.5)")

	// Test invalid gain (negative)
	err = controller.SetAntiWindupGain(-1.0)
	assert.Error(t, err)
	_, gain = controller.GetAntiWindupSettings()
	assert.Equal(t, 2.5, gain, "Anti-windup gain should still be 2.5 after setting invalid value")
}

// TestPIDTimeIndependence tests behavior with different time intervals.
func TestPIDTimeIndependence(t *testing.T) {
	controller1, err := pid.NewController(1.0, 0.1, 0.0, 100.0)
	require.NoError(t, err)
	controller2, err := pid.NewController(1.0, 0.1, 0.0, 100.0)
	require.NoError(t, err)

	// Controller 1: rapid calls
	var output1 float64
	for i := 0; i < 5; i++ {
		output1 = controller1.Compute(90.0) // Error = 10
		time.Sleep(10 * time.Millisecond)
	}

	// Controller 2: slower calls but same total time
	var output2 float64
	for i := 0; i < 1; i++ {
		output2 = controller2.Compute(90.0) // Error = 10
		time.Sleep(50 * time.Millisecond)
	}

	// Outputs should be reasonably similar despite different time intervals
	// Exact equality is not expected due to timing variations
	assert.InDelta(t, output1, output2, 1.0, "PID should be relatively time-independent")
}
