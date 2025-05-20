# Phoenix Quick Start Guide

This guide will help you get started with Phoenix (SA-OMF) quickly, from installation to basic configuration and monitoring.

## Prerequisites

- Go 1.24 or higher
- Docker (optional, for containerized deployment)
- Basic familiarity with OpenTelemetry concepts

## Installation Options

### Option 1: Direct Build from Source

```bash
# Clone the repository
git clone https://github.com/yourorg/Phoenix.git
cd Phoenix

# Build the binary
make build

# Run with default configuration
make run
```

### Option 2: Using Docker

```bash
# Clone the repository
git clone https://github.com/yourorg/Phoenix.git
cd Phoenix

# Build and run with Docker Compose
docker-compose up
```

### Option 3: Using VS Code Dev Container

1. Install [VS Code](https://code.visualstudio.com/) and the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. Clone this repository
3. Open the repository in VS Code
4. Click "Reopen in Container" when prompted
5. The container will set up all dependencies and tools automatically

## Basic Configuration

Phoenix uses two main configuration files:

1. **config.yaml**: Standard OpenTelemetry configuration
2. **policy.yaml**: Adaptive behavior configuration

### Core config.yaml Structure

```yaml
receivers:
  hostmetrics:
    collection_interval: 30s
    scrapers:
      cpu:
      memory:
      disk:
      filesystem:
      network:
      process:

processors:
  # Tag resources with priority levels
  priority_tagger:
    resource_attributes:
      service.name:
        - pattern: "database.*"
          priority: high
        - pattern: "auth.*"
          priority: high
        - pattern: ".*api.*"
          priority: medium
        - pattern: ".*"
          priority: low
  
  # Dynamically select top-k resources
  adaptive_topk:
    enabled: true
    metrics_pattern:
      - "process.cpu.time"
      - "process.memory.usage"
    dimension_key: "process.executable.name"
    k_value: 20
    k_min: 10
    k_max: 50
    coverage_target: 0.95
    
  # Aggregate metrics from less important resources
  others_rollup:
    enabled: true
    priority_attribute: "priority"
    low_priority_values:
      - "low"
    prefix: "others"
    metrics_pattern:
      - "process.cpu.time"
      - "process.memory.usage"

  # Monitor KPIs and provide insights
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

exporters:
  logging:
    verbosity: detailed
    
  prometheusremotewrite:
    endpoint: "http://localhost:9090/api/v1/write"
    
service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [priority_tagger, adaptive_topk, others_rollup, adaptive_pid]
      exporters: [logging, prometheusremotewrite]

extensions:
  health_check:
  pprof:
  zpages:

  pic_control_ext:
    policy_file: "/etc/sa-omf/policy.yaml"
    watch_policy: true
    autonomy_level: "active"
    
  service:
    extensions: [health_check, pprof, zpages, pic_control_ext]
```

### Core policy.yaml Structure

```yaml
version: "1.0"

processors_config:
  adaptive_topk:
    controller:
      kp: 0.5
      ki: 0.1
      kd: 0.05
      integral_windup_limit: 10
      hysteresis_percent: 5
      target_value: 0.95
      min_output: -5
      max_output: 5
      oscillation_detection:
        enabled: true
        window_size: 10
        threshold: 0.7
  
  others_rollup:
    threshold: "low"
    adaptation_interval: "30s"

safety:
  resource_limits:
    max_memory_percent: 90
    max_cpu_percent: 95
  rate_limiting:
    max_adaptation_frequency: 10  # Maximum adaptations per minute
    cooldown_period: 30s         # Minimum time between adaptations
  protection:
    enable_circuit_breakers: true  # Enable oscillation detection
```

## Testing Your Installation

After starting Phoenix, you can verify it's working correctly by:

1. Checking logs for startup messages:
   ```bash
   # If running directly
   tail -f logs/phoenix.log
   
   # If running with Docker Compose
   docker-compose logs -f
   ```

2. Viewing the metrics endpoint:
   ```bash
   curl http://localhost:8888/metrics
   ```

3. Checking health status:
   ```bash
   curl http://localhost:13133/health_check
   ```

4. Accessing the zPages interface (if enabled):
   ```bash
   # Open in browser
   http://localhost:55679/debug/zpages/
   ```

## Monitoring Adaptive Behavior

To observe Phoenix's adaptive behavior:

1. **Check Exported Metrics**: Look for metrics with these prefixes:
   - `aemf_pid_controller_*`: Information about PID controllers
   - `aemf_adaptive_topk_*`: Information about the adaptive_topk processor
   - `aemf_others_rollup_*`: Information about the others_rollup processor

2. **Watch Log Events**: Set verbosity to detailed to see adaptation events:
   ```
   2025-05-20T12:34:56.789Z INFO AdaptiveTopK: Adjusting k from 100 to 125 (target coverage: 0.95, current: 0.92)
   ```

3. **Use Grafana Dashboards**: If you're using the full Docker Compose setup, open Grafana at http://localhost:3000 and use the pre-configured dashboards.

### Key Monitoring Metrics

| Metric | Description | Normal Values |
|--------|-------------|---------------|
| `aemf_pid_controller_error{controller_name="coverage_controller"}` | Error between target and actual coverage | Near 0 (±0.05) |
| `aemf_adaptive_topk_current_k_value` | Current k parameter for adaptive_topk | Between k_min and k_max |
| `aemf_impact_adaptive_topk_resource_coverage_percent_avg_1m` | Coverage score | Near target (e.g., 0.95) |
| `aemf_controller_pid_circuit_breaker_trips_total` | Count of circuit breaker trips | Should not increase rapidly |
| `otelcol_process_memory_rss` | Collector memory usage | Below safety thresholds |

## Customizing Behavior

### Adaptive TopK Configuration

The `adaptive_topk` processor selects the most important resources to monitor based on a metric like CPU or memory usage:

```yaml
adaptive_topk:
  enabled: true
  metrics_pattern:
    - "process.cpu.time"  # Metric to use for ranking importance
  dimension_key: "process.executable.name"  # Dimension to filter on
  k_value: 20    # Initial number of top items to keep
  k_min: 10      # Minimum k value (safety limit)
  k_max: 50      # Maximum k value (safety limit)
  coverage_target: 0.95  # Target coverage (0-1)
```

Increasing `k_max` allows more unique resources to be monitored but increases cardinality. Adjust the `coverage_target` based on your requirements (higher values capture more data but require larger k).

### PID Controller Tuning

PID controllers can be tuned through the policy.yaml file:

```yaml
processors_config:
  adaptive_topk:
    controller:
      kp: 0.5            # Proportional gain - immediate response
      ki: 0.1            # Integral gain - eliminating steady-state error
      kd: 0.05           # Derivative gain - reducing oscillation
      integral_windup_limit: 10  # Prevent excessive integral buildup
      hysteresis_percent: 5      # Deadband to prevent small adjustments
```

For faster response, increase `kp`. For more stability, decrease `kp` and increase `kd`. To eliminate persistent errors, increase `ki` slightly.

## Common Environment Configurations

### For Low-Resource Environments

```yaml
# In config.yaml
hostmetrics:
  collection_interval: 60s  # Reduce collection frequency

# In policy.yaml
processors_config:
  adaptive_topk:
    controller:
      target_value: 0.9    # Slightly lower coverage target
      k_max: 50            # Lower maximum k value
```

### For High-Cardinality Environments

```yaml
# In policy.yaml
processors_config:
  adaptive_topk:
    controller:
      k_max: 200           # Higher maximum k value
      kp: 0.7              # More aggressive control
  
  others_rollup:
    threshold: "medium"    # More aggressive aggregation
```

### For Development/Testing

```yaml
# In config.yaml
processors:
  adaptive_pid:
    controllers:
      - name: "coverage_controller"
        debug_mode: true     # More verbose logging
        hysteresis_percent: 0  # Remove deadband for testing

# In policy.yaml
safety:
  rate_limiting:
    max_adaptation_frequency: 60  # Faster adaptation for testing
    cooldown_period: 1s
```

## Troubleshooting

### Common Issues

| Issue | Possible Solutions |
|-------|-------------------|
| Phoenix fails to start | Check port conflicts, file permissions, Go version |
| No metrics collected | Verify hostmetrics configuration, check collection_interval |
| Excessive resource usage | Adjust safety.resource_limits in policy.yaml, reduce k_max |
| Oscillating parameters | Tune PID parameters (reduce kp, ki), increase hysteresis_percent, enable circuit breakers |
| Low coverage score | Increase k_max, check if the dimension_key captures important resources |

### Diagnostic Commands

```bash
# Check if Phoenix is running
ps aux | grep sa-omf-otelcol

# View current metrics
curl -s localhost:8888/metrics | grep aemf

# Check policy file for syntax errors
python -c "import yaml; yaml.safe_load(open('/path/to/policy.yaml'))"

# View real-time adaptation logs
tail -f logs/phoenix.log | grep "Adjusting"
```

## Next Steps

After getting Phoenix running, here are some next steps:

1. **Read the [Architecture Overview](architecture.md)** to understand the system
2. **Learn about [Adaptive Processing](adaptive-processing.md)** concepts
3. **Explore [PID Controllers](pid-controllers.md)** for deeper tuning
4. **Consult the [Configuration Reference](configuration-reference.md)** for detailed options

## Getting Help

If you encounter issues not covered here:

1. Check the [full documentation](README.md)
2. Review the [Architecture Decision Records](architecture/adr/README.md) for design decisions
3. File an issue on GitHub with details about your environment and configuration