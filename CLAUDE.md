# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SA-OMF (Self-Aware OpenTelemetry Metrics Fabric) is an advanced metrics collection and processing system built on top of OpenTelemetry. It features adaptive processing that automatically adjusts parameters based on system behavior through PID control loops.

The architecture consists of:
- **Dual pipeline architecture**: Data pipeline for metrics processing and control pipeline for self-monitoring
- **Adaptive processors**: Self-tuning processors that can be dynamically reconfigured
- **PID control loops**: Self-regulation of key parameters to maintain optimal performance

## Development Commands

### Build and Run

```bash
# Build the collector binary
make build

# Run the collector with default config
make run

# Run all tests
make test

# Run only unit tests
make test-unit

# Run only integration tests
make test-integration

# Run tests with coverage
make test-coverage

# Run benchmarks
make benchmark

# Run linting
make lint

# Generate mocks for testing
make mocks

# Build Docker image
make docker

# Create a release tag
make release VERSION=x.y.z

# Show all available commands
make help
```

### Test-specific Commands

```bash
# Run tests in the test directory
cd test && make test

# Run unit tests only in test directory
cd test && make unit

# Run integration tests only in test directory
cd test && make integration

# Generate test coverage report
cd test && make coverage

# Run benchmarks in test directory
cd test && make benchmark
```

### Running in Containers

```bash
# Run with Docker Compose (bare environment)
cd test-environments/bare && docker-compose up -d

# Run with Docker Compose (Prometheus included)
cd test-environments/prometheus && docker-compose up -d 

# Run with Docker Compose (full stack with Grafana)
cd test-environments/full && docker-compose up -d
```

### Typical Development Workflow

1. Make code changes
2. Run linting: `make lint`
3. Run tests: `make test`
4. Build and run the collector: `make build && make run`
5. Verify changes work as expected through the collector's output and metrics

## Architecture Components

### Core Interfaces

- **UpdateableProcessor**: Key interface that allows processors to be dynamically reconfigured via ConfigPatch
  - `OnConfigPatch(ctx, patch)`: Apply a configuration change
  - `GetConfigStatus(ctx)`: Return current configuration

- **ConfigPatch**: Structure representing a proposed change to a processor's configuration

### Key Extensions

- **pic_control**: Central governance layer for config changes
  - Manages policy file watching
  - Handles configuration change requests
  - Enforces rate limiting and safety measures
  - Registers and manages all UpdateableProcessor instances

### Processors

1. **priority_tagger**: Tags resources with priority levels based on configurable rules
   - Implements UpdateableProcessor interface
   - Uses regexp matching for process prioritization

2. **adaptive_topk**: Dynamically adjusts k parameter based on coverage score
   - Uses Space-Saving algorithm for top-k tracking
   - Self-tunes to achieve target coverage with minimal k-value

3. **Other planned processors**:
   - cardinality_guardian: Controls metrics cardinality
   - reservoir_sampler: Provides statistical sampling with adjustable reservoir sizes
   - others_rollup: Aggregates non-priority processes

### Control Components

1. **pid_decider**: Generates configuration patches using PID control
   - Monitors KPIs and calculates configuration changes needed
   - Uses PID controllers for stable adjustments
   - Emits configuration patches via metrics

2. **pic_connector**: Connects pid_decider to pic_control
   - Extracts ConfigPatch objects from metrics
   - Submits them to pic_control

### PID Controller

The PID controller is a key component that provides feedback-based control:
- Computes error between target value and measured value
- Applies proportional, integral, and derivative terms
- Generates stable control signals for processor reconfiguration

### Policy Management

The system is configured through a policy.yaml file which defines:
- KPIs and target values
- PID controller parameters
- Processor configurations
- Safety thresholds and guard-rails

## Data Flow

1. **Data Pipeline**:
   - Collects metrics from hostmetrics receiver
   - Processes through various adaptive processors
   - Exports metrics to configured destinations

2. **Control Pipeline**:
   - Monitors self-metrics
   - Evaluates KPIs against targets
   - Generates and applies configuration patches
   - Ensures system stays within operational bounds

## Working with the Codebase

### Project Structure

```
sa-omf/
├── cmd/
│   └── sa-omf-otelcol/             # Main binary entrypoint
├── internal/
│   ├── interfaces/                  # Core interfaces (UpdateableProcessor, etc.)
│   ├── extension/
│   │   └── piccontrolext/           # pic_control implementation
│   ├── connector/
│   │   └── picconnector/            # pic_connector implementation
│   ├── processor/                   # All custom processors
│   └── control/                     # Control logic helpers
├── pkg/                             # Reusable packages
├── test/
│   ├── unit/                        # Unit tests for core algorithms
│   ├── interfaces/                  # Interface contract tests
│   ├── processors/                  # Processor-specific tests
│   ├── integration/                 # End-to-end tests
│   └── testutils/                   # Testing utilities
├── deploy/
│   ├── kubernetes/                  # K8s deployment manifests
│   └── docker/                      # Dockerfile
└── docs/                            # Documentation
```

### Adding a New Processor

1. Create factory.go and processor.go in internal/processor/yourprocessor/
2. Implement the UpdateableProcessor interface
3. Register your processor in the collector factory
4. Add processor configuration to policy schema
5. Update config.yaml and policy.yaml with default configs
6. Add unit and integration tests in test/processors/yourprocessor/

### Modifying PID Controllers

PID controllers are defined in the policy.yaml file:
- Adjust kp, ki, kd values to change control behavior
- Set target_value to define the desired KPI state
- Configure output_config_patches to specify what parameters get adjusted

### Testing Your Changes

1. Use processor_test_template.go for standard processor test structure
2. Test UpdateableProcessor compliance with interfaces/updateable_processor_test.go
3. For control components, use testutils/pid_helper.go to test PID behavior
4. Add benchmarks for performance-critical components
5. For integration testing, use testutils/metrics_generator.go to create synthetic test data

### Safety Mechanism Development

The system includes several safety mechanisms:
- Safe mode activation when resource limits are reached
- Configuration patch rate limiting
- Policy validation against schema
- Parameter bounds checking in patches

### Prerequisites

- Go 1.21 or higher
- OpenTelemetry Collector Contrib
- Docker (for containerized testing)
- Kubernetes (optional, for orchestrated deployment)