# Phoenix Documentation

Welcome to the Phoenix documentation! This guide will help you understand, deploy, and contribute to the Phoenix adaptive cardinality optimization system.

## 📚 Documentation Index

### Getting Started
- **[Project Overview](../README.md)** - Main README with quick start guide
- **[Architecture Overview](ARCHITECTURE.md)** - System design and components
- **[Monorepo Structure](MONOREPO_STRUCTURE.md)** - Project organization

### Operations
- **[Troubleshooting Guide](TROUBLESHOOTING.md)** - Common issues and solutions
- **[Pipeline Analysis](PIPELINE_ANALYSIS.md)** - Deep dive into pipeline behavior
- **[Migration Guide](MIGRATION_GUIDE.md)** - Migrating to the monorepo structure

### Development
- **[CLAUDE.md](../CLAUDE.md)** - AI assistant instructions for development

## 🗺️ Quick Navigation

### For New Users
1. Start with the [Project Overview](../README.md)
2. Understand the [Architecture](ARCHITECTURE.md)
3. Review the [Monorepo Structure](MONOREPO_STRUCTURE.md)

### For Operators
1. [Troubleshooting Guide](TROUBLESHOOTING.md) for issue resolution
2. [Pipeline Analysis](PIPELINE_ANALYSIS.md) for performance tuning
3. Monitor dashboards at http://localhost:3000

### For Developers
1. [Monorepo Structure](MONOREPO_STRUCTURE.md) for code organization
2. [Architecture Overview](ARCHITECTURE.md) for system understanding
3. [CLAUDE.md](../CLAUDE.md) for AI-assisted development

## 📖 Document Summaries

### [Architecture Overview](ARCHITECTURE.md)
Comprehensive system design covering:
- 3-pipeline processing system (Full, Optimized, Experimental)
- Adaptive control system with PID-like behavior
- Component interactions and data flow
- Performance considerations

### [Monorepo Structure](MONOREPO_STRUCTURE.md)
Project organization guide covering:
- Directory structure and conventions
- Package management with workspaces
- Service boundaries and interfaces
- Build system with Turborepo

### [Pipeline Analysis](PIPELINE_ANALYSIS.md)
Detailed analysis of:
- Pipeline-specific optimizations
- Cardinality reduction techniques
- Performance benchmarks
- Configuration tuning

### [Troubleshooting Guide](TROUBLESHOOTING.md)
Common issues and solutions:
- Container startup problems
- Memory and resource issues
- Network connectivity
- Control system debugging

### [Migration Guide](MIGRATION_GUIDE.md)
Step-by-step migration:
- From monolithic to modular structure
- Configuration changes
- API updates
- Rollback procedures

## 🛠️ Quick Reference

### Common Commands
```bash
make help          # Show all available commands
make deploy-dev    # Deploy development environment
make health        # Check service health
make monitor       # Open monitoring dashboards
make logs          # View all service logs
```

### Service Endpoints
- **OTLP Ingestion**: `localhost:4318`
- **Prometheus**: `http://localhost:9090`
- **Grafana**: `http://localhost:3000` (admin/admin)
- **Control API**: `http://localhost:8080/api/v1`

### Configuration Files
- **Environment**: `.env` (copy from `config/environments/dev/.env`)
- **Collector**: `services/collector/configs/main.yaml`
- **Control**: `config/defaults/control/optimization_mode.yaml`
- **Prometheus**: `monitoring/prometheus/prometheus.yaml`

## 🔄 Documentation Updates

To update documentation:
1. Edit the relevant `.md` file in the `docs/` directory
2. Follow existing formatting patterns
3. Test any code examples
4. Update cross-references if needed
5. Submit a pull request

## 📞 Getting Help

- **GitHub Issues**: Report bugs or request features
- **Discussions**: Ask questions and share ideas
- **Slack**: Join #phoenix channel (if available)

## 🏷️ Version Information

Current versions:
- Phoenix: v1.0.0
- OpenTelemetry Collector: v0.91.0
- Prometheus: v2.48.0
- Grafana: v10.2.2

Last updated: January 2025
