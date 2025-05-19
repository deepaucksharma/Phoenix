# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SA-OMF (Self-Aware OpenTelemetry Metrics Fabric) is an advanced metrics collection and processing system built on top of OpenTelemetry. It features adaptive processing that automatically adjusts parameters based on system behavior through PID control loops.

**Project Codename**: Phoenix  
**Current Implementation Timeline**: 18 months  
**Repository Structure**: Monorepo with modular packages

The architecture consists of:
- **Dual pipeline architecture**: Data pipeline for metrics processing and control pipeline for self-monitoring
- **Adaptive processors**: Self-tuning processors that can be dynamically reconfigured
- **PID control loops**: Self-regulation of key parameters to maintain optimal performance
- **Safety mechanisms**: Built-in guard rails to prevent resource exhaustion

## Development Commands

### Build and Run

```bash
# Build the collector binary
make build

# Run the collector with default config
make run

# Run with specific config
./bin/sa-omf-otelcol --config=configs/production/config.yaml

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

# Code consistency check for interdependent files
make drift-check

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

# Generate comprehensive test report
cd test && make report

# Run benchmarks in test directory
cd test && make benchmark

# Run targeted tests for specific components
go test -v ./test/processors/adaptive_pid/...
go test -v ./test/unit/hll/...

# Run benchmarks for specific algorithms
go test -v ./test/benchmarks/algorithms/... -bench=.
```

### Running in Containers

```bash
# Run with Docker Compose (bare environment - from project root)
docker-compose -f deploy/compose/bare/docker-compose.yaml up -d

# Run with Docker Compose (Prometheus included - from project root)
docker-compose -f deploy/compose/prometheus/docker-compose.yaml up -d

# Run with Docker Compose (full stack with Grafana - from project root)
docker-compose -f deploy/compose/full/docker-compose.yaml up -d

# Build and run using Docker directly
docker build -t sa-omf-otelcol:latest -f deploy/docker/Dockerfile .
docker run -p 8888:8888 -v $PWD/configs/default:/etc/sa-omf sa-omf-otelcol:latest --config=/etc/sa-omf/config.yaml

# Deploy to Kubernetes
kubectl apply -f deploy/kubernetes/prometheus-operator-resources.yaml
```

### Validation and Configuration Testing

