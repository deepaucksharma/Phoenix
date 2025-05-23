# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Phoenix-vNext is a production-ready adaptive cardinality optimization system for OpenTelemetry metrics. It features efficient 3-pipeline processing with Go-based PID control, real-time anomaly detection, automated benchmarking, and enterprise observability integration.

## Architecture Overview

### Core Components

1. **Main Collector** (`otelcol-main`): Efficient shared processing with 3 parallel pipelines
   - Shared processors reduce overhead by 40%
   - Dynamic configuration via control signals
   - Ports: 4317-4318 (OTLP), 8888-8890 (Prometheus), 13133 (health)

2. **Control Actuator** (`control-actuator-go`): Go-based PID controller
   - Advanced PID algorithm with hysteresis
   - Stability period enforcement (120s default)
   - Metrics endpoint at :8081
   - Automatic mode transitions

3. **Anomaly Detector** (`anomaly-detector`): Multi-algorithm detection
   - Statistical (Z-score), rate of change, pattern matching
   - Automatic remediation via webhooks
   - Alert management at :8082

4. **Benchmark Controller** (`benchmark-controller`): Performance validation
   - 4 predefined scenarios
   - Resource tracking and pass/fail validation
   - API at :8083

5. **Observer Collector** (`otelcol-observer`): KPI aggregation
   - Monitors pipeline outputs
   - Provides metrics for control decisions

### Pipeline Architecture

1. **Full Fidelity Pipeline**: Baseline without optimization
2. **Optimized Pipeline**: Smart cardinality reduction (15-40%)
3. **Experimental Pipeline**: Advanced TopK sampling

### Control System

- **PID Control**: `pidOutput = 0.5*error + 0.1*integral + 0.05*derivative`
- **Hysteresis**: 10% band prevents oscillation
- **Modes**: Conservative (<15k), Balanced (15-25k), Aggressive (>25k)

## Development Commands

### Initial Setup
```bash
# Clone and initialize
git clone https://github.com/deepaucksharma/Phoenix.git
cd phoenix-vnext
./scripts/initialize-environment.sh

# Configure environment
cp .env.template .env
# Edit .env with your settings
```

### Running the System
```bash
# Start full stack
docker-compose up -d

# Start specific services
docker-compose up -d prometheus grafana
docker-compose up -d otelcol-main control-actuator-go
docker-compose up -d anomaly-detector benchmark-controller

# View logs
docker-compose logs -f control-actuator-go
docker-compose logs -f anomaly-detector

# Stop everything
docker-compose down
```

### Development & Testing
```bash
# Run Go service locally
cd apps/control-actuator-go
go run main.go

# Run tests
go test -v -race ./...

# Build Docker image
docker-compose build control-actuator-go

# Run benchmarks
curl -X POST http://localhost:8083/benchmark/run \
  -H "Content-Type: application/json" \
  -d '{"scenario": "baseline_steady_state"}'

# Check anomalies
curl http://localhost:8082/alerts | jq

# Monitor control state
watch -n 5 'curl -s http://localhost:8081/metrics | jq'
```

### New Relic Integration
```bash
# Configure New Relic
export NEW_RELIC_LICENSE_KEY=your_key
./scripts/newrelic-integration.sh

# Verify integration
curl -s http://localhost:9090/api/v1/query?query=up | jq
```

## Configuration Files

### OpenTelemetry Configs
- `configs/otel/collectors/main-optimized.yaml`: Shared processing config
- `configs/otel/collectors/observer.yaml`: KPI monitoring
- `configs/otel/exporters/newrelic-enhanced.yaml`: NR integration

### Control System
- `configs/control/optimization_mode.yaml`: Dynamic control file
- Updated by control actuator every 60s
- Read by main collector via config_sources

### Monitoring
- `configs/monitoring/prometheus/rules/phoenix_comprehensive_rules.yml`: 25+ recording rules
- `monitoring/prometheus/prometheus.yaml`: Scrape configs for all services
- `configs/monitoring/grafana/`: Dashboard provisioning

## Key Environment Variables

