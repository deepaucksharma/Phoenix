# Phoenix Processor Monitoring Guide

This guide provides detailed instructions for monitoring and troubleshooting the Phoenix processors using their self-metrics capabilities. It focuses on setting up proper monitoring and alerting for the `timeseries_estimator` and `cpu_histogram_converter` processors.

## Self-Metrics Overview

Both processors emit comprehensive self-monitoring metrics that can be used to:

1. Track processor performance
2. Monitor memory usage and constraints
3. Detect operational issues
4. Optimize configuration

## Metric Categories

### Timeseries Estimator Metrics

| Metric | Type | Description | Recommended Alerting |
|--------|------|-------------|----------------------|
| `phoenix.timeseries.estimate` | Gauge | Estimated number of unique time series | >100,000 (adjustable based on capacity) |
| `phoenix.timeseries.memory_usage_mb` | Gauge | Memory usage in MB | >80% of configured limit |
| `phoenix.timeseries.mode` | Gauge | Operating mode (0=exact, 1=hll) | Change in value (indicates fallback) |
| `phoenix.timeseries.memory_constrained` | Gauge | Memory constraint status | Value=1 for >10 minutes |

### CPU Histogram Converter Metrics

| Metric | Type | Description | Recommended Alerting |
|--------|------|-------------|----------------------|
| `phoenix.cpu_histogram.processes_tracked` | Gauge | Number of processes tracked | Sudden drops (>30%) |
| `phoenix.cpu_histogram.processes_processed` | Gauge | Processes processed in batch | Zero for multiple intervals |
| `phoenix.cpu_histogram.histograms_generated` | Gauge | Histograms generated | Zero for multiple intervals |
| `phoenix.cpu_histogram.processing_time_ms` | Gauge | Processing time in ms | >1000ms consistently |

## Dashboard Setup

### New Relic Dashboard

Create a monitoring dashboard with the following components:

1. **Cardinality Overview**:
   ```
   SELECT latest(phoenix.timeseries.estimate) FROM Metric FACET host
   ```

2. **Memory Status**:
   ```
   SELECT latest(phoenix.timeseries.memory_usage_mb) FROM Metric FACET host
   SELECT latest(phoenix.timeseries.memory_constrained) FROM Metric FACET host
   ```

3. **CPU Histogram Performance**:
   ```
   SELECT latest(phoenix.cpu_histogram.processes_tracked) FROM Metric FACET host
   SELECT latest(phoenix.cpu_histogram.processing_time_ms) FROM Metric FACET host
   ```

4. **Alert Conditions**:
   ```
   SELECT latest(phoenix.timeseries.memory_constrained) FROM Metric WHERE host = 'production-host'
   FACET host WHERE latest(phoenix.timeseries.memory_constrained) > 0
   ```

### Prometheus/Grafana Dashboard

For Prometheus users, create a Grafana dashboard with these panels:

1. **Time Series Cardinality**:
   ```
   rate(phoenix_timeseries_estimate{job="sa-omf"}[5m])
   ```

2. **Memory Usage**:
   ```
   phoenix_timeseries_memory_usage_mb{job="sa-omf"}
   ```

3. **Memory Constraint Status**:
   ```
   phoenix_timeseries_memory_constrained{job="sa-omf"}
   ```

4. **CPU Histogram Stats**:
   ```
   phoenix_cpu_histogram_processes_tracked{job="sa-omf"}
   phoenix_cpu_histogram_processing_time_ms{job="sa-omf"}
   ```

## Alerting Recommendations

### Critical Alerts

1. **Memory Constraint Status**:
   - Condition: `phoenix.timeseries.memory_constrained = 1` for >10 minutes
   - Action: Increase memory limit or switch to HLL mode

2. **Processing Time Spike**:
   - Condition: `phoenix.cpu_histogram.processing_time_ms > 1000` for 3 consecutive periods
   - Action: Check for high process count or resource contention

### Warning Alerts

1. **High Cardinality**:
   - Condition: `phoenix.timeseries.estimate` increasing >20% week-over-week
   - Action: Review filtering and sampling settings

2. **Tracking Capacity**:
   - Condition: `phoenix.cpu_histogram.processes_tracked > 0.9 * max_processes_in_memory`
   - Action: Increase `max_processes_in_memory` or enable `top_k_only`

## Monitoring Checklist

- [ ] Verify all processor metrics are being emitted
- [ ] Create dashboard for daily monitoring
- [ ] Set up alerts for critical conditions
- [ ] Document normal baseline values
- [ ] Schedule periodic review of metrics trends

## Troubleshooting with Metrics

### Issue: High Memory Usage

**Metrics to check**:
- `phoenix.timeseries.memory_usage_mb`: Current memory usage
- `phoenix.timeseries.memory_constrained`: Constraint status
- `phoenix.timeseries.mode`: Current operating mode

**Resolution**:
1. If `memory_constrained=1` and `mode=0`, switch to HLL mode
2. Increase `memory_limit_mb` if resources available
3. Check for sudden increases in `timeseries.estimate` indicating high cardinality

### Issue: Missing CPU Histograms

**Metrics to check**:
- `phoenix.cpu_histogram.processes_processed`: Should be >0
- `phoenix.cpu_histogram.histograms_generated`: Should be >0
- `phoenix.cpu_histogram.processing_time_ms`: Check for normal processing

**Resolution**:
1. Verify `processes_processed` >0, indicating data is flowing
2. Check logs for errors during histogram generation
3. Wait for at least two metric batches to establish baseline

## Integration with Monitoring Systems

### Prometheus Integration

Add these scrape configs to `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'sa-omf'
    scrape_interval: 30s
    static_configs:
      - targets: ['localhost:8888']
```

### New Relic Integration

Ensure these settings in your configuration:

```yaml
exporters:
  otlphttp/newrelic:
    endpoint: https://otlp.nr-data.net:4317
    headers:
      api-key: ${NEW_RELIC_LICENSE_KEY}
```

Ensure all Phoenix self-metrics are included in the pipeline.