```bash
# Validate policy schema
scripts/validation/validate_policy_schema.sh

# Validate config schema
scripts/validation/validate_config_schema.sh

# Create a new ADR
scripts/dev/new-adr.sh "My New Architecture Decision"

# Create a new component
scripts/dev/new-component.sh processor my_new_processor

# Create a new branch
scripts/dev/create-branch.sh feature/new-feature

# Create a new task
scripts/dev/create-task.sh "Add new processor"
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

- **pic_control_ext**: Central governance layer for config changes
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

3. **adaptive_pid**: Generates configuration patches using PID control
   - Monitors KPIs and calculates configuration changes needed
   - Uses PID controllers for stable adjustments
   - Emits configuration patches via metrics

4. **Other planned processors**:
   - cardinality_guardian: Controls metrics cardinality
   - reservoir_sampler: Provides statistical sampling with adjustable reservoir sizes
   - others_rollup: Aggregates non-priority processes

### Control Components

1. **pid_controller**: Core controller implementation in control/pid package
   - Computes error between target value and measured value
   - Applies proportional, integral, and derivative terms
   - Generates stable control signals for processor reconfiguration
   - Handles anti-windup protection

2. **pic_connector**: Connects adaptive_pid to pic_control_ext
   - Extracts ConfigPatch objects from metrics
   - Submits them to pic_control_ext

3. **safety_monitor**: Provides safeguards against excessive resource usage
   - Monitors system resources
   - Can trigger safe mode when thresholds are exceeded

### PID Control System

The PID controller is a key component that provides feedback-based control:
- Computes error between target value and measured value
- Applies proportional, integral, and derivative terms to compute corrections
- Generates stable control signals for processor reconfiguration
- Includes configurable integral windup protection
- Handles hysteresis to prevent oscillation

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
├── configs/
│   ├── default/                    # Default configuration
│   ├── development/                # Development configuration
│   ├── production/                 # Production configuration
│   ├── testing/                    # Testing configuration
│   └── examples/                   # Example configurations
├── internal/
│   ├── interfaces/                 # Core interfaces (UpdateableProcessor, etc.)
│   ├── extension/
│   │   └── pic_control_ext/        # pic_control implementation
│   ├── connector/
│   │   └── pic_connector/          # pic_connector implementation
│   ├── processor/                  # All custom processors
│   │   ├── adaptive_pid/           # PID-based adaptive configuration
│   │   ├── adaptive_topk/          # Adaptive top-k filtering
│   │   ├── priority_tagger/        # Priority tagging processor
│   │   └── base/                   # Base processor implementation
│   └── control/                    # Control logic helpers
│       ├── pid/                    # PID controller implementation
│       ├── configpatch/            # Configuration patch validation
│       └── safety/                 # Safety monitoring
├── pkg/                            # Reusable packages
│   ├── metrics/                    # Metrics utilities
│   ├── policy/                     # Policy schema and validation
│   └── util/                       # Utility algorithms
│       ├── hll/                    # HyperLogLog for cardinality estimation
│       ├── reservoir/              # Reservoir sampling
│       └── topk/                   # Top-K frequency tracking
├── test/
│   ├── unit/                       # Unit tests for core algorithms
│   ├── interfaces/                 # Interface contract tests
│   ├── processors/                 # Processor-specific tests
│   ├── integration/                # End-to-end tests
│   ├── benchmarks/                 # Performance benchmarks
│   ├── testutils/                  # Testing utilities
│   └── generator/                  # Test data generation
├── deploy/
│   ├── kubernetes/                 # K8s deployment manifests
│   ├── docker/                     # Dockerfile
│   └── compose/                    # Docker Compose configurations
└── docs/                           # Documentation
    ├── architecture/               # Architecture documentation
    │   └── adr/                    # Architecture Decision Records
    ├── components/                 # Component documentation
    ├── concepts/                   # Concept documentation
    ├── operations/                 # Operational documentation
    └── tutorials/                  # Tutorials
```

### Adding a New Processor

1. Create factory.go and processor.go in internal/processor/yourprocessor/
2. Implement the UpdateableProcessor interface
3. Extend the BaseProcessor for common functionality
4. Register your processor in the collector factory
5. Add processor configuration to policy schema
6. Update config.yaml and policy.yaml with default configs
7. Add unit and integration tests in test/processors/yourprocessor/
8. Add performance benchmarks in test/benchmarks/processors/yourprocessor/

### Modifying PID Controllers

PID controllers are defined in the policy.yaml file:
- Adjust kp, ki, kd values to change control behavior
- Set target_value to define the desired KPI state
- Configure hysteresis_percent to prevent oscillation
- Set integral_windup_limit to prevent integral term from growing too large
- Configure output_config_patches to specify what parameters get adjusted

### Testing Your Changes

1. Use processor_test_template.go for standard processor test structure
2. Test UpdateableProcessor compliance with interfaces/updateable_processor_test.go
3. For control components, use testutils/pid_helper.go to test PID behavior
4. Add benchmarks for performance-critical components
5. For integration testing, use testutils/metrics_generator.go to create synthetic test data

### Configuration Management

The system uses two main types of configuration files:

1. **config.yaml**: Standard OpenTelemetry Collector configuration
   - Defines receivers, processors, exporters, and pipelines
   - Sets up component connections

2. **policy.yaml**: Self-adaptive behavior configuration
   - Defines KPIs and target values
   - Configures PID controller parameters
   - Sets safety thresholds and limits
   - Controls adaptive processor behavior

Different environment configurations are available in configs/[environment]/:
- **default/**: Standard baseline configuration
- **development/**: More verbose, faster adaptation for development
- **production/**: More conservative settings for stability
- **testing/**: Configuration optimized for tests

### Safety Mechanism Development

The system includes several safety mechanisms:
- Safe mode activation when resource limits are reached
- Configuration patch rate limiting
- Policy validation against schema
- Parameter bounds checking in patches
- Hysteresis to prevent oscillation

### Prerequisites

- Go 1.21 or higher
- OpenTelemetry Collector Contrib
- Docker (for containerized testing)
- Kubernetes (optional, for orchestrated deployment)