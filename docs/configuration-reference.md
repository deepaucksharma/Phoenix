# Phoenix Configuration Reference

This document provides a complete reference for configuring Phoenix (SA-OMF). The system uses two main configuration files:

1. **config.yaml**: Standard OpenTelemetry Collector configuration
2. **policy.yaml**: Self-adaptive behavior configuration

## Standard Configuration (config.yaml)

Phoenix uses the standard OpenTelemetry Collector configuration format with custom processors.

### Example Configuration

```yaml
receivers:
  hostmetrics:
    collection_interval: 10s
    scrape_metrics:
      process:
        metrics:
          process.cpu.time:
            enabled: true
          process.memory.usage:
            enabled: true
          process.memory.virtual:
            enabled: true

processors:
  # Tags resources with priority levels
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

  # Dynamically selects top-k resources
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
    adaptation_interval: 30s

  # Aggregates less important resources
  others_rollup:
    enabled: true
    priority_attribute: "priority"
    low_priority_values:
      - "low"
    prefix: "others"
    metrics_pattern:
      - "process.cpu.time"
      - "process.memory.usage"

  # Guards against excessive cardinality
  cardinality_guardian:
    enabled: true
    max_cardinality: 5000
    adaptation_interval: 60s
    max_decrease_percent: 15
    metrics_pattern:
      - ".*"
    priority_attribute: "priority"
    
  # Monitors KPIs and provides insights  
  adaptive_pid:
    controllers:
      - name: coverage_controller
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
    endpoint: "http://prometheus:9090/api/v1/write"
    timeout: 5s

service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [priority_tagger, adaptive_topk, others_rollup, adaptive_pid]
      exporters: [logging, prometheusremotewrite]
```

> **Important Note**: Older configuration files may reference components that are no longer used, such as a control pipeline with `pic_connector` exporter and `pic_control_ext` extension. These components have been removed in the current architecture. See [Current Architecture State](./architecture/CURRENT_STATE.md) for details.

### Processor Configurations

#### priority_tagger

Tags resources with priority levels based on pattern matching.

```yaml
priority_tagger:
  resource_attributes:
    <attribute_name>:
      - pattern: "<regex_pattern>"
        priority: "high|medium|low"
      - pattern: "<regex_pattern>"
        priority: "high|medium|low"
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resource_attributes` | Map of attributes to match patterns against | Required |
| `pattern` | Regular expression to match | Required |
| `priority` | Priority level to assign (high, medium, low) | Required |

#### adaptive_topk

Dynamically adjusts k-value to maintain a target coverage score.

```yaml
adaptive_topk:
  enabled: true
  metrics_pattern:
    - "<metric_name_pattern>"
  dimension_key: "<dimension_attribute>"
  k_value: <initial_k>
  k_min: <minimum_k>
  k_max: <maximum_k>
  coverage_target: <target_coverage>
  adaptation_interval: <interval>
  controller:
    kp: <proportional_term>
    ki: <integral_term>
    kd: <derivative_term>
    integral_windup_limit: <windup_limit>
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `enabled` | Enables or disables the processor | `true` |
| `metrics_pattern` | List of metric names to process | Required |
| `dimension_key` | Attribute to use for top-k filtering | Required |
| `k_value` | Initial k value | `20` |
| `k_min` | Minimum k value | `5` |
| `k_max` | Maximum k value | `100` |
| `coverage_target` | Target coverage score (0.0-1.0) | `0.95` |
| `adaptation_interval` | How often to adjust the k value | `30s` |
| `controller` | PID controller configuration | Optional |

#### others_rollup

Aggregates metrics from resources with low priority.

```yaml
others_rollup:
  enabled: true
  priority_attribute: "<priority_attribute_name>"
  low_priority_values:
    - "<priority_value>"
  prefix: "<prefix_for_others>"
  metrics_pattern:
    - "<metric_name_pattern>"
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `enabled` | Enables or disables the processor | `true` |
| `priority_attribute` | Attribute holding priority values | `"priority"` |
| `low_priority_values` | List of priority values to aggregate | `["low"]` |
| `prefix` | Prefix to add to aggregated metrics | `"others"` |
| `metrics_pattern` | List of metric names to aggregate | Required |

#### adaptive_pid

