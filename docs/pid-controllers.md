# PID Controllers in Phoenix

This document describes the PID controller implementation in Phoenix, its usage, and tuning guidelines.

## What is a PID Controller?

A PID (Proportional-Integral-Derivative) controller is a control loop mechanism that uses feedback to adjust a system parameter. In Phoenix, PID controllers adjust processor parameters based on observed metrics to maintain target performance.

## Core Implementation

The PID controller implementation includes these key features:

- Traditional PID algorithm with configurable P, I, and D terms
- Anti-windup protection to prevent integral term saturation
- Low-pass filtering for derivative term to reduce noise sensitivity
- Oscillation detection with circuit breaker capability
- Thread-safety for concurrent access
- Comprehensive metrics and logging

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

## Oscillation Detection

The PID controller includes a sophisticated oscillation detection mechanism:

```go
type OscillationDetector struct {
    // Configuration
    sampleWindow               int           // Number of samples to track
    oscillationThresholdPercent float64      // Percentage of zero crossings required
    minSignalMagnitude         float64       // Minimum magnitude for significance
    minDuration                time.Duration // Minimum duration before tripping
    resetDuration              time.Duration // Auto-reset duration
    
    // State
    signalHistory     []float64    
    valueHistory      []float64    
    signalTimeHistory []time.Time  
    isTripped         bool        
    tripTime          time.Time    
    overrideUntil     time.Time    
    
    lock              sync.RWMutex
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

## Tuning PID Controllers

Properly tuning PID controllers is critical for effective adaptation. Here are guidelines for each parameter:

### Basic Tuning Parameters

| Parameter | Description | Typical Values | Effect of Increase | Effect of Decrease |
|-----------|-------------|----------------|-------------------|-------------------|
| kp | Proportional gain | 0.1 - 2.0 | Faster response, may cause oscillation | Slower response, more stable |
| ki | Integral gain | 0.01 - 0.5 | Eliminates steady-state error, may cause overshoot | Less overshoot, but persistent error |
| kd | Derivative gain | 0.01 - 0.3 | Reduces overshoot, improves stability | Less damping, may oscillate |

### Advanced Parameters

| Parameter | Description | Typical Values |
|-----------|-------------|----------------|
| derivativeFilterCoeff | Low-pass filter coefficient | 0.1 - 0.3 |
| integralLimit | Maximum accumulated integral term | 10.0 - 100.0 |
| antiWindupGain | Gain for anti-windup back-calculation | 0.5 - 1.0 |

### Oscillation Detection Parameters

| Parameter | Description | Typical Values |
|-----------|-------------|----------------|
| sampleWindow | Number of samples to analyze | 20 - 50 |
| oscillationThresholdPercent | % of zero crossings required | 50% - 70% |
| minSignalMagnitude | Minimum signal magnitude | 0.01 - 0.1 |
| minDuration | Minimum duration before tripping | 30s - 2min |
| resetDuration | Auto-reset duration | 5min - 30min |

### Tuning Methodology

1. **Start with P-only**: Set ki=0, kd=0, and adjust kp until you get a reasonable response
2. **Add I term**: Increase ki slowly until steady-state error is eliminated
3. **Add D term**: Add a small amount of kd to improve stability
4. **Fine-tune**: Adjust all three parameters to optimize performance
5. **Add safety features**: Configure integral limits, anti-windup, and oscillation detection

### Example Configurations

#### Aggressive Tuning (Fast Response)
```yaml
pid:
  kp: 1.0
  ki: 0.2
  kd: 0.1
  integral_limit: 50.0
  derivative_filter_coefficient: 0.2
```

#### Conservative Tuning (More Stable)
```yaml
pid:
  kp: 0.3
  ki: 0.05
  kd: 0.02
  integral_limit: 20.0
  derivative_filter_coefficient: 0.1
```

## Best Practices

1. **Start Conservative**: Begin with low values for kp, ki, and kd, then increase gradually
2. **Monitor Oscillations**: Watch for oscillations and enable the circuit breaker
3. **Use Anti-Windup**: Always enable anti-windup protection for integral terms
4. **Filter Derivative**: Use derivative filtering to reduce noise sensitivity
5. **Set Appropriate Limits**: Configure output limits and integral limits based on parameter constraints
6. **Test with Real Workloads**: Final tuning should be done with representative workloads

## References

- [PID Controller Theory](https://en.wikipedia.org/wiki/PID_controller)
- [Anti-Windup Techniques](https://en.wikipedia.org/wiki/Integral_windup)
- [Digital PID Controllers](https://www.sciencedirect.com/topics/engineering/digital-pid-controller)