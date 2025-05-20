// Package pid implements a Proportional-Integral-Derivative controller
package pid

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/deepaucksharma/Phoenix/pkg/metrics"
)

// StreamlinedController implements a PID (Proportional-Integral-Derivative) controller
// with streamlined configuration and integrated metrics using the unified metrics system.
type StreamlinedController struct {
	// Configuration
	config ControllerConfig

	// State
	lastError     float64   // Last error value
	prevError     float64   // Error from two steps ago (for filtered derivative)
	integral      float64   // Accumulated error
	lastTime      time.Time // Last update time
	lastDeltaTime float64   // Last time step in seconds (for consistent derivative)
	
	// Output history for oscillation detection
	outputHistory []float64
	outputIndex   int
	
	// Metrics
	metricsCollector *metrics.UnifiedMetricsCollector
	logger           *zap.Logger
	
	lock sync.Mutex // For thread safety
}

// ControllerConfig holds the configuration for a PID controller
type ControllerConfig struct {
	// Identification
	Name string `json:"name" yaml:"name"`
	
	// PID constants
	Kp float64 `json:"kp" yaml:"kp"`
	Ki float64 `json:"ki" yaml:"ki"`
	Kd float64 `json:"kd" yaml:"kd"`
	
	// Target value and limits
	Setpoint      float64 `json:"setpoint" yaml:"setpoint"`
	OutputMin     float64 `json:"output_min" yaml:"output_min"`
	OutputMax     float64 `json:"output_max" yaml:"output_max"`
	IntegralLimit float64 `json:"integral_limit" yaml:"integral_limit"`
	
	// Tuning options
	AntiWindupEnabled     bool    `json:"anti_windup_enabled" yaml:"anti_windup_enabled"`
	AntiWindupGain        float64 `json:"anti_windup_gain" yaml:"anti_windup_gain"`
	DerivativeFilterCoeff float64 `json:"derivative_filter_coeff" yaml:"derivative_filter_coeff"`
	
	// Oscillation detection
	OscillationDetectionEnabled bool    `json:"oscillation_detection_enabled" yaml:"oscillation_detection_enabled"`
	OscillationThreshold        float64 `json:"oscillation_threshold" yaml:"oscillation_threshold"`
	OscillationWindowSize       int     `json:"oscillation_window_size" yaml:"oscillation_window_size"`
	
	// Hysteresis to prevent frequent changes
	HysteresisPercent float64 `json:"hysteresis_percent" yaml:"hysteresis_percent"`
}

// NewStreamlinedController creates a new PID controller with streamlined configuration
func NewStreamlinedController(config ControllerConfig, logger *zap.Logger) *StreamlinedController {
	// Initialize output history for oscillation detection
	outputHistory := make([]float64, config.OscillationWindowSize)
	
	// Validate configuration
	if config.IntegralLimit <= 0 {
		config.IntegralLimit = 100 // Default integral limit
	}
	
	if config.OscillationWindowSize <= 0 {
		config.OscillationWindowSize = 10 // Default window size
	}
	
	if config.DerivativeFilterCoeff < 0 || config.DerivativeFilterCoeff > 1 {
		config.DerivativeFilterCoeff = 0.2 // Default filter coefficient
	}
	
	// Create metrics collector
	metricsCollector := metrics.NewUnifiedMetricsCollector(logger)
	metricsCollector.AddDefaultAttribute("controller", config.Name)
	
	return &StreamlinedController{
		config:           config,
		lastError:        0,
		prevError:        0,
		integral:         0,
		lastTime:         time.Now(),
		lastDeltaTime:    0.1, // Default initial time step
		outputHistory:    outputHistory,
		outputIndex:      0,
		metricsCollector: metricsCollector,
		logger:           logger,
	}
}

