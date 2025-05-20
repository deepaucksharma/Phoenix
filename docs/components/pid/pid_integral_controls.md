# PID Integral Controls and Safety Mechanisms

The Phoenix PID controller implements various safety mechanisms to ensure stable and reliable operation in diverse operating conditions. This document covers integral management features, derivative term filtering, and oscillation detection.

## Anti-Windup Mechanisms

The controller implements two separate but complementary anti-windup mechanisms:

1. **Integral Limits** - Simple clamping of the integral term to a maximum value
2. **Back-Calculation** - Dynamically adjusts the integral term when output saturates

### When to Use Each Approach

- **Integral Limits**: A simple approach that prevents excessive buildup but doesn't help with recovery
- **Back-Calculation**: More sophisticated approach that actively reduces integral term during saturation, leading to faster recovery

## SetIntegralLimit

`SetIntegralLimit(limit float64)` constrains the absolute value of the
internal integral. Any accumulated error beyond `±limit` will be clipped.

```go
controller := pid.NewController(1.0, 0.5, 0.1, 100.0)
controller.SetIntegralLimit(10.0)
```

Use this to guard against excessive integral buildup when errors persist.

## Back-Calculation Anti-Windup

The controller also implements back-calculation anti-windup, which actively reduces the
integral term when the output saturates. This mechanism is controlled through two methods:

### SetAntiWindupEnabled

`SetAntiWindupEnabled(enabled bool)` enables or disables the back-calculation anti-windup mechanism.
Anti-windup is enabled by default.

```go
controller := pid.NewController(1.0, 0.5, 0.1, 100.0)
controller.SetAntiWindupEnabled(false) // Disable anti-windup
```

### SetAntiWindupGain

`SetAntiWindupGain(gain float64)` sets the gain for the back-calculation anti-windup mechanism.
Higher values result in faster integral term reduction during saturation, leading to quicker recovery.

```go
controller := pid.NewController(1.0, 0.5, 0.1, 100.0)
controller.SetAntiWindupGain(2.0) // More aggressive anti-windup
```

The default gain is 1.0, which provides a balance between reduction speed and stability.

### How Back-Calculation Works

When the controller output hits a limit (either min or max), the anti-windup mechanism:

1. Calculates the "saturation error" - how much the output exceeded the limit
2. Scales this error by the anti-windup gain
3. Uses this value to decrease the integral term in the appropriate direction

```go
// When output hits maximum limit
if output > c.outputMax {
    if c.antiWindupEnabled && c.ki != 0 {
        // Reduce integral term based on saturation amount
        saturationError := c.outputMax - output
        c.integral += (saturationError * c.antiWindupGain) / c.ki
    }
    output = c.outputMax
}
```

This adjustment helps the controller recover more quickly once the system starts responding.

## ResetIntegral

`ResetIntegral()` clears the integral term entirely. Call it if the setpoint
changes drastically or after a long period of saturation.

```go
controller.ResetIntegral()
output := controller.Compute(currentValue)
```

Resetting helps the controller react quickly to new conditions.

## GetAntiWindupSettings

`GetAntiWindupSettings()` returns the current anti-windup configuration as a tuple of (enabled, gain).

```go
enabled, gain := controller.GetAntiWindupSettings()
fmt.Printf("Anti-windup: enabled=%v, gain=%.2f\n", enabled, gain)
```

## Derivative Filtering

The derivative term can be sensitive to noise in the signal. Phoenix implements a first-order low-pass filter for the derivative term to improve noise tolerance.

### SetDerivativeFilterCoefficient

`SetDerivativeFilterCoefficient(coefficient float64)` sets the filtering coefficient for the derivative term.

```go
controller := pid.NewController(1.0, 0.5, 0.1, 100.0)
controller.SetDerivativeFilterCoefficient(0.2) // More filtering
```

The `coefficient` ranges from 0 to 1:
- 1.0: No filtering (raw derivative)
- 0.0: Maximum filtering (ignores current derivative)
- Default is 0.2: Good balance for most applications

### How Derivative Filtering Works

The filter is a weighted average of the current derivative and the previous filtered derivative:

```go
// Apply filtering to the derivative to reduce noise sensitivity
currentDerivative := (error - c.lastError) / dt
previousDerivative := (c.lastError - c.prevError) / c.lastDeltaTime

// Apply low-pass filter to derivative term
filteredDerivative := c.derivativeFilterCoeff*currentDerivative +
    (1.0-c.derivativeFilterCoeff)*previousDerivative

dTerm = c.kd * filteredDerivative
```

This substantially reduces the impact of noisy signals while preserving the derivative term's ability to anticipate changes.

## Oscillation Detection and Circuit Breaking

