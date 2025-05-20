# SA-OMF Processors

This directory contains documentation for the various processors implemented in SA-OMF.

## Available Processors

| Processor | Description | Status |
|-----------|-------------|--------|
| [priority_tagger](./priority_tagger.md) | Tags metrics with priority levels based on rules | Stable |
| [adaptive_topk](./adaptive_topk.md) | Dynamically adjusts k value for top-k filtering | Stable |
| [others_rollup](./others_rollup.md) | Aggregates low-priority metrics to reduce cardinality | Stable |
| [adaptive_pid](./adaptive_pid.md) | PID-based processor for monitoring KPIs and adapting behavior | Stable |
| [cardinality_guardian](./cardinality_guardian.md) | Controls metrics cardinality based on resource usage | Planning |
| [reservoir_sampler](./reservoir_sampler.md) | Provides statistical sampling with adjustable rates | Implemented |
| [timeseries_estimator](./timeseries_estimator.md) | Estimates active time series with circuit breaker | Implemented |

## Processor Concepts

Processors are components that operate on metrics as they flow through the OpenTelemetry Collector pipeline. In SA-OMF, processors have self-adapting capabilities built-in.

### Self-Adaptation

Each adaptive processor in Phoenix:
1. Monitors its own metrics and performance
2. Contains internal PID controllers
3. Dynamically adjusts its parameters based on measurements
4. Maintains metrics for observability

This design simplifies the architecture by embedding adaptation directly within processors rather than requiring an external control loop.

## Common Processor Structure

All processors follow a common implementation pattern:

1. **Config**: Defines the processor's configuration parameters
2. **Factory**: Creates and registers the processor in the collector
3. **Processor**: Implements the actual processing logic and adaptation mechanisms

## Implementing New Processors

To implement a new processor:

1. Create a new directory in `internal/processor/{processor_name}`
2. Implement the required files (config.go, factory.go, processor.go)
3. Implement internal adaptation if needed
4. Add tests in `test/processors/{processor_name}`
5. Register your processor in `cmd/sa-omf-otelcol/main.go`
6. Document in this directory

## Adaptation Mechanisms

Phoenix processors can use several adaptation mechanisms:

1. **PID Control**: Using proportional-integral-derivative control loops
2. **Bayesian Optimization**: For complex parameter spaces
3. **Rule-based Adaptation**: For simpler threshold-based adjustments
4. **Circuit Breakers**: To prevent oscillation or instability

See the [Adaptive Processing](../../concepts/adaptive-processing.md) documentation for more details on these mechanisms.