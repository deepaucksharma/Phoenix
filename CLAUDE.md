# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Phoenix-vNext is a production-ready 3-Pipeline Cardinality Optimization System for OpenTelemetry metrics collection and processing. The system uses adaptive cardinality management with dynamic switching between optimization profiles (conservative, balanced, aggressive) based on metric volume and system performance through a PID controller implemented in Go.

## Architecture

### Core System Components
- **Main Collector** (`otelcol-main`): Runs 3 parallel pipelines with different cardinality optimization levels
- **Observer Collector** (`otelcol-observer`): Control plane that monitors pipeline metrics and system performance
- **Control Actuator** (`control-actuator-go`): Go-based PID controller with hysteresis and stability management
- **Anomaly Detector** (`anomaly-detector`): Multi-algorithm detection (Z-score, rate of change, pattern matching)
- **Benchmark Controller** (`benchmark-controller`): Performance validation with 4 test scenarios
- **Synthetic Generator** (`synthetic-metrics-generator`): Go-based load generator for testing

### Pipeline Architecture
The system operates 3 distinct pipelines in parallel:
1. **Full Fidelity Pipeline** (`pipeline_full_fidelity`) - Complete metrics baseline without optimization
2. **Optimized Pipeline** (`pipeline_optimised`) - Moderate cardinality reduction with configurable aggregation
3. **Experimental TopK Pipeline** (`pipeline_experimental_topk`) - Advanced optimization using TopK sampling

### Adaptive Control System
- Observer monitors `phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate` metrics
- Control actuator applies discrete profile switching based on time series count thresholds:
  - Conservative: < 15,000 time series
  - Balanced: 15,000 - 25,000 time series  
  - Aggressive: > 25,000 time series
- Control signals written to `configs/control/optimization_mode.yaml` and read by main collector
- PID algorithm: `pidOutput = 0.5*error + 0.1*integral + 0.05*derivative`
- Hysteresis factor (10%) prevents rapid oscillation

## Development Commands

### Quick Start
```bash
# Initialize environment (creates data dirs, control files, .env from template)
./scripts/consolidated/core/initialize-environment.sh

# Start full stack
./run-phoenix.sh

# Or use docker-compose directly
docker-compose up -d

# Stop services
./run-phoenix.sh stop

# Clean everything
./run-phoenix.sh clean
```

### Makefile Commands
```bash
# Main targets
make help                   # Show all available commands
make setup-env             # Initialize environment
make build                 # Build all projects (Turborepo)
make build-docker          # Build all Docker images
make dev                   # Start development mode
make test                  # Run tests
make test-integration      # Run integration tests
make monitor               # Open monitoring dashboards
make clean                 # Clean build artifacts

# Service logs
make collector-logs        # View main collector logs
make observer-logs         # View observer logs
make actuator-logs         # View control actuator logs
make generator-logs        # View generator logs
make anomaly-logs          # View anomaly detector logs
make benchmark-logs        # View benchmark controller logs

# Utilities
make validate-config       # Validate YAML configurations
make docs-serve           # Serve documentation locally
```

### Docker Compose Operations
```bash
# Start specific services
docker-compose up -d otelcol-main otelcol-observer prometheus grafana

# Rebuild and restart a specific service
docker-compose build control-actuator-go
docker-compose up -d control-actuator-go

# View logs
docker-compose logs -f otelcol-main
docker-compose logs -f control-actuator-go

# Check service health
curl http://localhost:13133  # Main collector health
curl http://localhost:13134  # Observer health
curl http://localhost:8081/health  # Control actuator health
curl http://localhost:8082/health  # Anomaly detector health
```

### Testing & Validation
```bash
# Run integration tests
./tests/integration/test_core_functionality.sh

# Generate synthetic load
docker-compose up synthetic-metrics-generator

# Run benchmark scenarios
curl http://localhost:8083/benchmark/scenarios  # List scenarios
curl -X POST http://localhost:8083/benchmark/run \
  -H "Content-Type: application/json" \
  -d '{"scenario": "baseline_steady_state"}'

# Monitor control signal changes
watch cat configs/control/optimization_mode.yaml

# Validate configurations
docker-compose config
sha256sum configs/otel/collectors/*.yaml configs/templates/control/*.yaml > CHECKSUMS.txt
```

### Cloud Deployment
```bash
# AWS deployment
./deploy-aws.sh

# Azure deployment  
./deploy-azure.sh

# Terraform deployment
cd infrastructure/terraform/environments/aws
terraform init && terraform apply
```

## Configuration Architecture

### OpenTelemetry Configurations
- `configs/otel/collectors/main.yaml`: Core collector with 3-pipeline configuration (all processors/exporters defined inline)
- `configs/otel/collectors/observer.yaml`: Monitoring collector that exposes KPI metrics
- `configs/otel/processors/common_intake_processors.yaml`: Template/reference for common processor patterns (not actively included)
- `configs/otel/exporters/newrelic-enhanced.yaml`: Template/reference for New Relic integration (production uses pipeline-specific keys)

