# Adaptive Processing in Phoenix

This document explains the concepts of adaptive processing in Phoenix and how it enables automatic parameter adjustment based on observed metrics.

## What is Adaptive Processing?

Adaptive processing is Phoenix's ability to automatically adjust its internal parameters in response to changing conditions. Rather than using static configurations, Phoenix uses feedback control techniques to optimize its behavior dynamically.

### Key Benefits

1. **Resilience to changing workloads**: Automatic adjustment to unexpected traffic patterns
2. **Efficient resource utilization**: Uses just enough resources to maintain quality of service
3. **Reduced operational overhead**: Less need for manual tuning and reconfiguration
4. **Consistent performance**: Maintains target KPIs across varying conditions

## Core Adaptive Mechanisms

Phoenix implements several adaptive mechanisms, with PID control being the primary approach.

### PID Control

**P**roportional, **I**ntegral, **D**erivative (PID) control is a classical control theory technique used in industrial systems. In Phoenix, PID controllers are embedded within adaptive processors to guide parameter adjustments.

#### How PID Control Works

1. **Define a target value** for a key metric (e.g., coverage score = 0.95)
2. **Measure the current value** of that metric
3. **Calculate the error** (difference between target and measured value)
4. **Apply the PID formula**:
   - **P term**: Reacts proportionally to the current error
   - **I term**: Accounts for accumulated error over time
   - **D term**: Considers the rate of change of error
5. **Adjust parameters** based on the controller output

![PID Control Loop](images/pid-control-loop.png)

#### Example: Adjusting k in adaptive_topk

The `adaptive_topk` processor uses PID control to adjust its k value:

1. Target: Coverage score of 0.95 (95% of total metric values are captured by top-k)
2. Measurement: Current coverage score (e.g., 0.92)
3. Error: 0.03 (0.95 - 0.92)
4. PID calculation determines adjustment needed
5. k value is increased by the calculated amount
6. Process repeats at the next adaptation interval

### Safety Mechanisms

To prevent unstable behavior, Phoenix implements several safety features:

1. **Bounded outputs**: All parameter adjustments have min/max limits
2. **Hysteresis**: Small errors are ignored to prevent oscillation
3. **Anti-windup protection**: Prevents integral term from growing too large
4. **Oscillation detection**: Circuit breakers that temporarily disable adaptation when oscillation is detected
5. **Rate limiting**: Prevents too-frequent adaptation

### Beyond PID: Bayesian Optimization

For complex parameter spaces where PID control might not be sufficient, Phoenix can use Bayesian optimization:

1. **Gaussian Process Model**: Creates a probabilistic model of the parameter space
2. **Acquisition Function**: Balances exploration vs. exploitation (Upper Confidence Bound)
3. **Latin Hypercube Sampling**: Efficiently explores the parameter space
4. **Automatic Fallback**: Can be triggered when PID control stalls or oscillates

## Adaptive Components

Phoenix includes several processors that implement adaptive behavior:

### adaptive_topk

This processor dynamically adjusts the k parameter (number of top resources to track) to maintain a target coverage score.

**Adaptation Strategy**:
- Uses Space-Saving algorithm to track top resources
- Calculates coverage score for current k
- Applies PID control to adjust k up or down
- Aims to use the smallest k value that achieves target coverage

```yaml
adaptive_topk:
  enabled: true
  metrics_pattern:
    - "process.cpu.time"
  dimension_key: "process.executable.name"
  k_value: 20          # Initial value
  k_min: 10            # Minimum allowed value
  k_max: 50            # Maximum allowed value
  coverage_target: 0.95  # Target coverage score
  adaptation_interval: 30s
```

### others_rollup

Aggregates metrics from low-priority resources to reduce cardinality while preserving detail for important resources.

**Adaptation Strategy**:
- Uses priority values from priority_tagger
- Identifies low-priority resources for aggregation
- Preserves individual metrics for high-priority resources
- May adjust aggregation threshold based on current cardinality

```yaml
others_rollup:
  enabled: true
  priority_attribute: "priority"
  low_priority_values:
    - "low"
  prefix: "others"
  metrics_pattern:
    - "process.cpu.time"
```

### cardinality_guardian

Monitors and controls overall metrics cardinality to prevent resource exhaustion.

**Adaptation Strategy**:
- Monitors current metric cardinality
- Adjusts filtering parameters when cardinality exceeds thresholds
- Applies more aggressive filtering to low-priority metrics
- Uses system resource usage (memory, CPU) as inputs

```yaml
cardinality_guardian:
  enabled: true
  max_cardinality: 5000
  adaptation_interval: 60s
  max_decrease_percent: 15
  metrics_pattern:
    - ".*"
  priority_attribute: "priority"
```

### adaptive_pid

Monitors KPIs and adjusts parameters of other processors using PID controllers.

