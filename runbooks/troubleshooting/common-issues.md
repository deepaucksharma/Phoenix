# Phoenix-vNext Common Issues and Troubleshooting Guide

## Table of Contents
1. [Collector Issues](#collector-issues)
2. [Pipeline Issues](#pipeline-issues)
3. [Control Loop Issues](#control-loop-issues)
4. [Prometheus Issues](#prometheus-issues)
5. [Performance Issues](#performance-issues)
6. [Data Quality Issues](#data-quality-issues)

## Collector Issues

### Issue: Collector Pods Crash Looping
**Symptoms:**
- Pods in `CrashLoopBackOff` state
- Repeated restarts

**Diagnosis:**
```bash
# Check pod status
kubectl -n phoenix-vnext get pods -l component=collector

# View logs
kubectl -n phoenix-vnext logs <pod-name> --previous

# Check events
kubectl -n phoenix-vnext describe pod <pod-name>
```

**Common Causes & Solutions:**

1. **Configuration Error**
   ```bash
   # Validate configuration
   kubectl -n phoenix-vnext get configmap otel-collector-main-config -o yaml | yq e '.data."config.yaml"' - > /tmp/config.yaml
   otelcol validate --config /tmp/config.yaml
   ```

2. **Port Conflicts**
   ```bash
   # Check for port conflicts
   kubectl -n phoenix-vnext get svc
   netstat -tulpn | grep -E "4317|4318|8888"
   ```

3. **Resource Limits**
   ```bash
   # Increase limits temporarily
   kubectl -n phoenix-vnext patch deployment otel-collector-main --type='json' -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/resources/limits/memory", "value": "2Gi"}]'
   ```

### Issue: Collector Not Receiving Data
**Symptoms:**
- No metrics in Prometheus
- Zero accepted metric points

**Diagnosis:**
```bash
# Check collector metrics
curl http://localhost:8888/metrics | grep otelcol_receiver_accepted_metric_points

# Test connectivity
telnet otel-collector-main 4317
curl -v http://otel-collector-main:4318/v1/metrics
```

**Solutions:**
1. **Network Policy Issues**
   ```bash
   # Check network policies
   kubectl -n phoenix-vnext get networkpolicy
   
   # Temporarily disable (dev only)
   kubectl -n phoenix-vnext delete networkpolicy --all
   ```

2. **Service Discovery**
   ```bash
   # Verify service endpoints
   kubectl -n phoenix-vnext get endpoints otel-collector-main
   ```

## Pipeline Issues

### Issue: High Cardinality in Specific Pipeline
**Symptoms:**
- Alert: `HighCardinality`
- Memory pressure
- Slow queries

**Diagnosis:**
```promql
# Check cardinality by pipeline
phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate

# Find high-cardinality metrics
topk(20, count by (__name__)({__name__=~".+"}))
```

**Solutions:**
1. **Apply Emergency Filters**
   ```yaml
   processors:
     filter:
       metrics:
         exclude:
           match_type: regexp
           metric_names: [".*_debug.*", ".*_trace.*"]
   ```

2. **Enable Aggregation**
   ```yaml
   processors:
     metricstransform:
       transforms:
         - include: ".*"
           match_type: regexp
           action: update
           operations:
             - action: aggregate_labels
               label_set: [service, region, status]
   ```

### Issue: Pipeline Dropping Metrics
**Symptoms:**
- Refused metric points > 0
- Missing expected metrics

**Diagnosis:**
```bash
# Check drop reasons
kubectl -n phoenix-vnext logs deployment/otel-collector-main | grep -i "drop\|refuse\|reject"

# Monitor refused metrics
curl -s http://localhost:8888/metrics | grep refused
```

**Solutions:**
1. **Check Filters**
   ```bash
   # Review filter configuration
   kubectl -n phoenix-vnext get configmap otel-collector-main-config -o yaml | grep -A20 "filter:"
   ```

2. **Validate Metric Names**
   ```bash
   # List all metric names
   curl -s http://prometheus:9090/api/v1/label/__name__/values | jq
   ```

## Control Loop Issues

### Issue: Control Loop Not Updating Optimization Mode
**Symptoms:**
- Optimization mode stuck
- No mode changes despite threshold crossing

**Diagnosis:**
```bash
# Check control loop logs
kubectl -n phoenix-vnext logs deployment/control-actuator --tail=100

# Verify control file updates
kubectl -n phoenix-vnext exec deployment/control-actuator -- cat /config/optimization_mode.yaml

# Check Prometheus connectivity
kubectl -n phoenix-vnext exec deployment/control-actuator -- curl -s http://prometheus:9090/api/v1/query?query=up
```

**Solutions:**
1. **Fix Prometheus Query**
   ```bash
   # Test query manually
   QUERY='sum(phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate{pipeline="metrics/optimised"})'
   curl -s "http://prometheus:9090/api/v1/query?query=$QUERY"
   ```

2. **Reset Control State**
   ```bash
   # Force mode update
   kubectl -n phoenix-vnext exec deployment/control-actuator -- sh -c 'echo "optimization_mode: balanced" > /config/optimization_mode.yaml'
   ```

### Issue: Rapid Mode Oscillation
**Symptoms:**
- Frequent mode changes
- Instability in metrics

**Solutions:**
1. **Increase Stability Period**
   ```bash
   kubectl -n phoenix-vnext set env deployment/control-actuator ADAPTIVE_CONTROLLER_STABILITY_SECONDS=300
   ```

2. **Adjust Thresholds**
   ```bash
   kubectl -n phoenix-vnext set env deployment/control-actuator \
     THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS=12000 \
     THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS=28000
   ```

## Prometheus Issues

### Issue: Prometheus High Memory Usage
**Symptoms:**
- OOM kills
- Slow queries
- High cardinality warnings

**Diagnosis:**
```bash
# Check memory usage
kubectl -n phoenix-vnext top pod prometheus-0

# Analyze TSDB
kubectl -n phoenix-vnext exec prometheus-0 -- promtool tsdb analyze /prometheus

# Check cardinality
curl -s http://prometheus:9090/api/v1/query?query=prometheus_tsdb_cardinality | jq
```

**Solutions:**
1. **Increase Retention Limits**
   ```bash
   # Reduce retention
   kubectl -n phoenix-vnext patch statefulset prometheus --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--storage.tsdb.retention.time=3d"}]'
   ```

2. **Enable Admin API for Cleanup**
   ```bash
   # Delete specific series
   curl -X POST -g 'http://prometheus:9090/api/v1/admin/tsdb/delete_series?match[]={__name__=~"test_.*"}'
   ```

### Issue: Prometheus Scrape Failures
**Symptoms:**
- `up` metric shows 0
- Missing targets

**Diagnosis:**
```bash
# Check scrape targets
curl http://prometheus:9090/api/v1/targets | jq '.data.activeTargets[] | {job: .labels.job, health: .health}'

# View scrape errors
curl http://prometheus:9090/api/v1/targets | jq '.data.activeTargets[] | select(.health != "up")'
```

## Performance Issues

### Issue: Slow Metric Queries
**Symptoms:**
- Dashboard timeouts
- Query latency > 5s

**Diagnosis:**
```promql
# Check query performance
histogram_quantile(0.99, prometheus_engine_query_duration_seconds_bucket)

# Find slow queries
topk(10, prometheus_engine_query_duration_seconds_sum)
```

**Solutions:**
1. **Optimize Queries**
   ```promql
   # Bad: Unbounded query
   sum(rate(http_requests_total[5m]))
   
   # Good: Bounded by job
   sum by (job) (rate(http_requests_total{job="api"}[5m]))
   ```

2. **Use Recording Rules**
   ```yaml
   groups:
     - name: performance
       interval: 30s
       rules:
         - record: job:http_requests:rate5m
           expr: sum by (job) (rate(http_requests_total[5m]))
   ```

### Issue: Dashboard Loading Slowly
**Solutions:**
1. **Reduce Query Complexity**
   - Use longer intervals for historical data
   - Limit number of series per panel
   - Use recording rules for complex calculations

2. **Enable Caching**
   ```bash
   kubectl -n phoenix-vnext set env deployment/grafana GF_CACHING_ENABLED=true
   ```

## Data Quality Issues

### Issue: Missing Metrics
**Symptoms:**
- Gaps in graphs
- Metrics appear/disappear

**Diagnosis:**
```bash
# Check for gaps
curl -s "http://prometheus:9090/api/v1/query_range?query=up{job='otel-collector-main'}&start=$(date -d '1 hour ago' +%s)&end=$(date +%s)&step=15s" | jq '.data.result[0].values' | grep -E "\[.*,\"0\"\]"

# Verify data flow
watch -n 5 'curl -s http://prometheus:9090/api/v1/query?query=rate(prometheus_tsdb_samples_appended_total[1m]) | jq'
```

**Solutions:**
1. **Check Batch Settings**
   ```yaml
   processors:
     batch:
       timeout: 5s  # Reduce from 10s
       send_batch_size: 5000  # Reduce if dropping
   ```

2. **Monitor Pipeline Health**
   ```bash
   # Create dashboard for pipeline health
   cat <<EOF | kubectl apply -f -
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: pipeline-health-dashboard
     namespace: phoenix-vnext
   data:
     dashboard.json: |
       {
         "title": "Pipeline Health",
         "panels": [
           {
             "targets": [
               {
                 "expr": "rate(otelcol_processor_dropped_metric_points[5m])"
               }
             ]
           }
         ]
       }
   EOF
   ```

### Issue: Incorrect Metric Values
**Symptoms:**
- Unexpected spikes/drops
- Values don't match source

**Solutions:**
1. **Verify Transformations**
   ```bash
   # Check processor configuration
   kubectl -n phoenix-vnext get configmap otel-collector-main-config -o yaml | grep -A50 "processors:"
   ```

2. **Enable Debug Logging**
   ```bash
   kubectl -n phoenix-vnext set env deployment/otel-collector-main OTEL_LOG_LEVEL=debug
   ```

## General Debugging Tips

### Enable Debug Mode
```bash
# Collector debug
kubectl -n phoenix-vnext patch deployment otel-collector-main --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--log-level=debug"}]'

# Prometheus debug
kubectl -n phoenix-vnext patch statefulset prometheus --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--log.level=debug"}]'
```

### Useful Debug Endpoints
```bash
# Collector
curl http://localhost:8888/metrics          # Internal metrics
curl http://localhost:55679/debug/pprof/    # Profiling
curl http://localhost:13133/               # Health check

# Prometheus  
curl http://prometheus:9090/api/v1/label/__name__/values  # All metrics
curl http://prometheus:9090/api/v1/targets                 # Scrape targets
curl http://prometheus:9090/metrics                        # Internal metrics
```

### Emergency Recovery
```bash
# Full restart
kubectl -n phoenix-vnext delete pods --all

# Reset to defaults
kubectl -n phoenix-vnext delete configmap --all
kubectl apply -k k8s/base/

# Backup before changes
kubectl -n phoenix-vnext get all,cm,secret -o yaml > backup.yaml
```

## Getting Help

1. **Check Logs First**
   - Collector logs
   - Prometheus logs
   - Control actuator logs

2. **Gather Information**
   ```bash
   # System state snapshot
   kubectl -n phoenix-vnext get all
   kubectl -n phoenix-vnext top pods
   kubectl -n phoenix-vnext describe pods
   ```

3. **Contact Support**
   - Include error messages
   - Provide configuration
   - Share relevant metrics/dashboards