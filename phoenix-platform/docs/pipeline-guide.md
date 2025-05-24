# Phoenix Pipeline Configuration Guide

## Overview

Phoenix pipelines are OpenTelemetry Collector configurations optimized for process metrics. This guide covers pipeline architecture, available processors, and configuration best practices.

## Pipeline Architecture

```yaml
receivers:
  hostmetrics:
    # Collect process metrics from host

processors:
  memory_limiter:     # Always first - prevent OOM
  transform:          # Classify and enrich
  filter:            # Remove unwanted metrics
  groupbyattrs:      # Aggregate similar processes
  batch:             # Always last - optimize exports

exporters:
  otlphttp/newrelic:  # Send to New Relic
  prometheus:         # Local metrics

service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [memory_limiter, transform, filter, batch]
      exporters: [otlphttp/newrelic, prometheus]
```

## Core Components

### Receivers

#### hostmetrics Receiver

```yaml
receivers:
  hostmetrics:
    collection_interval: 10s
    root_path: /hostfs  # For containerized collectors
    scrapers:
      process:
        mute_process_name_error: true
        mute_process_exe_error: true
        mute_process_io_error: true
        metrics:
          process.cpu.time:
            enabled: true
          process.cpu.utilization:
            enabled: true
          process.memory.physical:
            enabled: true
          process.memory.virtual:
            enabled: true
          process.disk.io:
            enabled: true
          process.threads:
            enabled: true
          process.open_file_descriptors:
            enabled: true
```

### Processors

#### 1. Memory Limiter (Required)

Always include first to prevent collector OOM:

```yaml
processors:
  memory_limiter:
    check_interval: 1s
    limit_mib: 512
    spike_limit_mib: 128
```

#### 2. Transform Processor

Add classifications and calculated fields:

```yaml
processors:
  transform/classify_priority:
    metric_statements:
      - context: resource
        statements:
          # Classify by process name
          - set(attributes["process.priority"], "critical") 
            where attributes["process.executable.name"] =~ "^(nginx|mysql|postgres|redis)$"
          - set(attributes["process.priority"], "high") 
            where attributes["process.executable.name"] =~ "^(docker|kubelet|sshd)$"
          - set(attributes["process.priority"], "low") 
            where attributes["process.priority"] == nil
          
  transform/add_metadata:
    metric_statements:
      - context: resource
        statements:
          # Add environment tag
          - set(attributes["environment"], "production") 
            where attributes["host.name"] =~ "^prod-"
          # Normalize process names
          - replace_pattern(attributes["process.command"], "^/usr/bin/", "")
```

#### 3. Filter Processor

Remove unwanted metrics:

```yaml
processors:
  # Drop low-priority processes
  filter/priority:
    metrics:
      datapoint:
        - 'attributes["process.priority"] == "low"'
  
  # Keep only high consumers
  filter/resource_usage:
    metrics:
      datapoint:
        - 'value < 0.01 and name == "process.cpu.utilization"'
        - 'value < 10485760 and name == "process.memory.physical"'  # 10MB
  
  # Drop specific processes
  filter/exclude_system:
    metrics:
      resource:
        - 'attributes["process.executable.name"] =~ "^(kernel|systemd-|snapd)"'
```

#### 4. Group By Attributes

Aggregate similar processes:

```yaml
processors:
  groupbyattrs/aggregate_browsers:
    keys:
      - host.name
      - process.group  # Created by transform
    aggregation_type: sum
    
  transform/pre_aggregate:
    metric_statements:
      - context: resource
        statements:
          # Group browser processes
          - set(attributes["process.group"], "chrome") 
            where attributes["process.executable.name"] =~ "^chrome"
          - set(attributes["process.group"], "firefox") 
            where attributes["process.executable.name"] =~ "^firefox"
          # Keep original name for non-grouped
          - set(attributes["process.group"], attributes["process.executable.name"]) 
            where attributes["process.group"] == nil
```

