# Runbook: OpenTelemetry Collector Out of Memory (OOM)

## Alert Name
`CollectorOOM`

## Description
This alert fires when an OpenTelemetry Collector pod is killed due to exceeding memory limits or experiences repeated OOM events.

## Severity
**Critical**

## Impact
- Data loss for metrics during OOM period
- Increased load on remaining collectors
- Potential cascading failures
- Gaps in monitoring data

## Detection
Alert fires when:
- Pod restart with OOMKilled reason
- Memory usage > 90% of limit for 5 minutes
- Prometheus query: `container_memory_working_set_bytes{pod=~"otel-collector.*"} / container_spec_memory_limit_bytes > 0.9`

## Immediate Actions

### 1. Verify OOM Status
```bash
# Check pod status and restart reason
kubectl -n phoenix-vnext get pods -l app=otel-collector-main
kubectl -n phoenix-vnext describe pod <pod-name> | grep -A 10 "State:"

# Check events
kubectl -n phoenix-vnext get events --field-selector involvedObject.name=<pod-name> --sort-by='.lastTimestamp'
```

### 2. Emergency Scale Up
```bash
# Scale up deployment to handle load
kubectl -n phoenix-vnext scale deployment otel-collector-main --replicas=5

# Verify new pods are running
kubectl -n phoenix-vnext get pods -l app=otel-collector-main -w
```

### 3. Increase Memory Limits (Temporary)
```bash
# Patch deployment with higher memory limit
kubectl -n phoenix-vnext patch deployment otel-collector-main --type='json' -p='[
  {
    "op": "replace",
    "path": "/spec/template/spec/containers/0/resources/limits/memory",
    "value": "4Gi"
  },
  {
    "op": "replace",
    "path": "/spec/template/spec/containers/0/resources/requests/memory",
    "value": "2Gi"
  }
]'

# Monitor rollout
kubectl -n phoenix-vnext rollout status deployment/otel-collector-main
```

### 4. Enable Emergency Batching
```bash
# Apply aggressive batching to reduce memory usage
cat <<EOF | kubectl -n phoenix-vnext create configmap emergency-batch-config --from-file=config.yaml=/dev/stdin
processors:
  batch:
    send_batch_size: 1000  # Reduced from 10000
    timeout: 1s            # Reduced from 10s
    send_batch_max_size: 1000
  memory_limiter:
    check_interval: 1s
    limit_mib: 3072
    spike_limit_mib: 512
    ballast_size_mib: 0  # Disable ballast during emergency
EOF

# Update collector config
kubectl -n phoenix-vnext patch deployment otel-collector-main -p '{"spec":{"template":{"spec":{"containers":[{"name":"otel-collector","args":["--config=/etc/emergency/config.yaml"]}]}}}}'
```

## Root Cause Analysis

### 1. Analyze Memory Usage Pattern
```bash
# Get memory usage history
curl -s "http://prometheus:9090/api/v1/query_range?query=container_memory_working_set_bytes{pod=~'otel-collector.*'}&start=$(date -d '1 hour ago' +%s)&end=$(date +%s)&step=30s" | jq

# Check for memory leaks
kubectl -n phoenix-vnext exec <pod-name> -- curl -s http://localhost:1777/debug/pprof/heap > heap.pprof
go tool pprof -http=:8080 heap.pprof
```

### 2. Identify High-Volume Sources
```bash
# Check incoming metric rates
curl -s http://prometheus:9090/api/v1/query?query=rate(otelcol_receiver_accepted_metric_points[5m]) | jq

# Look for specific high-volume metrics
kubectl -n phoenix-vnext logs deployment/otel-collector-main --tail=1000 | grep -E "high_cardinality|large_batch"
```

### 3. Review Recent Changes
```bash
# Check for configuration changes
kubectl -n phoenix-vnext describe configmap otel-collector-main-config

# Review deployment history
kubectl -n phoenix-vnext rollout history deployment otel-collector-main
```

## Long-term Fixes

### 1. Optimize Collector Configuration

#### Memory Limiter Settings
```yaml
processors:
  memory_limiter:
    check_interval: 1s
    limit_percentage: 80      # Percentage of limit
    spike_limit_percentage: 20 # Headroom for spikes
    ballast_size_mib: 683     # 1/3 of 2GB limit
```

#### Batch Processor Tuning
```yaml
processors:
  batch:
    send_batch_size: 5000
    timeout: 5s
    send_batch_max_size: 10000
```

### 2. Implement Queue Management
```yaml
exporters:
  prometheusremotewrite:
    endpoint: http://prometheus:9090/api/v1/write
    sending_queue:
      enabled: true
      num_consumers: 10
      queue_size: 10000
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 300s
```

### 3. Resource Allocation Strategy
```yaml
# Production settings
resources:
  requests:
    memory: "2Gi"
    cpu: "1000m"
  limits:
    memory: "4Gi"
    cpu: "2000m"
```

## Prevention

### 1. Monitoring Setup
```yaml
# Add memory usage alerts
groups:
- name: collector_memory
  rules:
  - alert: CollectorHighMemory
    expr: container_memory_working_set_bytes{pod=~"otel-collector.*"} / container_spec_memory_limit_bytes > 0.8
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Collector {{ $labels.pod }} memory usage is high"
      
  - alert: CollectorMemoryTrend
    expr: predict_linear(container_memory_working_set_bytes{pod=~"otel-collector.*"}[30m], 3600) > container_spec_memory_limit_bytes
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "Collector {{ $labels.pod }} memory will exceed limit in 1 hour"
```

### 2. Load Testing
```bash
# Generate test load
docker run --rm -it \
  -e SYNTHETIC_HOST_COUNT=10 \
  -e SYNTHETIC_PROCESS_COUNT_PER_HOST=1000 \
  -e SYNTHETIC_METRIC_EMIT_INTERVAL_S=1 \
  phoenix-vnext/synthetic-generator
```

### 3. Capacity Planning
- Monitor growth trends
- Plan for 2x headroom
- Regular performance reviews
- Document metric growth projections

## Recovery Verification

### 1. Check System Health
```bash
# Verify all collectors are healthy
kubectl -n phoenix-vnext get pods -l component=collector
kubectl -n phoenix-vnext top pods | grep otel-collector

# Check metrics flow
curl -s http://prometheus:9090/api/v1/query?query=up{job=~"otel-collector.*"} | jq
```

### 2. Validate No Data Loss
```bash
# Check for gaps in data
curl -s "http://prometheus:9090/api/v1/query?query=prometheus_tsdb_sample_appends_total&start=$(date -d '1 hour ago' +%s)&end=$(date +%s)&step=60s" | jq
```

## Communication

### During Incident
1. Post in #phoenix-incidents with:
   - Affected collectors
   - Current status
   - ETA for resolution
2. Update status page if customer impact

### Post-Incident
1. Document timeline
2. Calculate data loss (if any)
3. Create follow-up tickets
4. Schedule post-mortem

## References
- [OTel Collector Performance](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/performance.md)
- [Memory Limiter Processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/memorylimiterprocessor)
- [Kubernetes Resource Management](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)