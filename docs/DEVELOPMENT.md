# Phoenix-vNext Development Guide

## Development Environment Setup

### Prerequisites

- **Git**: Version control
- **Docker Desktop**: With Compose V2 support
- **Go**: 1.21+ for synthetic generator development
- **Make**: For build automation (optional)
- **curl/wget**: For API testing
- **jq**: For JSON processing and API responses

### Initial Setup

```bash
# Clone the repository
git clone <repository-url>
cd phoenix-vnext

# Initialize development environment
./scripts/initialize-environment.sh

# Start development stack
docker-compose up -d

# Verify services are running
docker-compose ps
```

### IDE Configuration

#### VS Code Setup

Recommended extensions:
- Go (for synthetic generator)
- YAML (for OpenTelemetry configs)
- Docker
- Prometheus/Grafana dashboards

Create `.vscode/settings.json`:

```json
{
    "go.gopath": "${workspaceFolder}/apps/synthetic-generator",
    "yaml.schemas": {
        "https://json.schemastore.org/otelcol": [
            "configs/otel/collectors/*.yaml",
            "configs/otel/processors/*.yaml"
        ]
    },
    "files.exclude": {
        "**/data": true,
        "**/.env": false
    }
}
```

#### Environment Variables

Copy `.env.template` to `.env` and customize:

```bash
# Development-specific settings
SYNTHETIC_PROCESS_COUNT_PER_HOST=100  # Reduced for development
SYNTHETIC_HOST_COUNT=2
OTELCOL_MAIN_MEMORY_LIMIT_MIB=1024   # Lower for development

# Debug settings
OTEL_LOG_LEVEL=debug
ENABLE_PPROF=true
ENABLE_ZPAGES=true

# Disable New Relic export for local development
NEW_RELIC_LICENSE_KEY_FULL=""
NEW_RELIC_LICENSE_KEY_OPTIMISED=""
NEW_RELIC_LICENSE_KEY_EXPERIMENTAL=""
```

## Project Structure

### Directory Layout

```
phoenix-vnext/
├── apps/                              # Application services
│   ├── control-actuator/              # Control loop implementation
│   │   ├── Dockerfile.actuator        # Control actuator container
│   │   └── update-control-file.sh     # Control logic script
│   └── synthetic-generator/           # Load generator
│       ├── generator.go               # Main generator code
│       ├── go.mod                     # Go dependencies
│       └── Dockerfile                 # Generator container
├── configs/                           # Configuration files
│   ├── control/                       # Dynamic control signals
│   │   ├── optimization_mode.yaml     # Current control state
│   │   └── optimization_mode_template.yaml # Control template
│   ├── dashboards/                    # Grafana dashboards
│   ├── monitoring/                    # Monitoring stack configs
│   │   ├── grafana/                   # Grafana provisioning
│   │   └── prometheus/                # Prometheus config and rules
│   └── otel/                          # OpenTelemetry configurations
│       ├── collectors/                # Main and observer configs
│       │   ├── main.yaml              # Main collector config
│       │   └── observer.yaml          # Observer collector config
│       └── processors/                # Processor configurations
│           └── common_intake_processors.yaml
├── data/                              # Persistent data (gitignored)
├── docs/                              # Documentation
├── scripts/                           # Operational scripts
├── docker-compose.yaml               # Main orchestration
├── .env                              # Environment variables
└── README.md                         # Project overview
```

### Key Components

#### Main Collector (`configs/otel/collectors/main.yaml`)

The core OpenTelemetry configuration implementing three pipelines:

- **Receivers**: hostmetrics and OTLP
- **Processors**: Pipeline-specific processing chains
- **Exporters**: Prometheus endpoints and New Relic
- **Connectors**: Routing for pipeline fan-out

#### Observer Collector (`configs/otel/collectors/observer.yaml`)

Control plane collector for metrics aggregation and cardinality estimation.

#### Synthetic Generator (`apps/synthetic-generator/`)

Go application generating realistic process metrics with configurable patterns.

## Development Workflows

### Local Development

#### Starting Services

```bash
# Start core services only
docker-compose up -d otelcol-main otelcol-observer prometheus grafana

# Start with synthetic load
docker-compose up -d

# Start specific services for debugging
docker-compose up otelcol-main  # No -d flag for logs
```

#### Configuration Changes

```bash
# Edit collector configuration
nano configs/otel/collectors/main.yaml

# Restart collector to apply changes
docker-compose restart otelcol-main

# Check configuration reload
curl http://localhost:13133
```

#### Control System Testing

```bash
# Manual control signal update
echo "current_mode: aggressive" > configs/control/optimization_mode.yaml

# Watch control actuator logs
docker-compose logs -f control-loop-actuator

# Check control system metrics
curl "http://localhost:9090/api/v1/query?query=phoenix_control_profile_switches_total"
```

