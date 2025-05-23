# Phoenix-vNext: Production-Ready Cardinality Optimization System

## Overview

Phoenix-vNext is an advanced, production-ready implementation of a multi-pipeline cardinality optimization system for OpenTelemetry metrics. Building on the architectural review recommendations, this enhanced version includes:

- **Efficient Pipeline Architecture**: Optimized shared processing with reduced resource overhead
- **Go-based Control Loop**: Enhanced PID-like adaptive control with hysteresis and stability features
- **Comprehensive Monitoring**: Advanced Prometheus recording rules and alerting
- **Anomaly Detection**: Multi-algorithm anomaly detection system with automatic remediation
- **Benchmark Controller**: Automated performance validation and regression testing
- **CI/CD Integration**: Complete GitHub Actions pipeline with security scanning
- **Cloud Deployment**: Support for AWS EKS and Azure AKS deployment
- **New Relic Integration**: Enhanced observability with New Relic OTLP export

## Quick Start

### Prerequisites

- Docker and Docker Compose (v2.20+)
- 4GB+ available RAM
- Port availability: 3000, 4317-4318, 8080-8083, 8888-8890, 9090, 13133-13134

### Installation

```bash
# Clone the repository
git clone https://github.com/your-org/phoenix-vnext.git
cd phoenix-vnext

# Initialize environment
./scripts/initialize-environment.sh

# Start the enhanced stack
docker-compose up -d

# Verify health
docker-compose ps
curl http://localhost:13133/health
```

### Access Points

- **Grafana Dashboard**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Control Loop API**: http://localhost:8081/metrics
- **Anomaly Detector**: http://localhost:8082/alerts
- **Benchmark Controller**: http://localhost:8083/benchmark/scenarios

## Architecture Improvements

### 1. Pipeline Efficiency (Phase 1)

The enhanced collector configuration (`configs/otel/collectors/main-optimized.yaml`) implements:

- **Shared Processing**: Common processors (memory_limiter, batch, resource detection) shared across pipelines
- **Reduced Overhead**: Single receiver instance with efficient routing
- **Optimized Batching**: Tuned batch sizes for optimal throughput

```yaml
processors:
  memory_limiter:    # Shared across all pipelines
  batch:             # Common batching configuration
  resource:          # Unified resource detection
```

### 2. Enhanced Control Loop (Phase 1)

The Go-based control actuator (`apps/control-actuator-go/`) provides:

- **PID Controller**: Proportional-Integral-Derivative control for smooth transitions
- **Hysteresis**: Prevents oscillation with configurable thresholds
- **Stability Period**: Enforces minimum time between mode changes
- **Metrics Endpoint**: Exposes control loop metrics for monitoring

```go
// PID calculation with discrete control output
pidOutput := 0.5*error + 0.1*integralError + 0.05*derivative

// Hysteresis application
if currentMode == Conservative {
    conservativeThreshold *= (1 + hysteresisFactor)
}
```

### 3. Comprehensive Recording Rules (Phase 1)

Advanced Prometheus rules (`configs/monitoring/prometheus/rules/phoenix_comprehensive_rules.yml`):

- **Signal Preservation Metrics**: Track data fidelity across pipelines
- **Resource Efficiency Scores**: Monitor cost per datapoint
- **Anomaly Detection Prep**: Z-score calculations for anomaly detection
- **SLI/SLO Tracking**: Service level indicators and objectives

### 4. Benchmark Controller (Phase 2)

Automated performance validation (`services/benchmark/`):

- **Predefined Scenarios**: Baseline, spike, gradual growth, wave patterns
- **Automated Validation**: Pass/fail criteria for each scenario
- **Resource Tracking**: CPU, memory, latency metrics during tests
- **CI/CD Integration**: Automated performance regression testing

### 5. Anomaly Detection System (Phase 3)

Multi-algorithm detection (`apps/anomaly-detector/`):

- **Statistical Detection**: Z-score based anomaly detection
- **Rate of Change**: Identifies rapid metric changes
- **Pattern Matching**: Detects known bad patterns (cardinality explosion, memory leaks)
- **Automatic Remediation**: Triggers control loop adjustments for critical anomalies

## Configuration

### Environment Variables

Key configuration options in `.env`:

