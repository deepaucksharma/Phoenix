// Package pid implements a Proportional-Integral-Derivative controller
package pid

import (
	"context"
	"sync"
	"time"

	"github.com/deepaucksharma/Phoenix/pkg/metrics"
)

// Controller implements a PID (Proportional-Integral-Derivative) controller
type Controller struct {
	// PID constants
	kp float64 // Proportional gain
	ki float64 // Integral gain
	kd float64 // Derivative gain

	// State
	setpoint      float64   // Target value
	lastError     float64   // Last error value
	prevError     float64   // Error from two steps ago (for filtered derivative)
	integral      float64   // Accumulated error
	lastTime      time.Time // Last update time
	lastDeltaTime float64   // Last time step in seconds (for consistent derivative)

	// Limits
	integralLimit float64 // Maximum absolute value for integral term
	outputMin     float64 // Minimum output value
	outputMax     float64 // Maximum output value

	// Anti-windup
	antiWindupEnabled bool    // Whether anti-windup protection is enabled
	antiWindupGain    float64 // Gain for anti-windup back-calculation

	// Derivative filtering
	derivativeFilterCoeff float64 // Coefficient for derivative low-pass filter (between 0 and 1)

	// Circuit Breaker for oscillation detection
	circuitBreaker   *OscillationDetector // Detects and prevents oscillations
	circuitBreakerEnabled bool            // Whether circuit breaker is enabled

	// Metrics
	name             string              // Controller name for metrics
	metricsCollector *metrics.PIDMetrics // For collecting and emitting metrics

	lock sync.Mutex // For thread safety
}

// NewController creates a new PID controller with the specified gains
func NewController(kp, ki, kd, setpoint float64) *Controller {
	return &Controller{
		kp:                  kp,
		ki:                  ki,
		kd:                  kd,
		setpoint:            setpoint,
		lastError:           0,
		prevError:           0,
		integral:            0,
		lastTime:            time.Now(),
		lastDeltaTime:       0.1, // Default initial time step
		integralLimit:       1000, // Default, can be changed with SetIntegralLimit
		outputMin:           -1000,
		outputMax:           1000,
		antiWindupEnabled:   true,  // Enable anti-windup by default
		antiWindupGain:      1.0,   // Default gain for anti-windup
		derivativeFilterCoeff: 0.2, // Default filter coefficient (0.2 = moderate filtering)
		circuitBreaker:      NewOscillationDetector(), // Initialize circuit breaker
		circuitBreakerEnabled: true, // Enable circuit breaker by default
		name:                "pid_controller", // Default name
		metricsCollector:    nil,     // No metrics collection by default
	}
}

// NewControllerWithMetrics creates a new PID controller with metrics collection
func NewControllerWithMetrics(kp, ki, kd, setpoint float64, name string, parent *metrics.MetricsEmitter) *Controller {
	c := NewController(kp, ki, kd, setpoint)
	c.name = name
	c.metricsCollector = metrics.NewPIDMetrics(name, parent)
	return c
}

// SetName sets the controller name used in metrics
func (c *Controller) SetName(name string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.name = name
	if c.metricsCollector != nil {
		c.metricsCollector.ControllerName = name
	}
}

// EnableMetrics enables metrics collection for this controller
func (c *Controller) EnableMetrics(parent *metrics.MetricsEmitter) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.metricsCollector == nil {
		c.metricsCollector = metrics.NewPIDMetrics(c.name, parent)
	} else {
		c.metricsCollector.Parent = parent
	}
}

// SetIntegralLimit sets the maximum absolute value for the integral term
func (c *Controller) SetIntegralLimit(limit float64) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.integralLimit = limit

	// Clamp existing integral if needed
	if c.integral > c.integralLimit {
		c.integral = c.integralLimit
	} else if c.integral < -c.integralLimit {
		c.integral = -c.integralLimit
	}
}

// SetOutputLimits sets the minimum and maximum output values
func (c *Controller) SetOutputLimits(min, max float64) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if min >= max {
		return // Invalid limits
	}

	c.outputMin = min
	c.outputMax = max
}

