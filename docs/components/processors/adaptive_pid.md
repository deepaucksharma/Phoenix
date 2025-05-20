# Adaptive PID Processor

The Adaptive PID processor monitors system KPIs (Key Performance Indicators) and uses PID (Proportional-Integral-Derivative) controllers to adjust its behavior, providing insights into system performance.

## Overview

This processor:

1. Consumes metrics that contain KPI values (like coverage scores, cardinality reduction ratios)
2. Computes the error between actual KPI values and desired target values 
3. Uses PID control to monitor system behavior
4. Adapts its own parameters based on observations
5. Emits metrics about KPI status and control decisions

## Current Implementation

> **Important Note**: This processor has evolved from its original design. In the current implementation, it functions as a self-contained adaptive component rather than generating configuration patches for other processors.

Each adaptive processor (like adaptive_topk, others_rollup) now implements its own adaptation mechanisms internally, making the system more modular and simpler to maintain.

## Configuration

```yaml
adaptive_pid:
  controllers:
    - name: coverage_controller
      enabled: true
      kpi_metric_name: aemf_impact_adaptive_topk_resource_coverage_percent_avg_1m
      kpi_target_value: 0.90
      kp: 30
      ki: 5
      kd: 0
      hysteresis_percent: 3
      integral_windup_limit: 100
      use_bayesian: true
      stall_threshold: 3  
      min_value: 10
      max_value: 60
    
    - name: rollup_controller
      enabled: true
      kpi_metric_name: aemf_processor_metrics_count
      kpi_target_value: 5000.0
      kp: 0.5
      ki: 0.1
      kd: 0
      hysteresis_percent: 5
      integral_windup_limit: 50
      min_value: 0
      max_value: 3
```

### Configuration Parameters

#### For each controller:

- `name`: Unique name for the controller
- `enabled`: Whether this controller is active
- `kpi_metric_name`: The metric name that contains the KPI value to monitor
- `kpi_target_value`: The desired value for the KPI
- `kp`, `ki`, `kd`: PID controller gains
  - `kp`: Proportional gain (immediate response to current error)
  - `ki`: Integral gain (response to accumulated error over time)
  - `kd`: Derivative gain (response to rate of change of error)
- `hysteresis_percent`: Deadband to prevent oscillation (ignored if within this % of target)
- `integral_windup_limit`: Maximum value for the integral term to prevent windup
- `use_bayesian`: Enable Bayesian optimization fallback if PID control stalls
- `stall_threshold`: Number of consecutive ineffective adjustments before trying Bayesian optimization
- `min_value`: Minimum allowed value for the parameter
- `max_value`: Maximum allowed value for the parameter

## How It Works

### PID Control Loop

1. **Input**: The processor receives metrics containing KPI values
2. **Error Calculation**: It calculates the error between the actual KPI and target KPI
3. **PID Computation**: 
   - P term = Error × Proportional Gain
   - I term = Accumulated Error × Integral Gain
   - D term = Error Change Rate × Derivative Gain
   - Output = P + I + D
4. **Output Clamping**: Ensure the output stays within [min_value, max_value]
5. **Metrics Generation**: Create metrics about KPI status and controller behavior

### Bayesian Optimization Fallback

If PID control is not effective after several iterations:

1. The processor detects stalled progress using the stall_threshold
2. It switches to a Bayesian optimization approach
3. This explores the parameter space more intelligently
4. Once progress is made, it can switch back to PID control

## Metrics Emitted

| Metric Name | Description |
|-------------|-------------|
| `aemf_pid_controller_error` | Current error (difference between KPI and target) |
| `aemf_pid_controller_proportional_term` | Current P term contribution |
| `aemf_pid_controller_integral_term` | Current I term contribution |
| `aemf_pid_controller_derivative_term` | Current D term contribution |
| `aemf_pid_controller_output` | Raw controller output before clamping |
| `aemf_pid_output_clamped_total` | Count of times output was clamped to min/max |

## Use Cases

- **KPI Monitoring**: Track key system metrics against target values
- **Observability**: Generate metrics about controller behavior and system performance
- **Performance Insights**: Identify when KPIs deviate from targets

## Example Configuration

```yaml
# In the policy.yaml:
adaptive_pid_config:
  controllers:
    - name: coverage_controller
      enabled: true
      kpi_metric_name: aemf_impact_adaptive_topk_resource_coverage_percent_avg_1m
      kpi_target_value: 0.90
      kp: 30
      ki: 5
      kd: 0

# In the config.yaml pipeline:
service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [priority_tagger, adaptive_topk, adaptive_pid]
      exporters: [prometheusremotewrite]
```