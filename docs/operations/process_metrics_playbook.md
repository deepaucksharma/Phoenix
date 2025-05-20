# Process Metrics Operational Playbook

This playbook provides operational procedures for managing the process metrics collectors and processors in Phoenix. It covers common tasks, troubleshooting steps, and configuration optimization for production deployments.

## Table of Contents

1. [Processor Components Overview](#processor-components-overview)
2. [Common Operational Tasks](#common-operational-tasks)
   - [Adjusting Processor Configuration](#adjusting-processor-configuration)
   - [Tuning Memory Usage](#tuning-memory-usage)
   - [Monitoring Processor Health](#monitoring-processor-health)
3. [Troubleshooting](#troubleshooting)
   - [High Memory Usage](#high-memory-usage)
   - [Missing Histograms](#missing-histograms)
   - [Incorrect Cardinality Estimates](#incorrect-cardinality-estimates)
   - [Process State Loss](#process-state-loss)
4. [Performance Optimization](#performance-optimization)
5. [Deployment Scenarios](#deployment-scenarios)

## Processor Components Overview

The process metrics collection in Phoenix consists of two main processors:

1. **timeseries_estimator**: Estimates the number of unique time series being produced
   - Monitors cardinality in real-time
   - Supports exact counting and HyperLogLog algorithms
   - Self-adjusts under memory pressure

2. **cpu_histogram_converter**: Converts CPU time metrics to utilization histograms
   - Tracks process CPU usage over time
   - Generates distribution histograms for visualization
   - Persists state between restarts

## Common Operational Tasks

### Adjusting Processor Configuration

#### Changing Memory Limits

For `timeseries_estimator`:

```yaml
processors:
  timeseries_estimator:
    memory_limit_mb: 200  # Increase from default 100MB
```

For `cpu_histogram_converter`:

```yaml
processors:
  cpu_histogram_converter:
    max_processes_in_memory: 20000  # Increase from default 10000
```

#### Modifying Histogram Buckets

To adjust the histogram buckets for better visualization:

```yaml
processors:
  cpu_histogram_converter:
    histogram_buckets: [0.1, 1, 5, 10, 25, 50, 75, 100, 200, 400, 800]
```

#### Enabling State Persistence

To enable state persistence for the CPU histogram processor:

```yaml
processors:
  cpu_histogram_converter:
    state_storage_path: "/var/lib/sa-omf/cpu_state.json"
    state_flush_interval_seconds: 300
```

### Tuning Memory Usage

#### Memory-Constrained Environments

For low-memory environments:

1. Use HyperLogLog estimation instead of exact counting:
   ```yaml
   processors:
     timeseries_estimator:
       estimator_type: "hll"
       hll_precision: 8  # Lower precision uses less memory
   ```

2. Decrease process tracking limits:
   ```yaml
   processors:
     cpu_histogram_converter:
       max_processes_in_memory: 5000
   ```

3. Enable process eviction monitoring metrics to track memory pressure.

#### High-Cardinality Environments

For environments with many processes:

1. Enable in-memory only mode to avoid I/O overhead:
   ```yaml
   processors:
     cpu_histogram_converter:
       state_storage_path: ""  # Empty disables persistence
   ```

2. Focus on top-k processes only:
   ```yaml
   processors:
     cpu_histogram_converter:
       top_k_only: true
   ```

### Monitoring Processor Health

Create dashboard panels for the self-monitoring metrics:

1. **Timeseries Estimator Health**:
   - `phoenix.timeseries.estimate`: Estimated cardinality
   - `phoenix.timeseries.memory_usage_mb`: Memory usage
   - `phoenix.timeseries.memory_constrained`: Memory constraint status

2. **CPU Histogram Converter Health**:
   - `phoenix.cpu_histogram.processes_tracked`: Number of processes tracked
   - `phoenix.cpu_histogram.processing_time_ms`: Processing time per batch

## Troubleshooting

### High Memory Usage

**Symptoms**:
- Increasing memory usage in Phoenix
- `phoenix.timeseries.memory_constrained` metric shows `1`
- Process evictions occurring frequently

**Steps**:

1. Check current memory usage:
   ```bash
   ps -o pid,rss,command | grep sa-omf
   ```

2. Verify memory constraint status:
   ```bash
   curl http://localhost:8888/metrics | grep memory_constrained
   ```

3. Adjust memory limits in config:
   ```yaml
   processors:
     timeseries_estimator:
       estimator_type: "hll"  # Switch to HLL for lower memory
       memory_limit_mb: 200   # Increase limit if resources available
   ```

4. If using exact counting, switch to HLL temporarily:
   ```yaml
   processors:
     timeseries_estimator:
       estimator_type: "hll"
       hll_precision: 10
   ```

### Missing Histograms

**Symptoms**:
- CPU utilization histograms not appearing in dashboards
- No histogram metrics in output

**Steps**:

1. Verify input metrics exist:
   ```bash
   curl http://localhost:8888/metrics | grep process.cpu.time
   ```

2. Check for errors in logs:
   ```bash
   grep cpu_histogram /var/log/sa-omf.log
   ```

3. Verify processor configuration:
   ```yaml
   processors:
     cpu_histogram_converter:
       enabled: true
       input_metric_name: "process.cpu.time"  # Ensure matches input
       output_metric_name: "process.cpu.utilization.histogram"
   ```

4. Wait for second batch to arrive (first batch establishes baseline):
   - Histograms only generate after at least two batches of metrics
   - Default collection interval is 10 seconds

### Incorrect Cardinality Estimates

**Symptoms**:
- Cardinality estimates much lower or higher than expected
- Huge jumps in estimated values between runs

**Steps**:

1. Check current estimation mode:
   ```bash
   curl http://localhost:8888/metrics | grep phoenix.timeseries.mode
   ```

2. If memory constrained, increase limits:
   ```yaml
   processors:
     timeseries_estimator:
       memory_limit_mb: 300
   ```

3. For consistent but approximate estimates, use HLL with high precision:
   ```yaml
   processors:
     timeseries_estimator:
       estimator_type: "hll"
       hll_precision: 14  # Higher precision (max 16)
   ```

4. For accurate counts at the cost of memory, use exact counting:
   ```yaml
   processors:
     timeseries_estimator:
       estimator_type: "exact"
       memory_limit_mb: 500  # High memory limit
   ```

### Process State Loss

**Symptoms**:
- CPU utilization drops to zero after restart
- Gaps in histogram data after restarts

**Steps**:

1. Enable state persistence:
   ```yaml
   processors:
     cpu_histogram_converter:
       state_storage_path: "/var/lib/sa-omf/cpu_state.json"
       state_flush_interval_seconds: 300
   ```

2. Verify state file permissions:
   ```bash
   ls -la /var/lib/sa-omf/cpu_state.json
   ```

3. Ensure directory exists and is writable:
   ```bash
   mkdir -p /var/lib/sa-omf
   chown sa-omf:sa-omf /var/lib/sa-omf
   ```

4. Check logs for state loading failures:
   ```bash
   grep "state" /var/log/sa-omf.log
   ```

## Performance Optimization

For optimal performance in production:

1. **CPU Efficiency**:
   - Increase collection interval for less frequent processing:
     ```yaml
     processors:
       cpu_histogram_converter:
         collection_interval_seconds: 30  # Default is 10
     ```

2. **Memory Efficiency**:
   - Enable process eviction with appropriate limits:
     ```yaml
     processors:
       cpu_histogram_converter:
         max_processes_in_memory: 5000  # Adjust based on environment
     ```

3. **I/O Efficiency**:
   - Balance state persistence frequency:
     ```yaml
     processors:
       cpu_histogram_converter:
         state_flush_interval_seconds: 600  # Less frequent saves
     ```

4. **Visualizing CPU Patterns**:
   - Create a dashboard panel for the CPU utilization histogram:
     - New Relic NRQL: `SELECT histogram(process.cpu.utilization.histogram) FROM Metric`
     - Grafana: Use histogram visualization with process.cpu.utilization.histogram metric

## Deployment Scenarios

### Small Environment (< 100 processes)

```yaml
processors:
  timeseries_estimator:
    enabled: true
    estimator_type: "exact"  # Exact counting is fine for small environments
    memory_limit_mb: 50
    
  cpu_histogram_converter:
    enabled: true
    max_processes_in_memory: 200
    state_storage_path: "/var/lib/sa-omf/cpu_state.json"
```

### Medium Environment (100-1000 processes)

```yaml
processors:
  timeseries_estimator:
    enabled: true
    estimator_type: "hll"
    hll_precision: 12
    memory_limit_mb: 100
    
  cpu_histogram_converter:
    enabled: true
    top_k_only: true  # Focus on important processes
    max_processes_in_memory: 2000
    state_storage_path: "/var/lib/sa-omf/cpu_state.json"
```

### Large Environment (1000+ processes)

```yaml
processors:
  timeseries_estimator:
    enabled: true
    estimator_type: "hll"
    hll_precision: 10
    memory_limit_mb: 200
    refresh_interval: 3600s  # Longer refresh to handle scale
    
  cpu_histogram_converter:
    enabled: true
    top_k_only: true
    max_processes_in_memory: 10000
    collection_interval_seconds: 30  # Less frequent collection
    state_storage_path: "/var/lib/sa-omf/cpu_state.json"
    state_flush_interval_seconds: 1800  # Less frequent saves
```