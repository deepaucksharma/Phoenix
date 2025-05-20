# Architecture Documentation

This directory contains architectural documentation for the Phoenix project (Self-Aware OpenTelemetry Metrics Fabric).

## Contents

- **adr/** - Architecture Decision Records
  - Contains formal records of significant architectural decisions made during the project
  - Each ADR explains the context, decision, and consequences of a key architectural choice
- **[dual-pipeline-architecture.md](dual-pipeline-architecture.md)** - Detailed explanation of the dual-pipeline architecture
- **[CURRENT_STATE.md](CURRENT_STATE.md)** - Current architectural state and implementation details

## Current Architecture Overview

Phoenix implements a dual-pipeline architecture that separates data processing from control logic:

1. **Data Pipeline**:
   - Collects metrics from standard OpenTelemetry receivers (e.g., hostmetrics)
   - Processes metrics through the unified `metric_pipeline` processor for resource filtering and transformation
   - Exports processed metrics to configured destinations
   - Each processor emits self-metrics about its own operation

2. **Control Pipeline**:
   - Monitors self-metrics from the data pipeline
   - Runs PID controllers to calculate needed adjustments
   - Generates configuration patches for the data pipeline
   - Uses the `adaptive_pid` processor as its core component
   - Applies configuration patches directly through the config manager

3. **Integration Components**:
   - `UpdateableProcessor` interface: Allows processors to receive dynamic configuration

## Architectural Principles

1. **Separation of Concerns**: Clean separation between data processing and control logic
2. **Self-Adaptation**: System automatically adjusts to changing conditions
3. **Feedback Control**: PID controllers provide stable parameter adjustments
4. **Safety Limits**: All adaptive behavior is constrained by configurable limits
5. **Observable Decisions**: All adaptation decisions are exposed as metrics

## Detailed Component Relationships

```
                ┌──────────────────────────────────────────┐
                │        Host / App  Metrics (OTLP)        │
                └──────┬────────────────────────────────────┘
                       │
       ┌───────────────▼───────────────┐
       │  Phoenix Data  Pipeline        │
       │ (Adaptive Processors:         │
       │  - PriorityTagger             │
       │  - AdaptiveTopK               │
       │  - OthersRollup)              │
       └───────────────┬───────────────┘
                       │ Self-Metrics
                       ▼
       ┌──────────────────────────────────────────┐
       │        Phoenix Control Pipeline          │
       │  (Processors:                            │
       │   - AdaptivePID: PID → Bayesian Logic)   │
       │  (Outputs: ConfigPatch Objects)          │
       └──────────────────────┬───────────────────┘
                              │ ConfigPatch
                              ▼
                 ┌───────────────────────────┐
                 │ PIC Control Extension      │
                 └───────────────┬───────────┘
                                 │ OnConfigPatch
                                 ▼
                 ┌───────────────────────────┐
                 │ UpdateableProcessor.apply  │
                 └───────────────────────────┘
```

## Safety Mechanisms

Phoenix includes several built-in safety mechanisms:

1. **PID Controller Safeguards**:
   - Anti-windup protection prevents integral term saturation
   - Derivative filtering reduces noise sensitivity
   - Circuit breakers detect and mitigate oscillation
   - Output clamping ensures parameters stay within bounds

2. **System-Level Safeguards**:
   - Resource usage monitoring (CPU, memory)
   - Automatic safe mode activation under high resource pressure
   - Graduated adaptation response for smoother transitions
   - Rate limiting prevents too-frequent parameter changes

## Key Architecture Decisions

The core architectural decisions are documented in Architecture Decision Records (ADRs):

1. [ADR-001: Dual-Pipeline Architecture](adr/001-dual-pipeline-architecture.md) - Establishes the separation between data and control pipelines
2. [ADR-002: Self-Regulating PID Control](adr/20250519-use-self-regulating-pid-control-for-adaptive-processing.md) - Details the decision to use PID controllers for adaptation

## Implementation Notes

Current implementation considerations:

1. **UpdateableProcessor Interface**: The core interface that allows processors to receive configuration updates:
   ```go
   type UpdateableProcessor interface {
       OnConfigPatch(ctx context.Context, patch *ConfigPatch) error
       GetConfigStatus(ctx context.Context) (any, error)
   }
   ```

2. **Configuration Flow**:
   - Configuration originates in the `policy.yaml` file
   - PID controllers in `adaptive_pid` processor calculate adjustments
   - Changes flow through the system as `ConfigPatch` objects
   - The config manager validates and applies changes to processors

3. **Metrics Flow**:
   - Processors emit self-metrics with the `aemf_` prefix
   - Control pipeline monitors these metrics to calculate KPIs
   - Metrics become inputs to PID controllers
   - System also exports standard OTel metrics with the `otelcol_` prefix

For detailed documentation on specific components, please see the [Components Documentation](../components/README.md).