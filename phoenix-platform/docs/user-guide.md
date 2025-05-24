# Phoenix Platform User Guide

## Table of Contents
1. [Getting Started](#getting-started)
2. [Creating Your First Experiment](#creating-your-first-experiment)
3. [Understanding Pipeline Templates](#understanding-pipeline-templates)
4. [Monitoring Experiments](#monitoring-experiments)
5. [Analyzing Results](#analyzing-results)
6. [Best Practices](#best-practices)

## Getting Started

### Prerequisites
- Access to Phoenix dashboard
- New Relic account with OTLP endpoint
- Kubernetes cluster with Phoenix installed
- Basic understanding of process metrics

### Accessing the Dashboard

1. Navigate to your Phoenix dashboard URL (e.g., `https://phoenix.example.com`)
2. Log in with your credentials
3. You'll see the main dashboard with:
   - Active experiments
   - Pipeline library
   - System health metrics

## Creating Your First Experiment

### Step 1: Choose Your Baseline

1. Click **"New Experiment"** button
2. Select a baseline pipeline (recommended: `process-baseline-v1`)
3. This will be your control group with no optimizations

### Step 2: Select Optimization Strategy

Choose from available optimization pipelines:

- **Priority Filter**: Keep only critical processes
- **Top-K**: Focus on top resource consumers
- **Aggregated**: Combine similar processes
- **Adaptive**: Dynamic optimization based on load

### Step 3: Configure Experiment

```yaml
name: "webserver-optimization-test"
description: "Reduce web server fleet metrics by 60%"
duration: "24h"
target_nodes:
  selector:
    app: "webserver"
    environment: "staging"
critical_processes:
  - "nginx"
  - "php-fpm"
  - "redis"
```

### Step 4: Visual Pipeline Builder

1. Drag processors from the palette
2. Connect them in sequence
3. Configure each processor:
   - **Memory Limiter**: Always first
   - **Transform**: Add classifications
   - **Filter**: Apply rules
   - **Batch**: Always last

### Step 5: Launch Experiment

1. Review configuration
2. Click **"Launch Experiment"**
3. Monitor deployment status
4. Wait for initial metrics (5-10 minutes)

## Understanding Pipeline Templates

### process-baseline-v1
```yaml
processors:
  - memory_limiter:
      limit_mib: 512
  - batch:
      send_batch_size: 1000
```
**Use Case**: Control group, full visibility

### process-priority-filter-v1
```yaml
processors:
  - memory_limiter
  - transform/classify:
      process.priority: |
        if name =~ "^(nginx|mysql|redis)" then "critical"
        else if name =~ "^(sshd|systemd)" then "high"
        else "low"
  - filter/priority:
      exclude:
        match_type: strict
        process.priority: "low"
  - batch
```
**Use Case**: Production systems with known critical processes

### process-topk-v1
```yaml
processors:
  - memory_limiter
  - transform/metrics:
      - context: datapoint
        statements:
          - set(attributes["cpu_rank"], CPU_USAGE_RANK())
  - filter/topk:
      top_k: 20
      metric_name: "process.cpu.utilization"
  - batch
```
**Use Case**: Development environments, resource debugging

### process-aggregated-v1
```yaml
processors:
  - memory_limiter
  - transform/normalize:
      - context: resource
        statements:
          - replace_pattern(attributes["process.name"], "chrome.*", "chrome")
          - replace_pattern(attributes["process.name"], "firefox.*", "firefox")
  - groupbyattrs:
      keys: ["process.name", "host.name"]
  - batch
```
**Use Case**: Desktop environments, many similar processes

## Monitoring Experiments

### Real-Time Metrics

The experiment dashboard shows:
- **Cardinality Reduction**: Percentage of series reduced
- **Critical Process Coverage**: Are all critical processes retained?
- **Resource Usage**: Collector CPU and memory
- **Error Rate**: Any data loss or errors

### Key Indicators

🟢 **Healthy Experiment**:
- Cardinality reduction > 40%
- Critical processes: 100%
- Collector CPU < 5%
- Zero errors

🟡 **Needs Attention**:
- Cardinality reduction < 20%
- Critical processes < 100%
- Collector CPU > 10%

🔴 **Failing Experiment**:
- No cardinality reduction
- Missing critical processes
- Collector crashing
- High error rate

## Analyzing Results

### Cost Analysis Dashboard

View estimated savings:
```
Baseline Cost:    $1,250/month
Optimized Cost:   $375/month
Savings:          $875/month (70%)
Annual Savings:   $10,500
```

### Performance Comparison

| Metric | Baseline | Optimized | Change |
|--------|----------|-----------|--------|
| Time Series | 50,000 | 12,500 | -75% |
| Ingestion Rate | 1M DPM | 250K DPM | -75% |
| Critical Processes | 25/25 | 25/25 | 0% |
| P99 Latency | 45ms | 42ms | -7% |

### Making Decisions

**Promote to Production When**:
- Cost savings meet target (>40%)
- All critical processes retained
- No increase in collector resource usage
- Stable for 24+ hours

**Iterate When**:
- Savings below target
- Missing some processes
- Need to adjust thresholds

**Abort When**:
- Critical processes missing
- Collector instability
- Unacceptable data loss

## Best Practices

### 1. Start Conservative
- Begin with priority-based filtering
- Define critical processes explicitly
- Test in staging first

### 2. Incremental Optimization
- Don't jump to aggressive filtering
- Add one optimization at a time
- Monitor for 24h before adding more

### 3. Critical Process Definition
```yaml
critical_processes:
  # Web tier
  - "nginx"
  - "apache"
  
  # App tier
  - "java"
  - "python"
  - "node"
  
  # Data tier
  - "mysql"
  - "postgres"
  - "redis"
  
  # Infrastructure
  - "dockerd"
  - "kubelet"
```

### 4. Testing Strategy
1. **Dev Environment**: Aggressive optimization
2. **Staging**: Moderate optimization
3. **Production**: Conservative with gradual rollout

### 5. Monitoring Hygiene
- Set up alerts for critical process visibility
- Monitor collector health metrics
- Review cost savings weekly
- Document optimization decisions

## Common Patterns

### Web Server Fleet
```yaml
strategy: "process-priority-filter-v1"
critical: ["nginx", "php-fpm", "redis"]
expected_reduction: "60-70%"
```

### Kubernetes Nodes
```yaml
strategy: "process-aggregated-v1"
aggregate: ["docker", "containerd", "runc"]
expected_reduction: "50-60%"
```

### Database Servers
```yaml
strategy: "process-topk-v1"
top_k: 50
focus: ["postgres", "pgbouncer"]
expected_reduction: "40-50%"
```

## Troubleshooting Quick Reference

| Issue | Check | Solution |
|-------|-------|----------|
| No cardinality reduction | Pipeline config | Verify filters are active |
| Missing processes | Filter rules | Adjust priority/threshold |
| High CPU usage | Batch size | Increase batch size |
| Slow dashboard | Time range | Reduce query window |

## Getting Help

- **Documentation**: Check pipeline configuration guide
- **Support**: #phoenix-support Slack channel
- **Issues**: GitHub issues for bugs
- **Office Hours**: Weekly Q&A sessions