package pid_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/deepaucksharma/Phoenix/internal/control/pid"
	"github.com/deepaucksharma/Phoenix/pkg/metrics"
)

func TestStreamlinedController_Creation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	config := pid.ControllerConfig{
		Name:            "test_controller",
		Kp:              10.0,
		Ki:              2.0,
		Kd:              0.5,
		Setpoint:        100.0,
		OutputMin:       -50.0,
		OutputMax:       50.0,
		IntegralLimit:   30.0,
		AntiWindupEnabled: true,
		AntiWindupGain: 0.1,
		DerivativeFilterCoeff: 0.2,
		OscillationDetectionEnabled: true,
		OscillationThreshold: 0.01,
		OscillationWindowSize: 10,
		HysteresisPercent: 5.0,
	}
	
	controller := pid.NewStreamlinedController(config, logger)
	assert.NotNil(t, controller)
}

func TestStreamlinedController_ComputeOutput(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	config := pid.ControllerConfig{
		Name:            "test_controller",
		Kp:              1.0,  // Use simpler gains for predictable testing
		Ki:              0.1,
		Kd:              0.01,
		Setpoint:        100.0,
		OutputMin:       -50.0,
		OutputMax:       50.0,
		IntegralLimit:   30.0,
		AntiWindupEnabled: true,
		OscillationDetectionEnabled: false, // Disable for predictable testing
	}
	
	controller := pid.NewStreamlinedController(config, logger)
	
	// Test computation with a simple case
	// If setpoint is 100 and measurement is 90, with Kp=1.0, we should get an output around 10
	// (plus small amounts from I and D terms)
	output, err := controller.Compute(context.Background(), 90.0)
	assert.NoError(t, err)
	
	// With Kp=1.0, error of 10, should be close to 10 (plus small I and D terms)
	assert.InDelta(t, 10.0, output, 1.0) 
	
	// Make a second call to test derivative term
	output, err = controller.Compute(context.Background(), 95.0)
	assert.NoError(t, err)
	
	// Error is now 5, plus some I and D effects
	assert.InDelta(t, 5.0, output, 1.0)
}

func TestStreamlinedController_OutputLimits(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	config := pid.ControllerConfig{
		Name:            "test_controller",
		Kp:              10.0,  // Higher gain to test limits
		Ki:              0.0,   // No integral for this test
		Kd:              0.0,   // No derivative for this test
		Setpoint:        100.0,
		OutputMin:       -10.0,
		OutputMax:       10.0,  // Deliberately constrained
		IntegralLimit:   30.0,
		AntiWindupEnabled: true,
	}
	
	controller := pid.NewStreamlinedController(config, logger)
	
	// With Kp=10 and error of 20, raw output would be 200, but should be limited to 10
	output, err := controller.Compute(context.Background(), 80.0)
	assert.NoError(t, err)
	assert.Equal(t, 10.0, output) // Should be capped at max
	
	// Test lower limit
	output, err = controller.Compute(context.Background(), 120.0)
	assert.NoError(t, err)
	assert.Equal(t, -10.0, output) // Should be capped at min
}

func TestStreamlinedController_AntiWindup(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	config := pid.ControllerConfig{
		Name:            "test_controller",
		Kp:              0.5,
		Ki:              0.2,
		Kd:              0.0,   // No derivative for this test
		Setpoint:        100.0,
		OutputMin:       -10.0,
		OutputMax:       10.0,
		IntegralLimit:   100.0, // High limit to test anti-windup
		AntiWindupEnabled: true,
		AntiWindupGain:  0.5,
	}
	
	controller := pid.NewStreamlinedController(config, logger)
	
	// Create a situation where integral will grow quickly
	for i := 0; i < 10; i++ {
		output, err := controller.Compute(context.Background(), 80.0)
		assert.NoError(t, err)
		// Output should be capped at 10.0
		assert.Equal(t, 10.0, output)
	}
	
	// Now measurement approaches setpoint
	output, err := controller.Compute(context.Background(), 95.0)
	assert.NoError(t, err)
	
	// Anti-windup should have prevented excessive integral buildup,
	// so output should be less than if no anti-windup was used
	assert.True(t, output < 10.0)
}

