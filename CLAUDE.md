# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Phoenix-vNext is a 5-Pipeline Cardinality Optimization System for OpenTelemetry metrics collection and processing. The system uses adaptive cardinality management with dynamic switching between optimization modes (moderate, adaptive, ultra) based on metric volume thresholds.

## Architecture

The system consists of several key components:

### Core Components
- **Main Collector** (`otelcol-main`): Runs 5 parallel pipelines for different cardinality optimization levels
- **Observer Collector** (`otelcol-observer`): Control plane that monitors cardinality and writes optimization signals
- **Synthetic Metrics Generator**: Generates test metrics for benchmarking
- **Monitoring Stack**: Prometheus + Grafana for visualization

### Pipeline Structure
The main collector operates 5 pipelines:
1. **Full Pipeline** (baseline) - Complete metrics collection
2. **Process Full** - Process-focused metrics with full detail
3. **Process Opt** - Process-focused with moderate optimization
4. **Process Ultra** - Process-focused with maximum rollup/aggregation
5. **Experimental/Hybrid** - Testing ground for new optimization strategies

### Control Signal System
- Observer monitors metric cardinality in real-time
- Writes optimization mode to `/configs/control_signals/opt_mode.yaml`
- Main collector reads this file to adapt collection behavior
- Thresholds: moderate (300), adaptive (375), ultra (450) series

## Development Commands

### Running the System
```bash
# Start the full stack
cd Phoenix-vNext/phoenix-bench
docker-compose up -d

# View logs
docker-compose logs -f otelcol-main
docker-compose logs -f otelcol-observer

# Stop services
docker-compose down
```

### Monitoring
- **Grafana Dashboard**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Main Collector Metrics**: http://localhost:8888-8896 (various pipeline endpoints)
- **Health Checks**: http://localhost:13133, http://localhost:13134

### Configuration Structure
- `configs/collectors/`: OpenTelemetry collector configurations
- `configs/control_signals/`: Dynamic optimization mode files (read/write by system)
- `configs/processors/`: Reusable processor configurations
- `configs/monitoring/`: Prometheus and Grafana setup
- `configs/dashboards/`: Grafana dashboard definitions

### Testing Synthetic Data
```bash
# Run synthetic metrics generator
cd Phoenix-vNext/phoenix-bench/synthetic-metrics.sh
./direct-metrics-generator.sh
```

## Key Files

- `docker-compose.yaml`: Main orchestration file with all services
- `configs/collectors/otelcol-main.yaml`: Core collector configuration with 5 pipelines
- `configs/collectors/otelcol-observer.yaml`: Control plane observer configuration
- `configs/control_signals/opt_mode.yaml`: Dynamic optimization state (modified by observer)
- `process-cardinality-analysis.json`: Analysis output showing pipeline effectiveness

## Environment Configuration

Create `.env` file in `Phoenix-vNext/phoenix-bench/` with:
```
NR_FULL_KEY=your_newrelic_key_for_full_pipeline
NR_OPT_KEY=your_newrelic_key_for_opt_pipeline  
NR_ULTRA_KEY=your_newrelic_key_for_ultra_pipeline
```

## Cardinality Optimization

The system automatically switches between modes based on metric volume:
- **Mode transitions** are logged in control signals
- **Rollup configurations** in `configs/processors/multi_level_rollup.yaml`
- **Process filtering** excludes system processes (kworker, migration, etc.)