// Compute calculates a new output value based on the current error
func (c *Controller) Compute(currentValue float64) float64 {
	// Use local lock to minimize lock scope
	c.lock.Lock()
	defer c.lock.Unlock()

	// Calculate error
	error := c.setpoint - currentValue

	// Calculate time delta
	now := time.Now()
	dt := now.Sub(c.lastTime).Seconds()
	if dt <= 0 {
		// Use previous delta time for consistency instead of a fixed value
		dt = c.lastDeltaTime
	}

	// Proportional term
	pTerm := c.kp * error

	// Integral term - use trapezoidal rule for more accurate integration
	c.integral += (error + c.lastError) * 0.5 * dt

	// Apply integral limit
	if c.integral > c.integralLimit {
		c.integral = c.integralLimit
	} else if c.integral < -c.integralLimit {
		c.integral = -c.integralLimit
	}

	iTerm := c.ki * c.integral

	// Derivative term with low-pass filtering to reduce noise sensitivity
	var dTerm float64
	if dt > 0 {
		// Apply filtering to the derivative to reduce noise sensitivity
		// This implements a first-order low-pass filter on the derivative
		// Filtered derivative = α * current_derivative + (1-α) * previous_derivative
		currentDerivative := (error - c.lastError) / dt
		previousDerivative := (c.lastError - c.prevError) / c.lastDeltaTime
		
		// If this is the first or second iteration, use current derivative only
		if c.prevError == 0 && c.lastError == 0 {
			dTerm = c.kd * currentDerivative
		} else {
			// Apply low-pass filter to derivative term
			filteredDerivative := c.derivativeFilterCoeff * currentDerivative + 
				(1.0 - c.derivativeFilterCoeff) * previousDerivative
			
			dTerm = c.kd * filteredDerivative
		}
	}

	// Calculate raw output (before limits)
	rawOutput := pTerm + iTerm + dTerm

	// Start with raw output
	output := rawOutput

	// Check if circuit breaker is tripped
	if c.circuitBreakerEnabled && c.circuitBreaker != nil {
		// Add sample to circuit breaker
		oscillating := c.circuitBreaker.AddSample(output, currentValue)
		
		// If circuit breaker is tripped, output zero
		if oscillating && c.circuitBreaker.IsTripped() {
			// When oscillating, use proportional term only with reduced gain
			// This helps stabilize the system while still providing some control
			safeKp := c.kp * 0.1 // Use 10% of normal P gain when in safe mode
			output = safeKp * error
			
			// Reset integral to prevent windup
			c.integral = 0
			
			// Add additional safeguard limits
			if output > c.outputMax * 0.5 {
				output = c.outputMax * 0.5
			} else if output < c.outputMin * 0.5 {
				output = c.outputMin * 0.5
			}
			
			// Update metrics if enabled
			if c.metricsCollector != nil {
				c.metricsCollector.AddMetric("aemf.controller.pid.circuit_breaker_trips_total", 1)
			}
		}
	}

	// Apply output limits and anti-windup if enabled
	if output > c.outputMax {
		// Anti-windup back-calculation when output is saturated at max
		if c.antiWindupEnabled && c.ki != 0 {
			// Reduce integral term based on saturation amount
			saturationError := c.outputMax - output
			c.integral += (saturationError * c.antiWindupGain) / c.ki
		}
		output = c.outputMax
	} else if output < c.outputMin {
		// Anti-windup back-calculation when output is saturated at min
		if c.antiWindupEnabled && c.ki != 0 {
			// Reduce integral term based on saturation amount
			saturationError := c.outputMin - output
			c.integral += (saturationError * c.antiWindupGain) / c.ki
		}
		output = c.outputMin
	}

	// Update state
	c.prevError = c.lastError
	c.lastError = error
	c.lastDeltaTime = dt
	c.lastTime = now

	// Update metrics if enabled
	if c.metricsCollector != nil {
		c.metricsCollector.Update(
			c.setpoint,
			currentValue,
			error,
			pTerm,
			iTerm,
			dTerm,
			rawOutput,
			output,
		)

		// Emit metrics if interval has passed
		if c.metricsCollector.ShouldEmit() {
			c.metricsCollector.EmitMetrics(context.Background())
		}
	}

	return output
}

// SetSetpoint updates the controller's target value
func (c *Controller) SetSetpoint(setpoint float64) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.setpoint = setpoint
}

