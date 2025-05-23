# Phoenix Configuration Directory

This directory contains all configuration files for the Phoenix observability system, organized by technology and purpose.

## Directory Structure

```
configs/
├── control/                    # Control system configurations
│   └── optimization_mode.yaml  # Dynamic control file (modified by control actuator)
│
├── monitoring/                 # Monitoring stack configurations
│   ├── grafana/               # Grafana dashboards and provisioning
│   │   ├── dashboards/        # Dashboard JSON files
│   │   ├── dashboards_provider.yaml
│   │   └── grafana-datasource.yaml
│   │
│   └── prometheus/            # Prometheus configuration
│       ├── prometheus.yaml    # Main Prometheus config
│       ├── prometheus-*.yaml  # Alternative configs (backup, minimal, clean)
│       └── rules/            # Recording and alerting rules
│           ├── phoenix_core_rules.yml      # Core metrics and aggregations
│           ├── phoenix_advanced_rules.yml  # Advanced analytics
│           ├── phoenix_documented_metrics.yml
│           └── phoenix_rules.yml          # Main rule set
│
├── otel/                      # OpenTelemetry configurations
│   ├── collectors/           # Collector configurations
│   │   ├── main.yaml        # Main collector (3-pipeline architecture)
│   │   ├── main-prod.yaml   # Production variant
│   │   ├── main-backup.yaml # Backup configuration
│   │   ├── main-minimal.yaml # Minimal configuration
│   │   └── observer.yaml    # Observer collector for KPI metrics
│   │
│   ├── exporters/           # Exporter configurations
│   │   └── newrelic-enhanced.yaml  # New Relic OTLP integration
│   │
│   └── processors/          # Processor configurations
│       └── common_intake_processors.yaml  # Shared processors
│
└── templates/                # Configuration templates
    ├── benchmark/           # Benchmark configurations
    │   └── config.yaml
    │
    ├── control/             # Control system templates
    │   └── optimization_mode_template.yaml
    │
    ├── monitoring/          # Monitoring templates
    │   ├── grafana/
    │   │   ├── dashboards_provider.yaml
    │   │   └── grafana-datasource.yaml
    │   └── prometheus/
    │       ├── prometheus.yaml
    │       └── rules/
    │           └── phoenix_rules.yml
    │
    └── otel/               # OpenTelemetry templates
        └── collectors/
            ├── main.yaml
            └── observer.yaml
```

## Configuration Files

### Control System
- **optimization_mode.yaml**: The active control file that is dynamically updated by the control actuator. Contains the current optimization mode (conservative/balanced/aggressive) and version tracking.

### OpenTelemetry Collectors
- **main.yaml**: Core collector configuration with 3-pipeline architecture (full_fidelity, optimised, experimental_topk)
- **observer.yaml**: Monitoring collector that exposes KPI metrics for the control loop
- **main-prod.yaml**: Production-optimized variant with enhanced security and performance settings

### Monitoring Stack
- **prometheus.yaml**: Prometheus scrape configuration for all Phoenix components
- **prometheus rules**: Recording rules that create derived metrics with the `phoenix:` prefix
- **grafana configs**: Datasource and dashboard provisioning configurations

### Templates
Templates provide starting points for new configurations and are used by initialization scripts. They should not be directly referenced by running services.

## Usage

1. **Active Configurations**: Services reference files directly from their respective directories (e.g., `configs/otel/collectors/main.yaml`)

2. **Templates**: Used by initialization scripts and for creating new configurations. Located in `configs/templates/`

3. **Dynamic Files**: `configs/control/optimization_mode.yaml` is modified at runtime by the control actuator

4. **Monitoring Rules**: Prometheus rules in `configs/monitoring/prometheus/rules/` create aggregated metrics for dashboards and alerting

## Best Practices

1. **Version Control**: All configuration changes should be committed with descriptive messages
2. **Testing**: Test configuration changes in development before applying to production
3. **Backup**: Keep backup versions of critical configurations (e.g., main-backup.yaml)
4. **Documentation**: Update this README when adding new configuration types or directories