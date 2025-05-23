# Phoenix-vNext Streamlined Project Structure

## Directory Layout

```
phoenix-vnext/
├── apps/                      # Core application services
│   ├── anomaly-detector/      # Anomaly detection service
│   ├── control-actuator-go/   # Go-based PID controller
│   └── synthetic-generator/   # Metrics load generator
│
├── configs/                   # All configuration files
│   ├── control/              # Control loop configurations
│   │   └── optimization_mode.yaml
│   ├── monitoring/           # Monitoring stack configs
│   │   ├── grafana/
│   │   │   ├── dashboards/   # Grafana dashboards (canonical)
│   │   │   ├── datasources/
│   │   │   └── provisioning/
│   │   └── prometheus/
│   │       ├── prometheus.yaml
│   │       └── rules/        # Recording rules and alerts
│   ├── otel/                 # OpenTelemetry configurations
│   │   ├── collectors/       # Collector configs
│   │   │   ├── main.yaml
│   │   │   └── observer.yaml
│   │   └── processors/       # Shared processor configs
│   └── production/           # Production-ready configs
│       └── tls/             # TLS certificates
│
├── data/                     # Runtime data (gitignored)
│   ├── prometheus/
│   ├── grafana/
│   └── otelcol/
│
├── docs/                     # Documentation
│   ├── ARCHITECTURE.md
│   ├── PIPELINE_ANALYSIS.md
│   └── TROUBLESHOOTING.md
│
├── k8s/                      # Kubernetes manifests
│   ├── base/                 # Base Kustomize configs
│   └── overlays/             # Environment overlays
│       ├── dev/
│       ├── staging/
│       └── production/
│
├── runbooks/                 # Operational runbooks
│   ├── incident-response/
│   ├── operational-procedures/
│   └── troubleshooting/
│
├── scripts/                  # Utility scripts
│   ├── initialize-environment.sh
│   └── validate-config.sh
│
├── services/                 # Additional services
│   ├── analytics/            # Analytics API service
│   └── benchmark/            # Benchmark service
│
├── tools/                    # Operational tools
│   └── scripts/              # Health checks, backups
│
├── docker-compose.yaml       # Main compose file
├── docker-compose.override.yml # Dev overrides
├── .env.example              # Environment template
└── README.md                 # Project documentation
```

## Configuration Hierarchy

1. **Base Configurations** (`configs/`)
   - Single source of truth for all configs
   - Environment-specific overrides via environment variables

2. **Service Locations**
   - Core services: `apps/`
   - Extended services: `services/`
   - Utilities: `tools/scripts/`

3. **Data Flow**
   ```
   Synthetic Generator → OTel Collector Main → Prometheus
                              ↓
                      Observer Collector → Control Actuator
                              ↓
                      Anomaly Detector
   ```

## Key Improvements

1. **Consolidated Configurations**
   - Single Prometheus config: `configs/monitoring/prometheus/prometheus.yaml`
   - Unified recording rules: `configs/monitoring/prometheus/rules/phoenix_rules.yml`
   - Canonical dashboards: `configs/monitoring/grafana/dashboards/`

2. **Simplified Docker Compose**
   - Main file for production
   - Override file for development
   - Profile-based optional services

3. **Clear Service Separation**
   - `apps/`: Core Phoenix services
   - `services/`: Extended functionality
   - Clear dependencies and health checks

4. **Standardized Naming**
   - Service names: `phoenix-<component>`
   - Config files: `<component>.yaml`
   - Metrics: `phoenix:<category>_<metric>`

## Quick Start

```bash
# Initialize environment
./scripts/initialize-environment.sh

# Start core services
docker-compose up -d

# Start with generators (dev mode)
docker-compose --profile generators up -d

# View logs
docker-compose logs -f otelcol-main

# Access services
- Grafana: http://localhost:3000
- Prometheus: http://localhost:9090
- Control API: http://localhost:8081
- Analytics: http://localhost:8080
```

## Removed Redundancies

- Duplicate Grafana dashboards consolidated
- Multiple Prometheus configs merged
- Overlapping recording rules unified
- Redundant k8s structures removed
- Duplicate service implementations consolidated
- Old deployment scripts archived