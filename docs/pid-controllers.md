# PID Controllers in Phoenix

This document describes the PID controller implementation in Phoenix, its usage, and tuning guidelines, with a focus on advanced stability features.

## What is a PID Controller?

A PID (Proportional-Integral-Derivative) controller is a control loop mechanism that uses feedback to adjust a system parameter. In Phoenix, PID controllers adjust processor parameters based on observed metrics to maintain target performance.

## Core Implementation

The PID controller implementation in Phoenix includes these key features:

- Traditional PID algorithm with configurable P, I, and D terms
- Anti-windup protection to prevent integral term saturation
- Low-pass derivative filtering to reduce noise sensitivity
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

## Advanced Stability Features

### Anti-Windup Protection

Integral windup occurs when a large error causes the integral term to grow excessively large, resulting in overshoots when the system finally responds. Phoenix implements "back-calculation" anti-windup:

1. When the controller output hits the defined limits, it calculates how much "over" the limit it would have gone
2. This over-saturation value is used to reduce the integral term
3. The `antiWindupGain` parameter controls how aggressively to back-calculate the integral term

```go
// Anti-windup back-calculation when output is saturated at max
if output > c.outputMax {
    if c.antiWindupEnabled && c.ki != 0 {
        // Reduce integral term based on saturation amount
        saturationError := c.outputMax - output
        c.integral += (saturationError * c.antiWindupGain) / c.ki
    }
    output = c.outputMax
}
```

### Derivative Filtering

The derivative term can amplify noise in the measured signal. Phoenix applies a first-order low-pass filter to the derivative calculation to smooth out noise:

```go
// Apply filtering to the derivative to reduce noise sensitivity
currentDerivative := (error - c.lastError) / dt
previousDerivative := (c.lastError - c.prevError) / c.lastDeltaTime

// Apply low-pass filter to derivative term
filteredDerivative := c.derivativeFilterCoeff*currentDerivative +
    (1.0-c.derivativeFilterCoeff)*previousDerivative

dTerm = c.kd * filteredDerivative
```

The `derivativeFilterCoeff` ranges from 0 to 1:
- Value of 1.0: No filtering (use raw derivative)
- Value of 0.0: Ignore current derivative entirely
- Typical values: 0.1-0.3 for good noise reduction while maintaining responsiveness

### Oscillation Detection and Circuit Breaking

Phoenix includes an advanced oscillation detection mechanism that monitors the controller output for oscillation patterns and can temporarily disable the controller when unstable behavior is detected:

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

The detector:
1. Tracks a window of recent controller outputs and measurements
2. Counts "zero crossings" (sign changes) in the controller output
3. If the percentage of samples showing oscillation exceeds a threshold for a defined duration, it trips the circuit breaker
4. When the circuit breaker is tripped, a safer control strategy is used (proportional-only with reduced gain)
5. The circuit breaker automatically resets after a configurable duration, or it can be manually reset

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

## Integration with Bayesian Optimization

For complex parameter spaces or when PID control stalls, Phoenix can automatically switch to Bayesian optimization:

```go
// In adaptive_pid processor config
use_bayesian: true
stall_threshold: 3  // Number of consecutive ineffective PID adjustments before switching
```

The Bayesian optimizer:
1. Uses a Gaussian Process (GP) model to learn the relationship between parameters and KPIs
2. Uses an acquisition function (Upper Confidence Bound) to balance exploration and exploitation
3. Efficiently explores the parameter space using Latin Hypercube Sampling
4. Can discover optimal parameter values even in complex, non-linear parameter spaces

See [Bayesian Optimization](bayesian-optimization.md) for more details.

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
  circuit_breaker:
    enabled: true
    sample_window: 30
    threshold_percent: 60.0
    min_duration: 60s
```

## Key Metrics to Monitor

Phoenix emits detailed metrics about PID controller behavior. Key metrics to watch:

| Metric | Description | When to Investigate |
|--------|-------------|---------------------|
| `aemf_pid_controller_error` | Current error value | Large oscillations or persistent non-zero values |
| `aemf_pid_controller_output` | Controller output | Hitting min/max limits consistently |
| `aemf_pid_controller_p_term` | Proportional term contribution | Unusually large values |
| `aemf_pid_controller_i_term` | Integral term contribution | Growing continuously (windup) |
| `aemf_pid_controller_d_term` | Derivative term contribution | Spikes indicating noise |
| `aemf_controller_pid_circuit_breaker_trips_total` | Circuit breaker activations | Frequent tripping suggests instability |
| `aemf_pid_output_clamped_total` | Output limit hits | Frequent clamping suggests wrong output limits |

## Best Practices

1. **Start Conservative**: Begin with low values for kp, ki, and kd, then increase gradually
2. **Monitor Oscillations**: Watch for oscillations and enable the circuit breaker
3. **Use Anti-Windup**: Always enable anti-windup protection for integral terms
4. **Filter Derivative**: Use derivative filtering to reduce noise sensitivity
5. **Set Appropriate Limits**: Configure output limits and integral limits based on parameter constraints
6. **Test with Real Workloads**: Final tuning should be done with representative workloads
7. **Consider Bayesian Fallback**: For complex parameter spaces, enable Bayesian optimization fallback

## Troubleshooting

| Issue | Symptoms | Solutions |
|-------|----------|-----------|
| **Oscillation** | Parameter value rapidly changes back and forth | Reduce kp and ki values, increase derivative term, enable circuit breaker, increase hysteresis |
| **Slow Response** | System takes too long to reach target | Increase kp, check if output is hitting limits, decrease hysteresis |
| **Overshooting** | Parameter exceeds target significantly before settling | Increase kd, reduce kp and ki, add anti-windup |
| **Steady-State Error** | System stabilizes but doesn't reach target | Increase ki to eliminate persistent error |
| **Noisy Output** | Parameter changes too much in response to noise | Increase derivative filtering, reduce kd |
| **Windup** | Integral term grows excessively | Enable anti-windup, set appropriate integral limits |

## References

- [PID Controller Theory](https://en.wikipedia.org/wiki/PID_controller)
- [Anti-Windup Techniques](https://en.wikipedia.org/wiki/Integral_windup)
- [Digital PID Controllers](https://www.sciencedirect.com/topics/engineering/digital-pid-controller)
- [Architecture Decision: Self-Regulating PID Control](architecture/adr/20250519-use-self-regulating-pid-control-for-adaptive-processing.md)
- [Bayesian Optimization](bayesian-optimization.md)
- [PID Integral Controls](components/pid/pid_integral_controls.md)