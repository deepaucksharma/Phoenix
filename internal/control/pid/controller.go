// Package pid implements a Proportional-Integral-Derivative controller
package pid

import (
	"fmt"
	"sync"
	"time"
)

// Controller implements a PID (Proportional-Integral-Derivative) controller
type Controller struct {
	// PID constants
	kp float64 // Proportional gain
	ki float64 // Integral gain
	kd float64 // Derivative gain

	// State
	setpoint  float64   // Target value
	lastError float64   // Last error value
	integral  float64   // Accumulated error
	lastTime  time.Time // Last update time

	// Limits
	integralLimit float64 // Maximum absolute value for integral term
	outputMin     float64 // Minimum output value
	outputMax     float64 // Maximum output value

	// Anti-windup
	antiWindupEnabled bool    // Whether anti-windup protection is enabled
	antiWindupGain    float64 // Gain for anti-windup back-calculation

	lock sync.Mutex // For thread safety
}

// NewController creates a new PID controller with the specified gains.
// It validates that the gain values are non-negative.
func NewController(kp, ki, kd, setpoint float64) (*Controller, error) {
	if kp < 0 {
		return nil, fmt.Errorf("invalid Kp: %v", kp)
	}
	if ki < 0 {
		return nil, fmt.Errorf("invalid Ki: %v", ki)
	}
	if kd < 0 {
		return nil, fmt.Errorf("invalid Kd: %v", kd)
	}

	return &Controller{
		kp:                kp,
		ki:                ki,
		kd:                kd,
		setpoint:          setpoint,
		lastError:         0,
		integral:          0,
		lastTime:          time.Now(),
		integralLimit:     1000, // Default, can be changed with SetIntegralLimit
		outputMin:         -1000,
		outputMax:         1000,
		antiWindupEnabled: true, // Enable anti-windup by default
		antiWindupGain:    1.0,  // Default gain for anti-windup
	}, nil
}

// NewControllerCompat provides backwards compatibility by returning a controller
// without exposing validation errors. Invalid gains will result in zeroed gains.
func NewControllerCompat(kp, ki, kd, setpoint float64) *Controller {
	c, _ := NewController(kp, ki, kd, setpoint)
	if c == nil {
		// Fall back to a controller with zeroed gains if validation failed
		c, _ = NewController(0, 0, 0, setpoint)
	}
	return c
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
func (c *Controller) SetOutputLimits(min, max float64) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if min >= max {
		return fmt.Errorf("invalid output limits: min (%f) must be less than max (%f)", min, max)
	}

	c.outputMin = min
	c.outputMax = max
	return nil
}

// Compute calculates a new output value based on the current error
func (c *Controller) Compute(currentValue float64) float64 {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Calculate error
	error := c.setpoint - currentValue

	// Calculate time delta
	now := time.Now()
	dt := now.Sub(c.lastTime).Seconds()
	if dt <= 0 {
		dt = 0.1 // Minimum time delta to avoid division by zero
	}

	// Proportional term
	pTerm := c.kp * error

	// Integral term
	c.integral += error * dt

	// Apply integral limit
	if c.integral > c.integralLimit {
		c.integral = c.integralLimit
	} else if c.integral < -c.integralLimit {
		c.integral = -c.integralLimit
	}

	iTerm := c.ki * c.integral

	// Derivative term
	var dTerm float64
	if dt > 0 {
		dTerm = c.kd * (error - c.lastError) / dt
	}

	// Calculate output
	output := pTerm + iTerm + dTerm

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
	c.lastError = error
	c.lastTime = now

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

// SetAntiWindupEnabled enables or disables anti-windup protection
func (c *Controller) SetAntiWindupEnabled(enabled bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.antiWindupEnabled = enabled
}

// SetAntiWindupGain sets the gain for anti-windup back-calculation
// Higher values lead to faster integral recovery when output is saturated
func (c *Controller) SetAntiWindupGain(gain float64) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if gain < 0 {
		return fmt.Errorf("invalid anti-windup gain: %f", gain)
	}

	c.antiWindupGain = gain
	return nil
}

// GetAntiWindupSettings returns the current anti-windup settings
func (c *Controller) GetAntiWindupSettings() (bool, float64) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.antiWindupEnabled, c.antiWindupGain
}
