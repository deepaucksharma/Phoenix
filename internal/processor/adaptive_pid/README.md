# adaptive_pid Processor

The `adaptive_pid` processor (also called `pid_decider`) uses PID control theory to generate configuration patches for adaptive processors in the system.

## Configuration

```yaml
processors:
  pid_decider:
    controllers:
      - name: coverage_controller
        enabled: true
        kpi_metric_name: aemf_impact_adaptive_topk_resource_coverage_percent_avg_1m
        kpi_target_value: 0.90
        kp: 30
        ki: 5
        kd: 0
        hysteresis_percent: 3
        output_config_patches:
          - target_processor_name: adaptive_topk
            parameter_path: k_value
            change_scale_factor: -20
            min_value: 10
            max_value: 60
```

## Description

The `adaptive_pid` processor implements a feedback control mechanism based on PID (Proportional-Integral-Derivative) control theory. It monitors KPIs (Key Performance Indicators) and automatically adjusts parameters of other processors to maintain target values.

This processor is the brain of the self-regulating system, allowing processors to dynamically adapt to changing conditions to maintain optimal performance.

## How It Works

1. The processor continuously monitors metrics that represent KPIs
2. For each controller, it:
   - Computes error between the current value and target value
   - Feeds error into a PID controller to generate a control signal
   - Scales the control signal for each output parameter
   - Applies bounds checking and hysteresis to prevent oscillations
   - Generates ConfigPatch objects for other processors
   - Currently logs the patches (future versions will emit them as metrics)

## PID Controllers

Each PID controller has:

- `kp`: Proportional gain - responds to current error
- `ki`: Integral gain - responds to accumulated error over time
- `kd`: Derivative gain - responds to rate of change of error
- `hysteresis_percent`: Minimum percent change required to emit a patch

## Dynamic Configuration

The processor supports the following configuration patches:

- `enabled`: Boolean to enable/disable specific controllers
- `kpi_target_value`: Target value for specific controllers

Example patch:
```json
{
  "patch_id": "update-coverage-target",
  "target_processor_name": "coverage_controller",
  "parameter_path": "kpi_target_value",
  "new_value": 0.95,
  "reason": "Increasing target coverage",
  "severity": "normal",
  "source": "manual"
}
```
