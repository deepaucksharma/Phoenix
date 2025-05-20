# Architecture Documentation

This directory contains architectural documentation for the Phoenix project (Self-Aware OpenTelemetry Metrics Fabric).

## Contents

- **adr/** - Architecture Decision Records
  - Contains formal records of significant architectural decisions made during the project
  - Each ADR explains the context, decision, and consequences of a key architectural choice
- **[dual-pipeline-architecture.md](dual-pipeline-architecture.md)** - Detailed explanation of the dual-pipeline architecture
- **[CURRENT_STATE.md](CURRENT_STATE.md)** - Current architectural state and implementation details

## Current Architecture Overview

Phoenix separates metric collection from control logic using two pipelines:

1. **Data Pipeline**
   - Collects metrics from OpenTelemetry receivers (for example `hostmetrics`)
   - Processes data through a single `metric_pipeline` processor for resource filtering and metric transformation
   - Exports processed metrics to configured destinations
   - Every processor emits self-metrics about its own operation

2. **Control Pipeline**
   - Consumes self-metrics from the data pipeline
   - Uses the `adaptive_pid` processor to compute configuration patches
   - Applies patches directly to the `metric_pipeline` processor via the `UpdateableProcessor` interface

## Architectural Principles

1. **Separation of Concerns** – Data processing and control logic run in separate pipelines
2. **Self-Adaptation** – The system automatically adjusts to changing conditions
3. **Feedback Control** – PID controllers provide stable parameter adjustments
4. **Safety Limits** – All adaptive behaviour is constrained by configurable limits
5. **Observable Decisions** – All adaptation decisions are exposed as metrics

## Component Relationships

```
                ┌──────────────────────────────┐
                │  Host / App Metrics (OTLP)   │
                └──────────────┬───────────────┘
                               │
               ┌───────────────▼───────────────┐
               │    Phoenix Data Pipeline      │
               │       metric_pipeline         │
               └───────────────┬───────────────┘
                               │ Self-Metrics
                               ▼
               ┌──────────────────────────────┐
               │    Control Pipeline          │
               │     adaptive_pid             │
               └───────────────┬──────────────┘
                               │ ConfigPatch
                               ▼
               ┌──────────────────────────────┐
               │ metric_pipeline.OnConfigPatch│
               └──────────────────────────────┘
```

## Safety Mechanisms

Phoenix includes several built-in safety mechanisms:

1. **PID Controller Safeguards**
   - Anti-windup protection prevents integral term saturation
   - Derivative filtering reduces noise sensitivity
   - Circuit breakers detect and mitigate oscillation
   - Output clamping ensures parameters stay within bounds

2. **System-Level Safeguards**
   - Resource usage monitoring (CPU, memory)
   - Automatic safe mode activation under high resource pressure
   - Graduated adaptation response for smoother transitions
   - Rate limiting prevents too-frequent parameter changes

## Key Architecture Decisions

The core architectural decisions are documented in Architecture Decision Records (ADRs):

1. [ADR-001: Dual-Pipeline Architecture](adr/001-dual-pipeline-architecture.md) – Establishes the separation between data and control pipelines
2. [ADR-002: Self-Regulating PID Control](adr/20250519-use-self-regulating-pid-control-for-adaptive-processing.md) – Details the decision to use PID controllers for adaptation

## Implementation Notes

1. **UpdateableProcessor Interface** – allows processors to receive configuration updates:
   ```go
   type UpdateableProcessor interface {
       OnConfigPatch(ctx context.Context, patch *ConfigPatch) error
       GetConfigStatus(ctx context.Context) (any, error)
   }
   ```

2. **Configuration Flow**
   - Configuration originates in `policy.yaml`
   - The `adaptive_pid` processor computes adjustments based on self-metrics
   - Patches are applied directly to the `metric_pipeline` processor

3. **Metrics Flow**
   - Processors emit self-metrics with the `aemf_` prefix
   - The control pipeline monitors these metrics to calculate KPIs
   - The system also exports standard OTel metrics with the `otelcol_` prefix

For detailed documentation on specific components, see the [Components Documentation](../components/README.md).
