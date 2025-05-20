# SA-OMF Extensions

This directory contains documentation for the extensions implemented in SA-OMF.

## Available Extensions

| Extension | Description | Status |
|-----------|-------------|--------|
| [pic_control_ext](./pic_control_ext.md) | Central governance layer for configuration changes | Stable |

## Extension Concepts

Extensions in OpenTelemetry provide functionality that is not directly tied to the data processing pipeline. In SA-OMF, extensions play a crucial role in the control loop that enables self-adaptation.

### Extension Types

SA-OMF uses extensions for:
- Configuration management
- Policy enforcement
- Safety monitoring
- Control flow coordination

## Common Extension Structure

All extensions follow a common implementation pattern:

1. **Config**: Defines the extension's configuration parameters
2. **Factory**: Creates and registers the extension in the collector
3. **Extension**: Implements the actual extension logic

## Implementing New Extensions

To implement a new extension:

1. Create a new directory in `internal/extension/{extension_name}`
2. Implement the required files (config.go, factory.go, extension.go)
3. Add tests in `test/extensions/{extension_name}`
4. Register your extension in `cmd/sa-omf-otelcol/main.go`
5. Document in this directory

For setup details see [internal/extension/pic_control_ext/README.md](../../internal/extension/pic_control_ext/README.md).
