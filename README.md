# Phoenix: Self-Aware OpenTelemetry Metrics Fabric (SA-OMF)

Phoenix (codename for SA-OMF) is an advanced metrics collection and processing system built on top of OpenTelemetry. It features adaptive processing that automatically adjusts parameters based on system behavior through embedded PID control and optimization algorithms.

## Overview

SA-OMF enables intelligent, self-adaptive metrics processing by:

- **Dynamically adjusting** key parameters in real-time
- **Self-tuning** to maintain target KPIs
- **Monitoring** its own behavior through internal feedback loops
- **Protecting** against resource exhaustion with built-in safety mechanisms

## Key Features

- **Adaptive Processors**: Self-tuning processors that adjust their parameters automatically
- **PID Control Systems**: Industrial-grade control theory applied to software systems
- **Safety Mechanisms**: Built-in guard rails to prevent resource exhaustion
- **Configuration Policies**: Define target KPIs and acceptable operating parameters

## Architecture

Phoenix uses a dual-pipeline architecture for separation of concerns:

1. **Data Pipeline**: Processes incoming metrics data
   - Collects metrics from standard OpenTelemetry receivers
   - Processes through various adaptive processors
   - Exports metrics to configured destinations

2. **Control Pipeline**: Monitors and adjusts the data pipeline
   - Monitors metrics from the data pipeline
   - Calculates KPIs and control signals using PID controllers
   - Generates and applies configuration changes

For more details, see the [Architecture Overview](docs/architecture.md).

## Development Workflow

The Phoenix project uses `make` as its primary development interface:

```bash
# Build the project
make build

# Run with development config
make run

# Run unit tests
make test-unit

# Run integration tests
make test-integration

# Run benchmarks
make benchmark

# For help with all available commands
make help
```

For more detailed instructions, see the [Development Guide](docs/development-guide.md).

## Core Components

- **Adaptive Processors**:
  - **adaptive_topk**: Dynamically selects top-k resources by importance
    - Uses Space-Saving algorithm for accurate frequency tracking
    - Self-tunes to achieve target coverage with minimal k-value
  - **priority_tagger**: Tags resources with priority levels based on rules
    - Flexible rule matching with regexp support
    - Provides basis for priority-based processing
  - **others_rollup**: Aggregates low-priority metrics to reduce cardinality
    - Configurable priority threshold
    - Maintains detailed metrics for important resources
  - **adaptive_pid**: Monitors KPIs and provides insights into system performance
    - Uses PID controllers for stable monitoring
    - Provides both PID and Bayesian optimization approaches
    - Configurable control parameters for each target

- **PID Controllers**: Self-regulating feedback control systems
  - Proportional-Integral-Derivative terms for precise control
  - Anti-windup protection to prevent integral term saturation
  - Low-pass derivative filtering for noise reduction
  - Oscillation detection and circuit breaking for stability
  - Thread-safe implementation for concurrent access
  - Configurable limits and bounds for all parameters

- **Advanced Optimization**:
  - Bayesian optimization with Gaussian processes
  - Multi-dimensional parameter space exploration
  - Dynamic exploration/exploitation balance
  - Latin Hypercube Sampling for efficient parameter space exploration

## PID Control System

The PID controller is a key component that provides feedback-based control:

- Computes error between target value and measured value
- Applies proportional, integral, and derivative terms to compute corrections
- Generates stable control signals for processor reconfiguration
- Includes configurable integral windup protection
- Handles hysteresis to prevent oscillation
- Features circuit breaker for oscillation detection and mitigation

See [PID Controllers](docs/pid-controllers.md) for details.

## Quick Start

### Prerequisites

- Go 1.24 or higher
- OpenTelemetry Collector Contrib
- Docker (for containerized testing)

### Quick Start with Dev Container

The fastest way to get started is using the VS Code Dev Container:

1. Install [VS Code](https://code.visualstudio.com/) and the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. Clone this repository
3. Open the repository in VS Code
4. Click "Reopen in Container" when prompted
5. The container will set up all dependencies and tools automatically

### Local Development

```bash
# Set up development environment (installs tools and dependencies)
make dev-setup

# Build the collector
make build

# Run the collector with development config
make run 

# Run with a specific config
make run CONFIG=configs/production/config.yaml
```

### Docker Deployment

```bash
# Development environment with mounted source code
docker-compose up dev

# Run with Prometheus for metrics visualization
docker-compose up prometheus collector-development

# Run full stack with Grafana dashboards
docker-compose up
```

## Documentation

For more information, see:

- [Quick Start Guide](docs/quick-start.md) 👈 Start here to get up and running
- [Architecture Overview](docs/architecture.md)
- [Adaptive Processing](docs/adaptive-processing.md)
- [PID Controllers](docs/pid-controllers.md)
- [Configuration Reference](docs/configuration-reference.md)
- [Documentation Home](docs/README.md)

## Recent Improvements

The project has undergone significant stability and reliability improvements:

- [Enhanced PID Controllers with Derivative Filtering & Circuit Breakers](docs/improvements/stability-improvements.md#1-pid-controller-enhancements)
- [Fixed Space-Saving Algorithm for Accurate Error Tracking](docs/improvements/stability-improvements.md#2-space-saving-algorithm-corrections)
- [Improved Concurrency Handling & Thread Safety](docs/improvements/stability-improvements.md#3-concurrency-handling-improvements)
- [Advanced Bayesian Optimization with Multi-Dimensional Support](docs/improvements/stability-improvements.md#4-bayesian-optimization-enhancements)

See the [complete stability improvements documentation](docs/improvements/stability-improvements.md) for details.

## License

This project is licensed under the [LICENSE](LICENSE) file in the repository.