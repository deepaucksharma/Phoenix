# Phoenix - Adaptive Cardinality Optimization System

<div align="center">
  <img src="docs/assets/phoenix-logo.png" alt="Phoenix Logo" width="200"/>
  
  [![CI](https://github.com/deepaucksharma/Phoenix/actions/workflows/ci.yml/badge.svg)](https://github.com/deepaucksharma/Phoenix/actions)
  [![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
  [![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-Compatible-orange)](https://opentelemetry.io/)
</div>

## Overview

Phoenix is an adaptive cardinality optimization system for OpenTelemetry metrics collection and processing. It dynamically manages metric cardinality through intelligent pipeline switching and optimization profiles.

## 🏗️ Monorepo Structure

This project is organized as a monorepo with clear separation of concerns:

```
phoenix/
├── packages/          # Shared libraries
├── services/          # Microservices
├── infrastructure/    # Deployment configs
├── monitoring/        # Observability stack
├── tools/            # Dev tools
└── docs/             # Documentation
```

## 🚀 Quick Start

### Prerequisites
- Node.js >= 18.0.0
- Docker and Docker Compose
- Go 1.21+ (for generator services)

### Setup
```bash
# Clone the repository
git clone https://github.com/deepaucksharma/Phoenix.git
cd Phoenix

# Install dependencies
make install

# Setup environment
make setup-env

# Build all services
make build
make build-docker
```

### Running Phoenix
```bash
# Development mode
make deploy-dev

# Check service health
make health

# View logs
make logs

# Open monitoring dashboards
make monitor
```

## 📦 Sub-Projects

### Packages (Shared Libraries)
- **@phoenix/contracts** - API contracts and schemas
- **@phoenix/common** - Shared utilities
- **@phoenix/config** - Configuration management

### Services
- **@phoenix/collector** - Core OTEL collector with multi-pipeline processing
- **@phoenix/control-observer** - Metrics observation service
- **@phoenix/control-actuator** - Adaptive control logic
- **@phoenix/generator-synthetic** - Synthetic metrics generator
- **@phoenix/generator-complex** - Complex metrics generator
- **@phoenix/validator** - Performance validation service

### Infrastructure
- Docker Compose configurations
- Kubernetes manifests (coming soon)
- Terraform modules (coming soon)

## 🔧 Development

### Working with the Monorepo
```bash
# Run specific service in dev mode
cd services/collector && npm run dev

# Run all services in dev mode
npm run dev

# Run tests across all packages
npm test

# Lint all code
npm run lint
```

### Adding a New Service
1. Create directory: `services/your-service/`
2. Add standard structure (cmd/, internal/, api/, etc.)
3. Create package.json with workspace dependency
4. Update root package.json workspaces if needed

## 📊 Architecture

Phoenix uses a 3-pipeline architecture:
1. **Full Fidelity** - Complete metrics without optimization
2. **Optimized** - Moderate cardinality reduction
3. **Experimental TopK** - Advanced optimization

The control plane monitors metrics and dynamically switches between optimization profiles based on cardinality thresholds.

## 🔍 Monitoring

- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Health Checks**: `make health`

## 📚 Documentation

- [Architecture Overview](docs/architecture/ARCHITECTURE.md)
- [API Documentation](docs/api/README.md)
- [Development Guide](docs/guides/DEVELOPMENT.md)
- [Deployment Guide](docs/guides/DEPLOYMENT.md)

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- OpenTelemetry community for the excellent collector
- Prometheus and Grafana for monitoring capabilities
- All contributors to this project