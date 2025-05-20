# Phoenix: Current Architecture State

This document provides clarity on the current architecture and implementation state of Phoenix (SA-OMF), explaining recent architectural changes and their implications.

## Current Architecture Overview

Phoenix has evolved through several architectural changes aimed at simplifying its design while maintaining core capabilities. This document reflects the current state as of the latest main branch.

### Original vs. Current Architecture

#### Original Design (Historical Reference)
The original Phoenix architecture implemented a full closed-loop adaptive system with these components:

1. **Data Pipeline**: Standard processing pipeline for metrics
2. **Control Pipeline**: Monitored KPIs and generated configuration changes
3. **Adaptive Mechanism**: 
   - `adaptive_pid` processor generated ConfigPatch objects
   - `pic_connector` exporter transmitted patches 
   - `pic_control_ext` extension received and applied configuration changes to processors
   - `UpdateableProcessor` interface defined how processors could be reconfigured

#### Current Implementation
The system has been streamlined to focus on the most valuable components, with a simpler architecture:

1. **Data Pipeline**: Maintained as the primary metrics processing pipeline
2. **Simplified Adaptive Components**:
   - Processors now implement self-adaptation internally
   - Direct feedback loops within processors replace the complex control pipeline
   - Each adaptive processor monitors its own KPIs and adjusts itself accordingly

This change has removed the following components that were described in older documentation:
- `pic_control_ext` extension
- `pic_connector` exporter
- `ConfigPatch` mechanism
- The formal `UpdateableProcessor` interface

### Core Components

#### 1. Adaptive Processors

The following processors now implement internal self-adaptation:

| Processor | Self-Adaptation Strategy |
|-----------|--------------------------|
| `adaptive_topk` | Dynamically adjusts k value based on coverage scores using a Space-Saving algorithm |
| `others_rollup` | Adapts aggregation thresholds and policies based on cardinality metrics |
| `priority_tagger` | Fixed configuration but provides basis for other adaptive components |
| `adaptive_pid` | Monitors KPIs and provides insights into system performance |
| `reservoir_sampler` | Provides statistical sampling with adjustable rates |

#### 2. PID Controllers

PID controllers are still a core component, but now embedded within processors:

- Each adaptive processor has its own PID controller configuration
- Controllers are initialized with parameters from policy.yaml
- Sophisticated features maintained:
  - Anti-windup protection
  - Derivative filtering
  - Circuit breakers for oscillation detection
  - Min/max bounds with clamping

#### 3. Configuration

The system uses two core configuration files:

1. **config.yaml**: Standard OpenTelemetry pipeline configuration
   - **Important Note**: Older versions may still reference removed components (`pic_control`, `pic_connector`)
   - Only the configured processors are functional in the current architecture

2. **policy.yaml**: Defines adaptive behavior parameters
   - PID controller parameters
   - Target KPIs
   - Adaptation thresholds and limits
   - Now processed directly by individual processors

## Implications and Best Practices

### Configurations to Use

Use the configuration files found in `configs/default` directory as they have been updated to match the current architecture. Specifically:

- Configuration pipelines should only include the current processors:
  ```yaml
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [priority_tagger, adaptive_topk, others_rollup, adaptive_pid]
      exporters: [logging, prometheusremotewrite]
  ```

- The control pipeline from older configurations should be removed or ignored.

### Monitoring Adaptation

Since adaptation is now internal to processors:

1. Adaptation behavior can be monitored through:
   - Standard OpenTelemetry metrics emitted by processors
   - Processor logs (configure with appropriate verbosity level)
   
2. Each adaptive processor exposes metrics about its internal state and adaptation decisions.

### Developer Integration Points

Developers wanting to implement new adaptive components should:

1. Implement the internal adaptive logic directly in your processor
2. Use the common utilities in the `internal/control/pid` package
3. Expose monitoring metrics following the patterns in existing processors

## Future Directions

The Phoenix project continues to evolve with these planned improvements:

1. **Enhanced Self-diagnostics**: Better visibility into adaptive behavior
2. **Expanded Adaptive Algorithms**: Additional adaptation strategies beyond PID
3. **Inter-processor Coordination**: Future versions may reintroduce controlled coordination between processors
4. **Richer Policy Controls**: More sophisticated policy controls for adaptive behavior

## Conclusion

While Phoenix has moved away from the complex external control pipeline architecture, it maintains its core value proposition of adaptive processing. The simplified architecture improves reliability, reduces complexity, and makes the system more maintainable while preserving the essential adaptive capabilities that make Phoenix unique.

Refer to other documentation for specific component details and configurations.