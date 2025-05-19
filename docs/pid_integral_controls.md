# PID Integral Controls

The Phoenix PID controller exposes helper methods to manage the integral term.
This prevents windup and allows you to reset accumulated error when the
operating conditions change.

## SetIntegralLimit

`SetIntegralLimit(limit float64)` constrains the absolute value of the
internal integral. Any accumulated error beyond `\pm limit` will be clipped.

```go
controller := pid.NewController(1.0, 0.5, 0.1, 100.0)
controller.SetIntegralLimit(10.0)
```

Use this to guard against excessive integral buildup when errors persist.

## ResetIntegral

`ResetIntegral()` clears the integral term entirely. Call it if the setpoint
changes drastically or after a long period of saturation.

```go
controller.ResetIntegral()
output := controller.Compute(currentValue)
```

Resetting helps the controller react quickly to new conditions.

