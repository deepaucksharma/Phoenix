# Phoenix - Adaptive Cardinality Optimization System

<div align="center">
  
  [![CI](https://github.com/deepaucksharma/Phoenix/actions/workflows/ci.yml/badge.svg)](https://github.com/deepaucksharma/Phoenix/actions)
  [![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
  [![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-Compatible-orange)](https://opentelemetry.io/)
  
</div>

## 🚀 Overview

Phoenix is an adaptive cardinality optimization system for OpenTelemetry metrics collection and processing. It dynamically manages metric cardinality through intelligent pipeline switching and optimization profiles based on real-time system performance.

### Key Features
- **3-Pipeline Architecture**: Full fidelity, optimized, and experimental TopK pipelines
- **Adaptive Control**: PID-like control system for automatic optimization
- **Real-time Monitoring**: Grafana dashboards with comprehensive metrics
- **Modular Design**: Microservices architecture with clear boundaries
- **High Performance**: Handles millions of metrics with intelligent sampling

## 📋 Table of Contents

- [Quick Start](#-quick-start)
- [Architecture](#-architecture)
- [Project Structure](#-project-structure)
- [Installation](#-installation)
- [Usage](#-usage)
- [Configuration](#-configuration)
- [Monitoring](#-monitoring)
- [Development](#-development)
- [Documentation](#-documentation)
- [Contributing](#-contributing)

## 🏃 Quick Start

```bash
# Clone the repository
git clone https://github.com/deepaucksharma/Phoenix.git
cd Phoenix

# Setup environment
make setup-env

# Install dependencies
make install

# Build and deploy
make build
make deploy-dev

# Check health
make health

# View dashboards
make monitor
```

## 🏗️ Architecture

Phoenix uses a sophisticated 3-pipeline architecture to optimize metric cardinality:

```
┌─────────────────┐     ┌──────────────────────────────────────┐
│ Metric Sources  │────▶│         Phoenix Collector            │
└─────────────────┘     │  ┌────────────────────────────────┐ │
                        │  │   Full Fidelity Pipeline       │ │
                        │  ├────────────────────────────────┤ │
                        │  │   Optimized Pipeline           │ │
                        │  ├────────────────────────────────┤ │
                        │  │   Experimental TopK Pipeline   │ │
                        │  └────────────────────────────────┘ │
                        └──────────────────────────────────────┘
                                          │
                        ┌─────────────────┴─────────────────┐
                        │                                   │
                  ┌─────▼──────┐                   ┌───────▼────────┐
                  │ Prometheus  │                   │ Control Plane  │
                  └─────┬──────┘                   └───────┬────────┘
                        │                                   │
                  ┌─────▼──────┐                   ┌───────▼────────┐
                  │  Grafana    │                   │   Actuator     │
                  └─────────────┘                   └────────────────┘
```

### Core Components
- **Collector**: Multi-pipeline OTEL collector with cardinality management
- **Observer**: Monitors pipeline metrics and system KPIs
- **Actuator**: Implements adaptive control logic
- **Generators**: Synthetic and complex metric generators for testing
- **Validator**: Performance benchmarking and validation

For detailed architecture documentation, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## 📁 Project Structure

```
phoenix/
├── packages/              # Shared libraries
│   ├── contracts/        # API contracts and schemas
│   ├── common/          # Common utilities
│   └── config/          # Configuration management
├── services/             # Microservices
│   ├── collector/       # Core OTEL collector
│   ├── control-plane/   # Observer and actuator
│   ├── generators/      # Load generators
│   └── validator/       # Benchmarking service
├── infrastructure/       # Deployment configurations
│   └── docker/         # Docker compose files
├── monitoring/          # Observability stack
│   ├── prometheus/     # Metrics storage
│   └── grafana/        # Visualization
├── config/             # Environment configs
├── tools/              # Development tools
└── docs/               # Documentation
```

## 🛠️ Installation

### Prerequisites
- Docker and Docker Compose
- Node.js >= 18.0.0
- Go 1.21+ (for building services)
- Make

### Setup Steps

1. **Clone and Setup Environment**
   ```bash
   git clone https://github.com/deepaucksharma/Phoenix.git
   cd Phoenix
   make setup-env
   ```

2. **Install Dependencies**
   ```bash
   make install
   ```

3. **Build Services**
   ```bash
   make build
   make build-docker
   ```

4. **Deploy**
   ```bash
   # Development
   make deploy-dev
   
   # Production
   make deploy-prod
   ```

## 📊 Usage

### Basic Commands

```bash
# View logs
make logs

# Check service health
make health

# Open monitoring dashboards
make monitor

# Stop services
make stop

# Clean everything
make clean
```

### Service-Specific Logs

```bash
make collector-logs   # View collector logs
make observer-logs    # View observer logs
make actuator-logs    # View actuator logs
make generator-logs   # View generator logs
```

## ⚙️ Configuration

### Environment Variables

Key configuration options in `.env`:

```bash
# Control thresholds
TARGET_OPTIMIZED_PIPELINE_TS_COUNT=20000
THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS=15000
THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS=25000

# Resource limits
OTELCOL_MAIN_MEMORY_LIMIT_MIB=1024
OTELCOL_MAIN_GOMAXPROCS=2

# Control timing
ADAPTIVE_CONTROLLER_INTERVAL_SECONDS=60
ADAPTIVE_CONTROLLER_STABILITY_SECONDS=120
```

### Optimization Modes

Phoenix automatically switches between three optimization modes:
- **Conservative**: < 15,000 time series
- **Balanced**: 15,000 - 25,000 time series
- **Aggressive**: > 25,000 time series

## 📈 Monitoring

### Access Points
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Collector Metrics**: http://localhost:8888/metrics
- **Control API**: http://localhost:8080/api/v1

### Available Dashboards
1. **Phoenix Adaptive Control Loop**: Real-time control system monitoring
2. **Phoenix Ultra Overview**: Comprehensive system metrics
3. **Pipeline Performance**: Detailed pipeline analytics

## 💻 Development

### Development Mode

```bash
# Start in development mode
make dev

# Run tests
make test

# Lint code
make lint

# Validate configs
make validate-config
```

### Adding a New Service

1. Create directory: `services/your-service/`
2. Add standard structure (cmd/, internal/, config/)
3. Create `package.json` and `Dockerfile`
4. Update workspace configuration

### Working with the Monorepo

This project uses NPM workspaces and Turborepo for efficient builds:

```bash
# Build specific service
cd services/collector && npm run build

# Run all tests
npm test

# Lint everything
npm run lint
```

## 📚 Documentation

- [Architecture Overview](docs/ARCHITECTURE.md)
- [Monorepo Structure](docs/MONOREPO_STRUCTURE.md)
- [Pipeline Analysis](docs/PIPELINE_ANALYSIS.md)
- [Troubleshooting Guide](docs/TROUBLESHOOTING.md)
- [Migration Guide](docs/MIGRATION_GUIDE.md)

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- OpenTelemetry community for the excellent collector
- Prometheus and Grafana for monitoring capabilities
- All contributors to this project

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/deepaucksharma/Phoenix/issues)
- **Discussions**: [GitHub Discussions](https://github.com/deepaucksharma/Phoenix/discussions)
- **Email**: phoenix-support@example.com
## 🔍 Retrospective Documents

- [Project Retrospective](docs/RETROSPECTIVE.md) - How we'd do it from scratch
- [Missing Features](docs/MISSING_FEATURES.md) - Components not yet migrated
- [Ideal vs Current](docs/IDEAL_VS_CURRENT.md) - Gap analysis and roadmap
