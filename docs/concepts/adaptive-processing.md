# Adaptive Processing in Phoenix

This document provides a conceptual overview of how adaptive processing works in Phoenix, helping you understand the principles behind its self-tuning capabilities.

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

![PID Control Loop](../images/pid-control-loop.png)

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

### Beyond PID: Other Adaptive Techniques

In addition to PID control, Phoenix implements:

1. **Bayesian optimization**: Uses Gaussian processes for complex parameter spaces
2. **Cardinality control**: Dynamically adjusts metrics cardinality based on resource usage
3. **Statistical sampling**: Automatically adjusts sampling rates to balance detail and overhead

## Adaptive Components

### adaptive_topk

This processor dynamically adjusts the k parameter (number of top resources to track) to maintain a target coverage score.

**Adaptation Strategy**:
- Uses Space-Saving algorithm to track top resources
- Calculates coverage score for current k
- Applies PID control to adjust k up or down
- Aims to use the smallest k value that achieves target coverage

### others_rollup

Aggregates metrics from low-priority resources to reduce cardinality while preserving detail for important resources.

**Adaptation Strategy**:
- Uses priority values from priority_tagger
- Identifies low-priority resources for aggregation
- Preserves individual metrics for high-priority resources
- May adjust aggregation threshold based on current cardinality

### cardinality_guardian

Monitors and controls overall metrics cardinality to prevent resource exhaustion.

**Adaptation Strategy**:
- Monitors current metric cardinality
- Adjusts filtering parameters when cardinality exceeds thresholds
- Applies more aggressive filtering to low-priority metrics
- Uses system resource usage (memory, CPU) as inputs

## Configuring Adaptive Behavior

Adaptive behavior is configured through the policy.yaml file. Key parameters include:

1. **PID parameters** (kp, ki, kd): Control the responsiveness and stability
2. **Target values**: Define the desired steady state
3. **Safety limits**: Set boundaries on adaptation
4. **Adaptation intervals**: Control how frequently adaptation occurs

### Tuning PID Controllers

Tuning PID controllers requires balancing responsiveness and stability:

- **Increase kp** to make the system more responsive (but may cause oscillation)
- **Increase ki** to eliminate steady-state error (but may cause overshoot)
- **Increase kd** to dampen oscillations (but may amplify noise)
- **Adjust integral_windup_limit** to prevent large accumulated corrections

## Monitoring Adaptation

Phoenix's adaptive behavior can be monitored through:

1. **Metrics**: Each processor emits metrics about its current parameters and adaptation decisions
2. **Logs**: Set verbosity to detailed to see adaptation events
3. **OpenTelemetry**: Phoenix emits standard OpenTelemetry metrics that can be viewed in dashboards

## Best Practices

1. **Start conservative**: Use small PID values and increase gradually
2. **Set appropriate bounds**: Define realistic min/max values for all parameters
3. **Choose meaningful targets**: Select KPI targets that reflect your requirements
4. **Use circuit breakers**: Enable oscillation detection to prevent instability
5. **Tune for your workload**: Different workloads may require different PID values
6. **Monitor adaptation events**: Watch for signs of oscillation or sluggish response

## Conclusion

Adaptive processing is the core differentiator of Phoenix. By implementing industrial control techniques within a metrics processing system, Phoenix can maintain optimal performance in dynamic environments with minimal operator intervention.

While the implementation has evolved to embed adaptation directly within processors (rather than using an external control pipeline), the fundamental principles of feedback control remain the foundation of Phoenix's self-adaptive capabilities.