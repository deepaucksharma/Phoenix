# Phoenix Project Documentation

Welcome to the documentation for the **Phoenix** project (SA-OMF: Self-Aware OpenTelemetry Metrics Fabric), an advanced metrics collection and processing system built on top of OpenTelemetry that features adaptive processing through PID control loops.

## Getting Started

- [Project Overview](../README.md) - Introduction to Phoenix
- [Quick Start](./quickstarts/implementer.md) - Get up and running quickly
- [Development Guide](./development-guide.md) - Comprehensive guide for developers
- [Offline Building](./offline-build.md) - Building in network-restricted environments
- [CI/CD Workflows](./ci-cd.md) - Overview of the CI/CD process

## Core Documentation

### System Architecture

- [Architecture Overview](./architecture/README.md) - High-level system architecture
- [Dual Pipeline Architecture](./architecture/adr/001-dual-pipeline-architecture.md) - Core architectural decision
- [Self-Regulating PID Control](./architecture/adr/20250519-use-self-regulating-pid-control-for-adaptive-processing.md) - PID control approach
- [Implementation Plan](../implementation-plan.md) - Planned implementation timeline

### Component Documentation

- [Components Overview](./components/README.md) - All system components
- [Processors](./components/processors/README.md) - Metric processor documentation
- [Extensions](./components/extensions/README.md) - Extension documentation
- [Connectors](./components/connectors/README.md) - Connector documentation
- [PID Controllers](./components/pid/pid_integral_controls.md) - PID controller documentation

### Operational Guides

- [Deployment](./operations/deployment.md) - Deployment configurations and guides
- [Configuration](./configuration.md) - Configuration file reference
- [Operation](./operations/operation.md) - Day-to-day operation guide
- [Monitoring](./operations/monitoring.md) - Monitoring and observability

### Development

- [Contributing Guide](./contributing.md) - How to contribute to Phoenix
- [Testing Framework](./testing/validation-framework.md) - Framework for testing
- [Development Workflow](./development-guide.md) - Development workflow and practices
- [Agent Framework](./agents/README.md) - Claude Code agent configurations

### Audit Framework

- [Audit Overview](../audit/README.md) - Overview of the audit framework
- [Audit Agenda](../audit/AUDIT_AGENDA.md) - Structured audit process
- [Audit Metrics](../audit/AUDIT_METRICS.md) - Metrics for measuring audit effectiveness
- [Component Audit Results](../audit/summary.md) - Summary of audit findings

## Reference

- [Changelog](../CHANGELOG.md) - Version history and changes
- [Configuration Reference](./configuration-reference.md) - Complete configuration options
- [API Reference](./api-reference.md) - API details

## Concepts

- [Adaptive Processing](./concepts/adaptive-processing.md) - Adaptive metric processing
- [PID Control Theory](./concepts/pid-control.md) - PID control loop basics
- [Self-Regulating Systems](./concepts/self-regulating-systems.md) - Self-regulation concepts
- [Metric Processing Patterns](./concepts/metric-processing.md) - Common processing patterns