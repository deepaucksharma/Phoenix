# Phoenix Modern Adaptive Architecture

This document provides an overview of the modern, streamlined architecture currently implemented in Phoenix. It explains how the system achieves self-adaptation through embedded adaptive processors.

## Core Architecture Principles

The current Phoenix architecture is built on several key principles:

1. **Embedded Adaptation**: Each processor implements its own adaptation mechanisms
2. **Direct Feedback**: Processors directly monitor and adjust their own parameters
3. **Self-Contained Components**: Reduced interdependencies between components
4. **Simplified Configuration**: Clearer relationship between configuration and behavior

## Architecture Evolution

Phoenix has evolved from a more complex dual-pipeline architecture to a streamlined design:

### Original Design (Historical)

The original architecture used:
- A control pipeline separate from the data pipeline
- ConfigPatch objects to communicate parameter adjustments
- A central extension (pic_control_ext) to apply changes
- Processors implementing an UpdateableProcessor interface

While powerful, this approach introduced complexity and interdependencies between components.

### Modern Design

The current architecture simplifies the design by:
- Embedding adaptation directly in processors
- Eliminating the need for a separate control pipeline
- Removing component interdependencies
- Maintaining the same adaptive capabilities with less complexity

## How Self-Adaptation Works

In the current architecture, self-adaptation works as follows:

1. **Metrics Collection**:
   - Standard OpenTelemetry collectors gather metrics
   - Processors receive metrics through the data pipeline

2. **Internal Monitoring**:
   - Each adaptive processor tracks its own performance
   - Processors compute relevant KPIs (Key Performance Indicators)
   - KPIs are compared against target values

3. **Adaptation Logic**:
   - Embedded PID controllers calculate necessary adjustments
   - Processors apply changes to their own parameters directly
   - Safety mechanisms prevent oscillation and resource issues

4. **Observation**:
   - Processors expose metrics about their adaptation decisions
   - These metrics can be collected and visualized for monitoring

## Component Types

### Adaptive Processors

These processors implement internal self-adaptation:

| Processor | Adaptation Strategy | Purpose |
|-----------|---------------------|---------|
| adaptive_topk | Adjusts k value based on coverage score | Efficiently capture important resources |
| others_rollup | Adapts aggregation based on priority | Reduce cardinality while preserving detail |
| adaptive_pid | Monitors KPIs for system health | Provide visibility into system performance |
| reservoir_sampler | Adjusts sampling rates | Balance detail and performance |

### PID Controllers

PID (Proportional-Integral-Derivative) controllers are embedded within processors. They:
1. Calculate error between actual and target values
2. Apply proportional, integral, and derivative terms
3. Produce stable adjustment signals
4. Include safety features like:
   - Anti-windup protection
   - Hysteresis bands
   - Min/max clamping
   - Oscillation detection

### Safety Mechanisms

Multiple layers of safety ensure stable adaptation:
1. **Local Safety**: Each processor implements its own guardrails
2. **Policy-Based Limits**: Global safety parameters in policy.yaml
3. **Circuit Breakers**: Detect and prevent oscillation
4. **Rate Limiting**: Prevent too-frequent adaptation

## Configuration Model

The configuration model mirrors the architecture:

1. **Standard Pipeline Configuration** (config.yaml):
   - Defines the data flow through components
   - Sets up receivers, processors, and exporters
   - Configures basic processor parameters

2. **Adaptation Policy** (policy.yaml):
   - Defines target KPIs for each adaptive component
   - Configures PID controller parameters
   - Sets safety thresholds and limits
   - Controls adaptation behavior

## Benefits of the Current Architecture

1. **Reduced Complexity**: Fewer components and interactions to understand
2. **Better Encapsulation**: Processors fully own their adaptation behavior
3. **Improved Testability**: Components can be tested in isolation
4. **Enhanced Reliability**: Fewer moving parts means fewer potential failures
5. **Easier Configuration**: More intuitive relationship between config and behavior

## Example: Adaptive Top-K Flow

To illustrate the architecture, let's follow how adaptive_topk functions:

1. **Initialization**:
   - Processor initialized with parameters from config.yaml
   - PID controller initialized with parameters from policy.yaml

2. **Processing**:
   - Metrics flow into the processor
   - Processor applies current k value to filter resources
   - Space-Saving algorithm tracks top resources

3. **Adaptation**:
   - At each adaptation interval, coverage score is calculated
   - PID controller compares score to target and calculates adjustment
   - k value is adjusted within defined bounds
   - Updated k value is applied immediately to subsequent metrics

4. **Observation**:
   - Processor emits metrics about current k value
   - Processor emits metrics about coverage score
   - PID controller emits metrics about control decisions

## Developing with the Modern Architecture

When developing new components for Phoenix:

1. **Embed Adaptation in Processors**:
   - Implement adaptation logic directly in the processor
   - Use the PID controller utilities in internal/control/pid

2. **Use Policy for Adaptation Parameters**:
   - Keep basic configuration in config.yaml
   - Put adaptation parameters in policy.yaml

3. **Expose Metrics for Observability**:
   - Emit metrics for all key performance indicators
   - Emit metrics for all adaptation decisions

4. **Implement Safety Mechanisms**:
   - Include min/max bounds on all parameters
   - Implement hysteresis to prevent oscillation
   - Add circuit breakers for unstable conditions

## Conclusion

The current Phoenix architecture delivers on the original promise of self-adaptation with a more streamlined, maintainable design. By embedding adaptation directly within processors, the system achieves the same goals with reduced complexity and improved reliability.

This evolution represents a natural maturation of the system, focusing on the core value of adaptive processing while simplifying the implementation.