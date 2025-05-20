# New Relic OTLP Export Configuration

This guide explains how to configure Phoenix SA-OMF for optimal OTLP export to New Relic with a focus on process metrics.

## Overview

The configuration optimizes for:
1. **Low cardinality**: Ensures metric count stays within New Relic limits
2. **Process-level focus**: Collects only process metrics to reduce volume
3. **Adaptive filtering**: Prioritizes important processes using PID control
4. **Histogram optimization**: Ensures histograms are formatted optimally for New Relic

## Prerequisites

1. A New Relic account with Metrics API access
2. Your New Relic API key (available in your New Relic account)

## Configuration Steps

### 1. Set Environment Variable

```bash
export NEW_RELIC_API_KEY="your-new-relic-api-key"
```

### 2. Use OTLP Export Configuration

Use the prebuilt configuration in `configs/default/config.yaml` which includes:

- OTLP exporter configured for New Relic
- Process-only metrics collection
- Attribute filtering to control cardinality
- Histogram optimization for New Relic visualization
- Adaptive k-value control for top processes

### 3. Start the Collector

```bash
./bin/sa-omf-otelcol --config=configs/default/config.yaml
```

## Key Components

### Process-Only Metric Collection

The configuration restricts collection to process metrics only and filters attributes to include only the essential ones:

```yaml
receivers:
  hostmetrics:
    scrapers:
      process:
        include:
          match_type: regexp
          processes: [".*"]
        resource_attributes:
          process.executable.name: true
          process.pid: true
          process.owner: true
```

### Adaptive Processing Pipeline

The metrics pipeline uses these processors in sequence:

1. **priority_tagger**: Tags processes by importance (critical, high, medium, low)
2. **adaptive_topk**: Dynamically adjusts which processes pass through based on CPU usage
3. **others_rollup**: Aggregates low-priority processes into a single "others" metric
4. **histogram_aggregator**: Optimizes histograms for New Relic visualization
5. **attributes/process**: Filters and normalizes attributes to control cardinality

### PID Controllers

Two PID controllers manage adaptive behavior:

1. **coverage_controller**: Adjusts k-value to maintain 95% coverage of total CPU usage
2. **cardinality_controller**: Manages the rollup threshold to control total metric cardinality

## Cardinality Management

The configuration ensures metric cardinality stays within New Relic limits by:

1. Focusing only on process metrics
2. Including only necessary process attributes
3. Dynamically adapting which processes are individually tracked
4. Rolling up low-priority processes

## Monitoring and Tuning

Monitor the configuration through:

1. The self-metrics pipeline in Prometheus
2. Log output showing adaptive changes
3. New Relic's metric explorer showing incoming metrics

### Key Metrics to Watch

- `aemf_impact_adaptive_topk_resource_coverage_percent`: Should stay near 95%
- `aemf_metrics_cardinality`: Total unique metrics being exported
- `aemf_metrics_export_rate`: Rate of metrics being exported to New Relic

## Troubleshooting

### Common Issues

1. **High Cardinality Alerts**: If New Relic shows high cardinality alerts:
   - Decrease `k_max` in the adaptive_topk configuration
   - Increase priority threshold for `others_rollup`

2. **Missing Important Processes**: If key processes aren't visible:
   - Add them to the priority_tagger rules with "critical" priority
   - Increase `k_min` value in adaptive_topk

3. **Poor Histogram Visualization**: If histograms don't display well:
   - Adjust the `custom_boundaries` in histogram_aggregator processor

4. **Authentication Failures**:
   - Verify your NEW_RELIC_API_KEY environment variable is set correctly
   - Check network connectivity to New Relic's OTLP endpoint

## Further Reading

- [New Relic OTLP Documentation](https://docs.newrelic.com/docs/more-integrations/open-source-telemetry-integrations/opentelemetry/opentelemetry-setup/)
- [Phoenix SA-OMF Adaptive Processing Guide](../adaptive-processing.md)
- [Process Monitoring Best Practices](../guides/process-monitoring.md)