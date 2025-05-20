# Phoenix: Self-Aware OpenTelemetry Metrics Fabric (SA-OMF)

Phoenix (codename for SA-OMF) is an advanced metrics collection and processing system built on top of OpenTelemetry. It features adaptive processing that automatically adjusts parameters based on system behavior through PID control loops.

## Overview

SA-OMF enables intelligent, self-adaptive metrics processing by:

- **Dynamically adjusting** key parameters in real-time
- **Self-tuning** to maintain target KPIs
- **Monitoring** its own behavior through control feedback loops
- **Protecting** against resource exhaustion with built-in safety mechanisms

## Key Features

- **Dual Pipeline Architecture**: Separates data processing from control operations
- **Adaptive Processors**: Self-tuning processors that adjust their parameters automatically
- **PID Control Loops**: Industrial-grade control theory applied to software systems
- **Safety Mechanisms**: Built-in guard rails to prevent resource exhaustion
- **Configuration Policies**: Define target KPIs and acceptable operating parameters

## Architecture

The architecture consists of:

1. **Data Pipeline**: Processes incoming metrics data
   - Collects metrics from standard OpenTelemetry receivers
   - Processes through various adaptive processors
   - Exports metrics to configured destinations

2. **Control Pipeline**: Monitors and adjusts the data pipeline
   - Monitors self-metrics
   - Evaluates KPIs against targets
   - Generates and applies configuration patches
   - Maintains system within operational bounds

## Core Components

- **pid_controller**: Implementation of PID controllers with:
  - Anti-windup protection
  - Derivative filtering
  - Circuit breakers to prevent oscillation

- **pic_control_ext**: Central governance layer for config changes
  - Manages policy file watching
  - Handles configuration change requests
  - Enforces rate limiting and safety measures

- **adaptive_pid**: Generates configuration patches using PID control
  - Monitors KPIs and calculates needed configuration changes
  - Uses PID controllers for stable parameter adjustments

- **adaptive_topk**: Dynamically adjusts k parameter based on coverage score
  - Uses Space-Saving algorithm for top-k tracking
  - Self-tunes to achieve target coverage with minimal k-value

- **Bayesian Optimization**: Advanced parameter tuning
  - Multi-dimensional optimization for complex parameter spaces
  - Latin Hypercube Sampling for efficient space exploration
  - Configurable exploration-exploitation balance

## Recent Improvements

The project has undergone significant stability and reliability improvements:

- [Enhanced PID Controllers with Derivative Filtering & Circuit Breakers](docs/improvements/stability-improvements.md#1-pid-controller-enhancements)
- [Fixed Space-Saving Algorithm for Accurate Error Tracking](docs/improvements/stability-improvements.md#2-space-saving-algorithm-corrections)
- [Improved Concurrency Handling & Thread Safety](docs/improvements/stability-improvements.md#3-concurrency-handling-improvements)
- [Advanced Bayesian Optimization with Multi-Dimensional Support](docs/improvements/stability-improvements.md#4-bayesian-optimization-enhancements)

See the [complete stability improvements documentation](docs/improvements/stability-improvements.md) for details.

## Getting Started

### Prerequisites

- Go 1.24 or higher
- OpenTelemetry Collector Contrib
- Docker (for containerized testing)

### Build and Run

```bash
# Build the collector binary
make build

# Run the collector with default config
make run

# Run with specific config
./bin/sa-omf-otelcol --config=configs/production/config.yaml
```

### Docker Deployment

```bash
# Run with Docker Compose (bare environment)
docker-compose -f deploy/compose/bare/docker-compose.yaml up -d

# Run with Docker Compose (Prometheus included)
docker-compose -f deploy/compose/prometheus/docker-compose.yaml up -d

# Run with Docker Compose (full stack with Grafana)
docker-compose -f deploy/compose/full/docker-compose.yaml up -d
```

## Documentation

For more information, see:
- [Architecture Documentation](docs/architecture/README.md)
- [Component Documentation](docs/components/README.md)
- [Development Guide](docs/development-guide.md)
- [Configuration Reference](docs/configuration-reference.md)
- [Concept Documentation](docs/concepts/README.md)

## License

This project is licensed under the [LICENSE](LICENSE) file in the repository.