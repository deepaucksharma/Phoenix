# SA-OMF Connectors

This directory contains documentation for the connectors implemented in SA-OMF.

## Available Connectors

| Connector | Description | Status |
|-----------|-------------|--------|
| [pic_connector](./pic_connector.md) | Connects the PID decider to the PIC control extension | Stable |

## Connector Concepts

Connectors in SA-OMF function as specialized exporters that bridge different parts of the control loop. They are essential for the self-adaptive capabilities of the system.

### Control Loop Integration

Connectors are typically used to:
- Forward configuration patch metrics from processors to extensions
- Transform metrics into control actions
- Bridge the data and control pipelines

## Common Connector Structure

All connectors follow a common implementation pattern:

1. **Config**: Defines the connector's configuration parameters
2. **Factory**: Creates and registers the connector in the collector
3. **Exporter**: Implements the actual connector logic

## Implementing New Connectors

To implement a new connector:

1. Create a new directory in `internal/connector/{connector_name}`
2. Implement the required files (config.go, factory.go, exporter.go)
3. Add tests in `test/connectors/{connector_name}`
4. Register your connector in `cmd/sa-omf-otelcol/main.go`
5. Document in this directory