func TestStreamlinedController_Hysteresis(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	config := pid.ControllerConfig{
		Name:             "test_controller",
		Kp:               1.0,
		Ki:               0.0,   // No integral for this test
		Kd:               0.0,   // No derivative for this test
		Setpoint:         100.0,
		HysteresisPercent: 5.0,  // 5% hysteresis
	}
	
	controller := pid.NewStreamlinedController(config, logger)
	
	// Within hysteresis band (100 ± 5%)
	output, err := controller.Compute(context.Background(), 96.0)
	assert.NoError(t, err)
	assert.InDelta(t, 0.0, output, 0.001) // Should be zero due to hysteresis
	
	output, err = controller.Compute(context.Background(), 104.0)
	assert.NoError(t, err)
	assert.InDelta(t, 0.0, output, 0.001) // Should be zero due to hysteresis
	
	// Outside hysteresis band
	output, err = controller.Compute(context.Background(), 94.0)
	assert.NoError(t, err)
	assert.InDelta(t, 6.0, output, 0.001) // Error is 6%
	
	output, err = controller.Compute(context.Background(), 106.0)
	assert.NoError(t, err)
	assert.InDelta(t, -6.0, output, 0.001) // Error is -6%
}

func TestStreamlinedController_OscillationDetection(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	config := pid.ControllerConfig{
		Name:                        "test_controller",
		Kp:                          1.0,
		Ki:                          0.0,
		Kd:                          0.0,
		Setpoint:                    100.0,
		OscillationDetectionEnabled: true,
		OscillationThreshold:        0.01,  // Small threshold for this test
		OscillationWindowSize:       5,     // Small window for this test
	}
	
	controller := pid.NewStreamlinedController(config, logger)
	
	// Create oscillating measurements
	measurements := []float64{90, 110, 90, 110, 90, 110}
	
	// Process the oscillating measurements
	for _, m := range measurements {
		_, err := controller.Compute(context.Background(), m)
		assert.NoError(t, err)
	}
	
	// We can't directly test if oscillation was detected, but we can
	// reset and make sure it works
	controller.Reset()
	
	// After reset, a single compute should work as normal
	output, err := controller.Compute(context.Background(), 90.0)
	assert.NoError(t, err)
	assert.InDelta(t, 10.0, output, 0.001)
}

func TestStreamlinedController_Reset(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	config := pid.ControllerConfig{
		Name:            "test_controller",
		Kp:              1.0,
		Ki:              0.5,
		Kd:              0.1,
		Setpoint:        100.0,
	}
	
	controller := pid.NewStreamlinedController(config, logger)
	
	// Make some calls to build up state
	for i := 0; i < 5; i++ {
		_, err := controller.Compute(context.Background(), 90.0)
		assert.NoError(t, err)
	}
	
	// Reset the controller
	controller.Reset()
	
	// After reset, should behave like a new controller
	output, err := controller.Compute(context.Background(), 90.0)
	assert.NoError(t, err)
	
	// Should be close to pure proportional response since I and D terms reset
	assert.InDelta(t, 10.0, output, 0.5)
}

func TestStreamlinedController_UpdateConfig(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	initialConfig := pid.ControllerConfig{
		Name:            "test_controller",
		Kp:              1.0,
		Ki:              0.1,
		Kd:              0.01,
		Setpoint:        100.0,
	}
	
	controller := pid.NewStreamlinedController(initialConfig, logger)
	
	// Make an initial compute
	output, err := controller.Compute(context.Background(), 90.0)
	assert.NoError(t, err)
	assert.InDelta(t, 10.0, output, 0.5)
	
	// Update the controller config
	newConfig := pid.ControllerConfig{
		Name:            "test_controller", // Keep same name
		Kp:              2.0,               // Double the gain
		Ki:              0.1,
		Kd:              0.01,
		Setpoint:        100.0,
	}
	
	err = controller.UpdateConfig(newConfig)
	assert.NoError(t, err)
	
	// After config update, should have new behavior
	output, err = controller.Compute(context.Background(), 90.0)
	assert.NoError(t, err)
	
	// With double the gain, error of 10 should give output around 20
	assert.InDelta(t, 20.0, output, 1.0)
}

