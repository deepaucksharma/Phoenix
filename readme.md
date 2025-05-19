# SA-OMF: Self-Aware OpenTelemetry Metrics Fabric

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

### Installation

1. Clone this repository:
   ```bash
   git clone https://github.com/yourorg/sa-omf.git
   cd sa-omf
   ```

2. Build the collector:
   ```bash
   make build
   ```

3. Run with a local configuration:
   ```bash
   ./bin/sa-omf-otelcol --config=./examples/config.yaml
   ```

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

## Development

The implementation is structured into multiple phases:

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
