# SA-OMF Core Concepts

This directory contains documentation for the core concepts and principles behind the Self-Aware OpenTelemetry Metrics Fabric.

## Key Concepts

### Dual Pipeline Architecture

SA-OMF is built around a dual pipeline architecture:
- **Data Pipeline**: Processes metrics for export to backends
- **Control Pipeline**: Monitors system behavior and adjusts configuration

### Self-Adaptation

The system can dynamically adjust its own parameters in response to changing conditions:
- Metrics are analyzed to determine system performance
- PID controllers calculate optimal parameter adjustments
- Configuration patches are applied to processors

### PID Control

PID (Proportional-Integral-Derivative) control is used for stable, responsive adaptation:
- **Proportional**: Responds to the current error
- **Integral**: Addresses accumulated error over time
- **Derivative**: Anticipates future error based on rate of change

### UpdateableProcessor Interface

The core interface that enables dynamic reconfiguration:
- Processors implement OnConfigPatch to accept changes
- GetConfigStatus returns current configuration state
- Changes are applied safely with proper validation

### Policy-Based Governance

All adaptation is governed by policy:
- KPI definitions and target values
- Adjustment limits and safety bounds
- Control parameters and response curves

## Additional Resources

- [Dual Pipeline Architecture ADR](../architecture/adr/001-dual-pipeline-architecture.md)
- [PID Controller Documentation](../components/pid/pid_integral_controls.md)
- [System Architecture Overview](../architecture/README.md)