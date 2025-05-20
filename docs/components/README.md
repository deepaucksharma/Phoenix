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
| [priority_tagger](./processors/priority_tagger.md) | Tags resources with priority levels based on rules | Implemented |
| [adaptive_topk](./processors/adaptive_topk.md) | Dynamically adjusts k parameter based on coverage score | Implemented |
| [adaptive_pid](./processors/adaptive_pid.md) | Generates configuration patches using PID control | Implemented |
| [cardinality_guardian](./processors/cardinality_guardian.md) | Controls metrics cardinality | Planned |
| [reservoir_sampler](./processors/reservoir_sampler.md) | Statistical sampling with adjustable reservoir sizes | Planned |
| [others_rollup](./processors/others_rollup.md) | Aggregates non-priority processes | Planned |

All processors implement the `UpdateableProcessor` interface which allows them to be dynamically reconfigured at runtime.

### Extensions

Extensions provide additional functionality outside the data path:

| Extension | Description | Status |
|-----------|-------------|--------|
| [pic_control_ext](./extensions/pic_control_ext.md) | Central governance layer for config changes | Implemented |

### Connectors

Connectors bridge between components:

| Connector | Description | Status |
|-----------|-------------|--------|
| [pic_connector](./connectors/pic_connector.md) | Connects adaptive_pid to pic_control_ext | Implemented |

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
│  Exporters    │          │ pic_connector │
└───────────────┘          └───────┬───────┘
                                   │
                                   ▼
                           ┌───────────────┐
                           │ pic_control   │
                           │   extension   │
                           └───────┬───────┘
                                   │
                                   ▼
                           ┌───────────────┐
                           │ Configuration │
                           │    Patches    │
                           └───────────────┘
```

## Configuration

Components are configured through two primary configuration files:

1. `config.yaml`: Standard OpenTelemetry Collector configuration
2. `policy.yaml`: Self-adaptive behavior configuration

See the [Configuration Reference](../configuration-reference.md) for details.
