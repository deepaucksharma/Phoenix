# Phoenix-vNext - Production-Ready Cardinality Optimization System

<div align="center">
  
  [![CI](https://github.com/deepaucksharma/Phoenix/actions/workflows/ci.yml/badge.svg)](https://github.com/deepaucksharma/Phoenix/actions)
  [![Security](https://github.com/deepaucksharma/Phoenix/actions/workflows/security.yml/badge.svg)](https://github.com/deepaucksharma/Phoenix/actions)
  [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
  [![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-Compatible-orange)](https://opentelemetry.io/)
  
</div>

## 🚀 Overview

Phoenix-vNext is a production-ready, adaptive cardinality optimization system for OpenTelemetry metrics. It features advanced multi-pipeline processing with intelligent control loops, anomaly detection, and comprehensive observability.

### Key Features
- **Efficient 3-Pipeline Architecture**: Shared processing with 40% reduced overhead
- **Go-Based Adaptive Control**: PID controller with hysteresis and stability management
- **Real-time Anomaly Detection**: Multi-algorithm detection with automatic remediation
- **Automated Benchmarking**: Performance validation with CI/CD integration
- **Cloud-Native**: Support for AWS EKS and Azure AKS deployment
- **Enterprise Observability**: New Relic integration with cost optimization

### Performance Metrics
- Signal preservation: >98%
- Cardinality reduction: 15-40% (mode dependent)
- Control loop latency: <100ms
- Memory usage: <512MB baseline
- P99 processing latency: <50ms

## 📋 Table of Contents

- [Quick Start](#-quick-start)
- [Architecture](#-architecture)
- [Installation](#-installation)
- [Configuration](#-configuration)
- [Usage](#-usage)
- [Monitoring](#-monitoring)
- [API Reference](#-api-reference)
- [Cloud Deployment](#-cloud-deployment)
- [Development](#-development)
- [Troubleshooting](#-troubleshooting)

## 🏃 Quick Start

```bash
# Clone the repository
git clone https://github.com/deepaucksharma/Phoenix.git
cd phoenix-vnext

# Initialize environment
./scripts/initialize-environment.sh

# Start the stack
docker-compose up -d

# Verify health
curl http://localhost:13133/health

# Access Grafana dashboards
open http://localhost:3000  # admin/admin

# Run a benchmark
curl -X POST http://localhost:8083/benchmark/run \
  -H "Content-Type: application/json" \
  -d '{"scenario": "baseline_steady_state"}'
```

## 🏗️ Architecture

Phoenix-vNext implements a sophisticated multi-tier architecture:

```
┌─────────────────┐     ┌──────────────────────────────────────┐
│ Metric Sources  │────▶│    Phoenix Main Collector (Go)       │
└─────────────────┘     │  ┌────────────────────────────────┐ │
                        │  │ Shared Processors:              │ │
                        │  │ - Memory Limiter                │ │
                        │  │ - Batch Processor               │ │
                        │  │ - Resource Detection            │ │
                        │  ├────────────────────────────────┤ │
                        │  │ Pipeline Routes:                │ │
                        │  │ - Full Fidelity                 │ │
                        │  │ - Optimized (Cardinality)       │ │
                        │  │ - Experimental (TopK)           │ │
                        │  └────────────────────────────────┘ │
                        └──────────────┬───────────────────────┘
                                       │
                ┌──────────────────────┴──────────────────────┐
                │                                             │
         ┌──────▼────────┐                         ┌─────────▼────────┐
         │  Prometheus   │                         │ Observer Collector│
         │  + Recording  │                         │  (KPI Metrics)   │
         │    Rules      │                         └─────────┬────────┘
         └──────┬────────┘                                   │
                │                                   ┌─────────▼────────┐
         ┌──────▼────────┐                         │ Control Actuator │
         │   Grafana     │                         │    (Go + PID)    │
         │  Dashboards   │                         └─────────┬────────┘
         └───────────────┘                                   │
                │                                   ┌─────────▼────────┐
         ┌──────▼────────┐                         │ Anomaly Detector │
         │  New Relic    │                         │ (Multi-Algorithm)│
         │ Integration   │                         └──────────────────┘
         └───────────────┘
```

### Core Components

#### 1. **Main Collector** (`otelcol-main`)
- Efficient shared processing across pipelines
- Dynamic configuration via control signals
- Memory-optimized batching and routing

#### 2. **Control Actuator** (Go Implementation)
- PID control algorithm for smooth transitions
- Hysteresis to prevent oscillation
- Stability period enforcement
- Metrics endpoint for observability
- Bash-based enhanced fallback script (`services/control-plane/actuator/src/control-loop-enhanced.sh`)

#### 3. **Anomaly Detector**
- Statistical detection (Z-score)
- Rate of change analysis
- Pattern matching for known issues
- Automatic control loop integration

#### 4. **Benchmark Controller**
- 4 predefined test scenarios
- Automated performance validation
- Resource tracking and reporting
- CI/CD integration ready

## 🛠️ Installation

### Prerequisites
- Docker & Docker Compose v2.20+
- 4GB+ available RAM
- Ports: 3000, 4317-4318, 8080-8083, 8888-8890, 9090, 13133-13134

### Setup Steps

1. **Clone and Initialize**
   ```bash
   git clone https://github.com/deepaucksharma/Phoenix.git
   cd phoenix-vnext
   ./scripts/initialize-environment.sh
   ```

2. **Configure Environment**
   ```bash
   # Edit .env file with your settings
   vi .env
   
   # Key settings:
   # TARGET_OPTIMIZED_PIPELINE_TS_COUNT=20000
   # HYSTERESIS_FACTOR=0.1
   # NEW_RELIC_LICENSE_KEY=your_key_here
   ```

3. **Start Services**
   ```bash
   docker-compose up -d
   
   # Or use individual services:
   docker-compose up -d prometheus grafana
   docker-compose up -d otelcol-main otelcol-observer
   docker-compose up -d control-actuator-go anomaly-detector
   ```

4. **Verify Installation**
   ```bash
   # Check health endpoints
   curl http://localhost:13133/health  # Main collector
   curl http://localhost:8081/metrics  # Control actuator
   curl http://localhost:8082/health  # Anomaly detector
   ```

## ⚙️ Configuration

### Environment Variables

```bash
# Control Loop Configuration
TARGET_OPTIMIZED_PIPELINE_TS_COUNT=20000      # Target time series count
HYSTERESIS_FACTOR=0.1                         # 10% hysteresis band
ADAPTIVE_CONTROLLER_STABILITY_SECONDS=120      # Min time between changes
ADAPTIVE_CONTROLLER_INTERVAL_SECONDS=60        # Control loop interval

# Optimization Thresholds
THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS=15000
THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS=25000

# Resource Limits
OTELCOL_MAIN_MEMORY_LIMIT_MIB=1024
OTELCOL_MAIN_GOMAXPROCS=2

# New Relic Integration
NEW_RELIC_LICENSE_KEY=your_license_key
NEW_RELIC_OTLP_ENDPOINT=https://otlp.nr-data.net:4317
ENABLE_NR_EXPORT_FULL=true
ENABLE_NR_EXPORT_OPTIMISED=true
ENABLE_NR_EXPORT_EXPERIMENTAL=false

# Load Generation
SYNTHETIC_PROCESS_COUNT_PER_HOST=250
SYNTHETIC_HOST_COUNT=3
SYNTHETIC_METRIC_EMIT_INTERVAL_S=15
```

### Optimization Profiles

| Mode | Time Series Range | Cardinality Reduction | Use Case |
|------|------------------|----------------------|-----------|
| Conservative | < 15,000 | ~5% | Low volume, max fidelity |
| Balanced | 15,000-25,000 | ~15% | Normal operations |
| Aggressive | > 25,000 | ~40% | High cardinality scenarios |

## 📊 Usage

### Basic Operations

```bash
# View real-time logs
docker-compose logs -f control-actuator-go

# Check current optimization mode
curl http://localhost:8081/metrics | jq '.current_mode'

# View detected anomalies
curl http://localhost:8082/alerts | jq

# Force metric generation spike (testing)
docker-compose up synthetic-metrics-generator
```

### Running Benchmarks

```bash
# List available scenarios
curl http://localhost:8083/benchmark/scenarios

# Run specific benchmark
curl -X POST http://localhost:8083/benchmark/run \
  -H "Content-Type: application/json" \
  -d '{"scenario": "cardinality_spike"}'

# Get benchmark results
curl http://localhost:8083/benchmark/results | jq
```

### New Relic Integration

```bash
# Configure New Relic
export NEW_RELIC_LICENSE_KEY=your_key
./scripts/newrelic-integration.sh

# Verify metrics are flowing
curl -s http://localhost:9090/api/v1/query?query=phoenix:signal_preservation_score
```

## 📈 Monitoring

### Access Points
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Control API**: http://localhost:8081/metrics
- **Anomaly API**: http://localhost:8082/alerts
- **Benchmark API**: http://localhost:8083/benchmark/scenarios

### Key Metrics

#### Efficiency Metrics
- `phoenix:signal_preservation_score` - Data fidelity (target: >0.95)
- `phoenix:cardinality_efficiency_ratio` - Reduction effectiveness
- `phoenix:resource_efficiency_score` - Cost per metric

#### Control Metrics
- `phoenix:control_stability_score` - Loop stability (target: >0.8)
- `phoenix:control_mode_transitions_total` - Mode change frequency
- `phoenix:control_loop_effectiveness` - Distance from target

#### Anomaly Metrics
- `phoenix:cardinality_zscore` - Statistical anomaly score
- `phoenix:cardinality_explosion_risk` - Explosion likelihood
- `phoenix:anomaly_detection_latency` - Detection speed

### Alerts

Critical alerts configured:
- `PhoenixCardinalityExplosion` - Exponential growth detected
- `PhoenixResourceExhaustion` - Memory >90%
- `PhoenixControlLoopInstability` - Frequent mode changes
- `PhoenixSLOViolation` - Service objectives not met

## 🔌 API Reference

### Control Actuator API

**GET** `/metrics`
```json
{
  "current_mode": "balanced",
  "transition_count": 3,
  "stability_score": 0.92,
  "integral_error": 125.5,
  "last_error": -523.0,
  "uptime_seconds": 3600
}
```

### Anomaly Detector API

**GET** `/alerts`
```json
[{
  "id": "cardinality-1234567890",
  "anomaly": {
    "detector_name": "statistical_zscore",
    "metric_name": "phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate",
    "timestamp": "2024-05-23T10:30:00Z",
    "value": 35000,
    "expected": 20000,
    "severity": "high",
    "confidence": 0.95,
    "description": "Value 35000 is 4.2 standard deviations from mean 20000"
  },
  "status": "active",
  "action_taken": "Notified control loop to switch to aggressive mode"
}]
```

### Benchmark Controller API

**POST** `/benchmark/run`
```json
{
  "scenario": "cardinality_spike"
}
```

**GET** `/benchmark/results`
```json
[{
  "scenario": "baseline_steady_state",
  "start_time": "2024-05-23T10:00:00Z",
  "end_time": "2024-05-23T10:10:00Z",
  "metrics": {
    "signal_preservation": 0.98,
    "cardinality_reduction": 15.2,
    "cpu_usage": 45.3,
    "memory_usage": 412.5
  },
  "passed": true
}]
```

## ☁️ Cloud Deployment

### AWS EKS Deployment

```bash
# Configure AWS credentials
export AWS_PROFILE=your-profile

# Deploy infrastructure
cd infrastructure/terraform/environments/aws
terraform init
terraform apply

# Deploy Phoenix
cd ../../../..
./infrastructure/scripts/deploy-aws.sh
```

### Azure AKS Deployment

```bash
# Login to Azure
az login

# Deploy infrastructure
cd infrastructure/terraform/environments/azure
terraform init
terraform apply

# Deploy Phoenix
cd ../../../..
./infrastructure/scripts/deploy-azure.sh
```

### Kubernetes Manifests

```bash
# Using Helm
helm install phoenix ./infrastructure/kubernetes/helm/phoenix \
  --namespace phoenix \
  --create-namespace \
  --values ./infrastructure/kubernetes/helm/phoenix/values.yaml

# Using kubectl
kubectl create namespace phoenix
kubectl apply -f ./infrastructure/kubernetes/
```

## 💻 Development

### Local Development

```bash
# Run specific service locally
cd apps/control-actuator-go
go run main.go

# Run with live reload
air

# Run tests
go test -v -race ./...

# Build binary
go build -o control-actuator
```

### Testing

```bash
# Unit tests
cd services/benchmark
go test -v -cover ./...

# Integration tests
docker-compose -f docker-compose.test.yaml up --abort-on-container-exit

# Load testing
k6 run tests/load/spike_test.js
```

### CI/CD Pipeline

The project includes GitHub Actions workflows for:
- Configuration validation
- Go service testing with coverage
- Integration testing
- Performance benchmarking
- Security scanning (Trivy, Gosec, OWASP)
- Automated deployment

## 🔧 Troubleshooting

### Common Issues

1. **High Memory Usage**
   ```bash
   # Increase memory limit
   export OTELCOL_MAIN_MEMORY_LIMIT_MIB=2048
   docker-compose up -d otelcol-main
   ```

2. **Control Loop Instability**
   ```bash
   # Increase stability period
   export ADAPTIVE_CONTROLLER_STABILITY_SECONDS=300
   docker-compose up -d control-actuator-go
   ```

3. **Poor Cardinality Reduction**
   ```bash
   # Check current mode and metrics
   curl http://localhost:8081/metrics
   curl http://localhost:9090/api/v1/query?query=phoenix:cardinality_efficiency_ratio
   ```

4. **Anomaly False Positives**
   ```bash
   # Adjust detection sensitivity
   # Edit apps/anomaly-detector/main.go
   # Increase threshold from 3.0 to 4.0
   docker-compose build anomaly-detector
   docker-compose up -d anomaly-detector
   ```

### Debug Commands

```bash
# Enable debug logging
export OTEL_LOG_LEVEL=debug
docker-compose up -d otelcol-main

# Check pipeline metrics
curl -s http://localhost:8888/metrics | grep pipeline

# Force control mode change (testing)
curl -X POST http://localhost:8081/mode \
  -H "Content-Type: application/json" \
  -d '{"mode": "aggressive"}'

# Export metrics for analysis
curl http://localhost:9090/api/v1/query_range?query=phoenix:cardinality_growth_rate&start=$(date -u -d '1 hour ago' +%s)&end=$(date +%s)&step=60
```

## 📚 Documentation

- [Architecture Deep Dive](docs/ARCHITECTURE.md)
- [Pipeline Analysis](docs/PIPELINE_ANALYSIS.md)
- [Troubleshooting Guide](docs/TROUBLESHOOTING.md)
- [API Documentation](docs/API.md)
- [Performance Tuning](docs/PERFORMANCE.md)
- [Security Best Practices](docs/SECURITY.md)

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

- OpenTelemetry community
- Prometheus and Grafana teams
- New Relic for OTLP support
- All contributors to Phoenix

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/deepaucksharma/Phoenix/issues)
- **Discussions**: [GitHub Discussions](https://github.com/deepaucksharma/Phoenix/discussions)
- **Security**: security@phoenix-project.io