# Phoenix Architecture

This document describes the current architecture of Phoenix (SA-OMF), explaining key components and their interactions.

## Current Architecture Overview

Phoenix features a streamlined architecture with self-adaptive processors:

### Data Pipeline

- Collects metrics from hostmetrics receiver
- Processes through various adaptive processors
- Exports metrics to configured destinations

### Self-Adaptive Components

- Each processor implements internal self-adaptation
- PID controllers are embedded within processors
- Each processor monitors its own metrics and adjusts parameters

## Core Components

### Adaptive Processors

Processors that dynamically adjust their behavior:

| Processor | Self-Adaptation Strategy |
|-----------|--------------------------|
| `adaptive_topk` | Dynamically adjusts k value based on coverage scores using a Space-Saving algorithm |
| `others_rollup` | Adapts aggregation thresholds and policies based on cardinality metrics |
| `priority_tagger` | Fixed configuration but provides basis for other adaptive components |
| `adaptive_pid` | Monitors KPIs and provides insights into system performance |
| `reservoir_sampler` | Provides statistical sampling with adjustable rates |

### PID Controllers

Key implementation features:

- Proportional-Integral-Derivative control loops
- Anti-windup protection to prevent integral term saturation
- Low-pass filtering for the derivative term to reduce noise sensitivity
- Oscillation detection with circuit breaker capability
- Thread-safety for concurrent access
- Comprehensive metrics and logging

### Configuration System

The system uses two core configuration files:

1. **config.yaml**: Standard OpenTelemetry pipeline configuration
   - Defines receivers, processors, exporters, and pipeline connections
   - Configures basic processor parameters

2. **policy.yaml**: Defines adaptive behavior parameters
   - PID controller parameters
   - Target KPIs
   - Adaptation thresholds and limits
   - Now processed directly by individual processors

## Adaptation Mechanisms

### PID Control

**P**roportional, **I**ntegral, **D**erivative (PID) control is the primary adaptation mechanism:

1. **Define a target value** for a key metric (e.g., coverage score = 0.95)
2. **Measure the current value** of that metric
3. **Calculate the error** (difference between target and measured value)
4. **Apply the PID formula**:
   - **P term**: Reacts proportionally to the current error
   - **I term**: Accounts for accumulated error over time
   - **D term**: Considers the rate of change of error
5. **Adjust parameters** based on the controller output

### Safety Mechanisms

To prevent unstable behavior, Phoenix implements several safety features:

1. **Bounded outputs**: All parameter adjustments have min/max limits
2. **Hysteresis**: Small errors are ignored to prevent oscillation
3. **Anti-windup protection**: Prevents integral term from growing too large
4. **Oscillation detection**: Circuit breakers that temporarily disable adaptation when oscillation is detected
5. **Rate limiting**: Prevents too-frequent adaptation

### Bayesian Optimization

For more complex parameter spaces, Phoenix also supports Bayesian optimization:

- Uses Gaussian processes to model parameter space
- Balances exploration and exploitation
- Suitable for multi-dimensional parameter optimization

## Key Architectural Principles

1. **Self-Adaptation**: Components automatically adjust to changing conditions
2. **Feedback Control**: PID controllers provide stable parameter adjustments
3. **Safety Limits**: All adaptive behavior is constrained by configurable limits
4. **Observable Decisions**: All adaptation decisions are exposed as metrics

## Architecture Decisions

For details on key architectural decisions, see the [Architecture Decision Records](architecture/adr/README.md).