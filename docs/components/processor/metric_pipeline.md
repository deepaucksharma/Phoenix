# Metric Pipeline Processor

The `metric_pipeline` processor is a unified processor that combines resource filtering and metric transformation into a single processing step. It replaces multiple separate processors (priority_tagger, adaptive_topk, others_rollup, and attribute processors) to improve performance and simplify configuration.

## Architecture

The `metric_pipeline` processor is designed with a two-stage processing approach:

1. **Resource Filtering**: Filters and prioritizes resources (e.g., processes) based on configurable rules
   - Priority tagging: assigns priority levels to resources based on pattern matching
   - Top-K selection: identifies the most important resources based on a metric value
   - Hybrid approach: combines priority and top-K for optimal filtering
   - Rollup aggregation: aggregates metrics from filtered-out resources to maintain visibility

2. **Metric Transformation**: Transforms metrics to optimize for backend storage and visualization
   - Histogram generation: converts metrics to histograms for better visualization
   - Attribute processing: manages resource attributes to control cardinality

## Configuration

The processor is configured through two main sections:

### 1. Resource Filtering

```yaml
resource_filter:
  enabled: true
  filter_strategy: hybrid  # priority, topk, or hybrid
  priority_attribute: "aemf.process.priority"
  
  # Priority rules for tagging resources
  priority_rules:
    - match: "process.executable.name=~/java|javaw/"
      priority: high
    - match: "process.executable.name=~/nginx|httpd|apache2/"
      priority: high
    - match: "process.executable.name=~/mysql|postgres|mongod|redis-server|elasticsearch/"
      priority: critical
    - match: ".*"
      priority: low
  
  # TopK configuration
  topk:
    k_value: 20
    k_min: 10
    k_max: 40
    resource_field: "process.executable.name"
    counter_field: "process.cpu.time"
    coverage_target: 0.95
  
  # Rollup configuration
  rollup:
    enabled: true
    priority_threshold: low  # Rollup priority levels <= this threshold
    strategy: sum  # sum or avg
    name_prefix: "phoenix.others.process"
```

### 2. Metric Transformation

```yaml
transformation:
  # Histogram generation
  histograms:
    enabled: true
    max_buckets: 10
    metrics:
      process.cpu.time:
        boundaries: [0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0]
      process.memory.rss:
        boundaries: [1048576, 10485760, 104857600, 524288000, 1073741824, 2147483648]
  
  # Attribute processing
  attributes:
    actions:
      - key: "process.pid"
        action: "delete"
      - key: "collector.name"
        action: "insert"
        value: "SA-OMF"
```

## Filter Strategies

The processor supports three filtering strategies:

1. **Priority Strategy** (`priority`): Resources are filtered based solely on priority levels assigned via pattern matching rules. Only resources with priority levels above a configurable threshold are included individually.

2. **Top-K Strategy** (`topk`): Resources are ranked by a specified metric value (e.g., CPU usage), and only the top K resources are included individually. The K value can be statically configured or adaptively adjusted via the adaptive_pid processor.

3. **Hybrid Strategy** (`hybrid`): Combines both approaches - resources are first tagged with priority levels, then top-K selection is applied within each priority level or globally. This provides the best balance of capturing important resources while managing cardinality.

## Rollup Aggregation

Resources that are filtered out (not included individually) can be aggregated into rollup metrics. The rollup feature:

- Aggregates metrics for resources below a configurable priority threshold
- Uses a configurable aggregation strategy (sum or average)
- Prefixes rollup metrics with a configurable name prefix
- Maintains visibility for lower-priority resources without excessive cardinality

## Histogram Generation

The processor can convert individual metrics into histograms for better visualization in backends like New Relic. The histogram feature:

- Converts configured metrics into histograms with customizable buckets
- Preserves the original metrics alongside the histogram versions
- Optimizes for time-series databases that support histograms natively

## Attribute Processing

To control cardinality and enhance metrics for specific backends, the processor can modify resource attributes:

- Delete high-cardinality attributes (e.g., process.pid, process.command_line)
- Insert standard attributes (e.g., collector.name, service.name)
- Update attribute values to normalize formats
- Apply conditional attribute transformations

## Self-Metrics

The processor emits self-metrics to monitor its behavior and drive adaptive adjustments:

- **Resource Filter Metrics**:
  - `phoenix.priority_tagged_resources`: Resources tagged by priority level
  - `phoenix.topk.k_value`: Current K value used for filtering
  - `phoenix.topk.included_resources`: Resources included in the top-K set
  - `phoenix.filter.included_resources`: Total resources included after filtering
  - `phoenix.filter.coverage_percent`: Percentage of total metric value included
  - `phoenix.rollup.aggregated_resources`: Resources aggregated into rollup

- **Transformation Metrics**:
  - `phoenix.histogram.conversions`: Metrics converted to histograms
  - `phoenix.attribute.modifications`: Attribute modifications performed

These metrics can be used by adaptive controllers (like adaptive_pid) to dynamically adjust the processor's parameters (e.g., K value, rollup threshold) based on changing workloads.

## Dynamic Configuration

The processor implements the `UpdateableProcessor` interface, allowing its configuration to be dynamically updated at runtime. This enables:

- Adaptive adjustment of K values based on coverage metrics
- Threshold adjustments for prioritization
- On-the-fly changes to attribute handling and histogram configuration

Configuration patches can be applied programmatically or through the pic_control extension.

## Performance Considerations

The unified `metric_pipeline` processor offers significant performance benefits over separate processors:

- Reduced data copying between processors
- Single-pass filtering and transformation
- Optimized memory usage by avoiding intermediate metrics collections
- Benchmark tests show [X%] improvement for typical workloads

## Use Cases

The processor is particularly well-suited for:

1. **Process Metrics Collection**: Efficiently collects and filters process metrics, focusing on the most important processes while maintaining visibility via rollups.

2. **New Relic Integration**: Optimized for sending process metrics to New Relic with appropriate attribute handling, histogram generation, and cardinality control.

3. **Adaptive Monitoring**: When combined with the adaptive_pid processor, it provides a self-tuning monitoring solution that adjusts to changing environments.

## Example Use Case: Process Metrics for New Relic

```yaml
processors:
  metric_pipeline:
    resource_filter:
      enabled: true
      filter_strategy: hybrid
      priority_attribute: "aemf.process.priority"
      priority_rules:
        - match: "process.executable.name=~/java|javaw/"
          priority: high
        - match: "process.executable.name=~/mysql|postgres|mongod|redis-server|elasticsearch/"
          priority: critical
        - match: ".*"
          priority: low
      topk:
        k_value: 25
        k_min: 10
        k_max: 50
        resource_field: "process.executable.name"
        counter_field: "process.cpu.time"
        coverage_target: 0.95
      rollup:
        enabled: true
        priority_threshold: low
        strategy: sum
        name_prefix: "phoenix.others.process"
    
    transformation:
      histograms:
        enabled: true
        max_buckets: 10
        metrics:
          process.cpu.time:
            boundaries: [0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0]
          process.memory.rss:
            boundaries: [1048576, 10485760, 104857600, 524288000, 1073741824, 2147483648]
      
      attributes:
        actions:
          - key: "process.pid"
            action: "delete"
          - key: "process.command_line"
            action: "delete"
          - key: "container.id"
            action: "delete"
          - key: "collector.name"
            action: "insert"
            value: "Phoenix-SA-OMF"
          - key: "service.name"
            action: "insert"
            value: "phoenix-process-metrics"
          - key: "phoenix.priority"
            action: "insert"
            value: {{ .Resource.Attributes.GetString "aemf.process.priority" "unknown" }}
```

## Comparison with Individual Processors

The `metric_pipeline` processor replaces and consolidates the following processors:

1. **priority_tagger**: 
   - *Before*: Tagged resources with priority levels based on regex patterns.
   - *Now*: Handled by the `resource_filter` with `priority_rules`.

2. **adaptive_topk**:
   - *Before*: Selected top K resources based on a metric value.
   - *Now*: Handled by the `resource_filter` with `topk` configuration, with K values adjustable via adaptive_pid.

3. **others_rollup**:
   - *Before*: Aggregated metrics for non-priority processes.
   - *Now*: Handled by the `resource_filter` with `rollup` configuration.

4. **attributes** processor and **histogram_aggregator**:
   - *Before*: Separate processors for attribute manipulation and histogram generation.
   - *Now*: Integrated into the `transformation` section of the metric_pipeline.