#### 5. Resource Detection

Add host metadata:

```yaml
processors:
  resourcedetection:
    detectors: 
      - env
      - system
      - docker
      - ec2
      - gcp
      - azure
    system:
      hostname_sources: ["os", "dns"]
    timeout: 2s
    override: false
```

#### 6. Cumulative to Delta

Convert cumulative metrics:

```yaml
processors:
  cumulativetodelta:
    include:
      metrics:
        - process.cpu.time
        - process.disk.io
    max_staleness: 10m
```

#### 7. Batch Processor (Required)

Always include last for efficiency:

```yaml
processors:
  batch:
    send_batch_size: 1000
    send_batch_max_size: 1500
    timeout: 5s
```

### Exporters

#### New Relic OTLP Exporter

```yaml
exporters:
  otlphttp/newrelic:
    endpoint: https://otlp.nr-data.net:4318
    headers:
      api-key: ${NEW_RELIC_API_KEY}
    compression: gzip
    timeout: 30s
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 300s
    sending_queue:
      enabled: true
      num_consumers: 10
      queue_size: 1000
```

#### Prometheus Exporter

```yaml
exporters:
  prometheus:
    endpoint: "0.0.0.0:8888"
    namespace: phoenix
    const_labels:
      pipeline: "process-optimized"
    resource_to_telemetry_conversion:
      enabled: true
```

## Pipeline Templates

### 1. Baseline (No Optimization)

```yaml
# process-baseline-v1.yaml
processors:
  - memory_limiter:
      limit_mib: 512
  - resourcedetection
  - cumulativetodelta
  - batch:
      send_batch_size: 1000

service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [memory_limiter, resourcedetection, cumulativetodelta, batch]
      exporters: [otlphttp/newrelic, prometheus]
```

### 2. Priority-Based Filtering

```yaml
# process-priority-filter-v1.yaml
processors:
  - memory_limiter
  - transform/classify:
      metric_statements:
        - context: resource
          statements:
            - set(attributes["process.priority"], "critical") 
              where attributes["process.executable.name"] in ["nginx", "mysql", "redis"]
  - filter/priority:
      metrics:
        resource:
          - 'attributes["process.priority"] != "critical" and attributes["process.priority"] != "high"'
  - batch
```

### 3. Top-K Resource Consumers

```yaml
# process-topk-v1.yaml
processors:
  - memory_limiter
  - filter/top_consumers:
      metrics:
        datapoint:
          # Keep top CPU consumers
          - 'name == "process.cpu.utilization" and value < 1.0'
          # Keep top memory consumers (100MB threshold)
          - 'name == "process.memory.physical" and value < 104857600'
  - batch
```

### 4. Aggregated Processes

```yaml
# process-aggregated-v1.yaml
processors:
  - memory_limiter
  - transform/normalize:
      metric_statements:
        - context: resource
          statements:
            # Normalize Chrome variants
            - replace_pattern(attributes["process.executable.name"], "^Google Chrome.*", "chrome")
            - replace_pattern(attributes["process.executable.name"], "^firefox.*", "firefox")
  - groupbyattrs:
      keys: [process.executable.name, host.name]
  - batch
```

## Advanced Configurations

### Dynamic Filtering with Processor Chains

```yaml
processors:
  # Stage 1: Enrich with metadata
  transform/enrich:
    metric_statements:
      - context: resource
        statements:
          - set(attributes["process.age_seconds"], 
              (now() - attributes["process.create_time"]) / 1000000000)
          
  # Stage 2: Apply business logic
  filter/business_rules:
    metrics:
      resource:
        # Keep all processes younger than 5 minutes
        - 'attributes["process.age_seconds"] > 300 and attributes["process.priority"] == "low"'
        
  # Stage 3: Smart sampling for remaining
  probabilistic_sampler:
    hash_seed: 22
    sampling_percentage: 10.0
```

### Conditional Processing

