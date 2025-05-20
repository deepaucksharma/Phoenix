# Metric Pipeline Self-Metrics

The `metric_pipeline` processor is instrumented with comprehensive self-metrics to enable control feedback, troubleshooting, and visualization. This document describes the self-metrics emitted by the processor and how they can be used.

## Overview

Self-metrics provide insights into the internal behavior of the `metric_pipeline` processor, especially its resource filtering, transformation operations, and performance characteristics. These metrics are crucial for:

1. **Control feedback**: Providing feedback to adaptive controllers (like `adaptive_pid`) to adjust parameters.
2. **Troubleshooting**: Diagnosing issues during development and deployment.
3. **Visualization**: Creating dashboards to monitor the processor's behavior.

## Metrics Emitted

### Resource Filtering Metrics

| Metric Name | Type | Description | Attributes |
|-------------|------|-------------|------------|
| `phoenix.filter.resources.total` | Gauge | Total number of resources processed by the filter | processor, processor_id, filter_strategy |
| `phoenix.filter.resources.included` | Gauge | Number of resources included after filtering | processor, processor_id, filter_strategy |
| `phoenix.filter.coverage_ratio` | Gauge | Ratio of included resources to total resources | processor, processor_id, filter_strategy |

### Priority Tagging Metrics

| Metric Name | Type | Description | Attributes |
|-------------|------|-------------|------------|
| `phoenix.priority_tagged.resources` | Gauge | Number of resources tagged with each priority level | processor, processor_id, filter_strategy, priority |

### Top-K Metrics

| Metric Name | Type | Description | Attributes |
|-------------|------|-------------|------------|
| `phoenix.topk.k_value` | Gauge | Current value of K in the topK algorithm | processor, processor_id, filter_strategy |
| `phoenix.topk.included_resources` | Gauge | Number of resources included in the top K set | processor, processor_id, filter_strategy |

### Rollup Metrics

| Metric Name | Type | Description | Attributes |
|-------------|------|-------------|------------|
| `phoenix.rollup.aggregated_resources` | Gauge | Number of resources aggregated in the rollup | processor, processor_id, filter_strategy |

### Transformation Metrics

| Metric Name | Type | Description | Attributes |
|-------------|------|-------------|------------|
| `phoenix.histogram.conversions` | Counter | Number of metrics converted to histograms | processor, processor_id, filter_strategy, metric_name |

### Performance Metrics

| Metric Name | Type | Description | Attributes |
|-------------|------|-------------|------------|
| `phoenix.processing.duration_ms` | Gauge | Time taken to process metrics batch in milliseconds | processor, processor_id, filter_strategy |

### Configuration Metrics

| Metric Name | Type | Description | Attributes |
|-------------|------|-------------|------------|
| `phoenix.config.patches` | Counter | Configuration patches applied | processor, processor_id, filter_strategy, parameter |

## Metric Collection and Export

Self-metrics are collected using the `UnifiedMetricsCollector` from the `pkg/metrics` package, which provides a cohesive approach to metrics collection with a fluent builder pattern. 

The metrics are emitted at the end of each batch processing cycle in the `ConsumeMetrics` method and exported via the standard Phoenix self-metrics pipeline. This ensures that all metrics are available for both real-time control and historical analysis.

## Using Self-Metrics for Control

The `metric_pipeline` self-metrics are particularly valuable for implementing control feedback:

### Coverage-Based Adaptation

The `phoenix.filter.coverage_ratio` metric indicates what percentage of total resources are included after filtering. This can be used by the `adaptive_pid` processor to adjust the following parameters:

```yaml
controllers:
  - name: topk_coverage_controller
    kp: 2.0
    ki: 0.5
    kd: 0.1
    setpoint: 0.95  # Target 95% coverage
    input_metric: "phoenix.filter.coverage_ratio"
    output_config_patches:
      - processor: "metric_pipeline"
        parameter: "resource_filter.topk.k_value"
        scale_factor: 10.0  # Amplify controller output
```

### Resource Volume Adaptation

The `phoenix.filter.resources.included` metric can drive adaptation based on absolute resource count:

```yaml
controllers:
  - name: resource_limit_controller
    kp: 1.5
    ki: 0.3
    kd: 0.0
    setpoint: 50  # Target 50 resources
    input_metric: "phoenix.filter.resources.included"
    output_config_patches:
      - processor: "metric_pipeline"
        parameter: "resource_filter.topk.k_value"
        scale_factor: 0.5  # Dampen controller output
```

## Troubleshooting with Self-Metrics

During development and troubleshooting, these metrics provide valuable insights:

1. **Low Coverage Ratio?** Check `phoenix.topk.k_value` to see if the K value is too low, or examine `phoenix.priority_tagged.resources` to verify if priority rules are matching as expected.

2. **Performance Issues?** Monitor `phoenix.processing.duration_ms` to identify processing bottlenecks, especially when working with large metric volumes.

3. **Rollup Behavior?** Use `phoenix.rollup.aggregated_resources` to verify that low-priority resources are being properly aggregated.

4. **Histogram Generation?** Review `phoenix.histogram.conversions` to ensure that metrics are being converted to histograms as expected.

## Visualizing Self-Metrics

Phoenix self-metrics can be visualized in New Relic dashboards or other monitoring solutions. Here's an example dashboard configuration for visualizing key metrics:

### Coverage Dashboard

```json
{
  "name": "Process Coverage Dashboard",
  "widgets": [
    {
      "title": "Resource Coverage Ratio",
      "visualization": "line",
      "nrql": "SELECT latest(phoenix.filter.coverage_ratio) FROM Metric FACET processor_id TIMESERIES"
    },
    {
      "title": "Top-K Value",
      "visualization": "line",
      "nrql": "SELECT latest(phoenix.topk.k_value) FROM Metric FACET processor_id TIMESERIES"
    },
    {
      "title": "Resources by Priority",
      "visualization": "stacked_bar",
      "nrql": "SELECT latest(phoenix.priority_tagged.resources) FROM Metric FACET priority TIMESERIES"
    }
  ]
}
```

### Performance Dashboard

```json
{
  "name": "Processor Performance Dashboard",
  "widgets": [
    {
      "title": "Processing Duration (ms)",
      "visualization": "line",
      "nrql": "SELECT average(phoenix.processing.duration_ms) FROM Metric FACET processor_id TIMESERIES"
    },
    {
      "title": "Resources Processed",
      "visualization": "line",
      "nrql": "SELECT latest(phoenix.filter.resources.total) FROM Metric FACET processor_id TIMESERIES"
    },
    {
      "title": "Histogram Conversions",
      "visualization": "line",
      "nrql": "SELECT rate(count(phoenix.histogram.conversions), 1 minute) FROM Metric FACET metric_name TIMESERIES"
    }
  ]
}
```

## Extending Self-Metrics

The self-metrics implementation in the `metric_pipeline` processor is designed to be extensible. To add new metrics:

1. Add new field(s) to track the metric state in the `processorImpl` struct
2. Register the new metric in the `initializeMetrics` method
3. Update the relevant processing methods to record metric values
4. Add the metric to the `emitMetrics` method to ensure it's emitted

## Conclusion

The self-metrics system in the `metric_pipeline` processor provides comprehensive visibility into its internal operation, enabling control feedback, troubleshooting, and visualization. By leveraging these metrics, you can ensure the processor is operating efficiently and adapt its behavior dynamically based on observed performance.