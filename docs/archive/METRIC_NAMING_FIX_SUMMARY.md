# Phoenix Metric Naming Fix Summary

## Issue
The CLAUDE.md documentation referenced recording rule metrics using colon notation (e.g., `phoenix:signal_preservation_score`) while the actual Prometheus rules used underscore notation (e.g., `phoenix_signal_preservation_score`).

## Metrics That Were Missing/Mismatched

### Documentation Referenced (with `:`)
- `phoenix:signal_preservation_score`
- `phoenix:cardinality_efficiency_ratio`
- `phoenix:pipeline_latency_ms_p99`
- `phoenix:pipeline_throughput_metrics_per_sec`
- `phoenix:control_stability_score`
- `phoenix:control_loop_effectiveness`
- `phoenix:cardinality_zscore`
- `phoenix:cardinality_explosion_risk`
- `phoenix:resource_efficiency_score`
- `phoenix:collector_memory_usage_mb`
- `phoenix:cardinality_growth_rate`

### What Existed in Rules (with `_`)
- `phoenix_signal_preservation_score` (in phoenix_rules.yml and phoenix_core_rules.yml)
- `phoenix_cardinality_growth_rate` (in phoenix_core_rules.yml)
- `phoenix_control_loop_stability_score` (in phoenix_core_rules.yml)
- `phoenix_pipeline_efficiency_ratio` (in phoenix_core_rules.yml)

### Advanced Rules (already had `:` notation)
- `phoenix:cardinality_zscore`
- `phoenix:cardinality_growth_rate`
- `phoenix:cardinality_explosion_risk`
- Several other metrics in phoenix_advanced_rules.yml

## Resolution

Created a new file `/configs/monitoring/prometheus/rules/phoenix_documented_metrics.yml` that:

1. **Adds all missing metrics** referenced in CLAUDE.md with proper colon notation
2. **Provides proper expressions** for each metric based on available OpenTelemetry collector metrics
3. **Maintains compatibility** with existing rules (doesn't remove underscore versions)

## Key Metrics Added

1. **phoenix:signal_preservation_score** - Measures metric preservation (1 - drop rate)
2. **phoenix:cardinality_efficiency_ratio** - Measures cardinality reduction effectiveness
3. **phoenix:pipeline_latency_ms_p99** - 99th percentile latency in milliseconds
4. **phoenix:pipeline_throughput_metrics_per_sec** - Metrics processed per second
5. **phoenix:control_stability_score** - Control loop stability (based on mode change frequency)
6. **phoenix:control_loop_effectiveness** - How well control maintains target cardinality
7. **phoenix:resource_efficiency_score** - Combined memory and CPU efficiency
8. **phoenix:collector_memory_usage_mb** - Collector memory usage in MB

## Integration

The new rules file is automatically loaded by Prometheus because:
- Prometheus config loads all files from `/etc/prometheus/rules/*.yml`
- Docker compose mounts `./configs/monitoring/prometheus/rules:/etc/prometheus/rules:ro`
- The new file follows the naming pattern `phoenix_*.yml`

## Testing

To verify the metrics are working:

```bash
# Restart Prometheus to load new rules
docker-compose restart prometheus

# Query the new metrics
curl -s http://localhost:9090/api/v1/query?query=phoenix:signal_preservation_score
curl -s http://localhost:9090/api/v1/query?query=phoenix:cardinality_efficiency_ratio
curl -s http://localhost:9090/api/v1/query?query=phoenix:control_stability_score

# Check all phoenix: metrics
curl -s http://localhost:9090/api/v1/label/__name__/values | jq -r '.data[]' | grep "^phoenix:"
```

## Notes

- Both underscore and colon notations now exist to maintain backward compatibility
- The colon notation is preferred for consistency with Prometheus naming conventions
- Some metrics require the system to be running with data flowing to produce values
- The expressions use safe defaults (clamp_min, vector fallbacks) to avoid division by zero