Monitors KPIs and provides insights into system performance.

```yaml
adaptive_pid:
  controllers:
    - name: "<controller_name>"
      enabled: true
      kpi_metric_name: "<metric_name>"
      kpi_target_value: <target_value>
      kp: <proportional_term>
      ki: <integral_term>
      kd: <derivative_term>
      hysteresis_percent: <percent>
      integral_windup_limit: <limit>
      use_bayesian: true|false
      stall_threshold: <count>
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `name` | Unique name for the controller | Required |
| `enabled` | Whether this controller is active | `true` |
| `kpi_metric_name` | The metric name that contains the KPI value to monitor | Required |
| `kpi_target_value` | The desired value for the KPI | Required |
| `kp`, `ki`, `kd` | PID controller gains | Required |
| `hysteresis_percent` | Deadband to prevent oscillation | `3` |
| `integral_windup_limit` | Maximum value for the integral term | `10` |
| `use_bayesian` | Enable Bayesian optimization fallback | `false` |
| `stall_threshold` | Consecutive ineffective adjustments before using Bayesian | `3` |

#### reservoir_sampler

Provides statistical sampling with adjustable rates.

```yaml
reservoir_sampler:
  enabled: true
  reservoir_size: <size>
  metrics_pattern:
    - "<metric_name_pattern>"
  sampling_fraction: <fraction>
  adaptation_interval: <interval>
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `enabled` | Enables or disables the processor | `true` |
| `reservoir_size` | Size of the sampling reservoir | `100` |
| `metrics_pattern` | List of metric names to sample | Required |
| `sampling_fraction` | Initial sampling fraction (0.0-1.0) | `0.1` |
| `adaptation_interval` | How often to adjust sampling parameters | `60s` |

## Policy Configuration (policy.yaml)

The policy.yaml file controls the adaptive behavior of processors. It includes:

### Example Policy

```yaml
version: "1.0"
processors_config:
  adaptive_topk:
    enabled: true
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
  
  adaptive_pid:
    enabled: true
    controllers:
      - name: "resource_usage"
        kpi_metric_name: "system.cpu.utilization"
        kpi_target_value: 0.75
        kp: 0.8
        ki: 0.2
        kd: 0.1
        hysteresis_percent: 3
        
  others_rollup:
    enabled: true
    threshold: "low"
    adaptation_interval: "30s"
    
  reservoir_sampler:
    enabled: true
    size_range:
      min: 50
      max: 500
    target_memory_usage_mb: 100

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

### Policy Sections

#### Processor Configuration

Each processor can have specific policy configurations that control its adaptive behavior:

```yaml
processors_config:
  <processor_name>:
    enabled: true|false
    controller:
      kp: <proportional_gain>
      ki: <integral_gain>
      kd: <derivative_gain>
      integral_windup_limit: <windup_limit>
      hysteresis_percent: <hysteresis>
      target_value: <target>
      min_output: <min>
      max_output: <max>
      oscillation_detection:
        enabled: true|false
        window_size: <window_size>
        threshold: <threshold>
```

#### Safety Configuration

The safety section controls global safety limits and protection mechanisms:

```yaml
safety:
  resource_limits:
    max_memory_percent: <percent>
    max_cpu_percent: <percent>
  rate_limiting:
    max_adaptation_frequency: <frequency>
    cooldown_period: <duration>
  protection:
    enable_circuit_breakers: true|false
```

## Environment-Specific Configurations

Phoenix provides environment-specific configurations in separate directories:

- **configs/default/** - Balanced configuration for most environments
- **configs/development/** - More verbose logging and faster adaptation for development
- **configs/production/** - More conservative settings prioritizing stability
- **configs/testing/** - Configuration optimized for running tests

## Configuration Best Practices

1. **Start with an existing configuration**: Use configs/default as a starting point
2. **Tune PID parameters gradually**: Make small adjustments to kp, ki, and kd values
3. **Set appropriate safety limits**: Configure resource limits based on your environment
4. **Enable circuit breakers**: Use oscillation detection to prevent unstable behavior
5. **Use metric patterns carefully**: Be specific with metrics_pattern to target only relevant metrics
6. **Adjust adaptation intervals**: Faster intervals provide quicker adaptation but may increase overhead