### Synthetic Generator Development

#### Local Go Development

```bash
cd apps/synthetic-generator

# Install dependencies
go mod tidy

# Run locally (requires OTEL endpoint)
export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4318"
export SYNTHETIC_METRICS_PROCESSES=50
export SYNTHETIC_METRICS_HOSTS=1
go run generator.go
```

#### Building and Testing

```bash
# Build binary
go build -o generator .

# Run with custom configuration
./generator \
  --processes 100 \
  --hosts 2 \
  --interval 30s \
  --endpoint http://localhost:4318

# Test with Docker
docker-compose build synthetic-metrics-generator
docker-compose up synthetic-metrics-generator
```

#### Generator Configuration

Key environment variables:

```bash
# Process simulation
SYNTHETIC_METRICS_PROCESSES=250      # Processes per host
SYNTHETIC_METRICS_HOSTS=3           # Number of simulated hosts
SYNTHETIC_METRICS_INTERVAL=15s      # Emission interval

# Behavior patterns
SYNTHETIC_MEMORY_LEAK_PROBABILITY=0.1    # 10% chance of memory leak
SYNTHETIC_CPU_SPIKE_PROBABILITY=0.2      # 20% chance of CPU spike
SYNTHETIC_PROCESS_RESTART_PROBABILITY=0.05  # 5% chance of restart

# Export destination
OTEL_EXPORTER_OTLP_ENDPOINT=http://otelcol-main:4318
```

### Control Actuator Development

#### Script Development

The control actuator is implemented as a bash script:

```bash
# Edit control logic
nano apps/control-actuator/update-control-file.sh

# Test script locally
cd apps/control-actuator
./update-control-file.sh

# Check script output
cat ../../configs/control/optimization_mode.yaml
```

#### Control Logic Testing

```bash
# Set specific thresholds for testing
export THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS=1000
export THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS=2000
export TARGET_OPTIMIZED_PIPELINE_TS_COUNT=1500

# Run control loop once
docker-compose exec control-loop-actuator /app/update-control-file.sh

# Monitor control decisions
watch -n 5 "cat configs/control/optimization_mode.yaml"
```

### Configuration Development

#### Processor Development

Create new processors in `configs/otel/processors/`:

```yaml
# Example: custom_filter.yaml
filter/custom_high_value_processes:
  metrics:
    metric_name: "process.cpu.time"
    expression: |
      resource.attributes["process.executable.name"] IN ["nginx", "postgres", "redis"] OR
      metric.double_value > 1.0
```

Include in main configuration:

```yaml
processors:
  <% include /etc/otel/processors/custom_filter.yaml %>
```

#### Pipeline Development

Test new pipeline configurations:

```bash
# Validate configuration syntax
docker run --rm -v $(pwd)/configs/otel/collectors:/configs \
  otel/opentelemetry-collector-contrib:0.103.1 \
  --config=/configs/main.yaml --dry-run

# Test with minimal data
docker-compose up otelcol-main synthetic-metrics-generator
```

## Testing and Validation

### Unit Testing

#### Synthetic Generator Tests

```bash
cd apps/synthetic-generator

# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Benchmark tests
go test -bench=. ./...
```

#### Configuration Validation

```bash
# Validate YAML syntax
yaml-lint configs/otel/collectors/main.yaml

# Validate OpenTelemetry configuration
docker run --rm -v $(pwd)/configs/otel/collectors:/configs \
  otel/opentelemetry-collector-contrib:0.103.1 \
  --config=/configs/main.yaml --dry-run
```

### Integration Testing

#### End-to-End Pipeline Testing

```bash
# Start services
docker-compose up -d

# Wait for startup
sleep 30

# Generate test data
curl -X POST http://localhost:4318/v1/metrics \
  -H "Content-Type: application/json" \
  -d @test-data/sample-metrics.json

# Verify pipeline outputs
curl http://localhost:8888/metrics | grep phoenix_full_output
curl http://localhost:8889/metrics | grep phoenix_opt_output
curl http://localhost:8890/metrics | grep phoenix_exp_output

# Check cardinality estimates
curl "http://localhost:9090/api/v1/query?query=phoenix_pipeline_output_cardinality_estimate"
```

#### Control System Testing

```bash
# Test control system response to high cardinality
# Increase synthetic load
export SYNTHETIC_PROCESS_COUNT_PER_HOST=500
docker-compose restart synthetic-metrics-generator

# Monitor control decisions
watch -n 10 "cat configs/control/optimization_mode.yaml"

# Verify profile switching
curl "http://localhost:9090/api/v1/query?query=phoenix_control_profile_switches_total"
```

### Performance Testing

#### Load Testing