**Adaptation Strategy**:
- Monitors target KPI metrics
- Calculates adjustments using PID control logic
- Can fall back to Bayesian optimization for complex parameters
- Implements oscillation detection to prevent instability

```yaml
adaptive_pid:
  controllers:
    - name: "coverage_controller"
      enabled: true
      kpi_metric_name: "aemf_impact_adaptive_topk_resource_coverage_percent"
      kpi_target_value: 0.95
      kp: 0.5
      ki: 0.1
      kd: 0.05
      hysteresis_percent: 3
      integral_windup_limit: 10
      use_bayesian: true
      stall_threshold: 3
```

## Configuring Adaptive Behavior

The adaptive behavior is configured through two files:

1. **config.yaml**: Defines the initial processor settings
2. **policy.yaml**: Controls the adaptation parameters

### Policy Configuration

The `policy.yaml` file defines how the adaptive processing behaves:

```yaml
processors_config:
  adaptive_topk:
    controller:
      kp: 0.5
      ki: 0.1
      kd: 0.05
      integral_windup_limit: 10
      hysteresis_percent: 5
      target_value: 0.95

safety:
  resource_limits:
    max_memory_percent: 90
    max_cpu_percent: 95
  rate_limiting:
    max_adaptation_frequency: 10
    cooldown_period: 30s
  protection:
    enable_circuit_breakers: true
```

## The Adaptation Lifecycle

1. **Collection**: Self-metrics are collected from processors
2. **Evaluation**: KPIs are calculated and compared to targets
3. **Calculation**: PID controllers compute necessary adjustments
4. **Validation**: Adjustments are checked against safety limits
5. **Application**: Configuration changes are applied to processors
6. **Observation**: System observes the effects of changes
7. **Iteration**: Process repeats at the next adaptation interval

## Monitoring Adaptation

Phoenix's adaptive behavior can be monitored through:

1. **Metrics**: Each processor emits metrics about its current parameters and adaptation decisions
2. **Logs**: Set verbosity to detailed to see adaptation events
3. **OpenTelemetry**: Phoenix emits standard OpenTelemetry metrics that can be viewed in dashboards

### Key Metrics to Monitor

| Metric | Description | When to Investigate |
|--------|-------------|---------------------|
| `aemf_pid_controller_error` | Current error value | Large oscillations or consistent high values |
| `aemf_pid_controller_output` | Controller output | Hitting min/max limits consistently |
| `aemf_pid_controller_p_term` | Proportional term | Unusually large values |
| `aemf_pid_controller_i_term` | Integral term | Accumulating to large values |
| `aemf_pid_controller_d_term` | Derivative term | Spikes indicating noise |
| `aemf_controller_pid_circuit_breaker_trips_total` | Circuit breaker trips | When frequently tripped |
| `aemf_adaptive_topk_current_k_value` | Current k value | Hitting limits or plateauing |

## Best Practices

1. **Start conservative**: Use small PID values and increase gradually
2. **Set appropriate bounds**: Define realistic min/max values for all parameters
3. **Choose meaningful targets**: Select KPI targets that reflect your requirements
4. **Use circuit breakers**: Enable oscillation detection to prevent instability
5. **Tune for your workload**: Different workloads may require different PID values
6. **Monitor adaptation events**: Watch for signs of oscillation or sluggish response

## Example Use Cases

### Dynamic Resource Monitoring

For a system that needs to track the most active services while keeping overhead low:

1. Configure `adaptive_topk` to maintain 95% coverage
2. Set initial k value conservatively (e.g., 100)
3. Let the controller adjust k dynamically as service activity changes
4. During peak times, k might increase to 200-300
5. During quiet periods, k might decrease to 50-80
6. Result: Consistent monitoring quality with minimal resource usage

### Cardinality Management

For a system experiencing high cardinality from numerous ephemeral resources:

1. Configure `cardinality_guardian` with a target of 10,000 metrics
2. Set up `others_rollup` to aggregate low-priority resources
3. Let controllers dynamically adjust thresholds as workload changes
4. Result: Reliable cardinality control preventing resource exhaustion

## Troubleshooting

| Issue | Possible Solution |
|-------|-------------------|
| Parameter oscillating | Reduce kp and ki values, increase hysteresis, enable circuit breaker |
| Slow adaptation | Increase kp value, check if hitting output limits |
| Adaptation stalled | Check if circuit breaker is tripped, consider enabling Bayesian fallback |
| Excessive resource usage | Verify safety limits are properly configured |
| Unexpected adaptation | Check logs for PID controller state and metrics |

## References

- [PID Controller Documentation](pid-controllers.md)
- [Architecture Decision Record for PID Control](architecture/adr/20250519-use-self-regulating-pid-control-for-adaptive-processing.md)
- [Bayesian Optimization Guide](bayesian-optimization.md)
- [Configuration Reference](configuration-reference.md)