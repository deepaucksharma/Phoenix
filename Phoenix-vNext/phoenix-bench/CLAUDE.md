# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## System Overview

Phoenix-vNext is an OpenTelemetry-based benchmarking system that implements 5 parallel processing pipelines to evaluate cardinality optimization strategies. The system uses dynamic control signals to switch between optimization modes based on real-time metrics cardinality.

### Core Architecture

- **Main Collector** (`otelcol-main`): Implements 5 parallel pipelines (full, opt, ultra, exp, hybrid) with dynamic mode switching
- **Observer Collector** (`otelcol-observer`): Monitors cardinality and generates control signals by writing to `opt_mode.yaml`
- **Control Signal System**: File-based control mechanism where observer writes and main collector reads mode changes
- **Synthetic Metrics Generator**: Creates test data for pipeline comparison
- **Monitoring Stack**: Prometheus + Grafana for visualization

### 5-Pipeline Strategy

1. **Full Pipeline**: Baseline with minimal processing (250 series target)
2. **Opt Pipeline**: Moderate optimization with filtering (150 series target)  
3. **Ultra Pipeline**: Aggressive optimization (50 series target)
4. **Exp Pipeline**: Experimental Top-K algorithms (100 series target)
5. **Hybrid Pipeline**: Balanced multi-technique approach (125 series target)

## Common Development Commands

### System Management
```bash
# Start the entire system
docker compose up -d

# Run comprehensive system tests
./test-phoenix-system.sh

# Check system status
docker compose ps

# View logs for specific services
docker logs phoenix-bench-otelcol-main-1
docker logs phoenix-bench-otelcol-observer-1
```

### Configuration Validation
```bash
# Validate YAML syntax
python3 -c "import yaml; yaml.safe_load(open('configs/collectors/otelcol-main.yaml'))"
python3 -c "import yaml; yaml.safe_load(open('configs/collectors/otelcol-observer.yaml'))"
python3 -c "import yaml; yaml.safe_load(open('configs/metrics/synthetic-metrics.yaml'))"
```

### Monitoring and Debugging
```bash
# Check main collector metrics
curl -s http://localhost:8888/metrics | grep phoenix

# Check observer metrics  
curl -s http://localhost:8891/metrics | grep phoenix_observer_mode

# Check current control mode
grep -E "^mode:" configs/control_signals/opt_mode.yaml

# Check synthetic metrics
curl -s http://localhost:9999/metrics | grep phoenix
```

## Key Configuration Files

### Control Signal System
- `configs/control_signals/opt_mode.yaml` - Dynamic control file (read by main, written by observer)
- Schema requires: `mode`, `last_updated`, `config_version`, `correlation_id`
- Valid modes: `moderate`, `adaptive`, `ultra`

### Collector Configurations
- `configs/collectors/otelcol-main.yaml` - Main 5-pipeline collector with config_sources for dynamic control
- `configs/collectors/otelcol-observer.yaml` - Observer that monitors cardinality and writes control signals
- `configs/metrics/synthetic-metrics.yaml` - Synthetic data generator for testing

### Thresholds and Environment
- Default thresholds: moderate=300, adaptive=375, ultra=450 series
- Environment variables in `.env` file: `NR_*_KEY` for New Relic export, `THRESHOLD_*` for mode switching
- `docker-compose.yaml` orchestrates all services with proper dependencies

## Control Flow Understanding

1. Observer scrapes metrics from main collector (port 8888)
2. Observer's `transform/control_file_generator` processor analyzes cardinality 
3. Observer writes new mode to `opt_mode.yaml` via file exporter
4. Main collector's `config_sources.ctlfile` detects file changes
5. Main collector applies new mode to all 5 pipelines simultaneously
6. `test-phoenix-system.sh` validates end-to-end coherence

## Port Assignments

- 8888: Main collector metrics & telemetry
- 8889: Main collector consistency endpoint  
- 8890: Main collector feedback endpoint
- 8891: Observer collector metrics
- 9999: Synthetic metrics generator
- 9090: Prometheus
- 3000: Grafana (admin/admin)
- 4317/4318: OTLP control receivers (main collector)
- 4319/4320: OTLP receivers (observer from main)

## Testing Strategy

The `test-phoenix-system.sh` script validates:
- Schema coherence between control file and collector configs
- Component health via HTTP endpoints
- Control signal propagation timing
- Pipeline metrics generation
- YAML configuration syntax

Always run the test suite after configuration changes to ensure system coherence.