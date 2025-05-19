// Package pid implements a Proportional-Integral-Derivative controller
package pid

import (
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
	setpoint      float64     // Target value
	lastError     float64     // Last error value
	integral      float64     // Accumulated error
	lastTime      time.Time   // Last update time
	
	// Limits
	integralLimit float64     // Maximum absolute value for integral term
	outputMin     float64     // Minimum output value
	outputMax     float64     // Maximum output value
	
	lock          sync.Mutex  // For thread safety
}

// NewController creates a new PID controller with the specified gains
func NewController(kp, ki, kd, setpoint float64) *Controller {
	return &Controller{
		kp:            kp,
		ki:            ki,
		kd:            kd,
		setpoint:      setpoint,
		lastError:     0,
		integral:      0,
		lastTime:      time.Now(),
		integralLimit: 1000, // Default, can be changed with SetIntegralLimit
		outputMin:     -1000,
		outputMax:     1000,
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
	
	// Apply output limits
	if output > c.outputMax {
		output = c.outputMax
	} else if output < c.outputMin {
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