func TestStreamlinedController_ValidatePIDMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	config := pid.ControllerConfig{
		Name:            "test_controller",
		Kp:              1.0,
		Ki:              0.1,
		Kd:              0.01,
		Setpoint:        100.0,
	}
	
	controller := pid.NewStreamlinedController(config, logger)
	
	// Create a new metrics collector to pass to the controller
	metricsCollector := metrics.NewUnifiedMetricsCollector(logger)
	metricsCollector.AddDefaultAttribute("test", "metrics_integration")
	
	controller.UpdateMetricsCollector(metricsCollector)
	
	// Make a computation to generate metrics
	_, err := controller.Compute(context.Background(), 90.0)
	require.NoError(t, err)
	
	// Emit metrics
	err = controller.EmitMetrics(context.Background())
	require.NoError(t, err)
	
	// Verify metrics have been recorded in the collector
	// We can't easily check the emitted metrics, but we can verify they're being collected
	pidMeasurement := metricsCollector.GetMetric("aemf_pid_measurement")
	assert.NotNil(t, pidMeasurement)
	
	pidError := metricsCollector.GetMetric("aemf_pid_error")
	assert.NotNil(t, pidError)
	
	pidOutput := metricsCollector.GetMetric("aemf_pid_output")
	assert.NotNil(t, pidOutput)
}

func TestStreamlinedController_IntegrationTest(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	config := pid.ControllerConfig{
		Name:            "coverage_controller",
		Kp:              1.0,
		Ki:              0.1,
		Kd:              0.01,
		Setpoint:        0.95, // Target 95% coverage
		OutputMin:       -15.0,
		OutputMax:       15.0,
		IntegralLimit:   30.0,
		AntiWindupEnabled: true,
	}
	
	controller := pid.NewStreamlinedController(config, logger)
	metricsCollector := metrics.NewUnifiedMetricsCollector(logger)
	controller.UpdateMetricsCollector(metricsCollector)
	
	// Simulate a typical control scenario
	var measurements = []float64{0.80, 0.85, 0.90, 0.93, 0.96, 0.99, 0.97, 0.95}
	var outputs []float64
	
	// Process each measurement and collect outputs
	for _, m := range measurements {
		output, err := controller.Compute(context.Background(), m)
		assert.NoError(t, err)
		outputs = append(outputs, output)
		
		// Emit metrics after each computation
		err = controller.EmitMetrics(context.Background())
		assert.NoError(t, err)
		
		// Allow a small delay to simulate real-world interval
		time.Sleep(10 * time.Millisecond)
	}
	
	// Verify the general behavior of the PID controller:
	// 1. When measurement < setpoint, output should be positive
	// 2. When measurement > setpoint, output should be negative
	// 3. When measurement = setpoint, output should be close to zero
	
	// Verify outputs have correct signs based on error
	assert.True(t, outputs[0] > 0) // Measurement 0.80 < setpoint 0.95
	assert.True(t, outputs[1] > 0) // Measurement 0.85 < setpoint 0.95
	assert.True(t, outputs[2] > 0) // Measurement 0.90 < setpoint 0.95
	assert.True(t, outputs[5] < 0) // Measurement 0.99 > setpoint 0.95
	
	// Last output should be close to zero since measurement equals setpoint
	assert.InDelta(t, 0.0, outputs[7], 0.01)
	
	// Verify metrics were collected
	assert.NotNil(t, metricsCollector.GetMetric("aemf_pid_measurement"))
	assert.NotNil(t, metricsCollector.GetMetric("aemf_pid_error"))
	assert.NotNil(t, metricsCollector.GetMetric("aemf_pid_proportional"))
	assert.NotNil(t, metricsCollector.GetMetric("aemf_pid_integral"))
	assert.NotNil(t, metricsCollector.GetMetric("aemf_pid_derivative"))
	assert.NotNil(t, metricsCollector.GetMetric("aemf_pid_output"))
}