```bash
# Control Loop Tuning
TARGET_OPTIMIZED_PIPELINE_TS_COUNT=20000
HYSTERESIS_FACTOR=0.1
ADAPTIVE_CONTROLLER_STABILITY_SECONDS=120

# Resource Limits
OTELCOL_MAIN_MEMORY_LIMIT_MIB=1024
OTELCOL_MAIN_GOMAXPROCS=2

# New Relic Integration
NEW_RELIC_LICENSE_KEY=your_license_key
NEW_RELIC_OTLP_ENDPOINT=https://otlp.nr-data.net:4317
ENABLE_NR_EXPORT_FULL=true
ENABLE_NR_EXPORT_OPTIMISED=true
```

### Optimization Profiles

Three optimization modes with automatic switching:

1. **Conservative** (<15k time series): Minimal optimization, maximum fidelity
2. **Balanced** (15k-25k time series): Moderate cardinality reduction
3. **Aggressive** (>25k time series): Maximum optimization for high cardinality

## Monitoring & Alerting

### Key Metrics

- `phoenix:signal_preservation_score`: Data fidelity (target: >0.95)
- `phoenix:cardinality_efficiency_ratio`: Reduction effectiveness
- `phoenix:resource_efficiency_score`: Cost per metric ratio
- `phoenix:control_stability_score`: Control loop stability (target: >0.8)

### Alerts

Critical alerts configured:

- **PhoenixCardinalityExplosion**: Exponential cardinality growth detected
- **PhoenixResourceExhaustion**: Memory usage >90%
- **PhoenixControlLoopInstability**: Frequent mode transitions
- **PhoenixSLOViolation**: Service level objectives not met

## Performance Benchmarks

Run automated benchmarks:

```bash
# List available scenarios
curl http://localhost:8083/benchmark/scenarios

# Run baseline benchmark
curl -X POST http://localhost:8083/benchmark/run \
  -H "Content-Type: application/json" \
  -d '{"scenario": "baseline_steady_state"}'

# View results
curl http://localhost:8083/benchmark/results
```

Expected results:
- Signal preservation: >98%
- Cardinality reduction: 15-40% (mode dependent)
- Memory usage: <512MB baseline
- P99 latency: <50ms

## CI/CD Pipeline

GitHub Actions workflow (`.github/workflows/ci.yml`):

1. **Validation**: YAML linting, config validation
2. **Testing**: Go unit tests with race detection
3. **Integration**: Full stack testing with synthetic load
4. **Performance**: Automated benchmark validation
5. **Security**: Trivy, Gosec, OWASP dependency scanning
6. **Deployment**: Automated deployment to K8s clusters

## Cloud Deployment

### AWS EKS

```bash
cd infrastructure/terraform/environments/aws
terraform init
terraform apply
../../../scripts/deploy-aws.sh
```

### Azure AKS

```bash
cd infrastructure/terraform/environments/azure
terraform init
terraform apply
../../../scripts/deploy-azure.sh
```

## New Relic Integration

Enhanced integration with New Relic:

```bash
# Configure New Relic integration
export NEW_RELIC_LICENSE_KEY=your_key
./scripts/newrelic-integration.sh

# Metrics available in New Relic:
# - phoenix.* (all Phoenix-specific metrics)
# - Custom dashboards and alerts
# - Distributed tracing support
```

## Troubleshooting

### Common Issues

1. **High memory usage**: Adjust `OTELCOL_MAIN_MEMORY_LIMIT_MIB`
2. **Control loop instability**: Increase `ADAPTIVE_CONTROLLER_STABILITY_SECONDS`
3. **Poor cardinality reduction**: Check optimization mode and thresholds
4. **Anomaly false positives**: Tune detection thresholds in anomaly-detector

### Debug Commands

```bash
# Check control loop status
curl http://localhost:8081/metrics | jq

# View recent anomalies
curl http://localhost:8082/alerts | jq

# Inspect pipeline metrics
curl -s http://localhost:9090/api/v1/query?query=phoenix:cardinality_efficiency_ratio

# Force control mode change (testing)
docker exec control-actuator-go kill -USR1 1
```

## Contributing

See [CONTRIBUTING.md](./docs/CONTRIBUTING.md) for development guidelines.

## License

Apache License 2.0 - See [LICENSE](./LICENSE) for details.