Phoenix includes an oscillation detection mechanism that can temporarily disable or modify the controller's behavior if it detects unstable oscillations.

### EnableCircuitBreaker

`EnableCircuitBreaker(enabled bool)` enables or disables the oscillation detection circuit breaker.

```go
controller := pid.NewController(1.0, 0.5, 0.1, 100.0)
controller.EnableCircuitBreaker(true) // Enable circuit breaker (default)
```

### ConfigureCircuitBreaker

`ConfigureCircuitBreaker(sampleWindow int, thresholdPercent, minMagnitude float64, minDuration, resetDuration time.Duration)` configures the oscillation detector parameters.

```go
controller.ConfigureCircuitBreaker(
    20,               // Number of samples to analyze for oscillation
    60.0,             // % of zero crossings to consider oscillating
    0.05,             // Minimum signal magnitude to be significant
    time.Second * 30, // Minimum oscillation duration before tripping
    time.Minute * 5,  // Auto-reset time
)
```

### ResetCircuitBreaker

`ResetCircuitBreaker()` manually resets a tripped circuit breaker.

```go
if controller.GetCircuitBreakerStatus()["tripped"].(bool) {
    controller.ResetCircuitBreaker()
}
```

Use this if you want to immediately re-enable the controller after addressing the cause of oscillation.

### TemporaryOverrideCircuitBreaker

`TemporaryOverrideCircuitBreaker(duration time.Duration)` allows the controller to operate normally for a specified duration, even if the circuit breaker is tripped.

```go
// Override for 2 minutes
controller.TemporaryOverrideCircuitBreaker(time.Minute * 2)
```

This is useful for manual interventions or testing.

### GetCircuitBreakerStatus

`GetCircuitBreakerStatus()` returns detailed information about the circuit breaker's current state.

```go
status := controller.GetCircuitBreakerStatus()
fmt.Printf("Circuit breaker: tripped=%v, oscillation=%.1f%%, samples=%d\n",
    status["tripped"], status["oscillation_percent"], status["sample_count"])
```

### How Oscillation Detection Works

The oscillation detector:

1. Keeps a window of recent controller outputs (signal history)
2. Counts "zero crossings" (sign changes) in the output signal
3. If the percentage of samples showing oscillation exceeds the threshold for the minimum duration, it trips the circuit breaker
4. When tripped, the controller switches to a safer control strategy:
   ```go
   if oscillating && c.circuitBreaker.IsTripped() {
       // When oscillating, use proportional term only with reduced gain
       safeKp := c.kp * 0.1 // Use 10% of normal P gain when in safe mode
       output = safeKp * error
       
       // Reset integral to prevent windup
       c.integral = 0
   }
   ```
5. The circuit breaker will auto-reset after the configured reset duration

## Combining Safety Mechanisms

These safety mechanisms work best when used together. A typical configuration might look like:

```go
controller := pid.NewController(0.5, 0.1, 0.05, targetValue)

// Configure anti-windup
controller.SetIntegralLimit(20.0)
controller.SetAntiWindupEnabled(true)
controller.SetAntiWindupGain(1.0)

// Configure derivative filtering
controller.SetDerivativeFilterCoefficient(0.2)

// Configure oscillation detection
controller.EnableCircuitBreaker(true)
controller.ConfigureCircuitBreaker(
    20, 60.0, 0.05, 
    time.Second * 30, time.Minute * 5)
```

## Best Practices

1. **Always use integral limits**: Even with back-calculation anti-windup, setting an integral limit provides an additional safety layer
2. **Enable anti-windup for systems that can saturate**: Any system where the output might hit limits should use anti-windup
3. **Use derivative filtering when the signal is noisy**: For systems with measurement noise, set the derivative filter coefficient to 0.1-0.3
4. **Enable circuit breakers in production**: To prevent unstable oscillations, always enable the circuit breaker in production environments

## Reference Configurations

### For Noisy Systems
```go
// More filtering, less aggressive gains
controller.SetDerivativeFilterCoefficient(0.1) // Heavy filtering
controller.SetTunings(0.3, 0.05, 0.02) // Conservative tuning
```

### For Systems That Saturate Easily
```go
// Focus on anti-windup
controller.SetIntegralLimit(10.0) // Smaller integral limit
controller.SetAntiWindupGain(1.5) // More aggressive anti-windup
controller.SetOutputLimits(minValue, maxValue) // Define output limits
```

### For Mission-Critical Systems
```go
// Maximum safety
controller.EnableCircuitBreaker(true)
controller.ConfigureCircuitBreaker(15, 50.0, 0.1, time.Second*15, time.Minute*2)
controller.SetIntegralLimit(5.0) // Very conservative integral limit
controller.SetDerivativeFilterCoefficient(0.15) // Moderate filtering
```