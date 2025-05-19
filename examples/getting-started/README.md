# Getting Started with SA-OMF

This example provides a minimal configuration to get started with SA-OMF. It demonstrates:

1. Basic collector setup with host metrics collection
2. Simple adaptive processing configuration
3. Local visualization with Prometheus and Grafana

## Prerequisites

- Go 1.21 or later
- Docker and Docker Compose
- Make

## Files

- **config.yaml**: Minimal OpenTelemetry Collector configuration
- **policy.yaml**: Basic control policy
- **docker-compose.yaml**: Local development environment with Prometheus and Grafana
- **run.sh**: Script to run the example
- **cleanup.sh**: Script to clean up after running

## Running the Example

1. Build the collector (if you haven't already):
   ```bash
   cd ../../
   make build
   ```

2. Start the monitoring stack:
   ```bash
   ./run.sh
   ```

3. Access the dashboards:
   - Grafana: http://localhost:3000 (admin/admin)
   - Prometheus: http://localhost:9090

4. Generate some load (optional):
   ```bash
   ./generate-load.sh
   ```

5. Observe the system adapting as metrics flow through the pipeline

6. Clean up when finished:
   ```bash
   ./cleanup.sh
   ```

## What to Look For

1. Watch the console output for adaptation logs
2. In Grafana, open the "SA-OMF Overview" dashboard to see:
   - Metrics processing rate
   - Adaptive parameter changes
   - Resource usage

3. In Prometheus, query:
   - `saomf_adaptive_topk_k_value` to see the k-value change over time
   - `saomf_metrics_processed_total` to see throughput

## Next Steps

After this example, explore:
1. [Processor Configuration Examples](../processors/)
2. [Production Deployment Examples](../integrations/)
3. [Custom Component Examples](../custom-adapters/)