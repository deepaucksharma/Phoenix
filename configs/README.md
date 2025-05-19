# SA-OMF Configuration

This directory contains all configuration files for the Self-Aware OpenTelemetry Metrics Fabric.

## Directory Structure

- **default/**: Default configuration files used for standard deployments
  - **config.yaml**: Main collector configuration
  - **policy.yaml**: Control policy configuration
- **examples/**: Example configurations for different scenarios
  - **config.yaml**: Example collector configuration
  - **policy.yaml**: Example policy configuration

## Configuration File Types

### config.yaml

The main configuration file for the OpenTelemetry Collector. It defines:
- Receivers for data ingestion
- Processors for data transformation
- Exporters for data output
- Service pipelines that connect components

### policy.yaml

The policy configuration governs the adaptive behavior of the system:
- KPI definitions and target values
- PID controller parameters
- Safety thresholds and guard rails
- Adaptive processor configurations

## Using Configurations

To run the collector with a specific configuration:

```bash
# Default configuration
make run

# Custom configuration
./bin/sa-omf-otelcol --config=path/to/your/config.yaml
```

## Creating Custom Configurations

We recommend starting with the provided examples and modifying them to fit your specific needs. Ensure that all required components are properly defined and that the policy configuration is valid by running validation:

```bash
./hack/validate_policy_schema.sh
```