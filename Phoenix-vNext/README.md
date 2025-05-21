# Phoenix-vNext: New Relic-Optimized Stack

"Minimum ingest ➜ Maximum NRDB value" — a production-grade, no-code observability lab.

## Overview

Phoenix-vNext is an observability lab that:

- Runs multiple stress-ng workloads to mimic diverse host pressure
- Forks three parallel metric pipelines inside one OpenTelemetry Collector:
  - `baseline_full` – nearly raw data (truth source)
  - `opt_adaptive` – aggressively filtered + adaptive Top-K, delta conversion, histogram bucketing
  - `exp_space_saving` – experimental community Top-K processor (space-saving/HLL) for research
- Streams each pipeline to its own New Relic account/dataset for comparison
- Includes a config-file feedback loop: the observer collector flips the opt pipeline between moderate and ultra modes to keep active-series ≤ 450/node

## Quick Start

```bash
# Set New Relic API keys for each pipeline
export NR_FULL_KEY=XXXX
export NR_OPT_KEY=YYYY
export NR_EXP_KEY=ZZZZ

# Navigate to the phoenix-bench directory
cd phoenix-bench

# Start the containers
docker compose up -d
```

## Accessing Dashboards

- **Grafana**: http://localhost:3000 (admin/admin)
  - The starter board shows ingest deltas between pipelines
- **Prometheus**: http://localhost:9090
  - Query `phoenix_ts_active` to see series counts

## New Relic Comparison

Use three accounts or three nerdgraph metricAPIKeys to compare:
- Ingest volume
- Dashboard fidelity
- Alert accuracy

## Customization

Ready to push boundaries? Tweak:
- Filters
- Top-K k values
- Histogram buckets
- Stress-ng load parameters

Then watch NR ingest vs. fidelity live.

## Architecture

All containers use upstream images. The only "unofficial" piece is the community spacesavingprocessor (Top-K + HLL) pulled from ghcr.io/langstack/otel-spacesaving-processor:0.2.1.

### Directory Structure

```
phoenix-bench/
├── docker-compose.yaml
├── data/                       # ringbuffers, HLL state, etc.
├── configs/
│   ├── otelcol-main.yaml       # 3-pipeline collector
│   ├── otelcol-observer.yaml   # feedback controller
│   ├── prometheus.yaml         # scrapes both collectors
│   ├── grafana-datasource.yaml
│   ├── grafana-dashboard.json  # starter board
│   └── control_signals/
│       └── opt_mode.yaml       # {mode: moderate|ultra}
```