### Control System
- `configs/control/optimization_mode.yaml`: Dynamic control file modified by actuator
- `configs/templates/control/optimization_mode_template.yaml`: Template defining control file schema
- Version tracking with `config_version` field
- Correlation IDs for tracking changes

### Monitoring Stack
- `configs/monitoring/prometheus/prometheus.yaml`: Prometheus scrape configuration
- `configs/monitoring/prometheus/rules/phoenix_rules_consolidated.yml`: Canonical recording rules (colon-notation)
- `configs/monitoring/grafana/`: Datasource and dashboard provisioning

## Key Environment Variables

Critical variables in `.env`:
```bash
# Control thresholds for adaptive switching
TARGET_OPTIMIZED_PIPELINE_TS_COUNT=20000
THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS=15000
THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS=25000
HYSTERESIS_FACTOR=0.1

# Resource constraints
OTELCOL_MAIN_MEMORY_LIMIT_MIB="1024"
OTELCOL_MAIN_GOMAXPROCS="2"

# Control loop timing
ADAPTIVE_CONTROLLER_INTERVAL_SECONDS=60
ADAPTIVE_CONTROLLER_STABILITY_SECONDS=120

# Load generation
SYNTHETIC_PROCESS_COUNT_PER_HOST=250
SYNTHETIC_HOST_COUNT=3
SYNTHETIC_METRIC_EMIT_INTERVAL_S=15

# New Relic export
NEW_RELIC_LICENSE_KEY=your_key_here
NEW_RELIC_OTLP_ENDPOINT=https://otlp.nr-data.net:4317
ENABLE_NR_EXPORT_FULL="false"
ENABLE_NR_EXPORT_OPTIMISED="false"
ENABLE_NR_EXPORT_EXPERIMENTAL="false"
```

## Service Endpoints & APIs

### Core Service Endpoints
- **Grafana Dashboard**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Main Collector Metrics**: http://localhost:8888/metrics
- **Optimized Pipeline**: http://localhost:8889/metrics 
- **Experimental Pipeline**: http://localhost:8890/metrics
- **Observer Metrics**: http://localhost:9888/metrics

### API Endpoints
- **Control Actuator API**: http://localhost:8081
  - `GET /metrics` - Control state and metrics
  - `GET /health` - Health check
  - `POST /anomaly` - Webhook for anomaly events
  - `POST /mode` - Force mode change (testing)
- **Anomaly Detector API**: http://localhost:8082
  - `GET /alerts` - Active anomalies
  - `GET /health` - Health check
  - `GET /metrics` - Prometheus metrics
- **Benchmark Controller**: http://localhost:8083
  - `GET /benchmark/scenarios` - List test scenarios
  - `POST /benchmark/run` - Run benchmark
  - `GET /benchmark/results` - Get results
  - `GET /benchmark/validate` - Check SLO compliance

### Health Checks
- Main Collector: http://localhost:13133
- Observer Collector: http://localhost:13134
- Docker health checks configured with 20s intervals, 3 retries

### Debug Endpoints
- pprof: http://localhost:1777/debug/pprof
- zpages: http://localhost:55679

## Control Flow & Data Paths

1. **Metrics Ingestion**: Synthetic generator → Main collector OTLP endpoint (4318)
2. **Pipeline Processing**: 3 parallel processing chains with different optimization levels
3. **Metrics Export**: Each pipeline exports to dedicated Prometheus endpoints (8888-8890)
4. **Monitoring**: Observer scrapes main collector metrics and exposes KPIs (9888)
5. **Control Loop**: Actuator queries observer metrics → calculates profile → updates control file
6. **Adaptation**: Main collector reads control file changes → adjusts pipeline behavior
7. **Anomaly Detection**: Detector monitors metrics → sends webhooks to control actuator
8. **Benchmarking**: Controller generates load patterns → validates performance

## Development Patterns

### Adding New Processors
1. Create processor config in `configs/otel/processors/`
2. Include in pipeline via `configs/otel/collectors/main.yaml`
3. Test with synthetic load and monitor cardinality impact

### Modifying Control Logic
1. Update thresholds in `.env` file
2. Modify PID logic in `apps/control-actuator-go/main.go`
3. Test profile transitions using benchmark scenarios
4. Monitor stability score via recording rules

### Performance Tuning
```bash
# Enable debug logging
export OTEL_LOG_LEVEL=debug
docker-compose up -d otelcol-main

# Profile memory usage
curl http://localhost:1777/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Monitor resource usage
docker-compose top

# Check pipeline efficiency
curl -s http://localhost:9090/api/v1/query?query=phoenix:resource_efficiency_score
```

