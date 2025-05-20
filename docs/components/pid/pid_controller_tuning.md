# PID Controller Design and Tuning Guidelines

The Phoenix system uses PID (Proportional-Integral-Derivative) controllers for dynamic adaptation of processing parameters. This document provides an overview of the PID controller implementation and guidelines for tuning.

## Overview

The PID controller is a feedback control mechanism that continuously calculates an error value as the difference between a measured process variable and a desired setpoint (target value). The controller adjusts an output value based on proportional, integral, and derivative terms to minimize this error over time.

### Key Files

- `internal/control/pid/controller.go`: Core PID controller implementation
- `internal/control/pid/circuitbreaker.go`: Oscillation detection circuit breaker
- `internal/processor/adaptive_pid/processor.go`: Processor that uses PID controllers to generate config patches
- `internal/extension/pic_control_ext/extension.go`: Extension that applies config patches to processors

## Controller Features

### Basic PID Control

The controller implements standard PID control with configurable gains:

- **Proportional (P)**: Responds to the current error (how far we are from the target)
- **Integral (I)**: Responds to accumulated error over time (eliminates steady-state error)
- **Derivative (D)**: Responds to the rate of change of error (provides damping)

### Anti-Windup Protection

Integral windup occurs when the controller output saturates at its limits while a persistent error causes the integral term to grow excessively, leading to overshoot and oscillation when the error changes direction.

Our implementation uses back-calculation anti-windup, which reduces the integral term when the output is saturated, allowing faster recovery when the direction changes.

### Oscillation Detection

The oscillation detector monitors the controller output and measured value for oscillatory patterns by tracking zero-crossings (sign changes) in the control signal. When oscillation is detected, the circuit breaker can be tripped to prevent instability.

### Noise Reduction

Derivative action can amplify noise in the measured signal. Our implementation uses a first-order low-pass filter on the derivative term to reduce noise sensitivity while still providing damping action.

## Tuning Guidelines

### Basic Tuning Process

1. **Start with P-only control**: Set ki and kd to 0, increase kp until the system responds quickly but may oscillate.
2. **Add Integral term**: Gradually increase ki to eliminate steady-state error, but not so high it causes overshoot.
3. **Add Derivative term**: Add kd to dampen oscillations and improve stability, but keep it moderate to avoid noise amplification.

### Recommended Starting Points

| Parameter | Coverage Control | Cardinality Control | Resource Usage Control |
|-----------|------------------|---------------------|------------------------|
| kp        | 0.5              | 0.3                 | 0.2                    |
| ki        | 0.1              | 0.05                | 0.03                   |
| kd        | 0.05             | 0.02                | 0.01                   |

### Fine-Tuning Parameters

#### Anti-Windup Settings

```yaml
integral_windup_limit: 1000.0  # Maximum absolute value for integral term
anti_windup_gain: 1.0          # Gain for back-calculation (higher = faster recovery)
```

#### Oscillation Detection

```yaml
oscillation:
  sample_window: 20            # Number of samples to track
  threshold_percent: 60.0      # Percentage of zero crossings required to detect oscillation
  min_magnitude: 0.05          # Minimum magnitude for a signal to be considered significant
  min_duration: 30s            # Minimum duration of oscillation before tripping
  reset_duration: 5m           # Time after which to auto-reset the circuit breaker
```

#### Derivative Filtering

```yaml
derivative_filter_coefficient: 0.2  # Value between 0-1, lower = more filtering
```

### Diagnosing Problems

#### Slow Response

- Increase kp
- Increase ki (but watch for overshoot)
- Decrease derivative_filter_coefficient

#### Oscillation

- Decrease kp
- Decrease ki
- Increase kd
- Enable circuit_breaker if not already enabled

#### Overshoot

- Decrease kp
- Decrease ki
- Increase kd
- Reduce anti_windup_gain

#### Noise Sensitivity

- Decrease kd
- Decrease derivative_filter_coefficient
- Add time-based hysteresis

## Configuration Example

Below is an example PID controller configuration for the policy.yaml file:

```yaml
adaptive_pid:
  controllers:
    - name: "coverage_controller"
      enabled: true
      kpi_metric_name: "aemf.adaptive_topk.coverage_percent"
      kpi_target_value: 95.0
      kp: 0.5
      ki: 0.1
      kd: 0.05
      integral_windup_limit: 1000.0
      hysteresis_percent: 1.0
      output_config_patches:
        - target_processor_name: "adaptive_topk"
          parameter_path: "k_value"
          min_value: 10
          max_value: 1000
          change_scale_factor: 1.0
      use_bayesian: false
```

## Monitoring and Observability

The PID controller emits metrics for monitoring its behavior:

- `aemf.controller.pid.<name>.error`: The error value (target - current)
- `aemf.controller.pid.<name>.p_term`: The proportional term
- `aemf.controller.pid.<name>.i_term`: The integral term
- `aemf.controller.pid.<name>.d_term`: The derivative term
- `aemf.controller.pid.<name>.output`: The raw controller output
- `aemf.controller.pid.<name>.actual_output`: The limited/clamped output
- `aemf.controller.pid.<name>.circuit_breaker_trips_total`: Count of circuit breaker trips

## Advanced Topics

### Multi-Parameter Control

When controlling multiple parameters, consider:
1. **Parameter Independence**: Ensure parameters don't significantly affect each other
2. **Relative Scaling**: Scale parameters to similar ranges for consistent control
3. **Bayesian Optimization**: For multiple interacting parameters, enable Bayesian optimization

### Bayesian Optimization

For complex parameter spaces, the adaptive_pid processor supports Bayesian optimization as an alternative to PID control. This is particularly useful when:

- Parameters interact with each other
- The relationship between parameters and KPIs is non-linear
- Multiple parameters need to be tuned simultaneously

Enable Bayesian optimization with:

```yaml
use_bayesian: true
stall_threshold: 5  # Number of oscillations before trying a new point
```