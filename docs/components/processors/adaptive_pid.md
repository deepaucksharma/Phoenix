# Adaptive PID Processor

The Adaptive PID processor is a key component of the control loop architecture in SA-OMF. It monitors system KPIs (Key Performance Indicators) and uses PID (Proportional-Integral-Derivative) controllers to generate configuration patches that adapt the system's behavior.

## Overview

This processor sits in the control pipeline, not the data pipeline. It:

1. Consumes metrics that contain KPI values (like coverage scores, cardinality reduction ratios)
2. Computes the error between actual KPI values and desired target values 
3. Uses PID control to generate stable configuration adjustments
4. Emits configuration patches to adapt other processors' parameters

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
      output_config_patches:
        - target_processor_name: adaptive_topk
          parameter_path: k_value
          change_scale_factor: -20.0
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
      output_config_patches:
        - target_processor_name: others_rollup
          parameter_path: priority_threshold
          change_scale_factor: 1.0
          min_value: 0  # Corresponds to "low" 
          max_value: 3  # Corresponds to "critical"
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

#### For each output_config_patch:

- `target_processor_name`: Which processor to configure
- `parameter_path`: Which parameter in the processor to adjust
- `change_scale_factor`: Multiply PID output by this factor before applying
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
4. **Output Scaling**: Multiply by change_scale_factor (can be negative to invert relationship)
5. **Output Clamping**: Ensure the new parameter value stays within [min_value, max_value]
6. **Configuration Patch Generation**: Create a patch to update the target processor

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
| `aemf_pid_controller_output` | Raw controller output before scaling and clamping |
| `aemf_pid_output_clamped_total` | Count of times output was clamped to min/max |
| `aemf_pid_patch_generated_total` | Count of configuration patches generated |

## Use Cases

- **Adaptive Top-K Processing**: Dynamically adjust k value to maintain coverage target
- **Priority Threshold Adjustment**: Adjust which priority level metrics get aggregated
- **Cardinality Control**: Maintain desired reduction ratio by adjusting parameters
- **Resource Management**: Balance CPU/memory usage against data quality

## Example

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
      output_config_patches:
        - target_processor_name: adaptive_topk
          parameter_path: k_value
          change_scale_factor: -20.0
          min_value: 10
          max_value: 60

# In the config.yaml pipeline:
service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [priority_tagger, adaptive_topk]
      exporters: [prometheusremotewrite]
    
    control:
      receivers: [prometheus/self]
      processors: [adaptive_pid]
      exporters: [pic_connector]
```