```yaml
processors:
  routing/by_environment:
    from_attribute: "environment"
    default_pipelines: ["general"]
    table:
      - value: "production"
        pipelines: ["critical_only"]
      - value: "development"
        pipelines: ["verbose"]
        
  filter/critical_only:
    metrics:
      resource:
        - 'attributes["process.priority"] != "critical"'
        
  filter/verbose:
    # Keep everything in dev
    metrics: {}
```

### Cost-Optimized Pipeline

```yaml
processors:
  # Calculate cost score
  transform/cost_score:
    metric_statements:
      - context: datapoint
        statements:
          - set(attributes["cost_score"], 
              attributes["cardinality_contribution"] * 0.7 + 
              attributes["ingestion_rate"] * 0.3)
  
  # Filter by cost score
  filter/cost_based:
    metrics:
      datapoint:
        - 'attributes["cost_score"] < 0.1'
```

## Best Practices

### 1. Processor Ordering

Always follow this order:
1. `memory_limiter` - Prevent OOM
2. `resourcedetection` - Add metadata early
3. `transform` - Enrich and classify
4. `filter` - Remove unwanted data
5. `groupbyattrs` - Aggregate
6. `batch` - Optimize exports

### 2. Critical Process Protection

```yaml
# Never filter these without explicit override
critical_processes:
  databases:
    - postgres
    - mysql
    - mongodb
    - redis
  web_servers:
    - nginx
    - apache
    - caddy
  app_servers:
    - java
    - python
    - node
    - ruby
  infrastructure:
    - dockerd
    - containerd
    - kubelet
```

### 3. Performance Optimization

```yaml
# Optimize for high-volume environments
processors:
  batch:
    send_batch_size: 5000      # Larger batches
    timeout: 10s               # Longer timeout
    
  memory_limiter:
    limit_mib: 1024           # More memory for processing
    spike_limit_mib: 256      # Handle spikes
    
exporters:
  otlphttp/newrelic:
    sending_queue:
      num_consumers: 20       # More parallel exports
      queue_size: 5000        # Larger buffer
```

### 4. Debugging Pipelines

```yaml
# Add debug logging
exporters:
  logging/debug:
    loglevel: debug
    sampling_initial: 10
    sampling_thereafter: 100
    
service:
  pipelines:
    metrics/debug:
      receivers: [hostmetrics]
      processors: [memory_limiter, transform/classify]
      exporters: [logging/debug]
```

### 5. Gradual Rollout

Start conservative:
```yaml
# Week 1: Baseline
processors: [memory_limiter, batch]

# Week 2: Add classification
processors: [memory_limiter, transform/classify, batch]

# Week 3: Enable filtering
processors: [memory_limiter, transform/classify, filter/priority, batch]

# Week 4: Full optimization
processors: [memory_limiter, transform/classify, filter/priority, groupbyattrs, batch]
```

## Validation Checklist

Before deploying a pipeline:

- [ ] Memory limiter is first processor
- [ ] Batch processor is last
- [ ] Critical processes are explicitly protected
- [ ] Resource detection is configured
- [ ] Exporters have proper authentication
- [ ] Timeout values are reasonable
- [ ] Error handling is configured
- [ ] Metrics retention is tested

## Common Issues

### High Memory Usage

```yaml
# Fix: Reduce batch sizes and add stricter limits
processors:
  memory_limiter:
    limit_mib: 256  # Lower limit
  batch:
    send_batch_size: 500  # Smaller batches
```

### Missing Processes

```yaml
# Fix: Check filter conditions
processors:
  filter/debug:
    metrics:
      # Temporarily disable filtering
      resource: []
```

### Export Failures

```yaml
# Fix: Add retry and queue configuration
exporters:
  otlphttp/newrelic:
    retry_on_failure:
      enabled: true
      max_elapsed_time: 600s
    sending_queue:
      enabled: true
      persistent_storage_enabled: true
```