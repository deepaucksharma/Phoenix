# Timeseries Estimator Processor

The `timeseries_estimator` processor tracks the number of unique time series
seen in the metrics pipeline.  It begins in **exact** mode where every
series key is stored in memory.  When the count of unique series reaches the
configured `max_unique_time_series` limit the processor automatically switches to
a probabilistic estimator based on HyperLogLog.  This provides a circuit breaker
to prevent excessive memory usage while still producing an estimate of the
active series count.

## Configuration

```yaml
processors:
  timeseries_estimator:
    enabled: true
    estimator_type: "exact"
    max_unique_time_series: 5000
```

- `estimator_type` – starting estimator. `"exact"` stores every series and is
  accurate until the limit is reached. `"hll"` starts directly in probabilistic
  mode.
- `max_unique_time_series` – maximum number of series tracked exactly before
  falling back to HyperLogLog.

The processor exposes a self‑metric `phoenix.timeseries_estimator.memory_bytes`
which records the current memory usage of the estimator.
