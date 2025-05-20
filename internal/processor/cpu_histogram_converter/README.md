# CPU Histogram Converter Processor

The `cpu_histogram_converter` processor converts cumulative CPU time metrics into CPU utilization histogram metrics. This enables better visualization of CPU usage distribution across processes and facilitates analysis of resource consumption patterns.

## Features

- Converts CPU time metrics to CPU utilization percentage metrics
- Generates histograms for visualizing the distribution of CPU utilization
- Persists state between restarts to handle cumulative metrics correctly
- Memory-efficient with automatic LRU process eviction
- Supports filtering to focus on top-K processes only
- Emits self-monitoring metrics for performance and health tracking

## Configuration

```yaml
processors:
  cpu_histogram_converter:
    enabled: true
    input_metric_name: process.cpu.time
    output_metric_name: process.cpu.utilization.histogram
    collection_interval_seconds: 10
    host_cpu_count: 0  # Auto-detect
    top_k_only: false
    histogram_buckets: [0.1, 0.5, 1, 2, 5, 10, 25, 50, 75, 100, 200, 400, 800]
    state_storage_path: "/var/lib/phoenix/cpu_state.json"
    state_flush_interval_seconds: 300
    max_processes_in_memory: 10000
```

### Configuration Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | Enables or disables the processor |
| `input_metric_name` | string | `process.cpu.time` | Name of the input CPU time metric to convert |
| `output_metric_name` | string | `process.cpu.utilization.histogram` | Name of the output histogram metric |
| `collection_interval_seconds` | int | `10` | Interval between metric collections, used to calculate utilization |
| `host_cpu_count` | int | `0` | Number of CPU cores; 0 means auto-detect |
| `top_k_only` | bool | `false` | Whether to only process processes in the top-K set |
| `histogram_buckets` | float[] | `[0.1, 0.5, 1, 2, 5, 10, 25, 50, 75, 100, 200, 400, 800]` | Bucket boundaries for histograms (percentage of one CPU core) |
| `state_storage_path` | string | `""` | Path to persist state between restarts; empty disables persistence |
| `state_flush_interval_seconds` | int | `300` | How often to flush state to disk |
| `max_processes_in_memory` | int | `10000` | Maximum number of processes to track in memory before eviction |

## State Management

The processor maintains state to calculate deltas between CPU time measurements:

1. **In-memory state**: Tracks last CPU time and timestamp for each process
2. **Persistent state**: Optionally saves state to disk for handling restarts
3. **Memory management**: Implements LRU eviction when exceeding process limits

## How It Works

1. **Delta calculation**: CPU utilization is calculated as the change in CPU time divided by the elapsed time
2. **Percentage conversion**: Converts to percentage of one CPU core (0-100% = one core, 200% = two cores)
3. **Bucketing**: Distributes values into histogram buckets defined in configuration
4. **Aggregation**: Generates a histogram showing the distribution of CPU utilization

## Self-Monitoring Metrics

The processor emits the following metrics about its own operation:

| Metric | Type | Description |
|--------|------|-------------|
| `phoenix.cpu_histogram.processes_tracked` | gauge | Number of processes being tracked in memory |
| `phoenix.cpu_histogram.processes_processed` | gauge | Number of processes processed in each batch |
| `phoenix.cpu_histogram.histograms_generated` | gauge | Number of histograms generated per batch |
| `phoenix.cpu_histogram.processing_time_ms` | gauge | Time taken to process each batch in milliseconds |

## Example Use Cases

- Visualize CPU utilization distribution across all processes
- Identify processes consuming disproportionate CPU resources
- Track CPU usage patterns over time with histogram heatmaps
- Generate alerts based on excessive CPU utilization

## Implementation Details

The processor uniquely identifies processes using a combination of:
- Process name (`process.executable.name` or `process.name`)
- Process ID (`process.pid`)

It supports multi-core systems and adjusts calculations based on the configured or detected CPU count.