# Others Rollup Processor

The Others Rollup processor aggregates metrics for lower-priority resources into a summarized form to reduce cardinality while preserving detailed metrics for high priority resources.

## Overview

This processor:

1. Identifies resources below a configurable priority threshold
2. Aggregates their metrics (sum, average, etc.) into consolidated metrics
3. Preserves individual metrics for higher priority resources
4. Applies additional attributes to indicate the aggregation

This approach significantly reduces the total number of metrics while maintaining detailed visibility for important resources.

## Configuration

```yaml
others_rollup:
  enabled: true
  priority_threshold: "low"  # Can be "low", "medium", "high", or "critical" 
  metric_name_prefix: "others"
  aggregation_method: "sum"  # Optional, defaults to "sum"
```

### Configuration Parameters

- `enabled`: Whether the processor is active
- `priority_threshold`: Resources with this priority or lower will be aggregated
  - Valid values: "low", "medium", "high", "critical"
  - Resources are tagged with priorities by the priority_tagger processor
- `metric_name_prefix`: Prefix added to aggregated metric names
- `aggregation_method`: How to combine metrics (sum, avg, min, max)

## How It Works

1. **Input Filtering**: The processor examines all incoming resource metrics
2. **Priority Check**: It looks for the `aemf.process.priority` attribute for each resource
3. **Aggregation**:
   - High-priority resources (above threshold): passed through unchanged
   - Low-priority resources (at or below threshold): aggregated together
4. **Output Format**:
   - Original metrics are replaced with aggregated versions
   - Aggregated metrics include a count of how many resources were combined
   - Each metric type (gauge, sum, etc.) is handled appropriately

## Metrics Emitted

| Metric Name | Description |
|-------------|-------------|
| `aemf_rollup_processor_resources_total` | Total number of resources processed |
| `aemf_rollup_processor_resources_aggregated` | Number of resources that were aggregated |
| `aemf_rollup_processor_cardinality_reduction_ratio` | Ratio of cardinality reduction achieved |

## Sample Output

For example, if we have CPU metrics from 100 different low-priority processes, instead of sending 100 separate metrics, we might produce:

```
others.process.cpu.utilization{count=100, min=0.01, max=0.85, priority="low"} 12.5
```

This represents the sum of CPU utilization from all 100 processes, with additional metadata on the count and ranges.

## Use Cases

- **Reducing storage costs**: Limit the total number of metrics stored
- **Improving query performance**: Fewer series means faster queries
- **Focus on high-value data**: Keep detailed data only for important resources
- **Dynamically adjustable**: Can be tuned at runtime via adaptive_pid processor

## Example

```yaml
# In the policy.yaml:
processors_config:
  others_rollup:
    enabled: true
    priority_threshold: "low"
    metric_name_prefix: "others"

# In the config.yaml pipeline:
service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [priority_tagger, adaptive_topk, others_rollup]
      exporters: [prometheusremotewrite]
```

## Integration with Adaptive Control

The others_rollup processor can be dynamically configured by the adaptive_pid processor by adjusting the priority_threshold. For example, under high load, you could raise the threshold to "medium" or even "high" to aggregate more metrics and reduce the load on the system.

```yaml
adaptive_pid_config:
  controllers:
    - name: rollup_controller
      enabled: true
      kpi_metric_name: aemf_processor_metrics_count
      kpi_target_value: 5000.0
      kp: 0.5
      ki: 0.1
      kd: 0
      output_config_patches:
        - target_processor_name: others_rollup
          parameter_path: priority_threshold
          change_scale_factor: 1.0
          min_value: 0  # Corresponds to "low" 
          max_value: 3  # Corresponds to "critical"
```