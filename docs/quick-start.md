# Phoenix Quick Start Guide

This guide will help you get started with Phoenix (SA-OMF) quickly, from installation to basic configuration.

## Prerequisites

- Go 1.21 or higher
- Docker (optional, for containerized deployment)
- Basic familiarity with OpenTelemetry concepts

## Installation Options

### Option 1: Direct Build from Source

```bash
# Clone the repository
git clone https://github.com/deepaucksharma/Phoenix.git
cd Phoenix

# Build the binary
make build

# Run with default configuration
make run
```

### Option 2: Using Docker

```bash
# Clone the repository
git clone https://github.com/deepaucksharma/Phoenix.git
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
  priority_tagger:
    rules:
      - match_type: regexp
        resource_attributes:
          process.name: "(nginx|redis|mongodb)"
        priority: high
  
  adaptive_topk:
    k: 100  # Initial value, will be adapted automatically
    
  others_rollup:
    threshold: 0.5  # Initial value, will be adapted automatically

  adaptive_pid:
    enabled: true

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
```

### Core policy.yaml Structure

```yaml
adaptive_processors:
  adaptive_topk:
    coverage_controller:
      target_value: 0.95
      pid:
        kp: 0.5
        ki: 0.1
        kd: 0.05
        integral_limit: 20.0
      safety:
        min_value: 10
        max_value: 500
        adaption_interval: "30s"
        
  others_rollup:
    cardinality_controller:
      target_value: 10000
      pid:
        kp: 0.3
        ki: 0.05
        kd: 0.02
      safety:
        min_threshold: 0.1
        max_threshold: 0.9
```

## Testing Your Installation

After starting Phoenix, you can verify it's working correctly by:

1. Checking logs for startup messages:
   ```bash
   tail -f logs/phoenix.log
   ```

2. Viewing the metrics endpoint:
   ```bash
   curl http://localhost:8888/metrics
   ```

3. Checking Prometheus (if configured):
   ```bash
   # Open in browser
   http://localhost:9090/graph
   ```

## Monitoring Adaptive Behavior

To observe Phoenix's adaptive behavior:

1. **Check Exported Metrics**: Look for metrics with these prefixes:
   - `phoenix_pid_controller_*`: Information about PID controllers
   - `phoenix_adaptive_topk_*`: Information about the adaptive_topk processor
   - `phoenix_others_rollup_*`: Information about the others_rollup processor

2. **Watch Log Events**: Set verbosity to detailed to see adaptation events:
   ```
   2025-05-20T12:34:56.789Z INFO AdaptiveTopK: Adjusting k from 100 to 125 (target coverage: 0.95, current: 0.92)
   ```

3. **Use Grafana Dashboards**: If you're using the full Docker Compose setup, open Grafana at http://localhost:3000 and use the pre-configured dashboards.

## Common Configurations

### For Low-Resource Environments

```yaml
# In config.yaml
hostmetrics:
  collection_interval: 60s  # Reduce collection frequency

# In policy.yaml
adaptive_topk:
  coverage_controller:
    target_value: 0.9  # Slightly lower coverage target
    safety:
      max_value: 200   # Lower maximum k value
```

### For High-Cardinality Environments

```yaml
# In policy.yaml
others_rollup:
  cardinality_controller:
    target_value: 15000  # Higher cardinality target
    pid:
      kp: 0.7            # More aggressive control
```

### For Development/Testing

```yaml
# In config.yaml
processors:
  adaptive_pid:
    debug_mode: true     # More verbose logging of controller actions

# In policy.yaml
adaptive_processors:
  _global:
    safety:
      adaption_interval: "10s"  # Faster adaptation for testing
```

## Next Steps

After getting Phoenix running, here are some next steps:

1. **Read the [Architecture Overview](architecture.md)** to understand the system
2. **Learn about [Adaptive Processing](adaptive-processing.md)** concepts
3. **Explore [PID Controllers](pid-controllers.md)** for deeper tuning
4. **Consult the [Configuration Reference](configuration-reference.md)** for detailed options

## Troubleshooting

### Common Issues

| Issue | Possible Solutions |
|-------|-------------------|
| Phoenix fails to start | Check port conflicts, file permissions, Go version |
| No metrics collected | Verify hostmetrics configuration, check collection_interval |
| Excessive resource usage | Adjust target values in policy.yaml, reduce collection frequency |
| Oscillating parameters | Tune PID parameters (reduce kp, ki), increase adaptation interval |

### Getting Help

If you encounter issues not covered here:

1. Check the [full documentation](README.md)
2. File an issue on GitHub with details about your environment and configuration