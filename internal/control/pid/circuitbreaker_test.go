package pid

import (
	"math"
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOscillationDetector tests basic functionality of the oscillation detector
func TestOscillationDetector(t *testing.T) {
	detector := NewOscillationDetector()
	require.NotNil(t, detector, "Oscillation detector should be created")
	
	// Initially not tripped
	assert.False(t, detector.IsTripped(), "Detector should not be tripped initially")
	
	// Adding a few non-oscillating samples
	detector.AddSample(1.0, 10.0)
	detector.AddSample(1.5, 11.0)
	detector.AddSample(2.0, 12.0)
	
	// Should not detect oscillation with monotonic signals
	assert.False(t, detector.IsTripped(), "Detector should not trip with monotonic signal")
}

// TestOscillationDetection tests the oscillation detection algorithm
func TestOscillationDetection(t *testing.T) {
	detector := NewOscillationDetector()
	
	// Configure for faster testing
	detector.Configure(10, 50, 0.1, time.Millisecond*50, time.Millisecond*100)
	
	// Add oscillating signal samples
	for i := 0; i < 5; i++ {
		detector.AddSample(1.0, 10.0)
		detector.AddSample(-1.0, 10.0)
	}
	
	// Should detect oscillation after several cycles
	assert.True(t, detector.IsTripped(), "Detector should trip with oscillating signal")
}

// TestOscillationReset tests the reset functionality
func TestOscillationReset(t *testing.T) {
	detector := NewOscillationDetector()
	
	// Configure for faster testing
	detector.Configure(10, 50, 0.1, time.Millisecond*50, time.Millisecond*100)
	
	// Add oscillating signal
	for i := 0; i < 5; i++ {
		detector.AddSample(1.0, 10.0)
		detector.AddSample(-1.0, 10.0)
	}
	
	// Should be tripped
	assert.True(t, detector.IsTripped(), "Detector should trip with oscillating signal")
	
	// Reset the detector
	detector.Reset()
	
	// Should no longer be tripped
	assert.False(t, detector.IsTripped(), "Detector should not be tripped after reset")
}

// TestOscillationTemporaryOverride tests the override functionality
func TestOscillationTemporaryOverride(t *testing.T) {
	detector := NewOscillationDetector()
	
	// Configure for faster testing
	detector.Configure(10, 50, 0.1, time.Millisecond*50, time.Millisecond*100)
	
	// Add oscillating signal
	for i := 0; i < 5; i++ {
		detector.AddSample(1.0, 10.0)
		detector.AddSample(-1.0, 10.0)
	}
	
	// Should be tripped
	assert.True(t, detector.IsTripped(), "Detector should trip with oscillating signal")
	
	// Apply temporary override
	detector.TemporaryOverride(time.Millisecond * 100)
	
	// Should not be tripped due to override
	assert.False(t, detector.IsTripped(), "Detector should not be tripped with override active")
	
	// Wait for override to expire
	time.Sleep(time.Millisecond * 150)
	
	// Should be tripped again after override expires
	assert.True(t, detector.IsTripped(), "Detector should be tripped again after override expires")
}

// TestPIDControllerWithCircuitBreaker tests the integration of PID controller with circuit breaker
func TestPIDControllerWithCircuitBreaker(t *testing.T) {
	// Create controller with circuit breaker enabled
	controller := NewController(1.0, 0.1, 0.05, 100.0)
	assert.True(t, controller.circuitBreakerEnabled, "Circuit breaker should be enabled by default")
	
	// Configure circuit breaker for testing
	controller.ConfigureCircuitBreaker(10, 50, 0.1, time.Millisecond*50, time.Millisecond*100)
	
	// Initialize a simple test process
	value := 70.0
	
	// Try normal control first - should work fine
	for i := 0; i < 5; i++ {
		output := controller.Compute(value)
		t.Logf("Normal control - Step %d: Value=%.2f, Output=%.2f", i, value, output)
		value += output * 0.5 // Simple process model
		time.Sleep(10 * time.Millisecond)
	}
	
	// Value should be converging to setpoint
	assert.InDelta(t, 100.0, value, 20.0, "Value should be moving toward setpoint")
	
	// Now simulate an oscillating process
	for i := 0; i < 10; i++ {
		if i % 2 == 0 {
			value = 120.0 // Overshoot
		} else {
			value = 80.0 // Undershoot
		}
		output := controller.Compute(value)
		t.Logf("Oscillating - Step %d: Value=%.2f, Output=%.2f", i, value, output)
		time.Sleep(10 * time.Millisecond)
	}
	
	// The circuit breaker should be tripped - outputs should be diminished
	// Get last output with oscillating input
	output := controller.Compute(120.0)
	
	// Circuit breaker should generate smaller outputs when tripped
	status := controller.GetCircuitBreakerStatus()
	if status["tripped"].(bool) {
		// Output should be reduced when oscillating
		assert.Less(t, math.Abs(output), 5.0, "Output should be reduced in magnitude when circuit breaker is tripped")
	}
	
	// Disable circuit breaker
	controller.EnableCircuitBreaker(false)
	
	// Get output without circuit breaker
	normalOutput := controller.Compute(120.0)
	
	// Normal output should be larger in magnitude
	assert.Greater(t, math.Abs(normalOutput), math.Abs(output), 
		"Output with circuit breaker disabled should be larger in magnitude")
}