```bash
# High cardinality load test
export SYNTHETIC_PROCESS_COUNT_PER_HOST=1000
export SYNTHETIC_HOST_COUNT=5
docker-compose restart synthetic-metrics-generator

# Monitor resource usage
docker stats --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}"

# Check pipeline performance
curl http://localhost:8888/metrics | grep otelcol_processor_batch
```

#### Memory Profiling

```bash
# Enable pprof profiling
curl http://localhost:1777/debug/pprof/heap > heap.prof

# Analyze with go tool
go tool pprof heap.prof
```

## Debugging

### Service Debugging

#### Collector Debugging

```bash
# Check collector health
curl http://localhost:13133
curl http://localhost:13134

# View internal state
curl http://localhost:55679/debug/servicez
curl http://localhost:55679/debug/pipelinez

# Check configuration
curl http://localhost:55679/debug/configz
```

#### Log Analysis

```bash
# Structured log filtering
docker-compose logs otelcol-main | jq '.level == "error"'

# Follow specific component logs
docker-compose logs -f otelcol-main | grep "processor"

# Control system debugging
docker-compose logs control-loop-actuator | tail -50
```

### Metrics Debugging

#### Pipeline Output Analysis

```bash
# Compare pipeline outputs
curl http://localhost:8888/metrics | grep -E "process_(cpu|memory)" | wc -l
curl http://localhost:8889/metrics | grep -E "process_(cpu|memory)" | wc -l
curl http://localhost:8890/metrics | grep -E "process_(cpu|memory)" | wc -l

# Cardinality comparison
for port in 8888 8889 8890; do
  echo "Port $port cardinality:"
  curl -s http://localhost:$port/metrics | grep -v "^#" | wc -l
done
```

#### Control System Debugging

```bash
# Check control system state
cat configs/control/optimization_mode.yaml

# Verify threshold calculations
curl "http://localhost:9090/api/v1/query?query=phoenix_pipeline_output_cardinality_estimate"

# Control system metrics
curl "http://localhost:9090/api/v1/query?query=phoenix_control_last_switch_timestamp"
```

## Development Best Practices

### Code Standards

#### Go Code Style

```bash
# Format code
go fmt ./...

# Lint code
golangci-lint run

# Vet code
go vet ./...
```

#### Configuration Standards

- Use consistent indentation (2 spaces for YAML)
- Document processor purposes with comments
- Use environment variable substitution for configurable values
- Include validation for required parameters

### Git Workflow

#### Branch Strategy

```bash
# Feature development
git checkout -b feature/new-processor
git commit -m "Add custom filtering processor"
git push origin feature/new-processor

# Bug fixes
git checkout -b fix/memory-leak
git commit -m "Fix memory leak in synthetic generator"
git push origin fix/memory-leak
```

#### Commit Messages

Follow conventional commit format:

```
feat: add experimental TopK processor
fix: resolve cardinality estimation overflow
docs: update deployment guide with Kubernetes examples
test: add integration tests for control system
```

### Documentation

#### Code Documentation

```go
// ProcessMetrics generates synthetic process metrics with realistic patterns
// including memory usage, CPU time, and process lifecycle events.
func ProcessMetrics(ctx context.Context, cfg Config) error {
    // Implementation...
}
```

#### Configuration Documentation

```yaml
# Processor for selective process filtering based on priority and CPU usage
# Keeps approximately 30-40% of processes based on:
# - Critical priority: always kept
# - High priority: kept if CPU > 0.05
# - Medium priority: kept if CPU > 0.1
filter/optimised_selection:
  metrics:
    expression: |
      resource.attributes["phoenix.priority"] == "critical" OR
      (resource.attributes["phoenix.priority"] == "high" AND metric.double_value > 0.05)
```

### Environment Management

#### Multiple Environments

```bash
# Development
cp .env .env.dev
export ENV_FILE=.env.dev

# Testing
cp .env .env.test
export ENV_FILE=.env.test

# Load specific environment
docker-compose --env-file $ENV_FILE up -d
```

#### Secret Management

```bash
# Use environment-specific secret files
cp .env.secrets.template .env.secrets.dev

# Load secrets separately
set -a && source .env.secrets.dev && set +a
docker-compose up -d
```

## Contributing

### Pull Request Process

1. **Fork and Branch**: Create feature branch from main
2. **Develop**: Implement changes with tests
3. **Test**: Verify all tests pass locally
4. **Document**: Update relevant documentation
5. **Submit**: Create pull request with description

### Code Review Guidelines

- **Functionality**: Does the code work as intended?
- **Performance**: Are there any performance implications?
- **Security**: No sensitive data exposure
- **Documentation**: Is the code properly documented?
- **Testing**: Are there adequate tests?

### Release Process

```bash
# Version tagging
git tag -a v1.2.0 -m "Release version 1.2.0"
git push origin v1.2.0

# Release notes
git log --oneline v1.1.0..v1.2.0 > CHANGELOG.md
```