// UpdateMetricsCollector sets a new metrics collector
func (c *StreamlinedController) UpdateMetricsCollector(collector *metrics.UnifiedMetricsCollector) {
	c.lock.Lock()
	defer c.lock.Unlock()
	
	c.metricsCollector = collector
	c.metricsCollector.AddDefaultAttribute("controller", c.config.Name)
}

// Compute calculates a control output based on the current measurement
func (c *StreamlinedController) Compute(ctx context.Context, measurement float64) (float64, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	
	now := time.Now()
	dt := now.Sub(c.lastTime).Seconds()
	
	// Use the most consistent time step for more stable derivatives
	// If the time step is too small, use the last one to avoid numerical issues
	if dt < 0.001 {
		dt = c.lastDeltaTime
	} else {
		c.lastDeltaTime = dt
	}
	
	// Calculate error
	error := c.config.Setpoint - measurement
	
	// Check if within hysteresis band
	if c.config.HysteresisPercent > 0 {
		hysteresisThreshold := c.config.Setpoint * (c.config.HysteresisPercent / 100.0)
		if abs(error) < hysteresisThreshold {
			// Within hysteresis band, consider error as zero
			error = 0
		}
	}
	
	// Calculate proportional term
	proportional := c.config.Kp * error
	
	// Calculate integral term with anti-windup protection
	c.integral += c.config.Ki * error * dt
	
	// Apply integral limits to prevent windup
	if c.integral > c.config.IntegralLimit {
		c.integral = c.config.IntegralLimit
	} else if c.integral < -c.config.IntegralLimit {
		c.integral = -c.config.IntegralLimit
	}
	
	// Filtered derivative term to reduce noise sensitivity
	// Using a low-pass filter on the derivative
	derivativeRaw := (error - c.lastError) / dt
	derivativeFiltered := c.config.DerivativeFilterCoeff*derivativeRaw + 
		(1-c.config.DerivativeFilterCoeff)*(c.lastError-c.prevError)/c.lastDeltaTime
	derivative := c.config.Kd * derivativeFiltered
	
	// Calculate output
	output := proportional + c.integral + derivative
	
	// Apply output limits
	var limitedOutput float64
	if output > c.config.OutputMax {
		limitedOutput = c.config.OutputMax
	} else if output < c.config.OutputMin {
		limitedOutput = c.config.OutputMin
	} else {
		limitedOutput = output
	}
	
	// Anti-windup: adjust integral term if output is limited
	if c.config.AntiWindupEnabled && limitedOutput != output {
		// Back-calculation anti-windup
		windupError := limitedOutput - output
		c.integral += c.config.AntiWindupGain * windupError * dt
	}
	
	// Oscillation detection
	if c.config.OscillationDetectionEnabled {
		c.outputHistory[c.outputIndex] = limitedOutput
		c.outputIndex = (c.outputIndex + 1) % len(c.outputHistory)
		
		if c.detectOscillation() {
			c.logger.Warn("Oscillation detected in PID controller",
				zap.String("controller", c.config.Name),
				zap.Float64("measurement", measurement),
				zap.Float64("setpoint", c.config.Setpoint),
				zap.Float64("output", limitedOutput))
			
			// Reset integral term to dampen oscillations
			c.integral = 0
		}
	}
	
	// Update state for next iteration
	c.prevError = c.lastError
	c.lastError = error
	c.lastTime = now
	
	// Record metrics
	if c.metricsCollector != nil {
		c.recordMetrics(measurement, error, proportional, c.integral, derivative, limitedOutput)
	}
	
	return limitedOutput, nil
}

// detectOscillation checks for oscillatory behavior in the output history
func (c *StreamlinedController) detectOscillation() bool {
	if len(c.outputHistory) < 4 {
		return false // Need at least 4 points to detect oscillations
	}
	
	// Count sign changes in the differences between consecutive outputs
	signChanges := 0
	prevSign := 0
	
	for i := 1; i < len(c.outputHistory); i++ {
		diff := c.outputHistory[i] - c.outputHistory[i-1]
		if abs(diff) < c.config.OscillationThreshold {
			continue // Ignore small changes
		}
		
		currentSign := 0
		if diff > 0 {
			currentSign = 1
		} else if diff < 0 {
			currentSign = -1
		}
		
		if prevSign != 0 && currentSign != 0 && prevSign != currentSign {
			signChanges++
		}
		
		if currentSign != 0 {
			prevSign = currentSign
		}
	}
	
	// If more than half the points show sign changes, we're oscillating
	return signChanges >= len(c.outputHistory)/2
}

