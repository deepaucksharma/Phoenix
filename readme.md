# SA-OMF: Self-Aware OpenTelemetry Metrics Fabric

[![CI](https://github.com/yourorg/sa-omf/actions/workflows/ci.yml/badge.svg)](https://github.com/yourorg/sa-omf/actions/workflows/ci.yml)
[![CodeQL](https://github.com/yourorg/sa-omf/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/yourorg/sa-omf/actions/workflows/codeql-analysis.yml)
[![Cross-Platform Tests](https://github.com/yourorg/sa-omf/actions/workflows/cross-platform-tests.yml/badge.svg)](https://github.com/yourorg/sa-omf/actions/workflows/cross-platform-tests.yml)
[![Release](https://github.com/yourorg/sa-omf/actions/workflows/release.yml/badge.svg)](https://github.com/yourorg/sa-omf/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourorg/sa-omf)](https://goreportcard.com/report/github.com/yourorg/sa-omf)
[![Go Reference](https://pkg.go.dev/badge/github.com/yourorg/sa-omf.svg)](https://pkg.go.dev/github.com/yourorg/sa-omf)

A self-optimizing OpenTelemetry Collector designed to intelligently adapt its processing behavior based on real-time performance metrics.

## Project Overview

**Project Codename**: Phoenix  
**Target Implementation Timeline**: 18 months  
**Repository Structure**: Monorepo with modular packages  

The Self-Aware OpenTelemetry Metrics Fabric (SA-OMF) is an advanced metrics collection and processing system built on top of OpenTelemetry. It features:

- **Adaptive processing**: Automatically adjusts processing parameters based on system behavior
- **Dual pipeline architecture**: Data pipeline for metrics processing and control pipeline for self-monitoring
- **PID control loops**: Self-regulation of key parameters to maintain optimal performance
- **Safety mechanisms**: Built-in guard rails to prevent resource exhaustion

## Getting Started

### Prerequisites

- Go 1.21 or higher
- OpenTelemetry Collector Contrib
- Docker (for containerized deployment)
- Kubernetes (optional, for orchestrated deployment)

### Quick Start

1. Clone this repository:
   ```bash
   git clone https://github.com/yourorg/sa-omf.git
   cd sa-omf
   ```

2. Build the collector:
   ```bash
   make build
   ```

3. Run with the default configuration:
   ```bash
   make run
   ```

### Running with Docker

```bash
# Build Docker image
make docker

# Run with Docker Compose
cd deploy/compose/bare && docker-compose up -d
```

## Repository Structure

- **cmd/**: Application entrypoints
- **configs/**: Configuration files
  - **default/**: Default configurations
  - **examples/**: Example configurations
- **deploy/**: Deployment resources
  - **docker/**: Dockerfile and related resources
  - **kubernetes/**: Kubernetes deployment manifests
  - **compose/**: Docker Compose configurations
- **docs/**: Documentation
  - **architecture/**: System architecture documentation
  - **agents/**: Claude Code agent configurations
  - **quickstarts/**: Quick start guides
  - **testing/**: Testing documentation
- **internal/**: Internal packages (not intended for external use)
  - **connector/**: Connector implementations
  - **control/**: Control logic components
  - **extension/**: Extension implementations
  - **interfaces/**: Core interfaces
  - **processor/**: Processor implementations
- **pkg/**: Public packages (can be imported by external projects)
  - **metrics/**: Metrics utilities
  - **policy/**: Policy management
  - **util/**: Utility packages
- **scripts/**: Helper scripts
  - **ci/**: CI scripts
  - **dev/**: Development scripts
  - **validation/**: Validation scripts
- **test/**: Test code
  - **benchmarks/**: Performance benchmarks
  - **interfaces/**: Interface tests
  - **integration/**: Integration tests
  - **processors/**: Processor tests
  - **testutils/**: Test utilities
  - **unit/**: Unit tests

## Architecture

SA-OMF follows a dual pipeline architecture:

1. **Data Pathway (Pipeline A)**: Processes host & process metrics
   - Processors: priority_tagger, adaptive_topk, cardinality_guardian, etc.
   - Designed for high throughput, adaptive resource usage

2. **Control Pathway (Pipeline B)**: Self-monitors and adjusts Pipeline A
   - Components: pid_decider, pic_connector, pic_control
   - Generates and applies configuration patches based on observed KPIs

### Core Components

- **pic_control (Extension)**: Central governance layer for config changes
- **UpdateableProcessor (Interface)**: Contract for dynamic reconfiguration
- **pid_decider (Processor)**: Generates configuration patches using PID control
- **pic_connector (Exporter)**: Connects pid_decider to pic_control
- **priority_tagger (Processor)**: Assigns priorities to metrics based on process patterns
- **Policy Management**: policy.yaml defines KPIs, thresholds, and guard-rails

## Documentation

For more detailed documentation:

- [Architecture Documentation](docs/architecture/README.md)
- [Deployment Guide](deploy/README.md)
- [Configuration Reference](configs/README.md)
- [Development Scripts](scripts/README.md)
- [Testing Framework](docs/testing/validation-framework.md)
- [PID Integral Controls](docs/pid_integral_controls.md)

## Development

The implementation follows these development commands:

```bash
# Run tests
make test

# Run linting
make lint

# Check for code drift
make drift-check

# Generate test coverage
make test-coverage

# Show all available commands
make help
```

## Implementation Phases

The development is structured into four main phases:

### Phase 1: Foundation (Months 0-4)
- Core interfaces & framework
- pic_control extension
- First processors (priority_tagger, adaptive_topk)
- Basic control loop

### Phase 2: Enhanced Processors (Months 5-9)
- Advanced processors (cardinality_guardian, reservoir_sampler, others_rollup)
- Safety mechanisms
- Visualization dashboards

### Phase 3: Advanced Intelligence (Months 10-14)
- Learning capabilities (process_context_learner)
- OpAMP integration
- Bayesian optimization
- Causality detection

### Phase 4: Production Hardening (Months 15-18)
- Time series forecasting
- Final performance optimization
- Security hardening
- Production deployment tools

## License

This project is licensed under the Apache 2.0 License - see the LICENSE file for details.