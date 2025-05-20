# Phoenix Configuration Guide

This guide provides detailed instructions for configuring Phoenix (SA-OMF) for various environments and use cases.

## Configuration Overview

Phoenix uses two primary configuration files:

1. **config.yaml**: Standard OpenTelemetry Collector configuration defining the processing pipeline
2. **policy.yaml**: Phoenix-specific configuration controlling adaptive behavior

These files have distinct purposes:
- `config.yaml` defines **what** components are used and how they're connected
- `policy.yaml` defines **how** these components should adapt their behavior

## Configuration Locations

Configuration files are organized by environment:

```
configs/
├── default/              # Baseline configuration
│   ├── config.yaml
│   └── policy.yaml
├── development/          # Development configuration
│   ├── config.yaml
│   └── policy.yaml
├── production/           # Production configuration
│   ├── config.yaml
│   └── policy.yaml
└── testing/              # Testing configuration
    ├── config.yaml
    └── policy.yaml
```

## Configuration File: config.yaml

The `config.yaml` file follows the standard OpenTelemetry Collector configuration format with Phoenix's custom processors.

### Basic Structure

```yaml
receivers:
  # Define data sources
  hostmetrics:
    collection_interval: 10s
    # ...

processors:
  # Configure processors
  priority_tagger:
    # ...
  adaptive_topk:
    # ...

exporters:
  # Define data destinations
  logging:
    # ...
  prometheusremotewrite:
    # ...

service:
  # Define processing pipelines
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [priority_tagger, adaptive_topk, others_rollup]
      exporters: [logging, prometheusremotewrite]
```

### Common Receivers

```yaml
receivers:
  # Local host metrics
  hostmetrics:
    collection_interval: 10s
    scrape_metrics:
      process:
        metrics:
          process.cpu.time:
            enabled: true
          process.memory.usage:
            enabled: true
  
  # Prometheus metrics
  prometheus:
    config:
      scrape_configs:
        - job_name: 'prometheus'
          scrape_interval: 10s
          static_configs:
            - targets: ['localhost:9090']
  
  # OTLP receiver
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
```

### Phoenix Processors Configuration

```yaml
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
```

### Common Exporters

```yaml
exporters:
  # Console output
  logging:
    verbosity: detailed
    sampling_initial: 5
    sampling_thereafter: 200
  
  # Prometheus remote write
  prometheusremotewrite:
    endpoint: "http://prometheus:9090/api/v1/write"
    timeout: 5s
  
  # OTLP exporter
  otlp:
    endpoint: collector:4317
    tls:
      insecure: true
```

### Service Definition

The service section defines the telemetry pipelines:

```yaml
service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [priority_tagger, adaptive_topk, others_rollup, adaptive_pid]
      exporters: [logging, prometheusremotewrite]
    
    # Optional additional pipelines
    traces:
      receivers: [otlp]
      processors: []
      exporters: [otlp]
```

## Configuration File: policy.yaml

The `policy.yaml` file controls the adaptive behavior of processors:

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

## Environment-Specific Configurations

Phoenix comes with predefined environment configurations that you can use as a starting point:

### Development Configuration

Optimized for development with:
- Verbose logging
- Faster adaptation intervals
- More lenient safety limits

### Production Configuration

Optimized for stability with:
- Minimal logging
- Conservative adaptation settings
- Stricter safety limits
- More aggressive caching

### Testing Configuration

Optimized for testing with:
- Deterministic settings
- Mock data sources
- In-memory exporters

## Running with Different Configurations

### Using make commands

```bash
# Run with development config (default)
make fast-run

# Run with production config
make fast-run CONFIG=configs/production/config.yaml

# Run with custom config
make fast-run CONFIG=/path/to/custom/config.yaml
```

### Using Docker

```bash
# Run with mounted config directory
docker run -v $(pwd)/configs/production:/etc/sa-omf \
  sa-omf-otelcol:latest --config=/etc/sa-omf/config.yaml
```

### Direct Binary Execution

```bash
./bin/sa-omf-otelcol --config=configs/production/config.yaml
```

## Configuration Templates

### Minimal Config Example

A minimal configuration for getting started:

```yaml
# config.yaml
receivers:
  hostmetrics:
    collection_interval: 10s

processors:
  priority_tagger:
    resource_attributes:
      service.name:
        - pattern: ".*"
          priority: low

  adaptive_topk:
    enabled: true
    metrics_pattern: [".*"]
    dimension_key: "service.name"
    k_value: 20

exporters:
  logging:
    verbosity: detailed

service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [priority_tagger, adaptive_topk]
      exporters: [logging]
```

### High Performance Config Example

Configuration optimized for high throughput:

```yaml
# config.yaml
receivers:
  hostmetrics:
    collection_interval: 30s

processors:
  priority_tagger:
    resource_attributes:
      service.name:
        - pattern: "critical.*"
          priority: high
        - pattern: ".*"
          priority: low

  adaptive_topk:
    enabled: true
    metrics_pattern: [".*"]
    dimension_key: "service.name"
    k_value: 50
    adaptation_interval: 60s

  others_rollup:
    enabled: true
    priority_attribute: "priority"
    low_priority_values: ["low"]

exporters:
  prometheusremotewrite:
    endpoint: "http://prometheus:9090/api/v1/write"
    timeout: 10s
    sending_queue:
      enabled: true
      num_consumers: 4
      queue_size: 1000

service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [priority_tagger, adaptive_topk, others_rollup]
      exporters: [prometheusremotewrite]
```

## Best Practices

### Configuration Organization

1. **Environment-based Configuration**:
   - Keep separate configs for development, testing, and production
   - Use environment variables for dynamic values

2. **Component Grouping**:
   - Group related processors together in config.yaml
   - Organize policy.yaml sections to match config.yaml structure

### PID Controller Tuning

1. **Start Conservative**:
   - Begin with low kp, ki, kd values (e.g., kp=0.3, ki=0.1, kd=0.0)
   - Increase gradually while monitoring stability

2. **Stability First**:
   - Always enable oscillation detection (circuit breakers)
   - Set reasonable hysteresis values (3-5%)
   - Use integral windup limits to prevent overcorrection

### Performance Optimization

1. **Metrics Selection**:
   - Be specific with metrics_pattern to process only relevant metrics
   - Use regex patterns carefully to avoid excessive matching

2. **Pipeline Design**:
   - Place less intensive processors earlier in the pipeline
   - Use parallel pipelines for different metric types when possible

### Safety Configuration

1. **Resource Protection**:
   - Always set max_memory_percent and max_cpu_percent
   - Configure cooldown periods between adaptations

2. **Adaptation Rate Limiting**:
   - Set max_adaptation_frequency to prevent thrashing
   - Use longer adaptation intervals in production (60s+)

## Troubleshooting

### Common Configuration Issues

1. **Pipeline Configuration**:
   - Check that all referenced components are defined
   - Ensure processor order is logical (e.g., priority_tagger before others_rollup)

2. **PID Controller Issues**:
   - Oscillating values: Decrease kp and kd, increase hysteresis
   - Slow convergence: Increase ki slightly
   - Overshooting: Decrease ki, increase integral_windup_limit

3. **Metric Pattern Problems**:
   - Invalid regex patterns will silently fail to match
   - Too-broad patterns can impact performance

## Conclusion

Proper configuration is crucial for getting the most out of Phoenix's adaptive capabilities. Start with the provided environment configurations, understand the basic principles of PID controller tuning, and gradually adapt the configuration to your specific needs.

For a complete reference of all configuration options, see the [Configuration Reference](../configuration-reference.md).