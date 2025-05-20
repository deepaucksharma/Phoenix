# SA-OMF Configuration Files

This directory contains all configuration files for the Self-Aware OpenTelemetry Metrics Fabric.

## Directory Structure

- **default/**: Default configuration files used as a baseline
- **development/**: Configuration optimized for development environments
- **production/**: Configuration for production deployment
- **testing/**: Configuration for testing environments
- **examples/**: Example configurations demonstrating specific scenarios

## Environment-Specific Configuration

Each environment directory contains:

- **config.yaml**: Main OpenTelemetry Collector configuration
- **policy.yaml**: Control policy for self-adapting components

## Configuration Differences by Environment

### Development
- More verbose logging
- Lower processing intervals
- Lower metric retention
- Faster control loop adaptation

### Testing
- Configuration optimized for integration tests
- In-memory exporters
- Test-friendly component parameters
- Predictable adaptation values

### Production
- Production-ready component settings
- Optimized for stability and resource efficiency
- More conservative control parameters
- Safety limits for resource consumption

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

# Environment-specific configuration
./bin/sa-omf-otelcol --config=configs/production/config.yaml
```

## Configuration Best Practices

1. **Never modify default configurations directly** - Use them as reference only
2. **Keep environment-specific settings separate** - Use environment directories
3. **Document all custom settings** - Add comments to explain changes
4. **Validate before deployment** - Run policy validation with `scripts/validation/validate_policy_schema.sh`
5. **Version control changes** - Always commit configuration changes with descriptive messages
