# Phoenix Monitoring and Observability Guide

This guide explains how to monitor Phoenix (SA-OMF) and gain insights into its adaptive behavior and performance.

## Monitoring Overview

Phoenix is designed with observability in mind. It provides extensive metrics about:

1. **System Performance**: Standard telemetry about the collector's resource usage
2. **Adaptation Decisions**: Metrics about parameter adjustments and their reasons
3. **PID Controller Status**: Detailed metrics about controller behavior
4. **Processor Effectiveness**: Metrics showing the impact of each processor

## Key Metrics to Monitor

### System Health Metrics

These metrics indicate the overall health of the Phoenix system:

| Metric Name | Description | Normal Range |
|-------------|-------------|--------------|
| `process.runtime.go.mem.heap_alloc` | Current memory usage | Varies by load |
| `process.runtime.go.gc.pause_ns` | GC pause duration | < 100ms |
| `otelcol.processor.refused_spans` | Processing backpressure | Should be 0 |
| `otelcol.processor.batch_size` | Processing batch size | Stable value |

### Adaptive Parameter Metrics

These metrics show how Phoenix is adapting its parameters:

| Metric Name | Description |
|-------------|-------------|
| `aemf_adaptive_topk_k_value` | Current k value in adaptive_topk |
| `aemf_others_rollup_threshold` | Current threshold in others_rollup |
| `aemf_reservoir_sampler_size` | Current reservoir size |

### PID Controller Metrics

These metrics provide insights into the PID controllers' behavior:

| Metric Name | Description |
|-------------|-------------|
| `aemf_pid_controller_error` | Current error (difference from target) |
| `aemf_pid_controller_proportional_term` | P term contribution |
| `aemf_pid_controller_integral_term` | I term contribution |
| `aemf_pid_controller_derivative_term` | D term contribution |
| `aemf_pid_controller_output` | Raw controller output |
| `aemf_pid_output_clamped_total` | Count of output clamping events |
| `aemf_pid_oscillation_detected_total` | Count of oscillation detection events |

### Impact Metrics

These metrics show the effectiveness of Phoenix's processors:

| Metric Name | Description | Goal |
|-------------|-------------|------|
| `aemf_impact_adaptive_topk_resource_coverage_percent` | Percentage of total metric values captured | Near target (e.g., 0.95) |
| `aemf_impact_others_rollup_cardinality_reduction_ratio` | Reduction in metric cardinality | High value (e.g., > 0.5) |
| `aemf_impact_processor_latency_seconds` | Processing latency | Low, stable value |

## Setting Up Monitoring

### Basic Prometheus Setup

1. Configure the Prometheus exporter in `config.yaml`:

```yaml
exporters:
  prometheus:
    endpoint: 0.0.0.0:8889
```

2. Configure Prometheus to scrape Phoenix:

```yaml
scrape_configs:
  - job_name: 'phoenix'
    scrape_interval: 10s
    static_configs:
      - targets: ['phoenix:8889']
```

### Grafana Dashboard

Phoenix includes a pre-configured Grafana dashboard in the `dashboards/` directory:

1. Import `dashboards/autonomy-pulse.json` into Grafana
2. Connect it to your Prometheus data source
3. The dashboard provides:
   - System health overview
   - Adaptation parameter tracking
   - PID controller visualizations
   - Impact metrics analysis

### Using the Included Docker Compose Stack

The easiest way to set up monitoring is with the included Docker Compose stack:

```bash
# Start the full stack (Phoenix, Prometheus, Grafana)
docker-compose -f deploy/compose/full/docker-compose.yaml up -d

# Access Grafana at http://localhost:3000
# Default credentials: admin/admin
```

## Understanding Adaptation Behavior

### PID Controller Visualization

To understand PID controller behavior, monitor these metrics together:

1. **Error Value**: The difference between actual and target (KPI - Target)
2. **Controller Output**: The correction signal from the PID controller
3. **Parameter Value**: The resulting parameter being adjusted
4. **Target KPI**: The metric being controlled

A healthy PID controller should show:
- Error converging toward zero
- Output adjusting in response to error
- Parameter value stabilizing
- KPI approaching target value

![PID Controller Behavior](../images/pid-controller-visualization.png)

### Oscillation Detection

Watch for oscillation in parameter values, which indicates unstable controller settings:

1. **Signs of Oscillation**:
   - Parameters changing direction repeatedly
   - Error swinging between positive and negative
   - Output swinging between positive and negative

