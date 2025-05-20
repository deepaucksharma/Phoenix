# Phoenix Components

This document provides an overview of the key components in the SA-OMF (Phoenix) system.

## Architecture Overview

Phoenix is built on a dual-pipeline architecture:
- **Data Pipeline**: For primary metrics processing
- **Control Pipeline**: For self-monitoring and adaptive control

## Core Components

### Processors

Processors handle metrics data transformation and filtering:

| Processor | Description | Status |
|-----------|-------------|--------|
| [metric_pipeline](./processor/metric_pipeline.md) | Unified filtering and transformation processor | Implemented |
| [adaptive_pid](./processors/adaptive_pid.md) | Generates configuration patches using PID control | Implemented |
| [cardinality_guardian](./processors/cardinality_guardian.md) | Controls metrics cardinality | In Progress |
| [reservoir_sampler](./processors/reservoir_sampler.md) | Statistical sampling with adjustable reservoir sizes | In Progress |
| [process_context_learner](./processors/process_context_learner.md) | Learns and associates context with processes | In Progress |
| [multi_temporal_adaptive_engine](./processors/multi_temporal_adaptive_engine.md) | Advanced multi-timescale adaptive processing | In Progress |
| [semantic_correlator](./processors/semantic_correlator.md) | Discovers relationships between metrics | In Progress |

All processors implement the `UpdateableProcessor` interface which allows them to be dynamically reconfigured at runtime.

### Extensions

There are currently no extensions shipped with Phoenix. This section is reserved for future components that may operate outside the data pipeline.

### Connectors

Connectors bridge between components:

| Connector | Description | Status |
|-----------|-------------|--------|
| None currently in production | - | - |

*Note: The previously documented `pic_connector` has been removed as part of the codebase cleanup.*

### Control Components

| Component | Description | Status |
|-----------|-------------|--------|
| [pid_controller](./pid/pid_controller.md) | Core controller implementation | Implemented |
| [safety_monitor](./safety_monitor.md) | Provides safeguards against excessive resource usage | Implemented |

## Interfaces

The system is built around these key interfaces:

- **UpdateableProcessor**: Allows processors to be dynamically reconfigured
  - `OnConfigPatch(ctx, patch)`: Apply a configuration change
  - `GetConfigStatus(ctx)`: Return current configuration

- **ConfigPatch**: Structure representing a proposed change to a processor's configuration

## Component Relationships

### Dual Pipeline Architecture

Phoenix implements a dual pipeline architecture that separates data processing from control mechanisms:

```
┌───────────────┐          ┌───────────────┐
│   Collectors  │          │  Self-Metrics │
└───────┬───────┘          └───────┬───────┘
        │                          │
        ▼                          ▼
┌───────────────┐          ┌───────────────┐
│    Data       │          │   Control     │
│   Pipeline    │          │   Pipeline    │
└───────┬───────┘          └───────┬───────┘
        │                          │
        ▼                          ▼
┌───────────────┐          ┌───────────────┐
│  Exporters    │          │ ConfigPatch   │
└───────────────┘          └───────┬───────┘
                                   │
                                   ▼
                           ┌───────────────┐
                           │ UpdateableProcessor
                           │   Components  │
                           └───────────────┘
```

### PID Control Flow

The PID control system integrates with the dual pipeline architecture as follows:

```
┌───────────────┐          ┌───────────────┐
│ Target KPIs   │          │  Measured     │
│ (policy.yaml) │◄────────►│     KPIs      │
└───────┬───────┘          └───────┬───────┘
        │                          │
        ▼                          ▼
┌───────────────┐          ┌───────────────┐
│  PID          │          │  Safety       │
│  Controller   │◄────────►│  Monitor      │
└───────┬───────┘          └───────┬───────┘
        │                          │
        └──────────┬───────────────┘
                   │
                   ▼
           ┌───────────────┐
           │ Configuration │
           │    Patches    │
           └───────┬───────┘
                   │
                   ▼
           ┌───────────────┐
           │ Processor     │
           │ Parameters    │
           └───────────────┘
```

## Utility Packages

The system includes several utility packages that implement key algorithms:

| Package | Description | Status |
|---------|-------------|--------|
| [hll](./util/hll.md) | HyperLogLog for cardinality estimation | Implemented |
| [reservoir](./util/reservoir.md) | Reservoir sampling algorithms | Implemented |
| [topk](./util/topk.md) | Space-saving algorithm for top-k tracking | Implemented |
| [bayesian](./util/bayesian.md) | Gaussian process and Bayesian optimization | Implemented |
| [causality](./util/causality.md) | Granger causality and transfer entropy | Implemented |
| [timeseries](./util/timeseries.md) | Anomaly detection and forecasting | Implemented |

## Configuration

Components are configured through two primary configuration files:

1. `config.yaml`: Standard OpenTelemetry Collector configuration
   - Defines receivers, processors, exporters, and service pipelines
   - Configures component connections and basic settings

2. `policy.yaml`: Self-adaptive behavior configuration
   - Defines KPIs and target values
   - Configures PID controller parameters (kp, ki, kd)
   - Sets safety thresholds and limits
   - Defines parameter bounds for adaptive components
   - Includes rate limiting for configuration changes

Different environment configurations are available in `configs/[environment]/`:
- **default/**: Standard baseline configuration
- **development/**: More verbose, faster adaptation for development
- **production/**: More conservative settings for stability
- **testing/**: Configuration optimized for tests

See the [Configuration Reference](../configuration-reference.md) for detailed documentation.

## Safety Mechanisms

Phoenix implements multiple safety mechanisms to ensure operational stability:

| Mechanism | Description |
|-----------|-------------|
| **Safe Mode Activation** | When resource limits are reached, the system can enter a safe mode with conservative settings |
| **Config Patch Rate Limiting** | Limits the frequency of configuration changes to prevent oscillation |
| **Policy Validation** | Validates policy files against schema to prevent invalid configurations |
| **Parameter Bounds** | All configurable parameters have defined safe ranges |
| **Hysteresis** | Prevents oscillation by requiring significant change before adaptation |
| **Anti-windup Protection** | Prevents integral term from growing too large in PID controllers |

## Implementation Patterns

The system uses several key implementation patterns:

1. **Base Processor** - Common functionality for all processors is implemented in the base processor

2. **Interface-driven Design** - All components implement interfaces that define their behavior

3. **Factory Pattern** - Components are instantiated through factory methods for modularity

4. **PID Control Loop** - Feedback-based control with proportional, integral, and derivative terms

5. **Dynamic Configuration** - Runtime reconfiguration through well-defined configuration patches