# Phoenix-vNext Documentation

This directory contains comprehensive documentation for the Phoenix-vNext OpenTelemetry-based metrics collection and processing system.

## Documentation Index

### Getting Started
- **[Project README](../README.md)** - Quick start guide and project overview
- **[Architecture Overview](ARCHITECTURE.md)** - System design and component overview
- **[Deployment Guide](DEPLOYMENT.md)** - Production deployment and operations

### Development
- **[Development Guide](DEVELOPMENT.md)** - Local development setup and workflows
- **[API Documentation](API.md)** - Complete API reference for all endpoints
- **[Troubleshooting Guide](TROUBLESHOOTING.md)** - Common issues and solutions

## Quick Navigation

### For Operators
1. [Deployment Guide](DEPLOYMENT.md) - Production setup
2. [Troubleshooting Guide](TROUBLESHOOTING.md) - Problem resolution
3. [API Documentation](API.md) - Monitoring endpoints

### For Developers
1. [Development Guide](DEVELOPMENT.md) - Development environment
2. [Architecture Overview](ARCHITECTURE.md) - System understanding
3. [API Documentation](API.md) - Integration reference

### For Users
1. [Project README](../README.md) - Quick start
2. [Architecture Overview](ARCHITECTURE.md) - System concepts
3. [Troubleshooting Guide](TROUBLESHOOTING.md) - Self-service support

## Document Summaries

### [Architecture Overview](ARCHITECTURE.md)
Comprehensive system design documentation covering:
- High-level component architecture
- 3-pipeline processing system
- Control system design
- Data flow and processing patterns
- Performance and scalability considerations

### [Deployment Guide](DEPLOYMENT.md)
Complete deployment and operations guide including:
- Environment setup and prerequisites
- Docker Compose and Kubernetes deployment
- Production configuration best practices
- Monitoring and alerting setup
- Backup and recovery procedures

### [Development Guide](DEVELOPMENT.md)
Developer-focused documentation covering:
- Local development environment setup
- Code structure and conventions
- Testing and validation workflows
- Contributing guidelines
- Debugging and profiling techniques

### [API Documentation](API.md)
Complete API reference including:
- Data ingestion endpoints (OTLP)
- Metrics export endpoints
- Health check and debugging APIs
- Prometheus query examples
- SDK integration examples

### [Troubleshooting Guide](TROUBLESHOOTING.md)
Comprehensive troubleshooting reference covering:
- Common startup and configuration issues
- Performance and resource problems
- Control system debugging
- Network connectivity issues
- Emergency recovery procedures

## Additional Resources

### External Documentation
- [OpenTelemetry Collector Documentation](https://opentelemetry.io/docs/collector/)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)

### Community
- [OpenTelemetry Community](https://opentelemetry.io/community/)
- [CNCF Slack - #otel-collector](https://cloud-native.slack.com/channels/otel-collector)
- [Prometheus Community](https://prometheus.io/community/)

## Contributing to Documentation

### Documentation Standards
- Use clear, concise language
- Include practical examples
- Maintain consistent formatting
- Update examples when code changes
- Cross-reference related sections

### File Organization
```
docs/
├── README.md              # This index file
├── ARCHITECTURE.md        # System design documentation
├── DEPLOYMENT.md          # Operations and deployment guide
├── DEVELOPMENT.md         # Developer guide
├── API.md                 # API reference
└── TROUBLESHOOTING.md     # Problem resolution guide
```

### Updating Documentation
1. Edit the relevant `.md` file
2. Follow existing formatting patterns
3. Test any code examples provided
4. Update cross-references if needed
5. Submit pull request with documentation changes

## Version Information

This documentation is current as of:
- Phoenix-vNext: v1.0
- OpenTelemetry Collector: v0.103.1
- Prometheus: v2.53.0
- Grafana: v11.1.0

Last updated: 2024-01-15