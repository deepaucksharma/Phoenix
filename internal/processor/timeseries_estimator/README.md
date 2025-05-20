# Timeseries Estimator Processor

The `timeseries_estimator` processor estimates the number of unique time series being processed by the Phoenix collector and outputs this estimate as a metric. This is valuable for monitoring cardinality and understanding the impact of filtering, rollup, and other processors on data volume.

## Features

- Provides accurate estimates of the number of active time series
- Supports two estimation methods:
  - `exact`: Precise counting with a memory-efficient hash map
  - `hll`: HyperLogLog probabilistic counting algorithm for minimal memory usage
- Implements memory safety through automatic fallback to HLL when memory pressure exceeds limits
- Emits self-monitoring metrics for memory usage and constrained states
- Configurable refreshing to periodically reset counters
- Dynamically reconfigurable via ConfigPatch interface

## Configuration

```yaml
processors:
  timeseries_estimator:
    enabled: true
    output_metric_name: aemf_estimated_active_timeseries
    estimator_type: hll  # "exact" or "hll"
    hll_precision: 10  # Only used with "hll" mode, values 4-16
    memory_limit_mb: 100  # Memory limit that triggers HLL fallback
    refresh_interval: 1h  # How often to reset counters
```

### Configuration Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | Enables or disables the processor |
| `output_metric_name` | string | `aemf_estimated_active_timeseries` | Name of the metric that will contain the estimate |
| `estimator_type` | string | `hll` | Algorithm to use: `exact` for precise counting, `hll` for HyperLogLog probabilistic counting |
| `hll_precision` | int | `10` | Precision for HLL algorithm (4-16); higher is more accurate but uses more memory |
| `memory_limit_mb` | int | `100` | Memory usage limit in MB; exceeding this triggers fallback to HLL |
| `refresh_interval` | duration | `1h` | How often the processor resets its counters |

## Memory Safety

The processor includes safety mechanisms to prevent excessive memory usage:

1. **Memory monitoring**: Tracks memory usage and compares against configured limits
2. **Automatic fallback**: Switches to HLL algorithm when memory pressure is high, even if exact counting is configured
3. **Periodic refresh**: Resets counters at configurable intervals to prevent unbounded growth

## Output Metrics

The processor generates a gauge metric with the configured name (default: `aemf_estimated_active_timeseries`) that contains the current estimate of unique time series.

## Self-Monitoring Metrics

The processor emits the following metrics about its own operation:

| Metric | Type | Description |
|--------|------|-------------|
| `phoenix.timeseries.estimate` | gauge | Current estimate of unique time series |
| `phoenix.timeseries.memory_usage_mb` | gauge | Memory usage of the processor in MB |
| `phoenix.timeseries.mode` | gauge | Current operating mode (0=exact, 1=hll) |
| `phoenix.timeseries.memory_constrained` | gauge | Indicates if memory usage is constrained (0=no, 1=yes) |

## Example Use Cases

- Monitor the cardinality of metrics being processed
- Track the effectiveness of cardinality reduction techniques
- Alert on unexpected cardinality increases that could affect costs
- Visualize time series growth trends over time

## Implementation Details

The processor identifies unique time series by examining:
- Metric name
- Resource attributes
- Metrics data point attributes

It supports all OpenTelemetry metric types and calculates estimates using either exact counting (for precision) or HyperLogLog (for memory efficiency).