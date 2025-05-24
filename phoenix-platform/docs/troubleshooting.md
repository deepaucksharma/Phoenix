# Phoenix Platform Troubleshooting Guide

## Quick Diagnostics

Before diving into specific issues, run these diagnostic commands:

```bash
# Check Phoenix component status
kubectl get pods -n phoenix-system

# Check experiment status
phoenix experiment status <experiment-id>

# View recent logs
kubectl logs -n phoenix-system -l app=phoenix-api --tail=100

# Check collector health
kubectl exec -n phoenix-system <collector-pod> -- curl localhost:8888/metrics
```

## Common Issues

### 1. Experiment Not Starting

**Symptoms:**
- Experiment stuck in "pending" state
- No collector pods created

**Diagnosis:**
```bash
# Check experiment controller logs
kubectl logs -n phoenix-system deployment/experiment-controller

# Check CRD status
kubectl describe phoenixexperiment <experiment-name> -n phoenix-experiments

# Verify pipeline operator is running
kubectl get deployment pipeline-operator -n phoenix-system
```

**Solutions:**

1. **Git repository access issues:**
   ```bash
   # Verify Git token
   kubectl get secret git-credentials -n phoenix-system -o yaml
   
   # Test Git access
   kubectl exec -n phoenix-system deployment/config-generator -- \
     git ls-remote https://github.com/your-org/configs.git
   ```

2. **ArgoCD sync issues:**
   ```bash
   # Check ArgoCD application
   argocd app get phoenix-experiments
   
   # Force sync
   argocd app sync phoenix-experiments
   ```

3. **Invalid node selector:**
   ```yaml
   # Verify nodes match selector
   kubectl get nodes -l <your-selector>
   ```

### 2. No Metrics Appearing

**Symptoms:**
- Experiment running but no data in dashboards
- Zero cardinality reported

**Diagnosis:**
```bash
# Check collector logs
kubectl logs -n phoenix-experiments <collector-pod> | grep -i error

# Verify collector config
kubectl exec -n phoenix-experiments <collector-pod> -- \
  cat /etc/otel/config.yaml

# Check Prometheus targets
kubectl port-forward -n phoenix-system svc/prometheus 9090:9090
# Navigate to http://localhost:9090/targets
```

**Solutions:**

1. **Hostmetrics permission issues:**
   ```yaml
   # Ensure hostPID is set in DaemonSet
   spec:
     hostPID: true
     containers:
     - name: collector
       securityContext:
         privileged: true
   ```

2. **Wrong host path:**
   ```yaml
   # For containerized collectors
   receivers:
     hostmetrics:
       root_path: /hostfs
   
   # Mount host filesystem
   volumes:
   - name: hostfs
     hostPath:
       path: /
   ```

3. **New Relic authentication:**
   ```bash
   # Verify API key
   kubectl get secret newrelic-api-key -n phoenix-experiments -o yaml
   
   # Test OTLP endpoint
   curl -X POST https://otlp.nr-data.net:4318/v1/metrics \
     -H "Api-Key: <your-key>" \
     -H "Content-Type: application/json"
   ```

### 3. High Collector CPU/Memory Usage

**Symptoms:**
- Collector pods consuming excessive resources
- OOMKilled pods
- Host performance impact

**Diagnosis:**
```bash
# Check resource usage
kubectl top pods -n phoenix-experiments

# View collector metrics
curl http://<collector-pod>:8888/metrics | grep process_

# Check for processing bottlenecks
kubectl logs <collector-pod> | grep -E "(dropped|timeout|queue)"
```

**Solutions:**

1. **Adjust memory limits:**
   ```yaml
   processors:
     memory_limiter:
       check_interval: 1s
       limit_mib: 256  # Lower limit
       spike_limit_mib: 50
   ```

2. **Optimize batch processing:**
   ```yaml
   processors:
     batch:
       send_batch_size: 500  # Smaller batches
       timeout: 2s          # Shorter timeout
   ```

3. **Reduce collection frequency:**
   ```yaml
   receivers:
     hostmetrics:
       collection_interval: 30s  # From 10s
   ```

### 4. Missing Critical Processes

**Symptoms:**
- Important processes not showing in New Relic
- Alert firing for missing processes

**Diagnosis:**
```bash
# List all processes on host
kubectl exec -n phoenix-experiments <collector-pod> -- ps aux

# Check filter conditions
kubectl exec -n phoenix-experiments <collector-pod> -- \
  cat /etc/otel/config.yaml | grep -A20 "filter"

# Verify process names
kubectl exec <app-pod> -- ps aux | grep <process-name>
```

**Solutions:**

1. **Update critical process list:**
   ```yaml
   transform/classify:
     metric_statements:
       - context: resource
         statements:
           - set(attributes["process.priority"], "critical") 
             where attributes["process.executable.name"] =~ "^(nginx|httpd|apache2)$"
   ```

2. **Check process name variations:**
   ```yaml
   # Account for full paths
   - set(attributes["process.priority"], "critical") 
     where attributes["process.command_line"] contains "nginx"
   ```

3. **Disable aggressive filtering temporarily:**
   ```yaml
   # Comment out filter processor
   service:
     pipelines:
       metrics:
         processors: [memory_limiter, transform, batch]  # No filter
   ```

### 5. Dashboard Not Loading

