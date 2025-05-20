# PID Integral Controls

The Phoenix PID controller exposes helper methods to manage the integral term.
This prevents windup and allows you to reset accumulated error when the
operating conditions change.

## Anti-Windup Mechanisms

The controller implements two separate but complementary anti-windup mechanisms:

1. **Integral Limits** - Simple clamping of the integral term to a maximum value
2. **Back-Calculation** - Dynamically adjusts the integral term when output saturates

### When to Use Each Approach

- **Integral Limits**: A simple approach that prevents excessive buildup but doesn't help with recovery
- **Back-Calculation**: More sophisticated approach that actively reduces integral term during saturation, leading to faster recovery

> **Note**: `pid.NewController` now returns `(*Controller, error)` to report invalid gain values. Update existing code accordingly.

## SetIntegralLimit

`SetIntegralLimit(limit float64)` constrains the absolute value of the
internal integral. Any accumulated error beyond `±limit` will be clipped.

```go
controller, err := pid.NewController(1.0, 0.5, 0.1, 100.0)
if err != nil {
    log.Fatalf("controller init: %v", err)
}
controller.SetIntegralLimit(10.0)
```

Use this to guard against excessive integral buildup when errors persist.

## SetOutputLimits

`SetOutputLimits(min, max float64) error` defines the allowable output range for the controller.
An error is returned if `min` is not less than `max`.

```go
controller := pid.NewController(1.0, 0.5, 0.1, 100.0)
if err := controller.SetOutputLimits(-10.0, 10.0); err != nil {
    // handle invalid limits
}
```


## Back-Calculation Anti-Windup

The controller also implements back-calculation anti-windup, which actively reduces the
integral term when the output saturates. This mechanism is controlled through two methods:

### SetAntiWindupEnabled

`SetAntiWindupEnabled(enabled bool)` enables or disables the back-calculation anti-windup mechanism.
Anti-windup is enabled by default.

```go
controller, err := pid.NewController(1.0, 0.5, 0.1, 100.0)
if err != nil {
    log.Fatal(err)
}
controller.SetAntiWindupEnabled(false) // Disable anti-windup
```

### SetAntiWindupGain

`SetAntiWindupGain(gain float64) error` sets the gain for the back-calculation anti-windup mechanism.
Higher values result in faster integral term reduction during saturation, leading to quicker recovery.
The method returns an error if a negative gain value is supplied.

```go
controller, err := pid.NewController(1.0, 0.5, 0.1, 100.0)
if err != nil {
    log.Fatal(err)
}
err = controller.SetAntiWindupGain(2.0) // More aggressive anti-windup
if err != nil {
    // handle invalid gain
}
```

The default gain is 1.0, which provides a balance between reduction speed and stability.

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
