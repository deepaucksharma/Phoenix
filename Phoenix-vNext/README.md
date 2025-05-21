# Phoenix-vNext: 5-Pipeline Process Metric Optimization Lab

"Minimum process metric ingest ➜ Maximum NRDB value for host & process monitoring."

## Overview

Phoenix-vNext is an advanced observability lab for process metrics featuring a 5-pipeline architecture:

- Runs multiple `stress-ng` workloads for diverse host pressure.
- Utilizes a single OpenTelemetry Collector (`otelcol-main`) with five parallel process metric pipelines:
  - `metrics/full`: Baseline, minimal processing.
  - `metrics/opt`: "Moderate" optimization (aggregation, cardinality estimation).
  - `metrics/ultra`: "Ultra" optimization (more aggressive filtering and aggregation).
  - `metrics/exp`: Experimental, using `spacesaving` processor for Top-K.
  - `metrics/hybrid`: A balanced approach combining filtering, aggregation, and Top-K.
- The active mode for optimization can be toggled between "moderate" and "ultra" via `configs/control_signals/opt_mode.yaml`.
- Streams each pipeline's output to New Relic (via distinct API keys: `NR_FULL_KEY`, `NR_OPT_KEY`, `NR_ULTRA_KEY`, `NR_EXP_KEY`, `NR_HYBRID_KEY`).
- Includes `otelcol-observer` which monitors `otelcol-main` and updates `opt_mode.yaml` based on series count, implementing a feedback loop.
- Features new control mechanisms: `otelcol-main` can receive OTLP signals and monitors `opt_mode.yaml` as a log for changes.
- Exposes detailed metrics to local Prometheus & Grafana.

## Architecture

![Phoenix-vNext Architecture](https://i.imgur.com/nVsXjGm.png)

### Key Components

1. **Main Collector (otelcol-main)** - The central component with 5 distinct pipelines:
   - Scrapes host metrics and processes them through 5 different processing strategies
   - Uses a replicate connector to fan-out metrics to the 5 pipelines
   - Exports data to both New Relic and local Prometheus

2. **Observer Collector (otelcol-observer)** - The control plane:
   - Monitors the cardinality (active series count) of the main collector
   - Dynamically updates the control file (`opt_mode.yaml`)
   - Provides feedback on pipeline performance

3. **Control Signal File** - Enhanced with correlation and versioning:
   - Used by main collector to determine optimization modes
   - Contains metadata for tracking changes and stability

4. **Workload Generators**:
   - CPU stress test (`stress-cpu`)
   - I/O stress test (`stress-io`)
   - Generates consistent metrics for pipeline comparison

5. **Monitoring**:
   - Prometheus for metrics collection
   - Grafana for visualization
   - Comprehensive dashboard showing all 5 pipelines

## Quick Start

1. **Set Environment Variables:**
   Replace `YOUR_NR_..._KEY` with your New Relic Insert API keys.

   ```bash
   export NR_FULL_KEY="YOUR_NR_FULL_KEY"
   export NR_OPT_KEY="YOUR_NR_OPT_KEY"
   export NR_ULTRA_KEY="YOUR_NR_ULTRA_KEY"
   export NR_EXP_KEY="YOUR_NR_EXP_KEY"
   export NR_HYBRID_KEY="YOUR_NR_HYBRID_KEY"
   export BENCHMARK_ID="phoenix_vNext_5pipe_$(date +%s)" # Optional
   export DEPLOYMENT_ENV="development" # Optional
   export CORRELATION_ID="run_$(date +%s)" # Optional
   
   # Optional threshold configurations (defaults shown)
   export THRESHOLD_MODERATE=300.0
   export THRESHOLD_CAUTION=350.0
   export THRESHOLD_WARNING=400.0
   export THRESHOLD_ULTRA=450.0
   ```

2. **Start the Stack:**
   ```bash
   cd phoenix-bench
   docker compose up -d
   ```

## Accessing Services

- **Grafana:** `http://localhost:3000` (admin/admin) - Open the "Phoenix-vNext 5-Pipeline Dashboard".
- **Prometheus:** `http://localhost:9090`
  - Query `phoenix_opt_ts_active`, `phoenix_ultra_ts_active`, `phoenix_hybrid_ts_active`, `phoenix_exp_ts_active`
  - Query `otel_resource_attributes{otel_resource_observability_mode!=""}`
- **Otelcol-Main Health/Debug:**
  - Health Check: `http://localhost:13133`
  - pprof: `http://localhost:1777/debug/pprof/`
  - zPages: `http://localhost:55679/debug/zpages/`

## Testing Pipeline Behavior

1. **Monitor Mode Transitions:**
   - Watch the Grafana dashboard as the system automatically transitions between modes based on cardinality
   - View the correlation IDs in the Control File Status panel

2. **Manual Control Override:**
   - Edit `phoenix-bench/configs/control_signals/opt_mode.yaml` to manually change modes
   - Increment the `config_version` when making manual changes
   - Watch otelcol-main's reaction in the Grafana dashboard

3. **Pipeline Comparison:**
   - Compare metrics in the Active Time Series panel to see cardinality differences
   - Check New Relic Export Rate to see data volume differences between pipelines

## New Relic Comparison

Log in to New Relic to compare ingest and fidelity from the five pipelines:
- Use `benchmark.id = "<your_benchmark_id>"` to find your test run
- Filter by `pipeline.id = "{full|opt|ultra|exp|hybrid}"` to compare pipelines 
- Example NRQL: `SELECT count(*) FROM Metric WHERE benchmark.id = '${BENCHMARK_ID}' FACET pipeline.id TIMESERIES`

## Configurations

Key configuration aspects:

1. **Pipeline Processors:**
   - `full`: Minimal processing
   - `opt`: Moderate filtering, aggregation by host/service
   - `ultra`: Aggressive filtering, higher aggregation
   - `exp`: Space-saving Top-K algorithm for high cardinality metrics
   - `hybrid`: Balanced approach with selective filtering and Top-K

2. **Control File Schema:**
   ```yaml
   mode: "moderate"
   last_updated: "2025-05-22T10:00:00Z"
   config_version: 1
   correlation_id: "init-phoenix-vnext-20250522"
   optimization_level: 0
   thresholds:
     moderate: 300.0
     caution: 350.0
     warning: 400.0
     ultra: 450.0
   state:
     previous_mode: "initial"
     transition_timestamp: "2025-05-22T10:00:00Z"
     transition_duration_seconds: 0
     stability_period_seconds: 300
   ```

## Stopping the Stack

```bash
# From the phoenix-bench directory
docker compose down
docker compose down -v # To remove persistent data
```
