# Phoenix-vNext: New Relic-Optimized 5-Pipeline Process Metric Stack

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
- The active mode for `opt` vs `ultra` can be influenced by `configs/control_signals/opt_mode.yaml`.
- Streams each pipeline's output to New Relic (via distinct API keys: `NR_FULL_KEY`, `NR_OPT_KEY`, `NR_ULTRA_KEY`, `NR_EXP_KEY`, `NR_HYBRID_KEY`).
- Includes `otelcol-observer` which monitors `otelcol-main` and attempts to update `opt_mode.yaml` based on series count, implementing a feedback loop.
- Features new control mechanisms: `otelcol-main` can receive OTLP signals and monitors `opt_mode.yaml` as a log for changes.
- Exposes detailed metrics to local Prometheus & Grafana.

## Quick Start

1.  **Set Environment Variables:**
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
    ```

2.  **Prepare Directories and Initial Control File:**
    (The `phoenix-bench/data` directory will be created by Docker Compose.)

    ```bash
    # Navigate to the Phoenix-vNext directory
    # cd Phoenix-vNext

    mkdir -p phoenix-bench/configs/control_signals
    cat <<EOF > phoenix-bench/configs/control_signals/opt_mode.yaml
    mode: "moderate"
    last_updated: "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
    config_version: 1
    correlation_id: "initial-$(date +%s)"
    EOF
    ```

3.  **Navigate to the `phoenix-bench` directory:**
    ```bash
    cd phoenix-bench
    ```

4.  **Start the Stack:**
    ```bash
    docker compose up -d
    ```

## Accessing Services

-   **Grafana:** `http://localhost:3000` (admin/admin) - "Phoenix-vNext 5-Pipeline Dashboard".
-   **Prometheus:** `http://localhost:9090`
    -   Query `phoenix_opt_ts_active`, `phoenix_ultra_ts_active`, `phoenix_hybrid_ts_active`.
    -   Query `otel_resource_attributes{otel_resource_observability_mode!=""}`.
-   **Otelcol-Main Health/Debug:**
    -   Health Check: `http://localhost:13133`
    -   pprof: `http://localhost:1777/debug/pprof/`
    -   zPages: `http://localhost:55679/debug/zpages/`

## New Relic Comparison

Log in to New Relic to compare ingest and fidelity from the five pipelines. Use `benchmark.id = "<your_benchmark_id>"` and `pipeline.id = "{full|opt|ultra|exp|hybrid}"`.

## Customization & Experimentation

-   Manually edit `phoenix-bench/configs/control_signals/opt_mode.yaml` to change `mode`, `config_version`, `correlation_id`. `otelcol-main` and `otelcol-observer` will react.
-   Adjust `otelcol-main.yaml` and `otelcol-observer.yaml` configurations.

## Stopping the Stack

```bash
# From the phoenix-bench directory
docker compose down
docker compose down -v # To remove persistent data
```