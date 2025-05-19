# SA-OMF Examples

This directory contains example configurations, use cases, and implementations for the Self-Aware OpenTelemetry Metrics Fabric.

## Example Categories

### [Getting Started](./getting-started/)

Basic examples to help you get up and running with SA-OMF:
- Simple collector configuration
- Basic policy setup
- Hello world implementation

### [Processors](./processors/)

Examples demonstrating the use of specific processors:
- Priority tagger configuration
- Adaptive TopK setup
- PID controller examples

### [Integrations](./integrations/)

Example integrations with other systems:
- Prometheus setup
- Grafana dashboards
- Kubernetes deployment

### [Custom Adapters](./custom-adapters/)

Examples of extending SA-OMF with custom components:
- Custom processor implementation
- Custom policy implementation
- Custom control loop

## Running Examples

Each example directory contains its own README with specific instructions. In general:

1. Navigate to the example directory
2. Review the README.md file
3. Follow the step-by-step instructions

Most examples can be run with:

```bash
cd examples/[example-dir]
./run.sh
```

## Example Structure

Each example follows a consistent structure:

- **README.md**: Explanation and instructions
- **config.yaml**: OpenTelemetry Collector configuration
- **policy.yaml**: Control policy configuration
- **run.sh**: Script to run the example
- **cleanup.sh**: Script to clean up after running

## Contributing Examples

New examples are welcome! Please follow these guidelines:

1. Create a directory in the appropriate category
2. Include all necessary configuration files
3. Write a clear README with explanations
4. Include run and cleanup scripts
5. Ensure the example is self-contained