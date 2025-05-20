# SA-OMF Processors

This directory contains documentation for the various processors implemented in SA-OMF.

## Available Processors

| Processor | Description | Status |
|-----------|-------------|--------|
| [priority_tagger](./priority_tagger.md) | Tags metrics with priority levels based on rules | Stable |
| [adaptive_topk](./adaptive_topk.md) | Dynamically adjusts k value for top-k filtering | Beta |
| [adaptive_pid](./adaptive_pid.md) | PID-based control loop for configuration changes | Beta |

## Processor Concepts

Processors are components that operate on metrics as they flow through the OpenTelemetry Collector pipeline. In SA-OMF, processors can have self-adapting capabilities.

### Dynamic Configuration

Some SA-OMF processors support dynamic configuration adjustments at runtime. This is a core concept for enabling self-adaptation. The mechanism for this has been refactored in the latest version to use a more standardized approach.

## Common Processor Structure

All processors follow a common implementation pattern:

1. **Config**: Defines the processor's configuration parameters
2. **Factory**: Creates and registers the processor in the collector
3. **Processor**: Implements the actual processing logic and UpdateableProcessor interface

## Implementing New Processors

To implement a new processor:

1. Create a new directory in `internal/processor/{processor_name}`
2. Implement the required files (config.go, factory.go, processor.go)
3. Add tests in `test/processors/{processor_name}`
4. Register your processor in `cmd/sa-omf-otelcol/main.go`
5. Document in this directory

See the [Processor Implementation Guide](../../tutorials/implementing_processors.md) for detailed instructions.