**Symptoms:**
- 502 Bad Gateway errors
- Blank dashboard page
- WebSocket connection failures

**Diagnosis:**
```bash
# Check dashboard pod
kubectl logs -n phoenix-system deployment/phoenix-dashboard

# Verify API connectivity
kubectl exec -n phoenix-system deployment/phoenix-dashboard -- \
  curl http://phoenix-api:8080/health

# Check ingress
kubectl describe ingress phoenix-dashboard -n phoenix-system
```

**Solutions:**

1. **Fix CORS issues:**
   ```yaml
   # In API deployment
   env:
   - name: CORS_ALLOWED_ORIGINS
     value: "https://phoenix.example.com"
   ```

2. **Verify WebSocket upgrade:**
   ```yaml
   # In ingress annotations
   nginx.ingress.kubernetes.io/proxy-http-version: "1.1"
   nginx.ingress.kubernetes.io/proxy-set-headers: |
     Upgrade $http_upgrade;
     Connection "upgrade";
   ```

### 6. Experiment Comparison Not Working

**Symptoms:**
- Both variants showing same metrics
- No difference in cardinality
- Comparison dashboard empty

**Diagnosis:**
```bash
# Verify both collectors running
kubectl get pods -n phoenix-experiments -l experiment=<id>

# Check variant labels
kubectl exec <collector-pod> -- env | grep PHOENIX_VARIANT

# Verify distinct configurations
diff <(kubectl exec <baseline-pod> -- cat /etc/otel/config.yaml) \
     <(kubectl exec <candidate-pod> -- cat /etc/otel/config.yaml)
```

**Solutions:**

1. **Ensure variant labels:**
   ```yaml
   exporters:
     otlphttp/newrelic:
       headers:
         phoenix-variant: "${PHOENIX_VARIANT}"
   ```

2. **Fix configuration mounting:**
   ```yaml
   # Separate ConfigMaps for each variant
   volumes:
   - name: config
     configMap:
       name: collector-config-${PHOENIX_VARIANT}
   ```

## Performance Issues

### Slow Dashboard Queries

**Problem:** Dashboards take long to load

**Solution:**
```yaml
# Optimize Prometheus queries
# Bad: High cardinality query
sum by (process_name) (process_cpu_usage)

# Good: Pre-filtered query
sum by (process_name) (
  process_cpu_usage{process_priority="critical"}
)
```

### High Cardinality Warnings

**Problem:** Prometheus complaining about cardinality

**Solution:**
1. Check cardinality:
   ```promql
   prometheus_tsdb_symbol_table_size_bytes
   ```

2. Identify high-cardinality labels:
   ```bash
   curl -G http://localhost:9090/api/v1/label/__name__/values | \
     jq '.data | length'
   ```

3. Add recording rules for common queries

## Debugging Tools

### 1. Pipeline Validation Script

```bash
#!/bin/bash
# validate-pipeline.sh

CONFIG_FILE=$1

# Check YAML syntax
yamllint $CONFIG_FILE || exit 1

# Verify required processors
grep -q "memory_limiter" $CONFIG_FILE || echo "WARNING: No memory_limiter"
grep -q "batch" $CONFIG_FILE || echo "WARNING: No batch processor"

# Test with otelcol
otelcol validate --config=$CONFIG_FILE
```

### 2. Metrics Investigation

```bash
# Get all process metrics
curl -s http://localhost:8888/metrics | grep process_ | sort | uniq

# Count unique time series
curl -s http://localhost:8888/metrics | grep "^process_" | wc -l

# Find high-cardinality processes
curl -s http://localhost:8888/metrics | \
  grep process_cpu | \
  cut -d'{' -f2 | cut -d'}' -f1 | \
  sort | uniq -c | sort -nr | head -20
```

### 3. Emergency Recovery

```bash
#!/bin/bash
# emergency-stop.sh

# Stop all experiments
kubectl get phoenixexperiment -A -o name | \
  xargs -I {} kubectl patch {} --type=merge -p '{"spec":{"state":"stopped"}}'

# Scale down collectors
kubectl scale daemonset -n phoenix-experiments --all --replicas=0

# Clear stuck pipelines
kubectl delete pods -n phoenix-experiments --field-selector status.phase=Failed
```

## Escalation Path

1. **Level 1: Self-Service**
   - Check this troubleshooting guide
   - Review logs and metrics
   - Try suggested solutions

2. **Level 2: Team Support**
   - Post in #phoenix-support Slack
   - Include diagnostic output
   - Share experiment ID

3. **Level 3: Engineering**
   - Create GitHub issue with:
     - Experiment configuration
     - Collector logs
     - Error messages
     - Steps to reproduce

## Prevention Best Practices

1. **Test in Staging First**
   - Always run new pipelines in staging
   - Validate for at least 24 hours
   - Monitor resource usage

2. **Gradual Rollout**
   - Start with single node
   - Expand to 10% of fleet
   - Full rollout after validation

3. **Set Resource Limits**
   ```yaml
   resources:
     requests:
       cpu: 100m
       memory: 128Mi
     limits:
       cpu: 500m
       memory: 512Mi
   ```

4. **Monitor Key Metrics**
   - Set alerts for collector health
   - Monitor cardinality trends
   - Track cost impact

5. **Regular Reviews**
   - Weekly pipeline performance review
   - Monthly cost optimization
   - Quarterly critical process audit