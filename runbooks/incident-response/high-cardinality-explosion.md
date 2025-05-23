# Runbook: High Cardinality Explosion

## Alert Name
`HighCardinalityExplosion`

## Description
This alert fires when the cardinality of metrics in any pipeline exceeds critical thresholds, indicating a potential cardinality explosion that could impact system performance and cost.

## Severity
**Critical**

## Impact
- Increased memory usage in collectors and Prometheus
- Slower query performance
- Increased storage costs
- Potential OOM crashes
- Degraded dashboard performance

## Detection
Alert fires when:
```promql
phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate > 50000
```

## Immediate Actions

### 1. Verify the Alert
```bash
# Check current cardinality across all pipelines
curl -s http://prometheus:9090/api/v1/query?query=phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate | jq

# Check rate of increase
curl -s http://prometheus:9090/api/v1/query?query=rate(phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate[5m]) | jq
```

### 2. Identify High-Cardinality Metrics
```bash
# Top 10 metrics by cardinality
curl -s http://prometheus:9090/api/v1/query?query=topk(10,phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate) | jq

# Check specific metric patterns
kubectl exec -n phoenix-vnext prometheus-0 -- promtool tsdb analyze /prometheus
```

### 3. Emergency Mitigation

#### Option A: Force Aggressive Optimization
```bash
# Override control actuator to aggressive mode
kubectl -n phoenix-vnext create configmap emergency-control --from-literal=optimization_mode=aggressive --dry-run=client -o yaml | kubectl apply -f -

# Patch control actuator to use emergency config
kubectl -n phoenix-vnext patch deployment control-actuator -p '{"spec":{"template":{"spec":{"volumes":[{"name":"control-config","configMap":{"name":"emergency-control"}}]}}}}'
```

#### Option B: Enable Emergency Sampling
```bash
# Apply emergency sampling configuration
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: emergency-sampling
  namespace: phoenix-vnext
data:
  sampling.yaml: |
    processors:
      probabilistic_sampler:
        sampling_percentage: 10
EOF

# Restart collectors to apply
kubectl -n phoenix-vnext rollout restart deployment otel-collector-main
```

### 4. Monitor Recovery
```bash
# Watch cardinality metrics
watch -n 5 'curl -s http://prometheus:9090/api/v1/query?query=phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate | jq ".data.result[]|{pipeline:.metric.pipeline,value:.value[1]}"'

# Monitor memory usage
kubectl -n phoenix-vnext top pods | grep otel-collector
```

## Root Cause Analysis

### 1. Check Recent Changes
```bash
# Review recent deployments
kubectl -n phoenix-vnext rollout history deployment otel-collector-main

# Check for new instrumentation
git log --oneline -n 20 -- configs/
```

### 2. Analyze Metric Patterns
```bash
# Export metric names for analysis
curl -s http://prometheus:9090/api/v1/label/__name__/values > /tmp/metric_names.json

# Look for patterns
cat /tmp/metric_names.json | jq -r '.data[]' | grep -E "(test|debug|temp)" | sort | uniq -c
```

### 3. Identify Source
```bash
# Check which services are sending high-cardinality metrics
kubectl -n phoenix-vnext logs deployment/otel-collector-main --tail=1000 | grep -E "received|metric_name"
```

## Long-term Fixes

### 1. Update Metric Instrumentation
- Review application code for unbounded label values
- Remove high-cardinality labels (user IDs, request IDs, timestamps)
- Use bounded label values (status codes, regions, service names)

### 2. Configure Metric Filters
```yaml
processors:
  filter:
    metrics:
      exclude:
        match_type: regexp
        metric_names:
          - ".*_debug_.*"
          - ".*_test_.*"
```

### 3. Implement Cardinality Limits
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

## Prevention

1. **Pre-deployment Checks**
   - Test new metrics in dev environment
   - Use cardinality estimation tools
   - Review PR for unbounded labels

2. **Monitoring**
   - Set up gradual alerts (warning at 30k, critical at 50k)
   - Track cardinality trends in dashboards
   - Regular cardinality audits

3. **Governance**
   - Metric naming conventions
   - Label allowlists
   - Regular training on cardinality best practices

## Communication

### During Incident
- **Slack**: Post in #phoenix-incidents
- **Page**: On-call engineer if after hours
- **Status Page**: Update if customer impact

### Post-Incident
- Create incident report
- Schedule post-mortem
- Update this runbook with learnings

## References
- [Prometheus Cardinality Best Practices](https://prometheus.io/docs/practices/naming/#labels)
- [OpenTelemetry Metric Guidelines](https://opentelemetry.io/docs/reference/specification/metrics/semantic_conventions/)
- [Phoenix Metric Standards](../operational-procedures/metric-standards.md)