// SetTunings updates the PID gains
func (c *Controller) SetTunings(kp, ki, kd float64) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.kp = kp
	c.ki = ki
	c.kd = kd
}

// GetTunings returns the current PID gain values
func (c *Controller) GetTunings() (float64, float64, float64) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.kp, c.ki, c.kd
}

// ResetIntegral clears the accumulated integral term
func (c *Controller) ResetIntegral() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.integral = 0
}

// GetState returns the current internal state of the controller
func (c *Controller) GetState() (float64, float64, float64) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.lastError, c.integral, c.setpoint
}

// SetDerivativeFilterCoefficient sets the filtering coefficient for the derivative term
// A value of 1.0 means no filtering (use raw derivative)
// Lower values increase filtering: 0.2-0.3 is typically a good balance
// Very low values may make the derivative term less responsive
func (c *Controller) SetDerivativeFilterCoefficient(coefficient float64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	
	// Clamp coefficient to valid range
	if coefficient < 0.0 {
		coefficient = 0.0
	} else if coefficient > 1.0 {
		coefficient = 1.0
	}
	
	c.derivativeFilterCoeff = coefficient
}

// GetDerivativeFilterCoefficient returns the current derivative filtering coefficient
func (c *Controller) GetDerivativeFilterCoefficient() float64 {
	c.lock.Lock()
	defer c.lock.Unlock()
	
	return c.derivativeFilterCoeff
}

// EnableCircuitBreaker enables or disables the oscillation detection circuit breaker
func (c *Controller) EnableCircuitBreaker(enabled bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	
	c.circuitBreakerEnabled = enabled
}

// ConfigureCircuitBreaker configures the oscillation detector parameters
func (c *Controller) ConfigureCircuitBreaker(sampleWindow int, thresholdPercent, minMagnitude float64, 
                                           minDuration, resetDuration time.Duration) {
	c.lock.Lock()
	defer c.lock.Unlock()
	
	if c.circuitBreaker != nil {
		c.circuitBreaker.Configure(sampleWindow, thresholdPercent, minMagnitude, minDuration, resetDuration)
	}
}

// ResetCircuitBreaker manually resets the circuit breaker
func (c *Controller) ResetCircuitBreaker() {
	c.lock.Lock()
	defer c.lock.Unlock()
	
	if c.circuitBreaker != nil {
		c.circuitBreaker.Reset()
	}
}

// TemporaryOverrideCircuitBreaker allows the controller to operate despite the circuit breaker
// being tripped, for a specified duration
func (c *Controller) TemporaryOverrideCircuitBreaker(duration time.Duration) {
	c.lock.Lock()
	defer c.lock.Unlock()
	
	if c.circuitBreaker != nil {
		c.circuitBreaker.TemporaryOverride(duration)
	}
}

// GetCircuitBreakerStatus returns the current status of the oscillation detector
func (c *Controller) GetCircuitBreakerStatus() map[string]interface{} {
	c.lock.Lock()
	defer c.lock.Unlock()
	
	if c.circuitBreaker != nil {
		status := c.circuitBreaker.GetStatus()
		status["enabled"] = c.circuitBreakerEnabled
		return status
	}
	
	return map[string]interface{}{
		"enabled": c.circuitBreakerEnabled,
		"available": false,
	}
}

// GetMetrics returns the metrics collector for this controller
func (c *Controller) GetMetrics() *metrics.PIDMetrics {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.metricsCollector
}

// EmitMetrics immediately emits metrics for this controller
func (c *Controller) EmitMetrics(ctx context.Context) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.metricsCollector != nil {
		c.metricsCollector.EmitMetrics(ctx)
	}
}

// SetAntiWindupEnabled enables or disables anti-windup protection
func (c *Controller) SetAntiWindupEnabled(enabled bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.antiWindupEnabled = enabled
}

// SetAntiWindupGain sets the gain for anti-windup back-calculation
// Higher values lead to faster integral recovery when output is saturated
func (c *Controller) SetAntiWindupGain(gain float64) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if gain < 0 {
		return // Invalid gain
	}

	c.antiWindupGain = gain
}

// GetAntiWindupSettings returns the current anti-windup settings
func (c *Controller) GetAntiWindupSettings() (bool, float64) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.antiWindupEnabled, c.antiWindupGain
}
