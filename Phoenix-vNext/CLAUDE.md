# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project: Phoenix-vNext

Phoenix-vNext is a production-grade, no-code observability lab that provides a New Relic-optimized benchmarking stack. The project focuses on "Minimum ingest ➜ Maximum NRDB value" by comparing different metric pipeline optimizations.

## Architecture Overview

The system consists of:

1. **Workload generators**: Uses stress-ng to simulate diverse host pressure
2. **OpenTelemetry Collector**: Runs three parallel metric pipelines:
   - `baseline_full`: Nearly raw data (truth source)
   - `opt_adaptive`: Aggressively filtered + adaptive Top-K, delta conversion, histogram bucketing
   - `exp_space_saving`: Experimental community Top-K processor (space-saving/HLL) for research
3. **Observer collector**: Implements a config-file feedback loop to toggle the optimization pipeline between moderate and ultra modes
4. **Monitoring components**: Prometheus and Grafana for local visualization

The pipelines stream data to separate New Relic accounts/datasets for comparison of ingest volume, dashboard fidelity, and alert accuracy.

## Key Components

- **Docker Compose Stack**: All components run as Docker containers
- **OpenTelemetry Collectors**:
  - `otelcol-main`: Implements the three parallel pipelines
  - `otelcol-observer`: Controls optimization mode based on active series count
- **Stress Workloads**:
  - `stress-cpu`: CPU-intensive workload
  - `stress-io`: IO-intensive workload
- **Monitoring**:
  - Prometheus: Collects metrics from both collectors
  - Grafana: Visualizes pipeline comparison

## Commands

### Run the Stack

```bash
# Set New Relic API keys
export NR_FULL_KEY=XXXX NR_OPT_KEY=YYYY NR_EXP_KEY=ZZZZ

# Create required directories
mkdir -p phoenix-bench/{data,configs/control_signals}

# Copy initial configuration
cp configs/control_signals/opt_mode.yaml phoenix-bench/configs/control_signals/opt_mode.yaml

# Start the containers
cd phoenix-bench
docker compose up -d
```

### Access Monitoring Interfaces

- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090

### Stop the Stack

```bash
docker compose down
```

## Configuration Files

1. **docker-compose.yaml**: Defines all services and their configurations
2. **configs/otelcol-main.yaml**: Configuration for the main collector with three pipelines
3. **configs/otelcol-observer.yaml**: Configuration for the control-plane collector
4. **configs/control_signals/opt_mode.yaml**: Control file for optimization mode

## Project Structure

```
phoenix-bench/
├── docker-compose.yaml
├── data/                     # ringbuffers, HLL state, etc.
├── configs/
│   ├── otelcol-main.yaml     # 3-pipeline collector
│   ├── otelcol-observer.yaml # feedback controller  
│   ├── prometheus.yaml       # scrapes both collectors
│   ├── grafana-datasource.yaml
│   ├── grafana-dashboard.json # starter board
│   └── control_signals/
│       └── opt_mode.yaml      # {mode: moderate|ultra}
```

## Development Guidelines

When modifying this codebase:

1. **Configuration Updates**: Any changes to the collector configurations should maintain the three-pipeline architecture for comparison
2. **Performance Testing**: After changes, compare the metrics in New Relic to evaluate the impact on ingest volume vs. fidelity
3. **Optimization Parameters**: Major tuning points include:
   - Filter patterns
   - Top-K values
   - Histogram bucket configurations
   - Stress load parameters

## Customization

The project is designed to be a testbed for optimization strategies. Common modifications include:
- Adjusting Top-K values in the processors
- Modifying filter patterns to exclude low-value metrics
- Changing histogram bucket boundaries
- Adjusting stress-ng parameters to simulate different workloads