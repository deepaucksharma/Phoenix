# PID Controller Implementation

The PID controller is a key component of the Phoenix SA-OMF system, providing feedback-based control for adaptive processors. This document describes its implementation and usage.

## Overview

The PID controller implements a standard Proportional-Integral-Derivative control algorithm with several enhancements for stability and reliability:

- Anti-windup protection to prevent integral term saturation
- Low-pass filtering for the derivative term to reduce noise sensitivity
- Oscillation detection with circuit breaker capability
- Thread-safety for concurrent access
- Comprehensive metrics and logging

## Core Implementation 

The implementation is divided into two main files:

1. `controller.go` - The main PID controller implementation
2. `circuitbreaker.go` - Oscillation detection and circuit breaking

### Controller Structure

```go
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
    lastDeltaTime float64   // Last time step in seconds

    // Limits
    integralLimit float64 // Maximum absolute value for integral term
    outputMin     float64 // Minimum output value
    outputMax     float64 // Maximum output value

    // Anti-windup
    antiWindupEnabled bool    // Whether anti-windup protection is enabled
    antiWindupGain    float64 // Gain for anti-windup back-calculation

    // Derivative filtering
    derivativeFilterCoeff float64 // Coefficient for derivative low-pass filter

    // Circuit Breaker
    circuitBreaker        *OscillationDetector 
    circuitBreakerEnabled bool

    // Metrics
    name             string
    metricsCollector *metrics.PIDMetrics

    lock sync.Mutex // For thread safety
}
```

### Key Methods

- `NewController(kp, ki, kd, setpoint float64) *Controller`
- `Compute(currentValue float64) float64`
- `SetTunings(kp, ki, kd float64)`
- `SetOutputLimits(min, max float64)`
- `SetIntegralLimit(limit float64)`
- `ResetIntegral()`
- `SetAntiWindupEnabled(enabled bool)`
- `SetDerivativeFilterCoefficient(coefficient float64)`
- `EnableCircuitBreaker(enabled bool)`
- `ConfigureCircuitBreaker(params...)`

### Oscillation Detection

The `OscillationDetector` monitors the controller output for patterns of oscillation:

```go
type OscillationDetector struct {
    // Configuration
    sampleWindow       int           // Number of samples to track
    oscillationThresholdPercent float64 // Percentage of zero crossings required
    minSignalMagnitude float64      // Minimum magnitude for significance
    minDuration       time.Duration // Minimum duration before tripping
    resetDuration     time.Duration // Auto-reset duration
    
    // State
    signalHistory      []float64    
    valueHistory       []float64    
    signalTimeHistory  []time.Time  
    isTripped          bool        
    tripTime           time.Time    
    overrideUntil      time.Time    
    
    lock               sync.RWMutex
}
```

The detector analyzes the pattern of zero-crossings (sign changes) in the controller output signal and trips the circuit breaker if oscillations exceed the configured thresholds.

## Using the PID Controller

### Basic Usage

```go
// Create a new controller with P=0.5, I=0.1, D=0.05, and setpoint=90.0
controller := pid.NewController(0.5, 0.1, 0.05, 90.0)

// Set limits on output (e.g., for a parameter that must be between 10 and 100)
controller.SetOutputLimits(10.0, 100.0)

// Set integral limit to prevent excessive windup
controller.SetIntegralLimit(100.0)

// In your control loop, compute the output based on the current measurement
for {
    // Get the current measurement from your system
    currentValue := getMeasurement()
    
    // Compute the control output
    output := controller.Compute(currentValue)
    
    // Apply the output to your system
    applyControlOutput(output)
    
    // Wait for next cycle
    time.Sleep(time.Second)
}
```

### With Metrics Collection

```go
// Create a metrics emitter
metricsEmitter := metrics.NewMetricsEmitter("pid_controller", componentID)

// Create a controller with metrics
controller := pid.NewControllerWithMetrics(0.5, 0.1, 0.05, 90.0, "coverage_controller", metricsEmitter)

// Same usage as basic controller, but metrics will be automatically collected
```

### With Oscillation Detection

```go
// Enable the circuit breaker (enabled by default)
controller.EnableCircuitBreaker(true)

// Configure the circuit breaker parameters
controller.ConfigureCircuitBreaker(
    20,               // sampleWindow
    60.0,             // thresholdPercent
    0.05,             // minSignalMagnitude
    time.Second * 30, // minDuration
    time.Minute * 5,  // resetDuration
)

// Get the circuit breaker status
status := controller.GetCircuitBreakerStatus()
```

## Integration with Adaptive Processors

The PID controller is designed to be integrated with adaptive processors via the `adaptive_pid` processor, which:

1. Monitors KPI metrics through the metrics pipeline
2. Uses PID controllers to calculate configuration adjustments
3. Generates configuration patches for target processors
4. Submits these patches to the `pic_control_ext` extension

This enables a closed feedback loop where processors adjust their parameters based on their observed effects on system KPIs.

## Best Practices

1. **Start Conservative**: Begin with low values for kp, ki, and kd, then increase gradually
2. **Monitor Oscillations**: Watch for oscillations and enable the circuit breaker
3. **Use Anti-Windup**: Always enable anti-windup protection for integral terms
4. **Filter Derivative**: Use derivative filtering (coefficient 0.1-0.3) to reduce noise sensitivity
5. **Set Appropriate Limits**: Configure output limits and integral limits based on parameter constraints

## References

- [PID Controller Theory](https://en.wikipedia.org/wiki/PID_controller)
- [Anti-Windup Techniques](https://en.wikipedia.org/wiki/Integral_windup)
- [PID Controller Tuning Guide](pid_controller_tuning.md)
- [PID Integral Controls](pid_integral_controls.md)