### Local Development
```bash
# Run Go service locally
cd apps/control-actuator-go
go run main.go

# With live reload
air

# Run tests
go test -v -race ./...

# Build binary
go build -o control-actuator
```

## Monorepo Structure

- **`apps/`**: Go-based microservices (control-actuator, anomaly-detector)
- **`services/`**: Service implementations with Dockerfiles
- **`configs/`**: Technology-grouped configurations (otel, monitoring, control)
- **`infrastructure/`**: Cloud deployment (Terraform, Helm charts)
- **`packages/`**: Shared packages (managed by npm workspaces)
- **`scripts/`**: Operational utilities and environment setup
- **`tests/`**: Integration and performance tests
- **`tools/`**: Development and migration utilities
- **`data/`**: Persistent storage directories (gitignored)

## Build System

- **Turborepo**: Parallel builds with caching (`turbo.json`)
- **Make**: Developer-friendly commands (see `make help`)
- **npm workspaces**: Package management
- **Docker multi-stage builds**: Optimized images
- **Go modules**: Dependency management for Go services

## Key Metrics & Alerts

### Recording Rules (25+ rules)
- **Efficiency**: `phoenix:signal_preservation_score`, `phoenix:cardinality_efficiency_ratio`
- **Performance**: `phoenix:pipeline_latency_ms_p99`, `phoenix:pipeline_throughput_metrics_per_sec`
- **Control**: `phoenix:control_stability_score`, `phoenix:control_loop_effectiveness`
- **Anomaly**: `phoenix:cardinality_zscore`, `phoenix:cardinality_explosion_risk`
- **Resource**: `phoenix:resource_efficiency_score`, `phoenix:collector_memory_usage_mb`

### Critical Alerts
- `PhoenixCardinalityExplosion`: Exponential growth detected
- `PhoenixResourceExhaustion`: Memory >90%
- `PhoenixControlLoopInstability`: Frequent mode changes
- `PhoenixSLOViolation`: Service objectives not met

## Benchmark Scenarios

1. **baseline_steady_state**: Normal operation validation
2. **cardinality_spike**: Sudden 3x increase testing
3. **gradual_growth**: Linear growth over time
4. **wave_pattern**: Sinusoidal load pattern

## CI/CD Integration

### GitHub Actions Workflows
- `.github/workflows/ci.yml`: Full CI/CD pipeline
- `.github/workflows/security.yml`: Security scanning (Trivy, Gosec, OWASP)

### Pipeline Stages
1. Configuration validation
2. Go service testing with coverage
3. Integration testing
4. Docker image building
5. Performance benchmarking
6. Deployment (on main branch)

## Important Implementation Notes

### Service Ports (Updated)
- Control Actuator: **8081** (was 8080 in early versions)
- Anomaly Detector: **8082** 
- Benchmark Controller: **8083**
- Main Collector Health: **13133**
- Observer Health: **13134**

### PID Control Implementation
The control actuator implements a full PID controller with:
- Configurable gains: `PID_KP=0.5`, `PID_KI=0.1`, `PID_KD=0.05`
- Anti-windup for integral term with limit
- Time-based derivative calculation
- Soft integral reset on mode changes

### Recent Architecture Changes
1. **Consolidated Scripts**: Major scripts moved to `scripts/` directory
2. **Cleaned Codebase**: 96% reduction in code size, removed obsolete files
3. **All APIs Implemented**: Every documented endpoint now exists
4. **Recording Rules**: Added colon-notation metrics (e.g., `phoenix:signal_preservation_score`)

## Troubleshooting

### Common Issues
1. **High memory usage**: Increase `OTELCOL_MAIN_MEMORY_LIMIT_MIB`
2. **Control instability**: Increase `ADAPTIVE_CONTROLLER_STABILITY_SECONDS`
3. **Poor reduction**: Check mode via :8081/metrics, adjust thresholds
4. **Anomaly noise**: Modify detector threshold in `apps/anomaly-detector/main.go`
5. **Port conflicts**: Ensure ports 8081-8083 are available for services

### Debug Commands
```bash
# Check control decisions
curl http://localhost:8081/metrics | jq '.current_mode'

# View pipeline cardinality
curl -s http://localhost:9090/api/v1/query?query=phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate

# Force mode change (testing)
curl -X POST http://localhost:8081/mode \
  -H "Content-Type: application/json" \
  -d '{"mode": "aggressive"}'

# Export metrics for analysis
curl "http://localhost:9090/api/v1/query_range?query=phoenix:cardinality_growth_rate&start=$(date -u -d '1 hour ago' +%s)&end=$(date +%s)&step=60"

# Check for memory leaks
docker stats --no-stream

# Verify all services are healthy
for port in 8081 8082 8083; do echo "Port $port:"; curl -s http://localhost:$port/health | jq; done
```