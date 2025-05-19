# SA-OMF Documentation

This directory contains all documentation for the Self-Aware OpenTelemetry Metrics Fabric project.

## Documentation Structure

### Architecture
- **[architecture/](./architecture/README.md)**: System architecture documentation
  - **[adr/](./architecture/adr/)**: Architecture Decision Records
  - **README.md**: Architecture overview

### Components
- **[components/](./components/)**: Documentation for SA-OMF components
  - **[processors/](./components/processors/)**: Metric processor documentation
  - **[extensions/](./components/extensions/)**: Extension documentation
  - **[connectors/](./components/connectors/)**: Connector documentation
  - **[pid/](./components/pid/)**: PID controller documentation

### Concepts
- **[concepts/](./concepts/)**: Core concepts and design principles
  - Adaptive processing
  - Self-regulating systems
  - PID control theory
  - Metric processing patterns

### Tutorials
- **[tutorials/](./tutorials/)**: Step-by-step guides
  - **[getting-started/](./tutorials/getting-started/)**: First-time setup guides
  - Implementing custom processors
  - Extending the system

### Operations
- **[operations/](./operations/)**: Operational documentation
  - **[deployment/](./operations/deployment/)**: Deployment guides
  - **[monitoring/](./operations/monitoring/)**: Monitoring and observability
  - Production best practices
  - Troubleshooting

### Developer Resources
- **[quickstarts/](./quickstarts/)**: Quick start guides for different personas
- **[testing/](./testing/)**: Testing framework documentation
- **[agents/](./agents/)**: Claude Code agent configurations

## Key Resources

- [Architecture Overview](./architecture/README.md)
- [Architecture Decision Records](./architecture/adr/)
- [Testing Framework](./testing/validation-framework.md)
- [Agent Configuration](./agents/AGENTS.md)
- [Implementation Quickstart](./quickstarts/implementer.md)
- [PID Controller Documentation](./components/pid/pid_integral_controls.md)

## Documentation Conventions

1. **File Naming**: Use lowercase with underscores (snake_case) for all documentation files
2. **Headers**: Use ATX-style headers (`#` for top level, not underlines)
3. **Code Examples**: Include language specifiers with code blocks (```go, ```yaml, etc.)
4. **Images**: Store images in an `images/` directory adjacent to markdown files
5. **Links**: Use relative links to other documentation files
6. **Tables**: Use standard markdown tables for tabular data