```bash
# Control System
TARGET_OPTIMIZED_PIPELINE_TS_COUNT=20000     # PID target
HYSTERESIS_FACTOR=0.1                        # 10% band
ADAPTIVE_CONTROLLER_STABILITY_SECONDS=120     # Min between changes

# Thresholds
THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS=15000
THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS=25000

# Resources
OTELCOL_MAIN_MEMORY_LIMIT_MIB=1024
OTELCOL_MAIN_GOMAXPROCS=2

# New Relic
NEW_RELIC_LICENSE_KEY=your_key_here
NEW_RELIC_OTLP_ENDPOINT=https://otlp.nr-data.net:4317
ENABLE_NR_EXPORT_FULL=true
ENABLE_NR_EXPORT_OPTIMISED=true

# Load Generation
SYNTHETIC_PROCESS_COUNT_PER_HOST=250
SYNTHETIC_HOST_COUNT=3
```

## Service Endpoints

### APIs
- Control Actuator: http://localhost:8081/metrics
- Anomaly Detector: http://localhost:8082/alerts
- Benchmark Controller: http://localhost:8083/benchmark/scenarios
- Main Collector Health: http://localhost:13133

### Monitoring
- Grafana: http://localhost:3000 (admin/admin)
- Prometheus: http://localhost:9090
- Pipeline Metrics: http://localhost:8888-8890/metrics
- Observer KPIs: http://localhost:9888/metrics

### Debug
- pprof: http://localhost:1777/debug/pprof
- zpages: http://localhost:55679

## Key Metrics

### Efficiency
- `phoenix:signal_preservation_score` (target: >0.95)
- `phoenix:cardinality_efficiency_ratio`
- `phoenix:resource_efficiency_score`

### Control
- `phoenix:control_stability_score` (target: >0.8)
- `phoenix:control_mode_transitions_total`
- `phoenix:control_loop_effectiveness`

### Anomaly
- `phoenix:cardinality_zscore`
- `phoenix:cardinality_explosion_risk`

## Development Patterns

### Adding New Features
1. Create service in `apps/` or `services/`
2. Add Dockerfile and go.mod
3. Update docker-compose.yaml
4. Add Prometheus scrape config
5. Create recording rules if needed

### Modifying Control Logic
1. Edit `apps/control-actuator-go/main.go`
2. Adjust PID parameters or thresholds
3. Test with benchmark scenarios
4. Monitor stability score

### Performance Tuning
```bash
# Enable debug logging
export OTEL_LOG_LEVEL=debug

# Profile memory usage
curl http://localhost:1777/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Check pipeline efficiency
curl -s http://localhost:9090/api/v1/query?query=phoenix:resource_efficiency_score
```

## CI/CD Integration

### GitHub Actions
- `.github/workflows/ci.yml`: Full CI/CD pipeline
- `.github/workflows/security.yml`: Security scanning

### Pipeline Stages
1. Validate configs
2. Test Go services
3. Integration tests
4. Build images
5. Run benchmarks
6. Deploy (on main)

## Troubleshooting

### Common Issues
1. **High memory**: Increase `OTELCOL_MAIN_MEMORY_LIMIT_MIB`
2. **Control instability**: Increase `ADAPTIVE_CONTROLLER_STABILITY_SECONDS`
3. **Poor reduction**: Check mode via :8081/metrics
4. **Anomaly noise**: Adjust detector thresholds

### Debug Commands
```bash
# Check control decisions
curl http://localhost:8081/metrics | jq '.current_mode'

# View pipeline cardinality
curl -s http://localhost:9090/api/v1/query?query=phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate

# Force mode change (testing)
docker exec control-actuator-go kill -USR1 1

# Export metrics for analysis
curl "http://localhost:9090/api/v1/query_range?query=phoenix:cardinality_growth_rate&start=$(date -u -d '1 hour ago' +%s)&end=$(date +%s)&step=60"
```

## Project Structure

```
phoenix-vnext/
├── apps/                    # Go services
│   ├── control-actuator-go/ # PID controller
│   ├── anomaly-detector/    # Detection system
│   └── synthetic-generator/ # Load generator
├── services/               
│   ├── benchmark/          # Performance validation
│   ├── collector/          # OTEL configs
│   └── control-plane/      # Observer configs
├── configs/
│   ├── otel/              # Collector configs
│   ├── control/           # Control files
│   └── monitoring/        # Prometheus/Grafana
├── scripts/               # Utilities
├── docker-compose.yaml    # Service definitions
└── .github/workflows/     # CI/CD pipelines
```