// recordMetrics records performance metrics for the PID controller
func (c *StreamlinedController) recordMetrics(measurement, error, proportional, integral, derivative, output float64) {
	// Record process variable
	c.metricsCollector.AddGauge("aemf_pid_measurement", "Current measured value", "").
		WithValue(measurement).
		WithAttributes(map[string]string{
			"controller": c.config.Name,
		})
	
	// Record error
	c.metricsCollector.AddGauge("aemf_pid_error", "Error (setpoint - measurement)", "").
		WithValue(error).
		WithAttributes(map[string]string{
			"controller": c.config.Name,
		})
	
	// Record PID terms
	c.metricsCollector.AddGauge("aemf_pid_proportional", "Proportional term", "").
		WithValue(proportional).
		WithAttributes(map[string]string{
			"controller": c.config.Name,
		})
	
	c.metricsCollector.AddGauge("aemf_pid_integral", "Integral term", "").
		WithValue(integral).
		WithAttributes(map[string]string{
			"controller": c.config.Name,
		})
	
	c.metricsCollector.AddGauge("aemf_pid_derivative", "Derivative term", "").
		WithValue(derivative).
		WithAttributes(map[string]string{
			"controller": c.config.Name,
		})
	
	// Record output
	c.metricsCollector.AddGauge("aemf_pid_output", "Controller output", "").
		WithValue(output).
		WithAttributes(map[string]string{
			"controller": c.config.Name,
		})
}

// Reset resets the controller state (integral, errors)
func (c *StreamlinedController) Reset() {
	c.lock.Lock()
	defer c.lock.Unlock()
	
	c.integral = 0
	c.lastError = 0
	c.prevError = 0
	c.lastTime = time.Now()
	c.lastDeltaTime = 0.1
	
	// Clear output history
	for i := range c.outputHistory {
		c.outputHistory[i] = 0
	}
	c.outputIndex = 0
}

// UpdateConfig updates the controller configuration
func (c *StreamlinedController) UpdateConfig(config ControllerConfig) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	
	// Validate the new configuration
	if config.Name == "" {
		return fmt.Errorf("controller name cannot be empty")
	}
	
	// Update the configuration
	oldConfig := c.config
	c.config = config
	
	// Reset if setpoint has changed significantly
	if abs(oldConfig.Setpoint-config.Setpoint)/abs(oldConfig.Setpoint) > 0.1 {
		c.integral = 0
	}
	
	// If oscillation window size has changed, create a new history buffer
	if len(c.outputHistory) != config.OscillationWindowSize && config.OscillationWindowSize > 0 {
		c.outputHistory = make([]float64, config.OscillationWindowSize)
		c.outputIndex = 0
	}
	
	c.logger.Info("PID controller configuration updated",
		zap.String("controller", c.config.Name),
		zap.Float64("kp", c.config.Kp),
		zap.Float64("ki", c.config.Ki),
		zap.Float64("kd", c.config.Kd),
		zap.Float64("setpoint", c.config.Setpoint))
	
	return nil
}

// GetConfig returns the current controller configuration
func (c *StreamlinedController) GetConfig() ControllerConfig {
	c.lock.Lock()
	defer c.lock.Unlock()
	
	return c.config
}

// EmitMetrics emits all collected metrics
func (c *StreamlinedController) EmitMetrics(ctx context.Context) error {
	if c.metricsCollector != nil {
		return c.metricsCollector.Emit(ctx)
	}
	return nil
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}