2. **Addressing Oscillation**:
   - Decrease kp (proportional gain)
   - Increase hysteresis_percent (deadband)
   - Enable circuit breakers
   - Increase adaptation_interval

## Monitoring Dashboards

### System Overview Dashboard

Key panels to include:
- Memory usage over time
- CPU usage over time
- Throughput (metrics/second)
- Error rates
- Count of active time series

### Adaptation Dashboard

Key panels to include:
- Parameter values over time (e.g., k_value, thresholds)
- Target KPIs vs actual values
- PID controller terms (P, I, D contributions)
- Adaptation events frequency

### Performance Impact Dashboard

Key panels to include:
- Cardinality before and after processing
- Processing latency per processor
- Coverage score vs target
- Resource usage correlation with parameter changes

## Alerting

### Recommended Alerts

1. **System Health Alerts**:
   - Memory usage > 80% of limit
   - Sustained high CPU usage (>85% for 10+ minutes)
   - Metric processing delay > 30s

2. **Adaptation Alerts**:
   - Parameter hitting min/max bounds repeatedly
   - Oscillation detection triggering frequently
   - KPI consistently far from target (±20%)

3. **Circuit Breaker Alerts**:
   - Circuit breaker activated
   - Multiple adaptation failures
   - Safety mode activation

### Example Prometheus Alert Rules

```yaml
groups:
- name: Phoenix Alerts
  rules:
  - alert: PhoenixHighMemoryUsage
    expr: process_runtime_go_mem_heap_alloc_bytes{job="phoenix"} > 1000000000
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High memory usage"
      description: "Phoenix memory usage is high for 5+ minutes"

  - alert: PhoenixKpiTargetDeviation
    expr: abs(aemf_impact_adaptive_topk_resource_coverage_percent - 0.95) > 0.2
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "KPI far from target"
      description: "Coverage KPI is deviating significantly from target for 10+ minutes"

  - alert: PhoenixOscillationDetected
    expr: rate(aemf_pid_oscillation_detected_total[5m]) > 0
    labels:
      severity: warning
    annotations:
      summary: "Controller oscillation detected"
      description: "PID controller oscillation detected in the last 5 minutes"
```

## Advanced Monitoring

### Exporting Metrics to External Systems

Configure additional exporters to send metrics to other systems:

```yaml
exporters:
  otlp:
    endpoint: "monitoring.example.com:4317"
    tls:
      insecure: false
      cert_file: /certs/client.crt
      key_file: /certs/client.key
```

### Correlating with Application Metrics

To get the full picture, correlate Phoenix metrics with application metrics:

1. **Tag Application Metrics**: Use the same tags on application metrics
2. **Create Combined Dashboards**: Show Phoenix and application metrics together
3. **Look for Relationships**: Observe how Phoenix adaptations affect application metrics

### Monitoring in Kubernetes

When running in Kubernetes:

1. Use the Prometheus Operator for automatic service discovery
2. Configure ServiceMonitor resources to scrape Phoenix
3. Integrate with Kubernetes metrics via the kube-state-metrics exporter
4. Set up HorizontalPodAutoscalers based on custom metrics from Phoenix

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: phoenix-monitor
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: phoenix
  endpoints:
  - port: metrics
    interval: 10s
```

## Troubleshooting with Metrics

### Identifying Performance Issues

1. **High Latency**:
   - Check `aemf_impact_processor_latency_seconds` by processor
   - Look for correlation with high k values or complex patterns

2. **Memory Issues**:
   - Monitor heap usage over time
   - Check cardinality metrics for explosion (high unique time series)
   - Verify others_rollup is reducing cardinality effectively

3. **Unstable Adaptation**:
   - Look for oscillating parameter values
   - Check PID controller terms for large swings
   - Verify circuit breakers are configured correctly

### Debugging PID Controller Issues

1. **Slow Convergence**:
   - Increase kp (proportional term) carefully
   - Check if target is realistic given the system

2. **Overshooting**:
   - Decrease kp and ki
   - Increase derivative term (kd) slightly
   - Verify integral_windup_limit is set appropriately

3. **No Reaction**:
   - Verify metrics are being received
   - Check if controller is enabled
   - Verify target metrics exist and are correctly named

## Conclusion

Effective monitoring is essential to understand and optimize Phoenix's adaptive behavior. By tracking the key metrics outlined in this guide and setting up appropriate dashboards and alerts, you can ensure that Phoenix operates efficiently and adapt its configuration as needed.

For more information on specific metrics and their interpretation, refer to the individual processor documentation in the [Components](../components/) section.