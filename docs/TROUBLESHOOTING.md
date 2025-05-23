# Phoenix-vNext Troubleshooting Guide

## Table of Contents

- [Service Issues](#service-issues)
- [Control System Issues](#control-system-issues)
- [Pipeline Data Issues](#pipeline-data-issues)
- [Performance Issues](#performance-issues)
- [New Services Issues](#new-services-issues)
- [Diagnostic Commands](#diagnostic-commands)
- [Emergency Procedures](#emergency-procedures)

## Service Issues

### Services Won't Start

**Symptoms:**
- `docker-compose up` fails
- Services exit immediately
- Port binding errors

**Diagnosis:**
```bash
# Check service status
docker-compose ps

# View specific service logs
docker-compose logs control-actuator-go
docker-compose logs anomaly-detector
docker-compose logs benchmark-controller

# Check port availability
netstat -tulpn | grep -E ":(3000|4317|4318|8080|8081|8082|8083|8888|9090|13133)"
```

**Solutions:**

1. **Port conflicts:**
```bash
# Find processes using required ports
sudo lsof -i :8081  # Control actuator
sudo lsof -i :8082  # Anomaly detector
sudo lsof -i :8083  # Benchmark controller

# Kill conflicting processes or change ports in docker-compose.yaml
```

2. **Missing environment file:**
```bash
# Initialize environment
./scripts/initialize-environment.sh

# Verify .env exists and has required variables
grep -E "(NEW_RELIC|HYSTERESIS|TARGET_OPTIMIZED)" .env
```

3. **Go service build failures:**
```bash
# Rebuild Go services
docker-compose build control-actuator-go anomaly-detector benchmark-controller

# Check for missing dependencies
cd apps/control-actuator-go && go mod tidy
```

## Control System Issues

### Go Control Actuator Not Working

**Symptoms:**
- Control mode not changing
- No metrics at :8081/metrics
- PID controller errors

**Diagnosis:**
```bash
# Check control actuator health
curl http://localhost:8081/metrics

# View control actuator logs
docker-compose logs -f control-actuator-go | grep -E "(error|warn|mode)"

# Verify Prometheus connectivity
docker-compose exec control-actuator-go curl http://prometheus:9090/-/healthy
```

**Solutions:**

1. **Prometheus query failures:**
```bash
# Test the cardinality query manually
curl "http://localhost:9090/api/v1/query?query=phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate{pipeline=\"optimized\"}"

# Check if metrics exist
curl http://localhost:9090/api/v1/label/__name__/values | grep phoenix_observer
```

2. **Configuration issues:**
```bash
# Verify environment variables
docker-compose exec control-actuator-go env | grep -E "(TARGET|THRESHOLD|HYSTERESIS)"

# Check control file permissions
ls -la configs/control/optimization_mode.yaml
chmod 666 configs/control/optimization_mode.yaml
```

3. **PID tuning needed:**
```bash
# Adjust PID parameters in .env
HYSTERESIS_FACTOR=0.15  # Increase from 0.1
ADAPTIVE_CONTROLLER_STABILITY_SECONDS=300  # Increase from 120

# Restart control actuator
docker-compose restart control-actuator-go
```

### Control Loop Oscillation

**Symptoms:**
- Rapid mode switching
- Stability score < 0.5
- High transition count

**Solutions:**

1. **Increase stability period:**
```bash
export ADAPTIVE_CONTROLLER_STABILITY_SECONDS=300
docker-compose up -d control-actuator-go
```

2. **Adjust hysteresis:**
```bash
export HYSTERESIS_FACTOR=0.2  # 20% band
docker-compose up -d control-actuator-go
```

3. **Monitor stability:**
```bash
# Watch stability score
watch -n 5 'curl -s http://localhost:8081/metrics | jq ".stability_score"'
```

## Pipeline Data Issues

### No Metrics in Pipelines

**Symptoms:**
- Empty Prometheus endpoints
- No data in Grafana
- Zero cardinality estimates

**Diagnosis:**
```bash
# Check pipeline outputs
for port in 8888 8889 8890; do
  echo "Pipeline $port metrics count:"
  curl -s http://localhost:$port/metrics | grep -v "^#" | wc -l
done

# Verify OTLP receiver
docker-compose logs otelcol-main | grep "Started servers"
```

**Solutions:**

1. **Check shared processor configuration:**
```bash
# Verify main-optimized.yaml is being used
docker-compose exec otelcol-main ls /etc/otel-collector-config.yaml

# Test configuration
docker run --rm -v $(pwd)/configs/otel/collectors:/configs \
  otel/opentelemetry-collector-contrib:0.91.0 \
  --config=/configs/main-optimized.yaml --dry-run
```

2. **Verify routing:**
```bash
# Check if routing connector is working
docker-compose logs otelcol-main | grep "routing"
```

### Pipeline Performance Issues

**Symptoms:**
- High latency (>50ms p99)
- Memory pressure warnings
- Batch timeout errors

**Solutions:**

1. **Tune batch processor:**
```bash
# Edit configs/otel/collectors/main-optimized.yaml
# Adjust batch settings:
# send_batch_size: 5000  # Reduce from 10000
# timeout: 10s  # Reduce from 30s
```

2. **Increase memory limit:**
```bash
export OTELCOL_MAIN_MEMORY_LIMIT_MIB=2048
docker-compose up -d otelcol-main
```

## New Services Issues

### Anomaly Detector Issues

**Symptoms:**
- No alerts at :8082/alerts
- Detection algorithms not triggering
- Webhook failures

**Diagnosis:**
```bash
# Check anomaly detector health
curl http://localhost:8082/health

# View recent alerts
curl http://localhost:8082/alerts | jq

# Check detector logs
docker-compose logs anomaly-detector | grep -E "(detected|alert|error)"
```

**Solutions:**

1. **Adjust detection thresholds:**
```bash
# For high false positives, increase Z-score threshold
# Edit apps/anomaly-detector/main.go
# Change: threshold: 3.0 to threshold: 4.0
docker-compose build anomaly-detector
docker-compose up -d anomaly-detector
```

2. **Fix webhook connectivity:**
```bash
# Test control actuator webhook
docker-compose exec anomaly-detector curl -X POST http://control-actuator-go:8080/anomaly \
  -H "Content-Type: application/json" \
  -d '{"test": true}'
```

### Benchmark Controller Issues

**Symptoms:**
- Benchmarks not running
- No results at :8083/benchmark/results
- Generator configuration failures

**Diagnosis:**
```bash
# List available scenarios
curl http://localhost:8083/benchmark/scenarios

# Check benchmark logs
docker-compose logs benchmark-controller

# Verify generator connectivity
docker-compose exec benchmark-controller curl http://synthetic-metrics-generator:8080
```

**Solutions:**

1. **Generator communication:**
```bash
# Ensure synthetic generator is running
docker-compose up -d synthetic-metrics-generator

# Check generator endpoint
export SYNTHETIC_GENERATOR_URL=http://synthetic-metrics-generator:8080
docker-compose up -d benchmark-controller
```

2. **Prometheus queries failing:**
```bash
# Verify benchmark metrics exist
curl -s http://localhost:9090/api/v1/query?query=phoenix:signal_preservation_score
```

## Performance Issues

### High Memory Usage (New Services)

**Symptoms:**
- Go services using excessive memory
- OOM kills
- Slow response times

**Diagnosis:**
```bash
# Monitor Go service memory
docker stats control-actuator-go anomaly-detector benchmark-controller

# Check for memory leaks
curl http://localhost:8081/debug/pprof/heap > control-heap.prof
go tool pprof control-heap.prof
```

**Solutions:**

1. **Set GOMEMLIMIT:**
```bash
# Add to docker-compose.yaml environment
GOMEMLIMIT: 128MiB  # For control actuator
GOMEMLIMIT: 256MiB  # For anomaly detector
```

2. **Tune garbage collection:**
```bash
GOGC: 50  # More aggressive GC
```

### Recording Rules Performance

**Symptoms:**
- Prometheus high CPU usage
- Slow rule evaluation
- Recording rule failures

**Diagnosis:**
```bash
# Check rule evaluation time
curl http://localhost:9090/api/v1/rules | jq '.data.groups[].rules[] | select(.type=="recording") | {name, evaluationTime}'

# Monitor rule health
curl http://localhost:9090/api/v1/rules | jq '.data.groups[].rules[] | select(.health!="ok")'
```

**Solutions:**

1. **Optimize expensive rules:**
```bash
# Increase evaluation interval for complex rules
# Edit configs/monitoring/prometheus/rules/phoenix_comprehensive_rules.yml
# Change interval from 30s to 60s for expensive rules
```

2. **Reduce rule complexity:**
```bash
# Simplify aggregations
# Use recording rules to pre-compute expensive queries
```

## Diagnostic Commands

### System Health Check

```bash
# Complete health check
echo "=== Service Health ==="
curl -s http://localhost:13133/health || echo "Main collector: DOWN"
curl -s http://localhost:8081/metrics > /dev/null && echo "Control actuator: UP" || echo "Control actuator: DOWN"
curl -s http://localhost:8082/health || echo "Anomaly detector: DOWN"
curl -s http://localhost:8083/health || echo "Benchmark controller: DOWN"

echo -e "\n=== Control State ==="
curl -s http://localhost:8081/metrics | jq '{mode: .current_mode, transitions: .transition_count, stability: .stability_score}'

echo -e "\n=== Recent Anomalies ==="
curl -s http://localhost:8082/alerts | jq '.[0:3] | .[] | {metric: .anomaly.metric_name, severity: .anomaly.severity, time: .anomaly.timestamp}'
```

### Performance Metrics

```bash
# Pipeline efficiency
curl -s http://localhost:9090/api/v1/query?query=phoenix:resource_efficiency_score | jq '.data.result[0].value[1]'

# Signal preservation
curl -s http://localhost:9090/api/v1/query?query=phoenix:signal_preservation_score | jq '.data.result[0].value[1]'

# Cardinality reduction
curl -s http://localhost:9090/api/v1/query?query=phoenix:cardinality_reduction_percentage | jq '.data.result[0].value[1]'
```

### Debug Data Collection

```bash
# Create debug bundle
DEBUG_DIR="debug_$(date +%Y%m%d_%H%M%S)"
mkdir -p $DEBUG_DIR

# Collect logs
docker-compose logs > $DEBUG_DIR/docker-compose.logs
docker-compose logs control-actuator-go > $DEBUG_DIR/control-actuator.log
docker-compose logs anomaly-detector > $DEBUG_DIR/anomaly-detector.log

# Collect metrics
curl -s http://localhost:8081/metrics > $DEBUG_DIR/control-metrics.json
curl -s http://localhost:8082/alerts > $DEBUG_DIR/anomaly-alerts.json
curl -s http://localhost:9090/api/v1/query?query='{__name__=~"phoenix:.*"}' > $DEBUG_DIR/phoenix-metrics.json

# Collect configs
cp .env $DEBUG_DIR/
cp configs/control/optimization_mode.yaml $DEBUG_DIR/
cp docker-compose.yaml $DEBUG_DIR/

echo "Debug bundle created: $DEBUG_DIR"
```

## Emergency Procedures

### Complete System Reset

```bash
# Stop everything
docker-compose down -v

# Clean build cache
docker system prune -f
docker builder prune -f

# Remove data
rm -rf data/
rm -f configs/control/optimization_mode.yaml

# Reinitialize
./scripts/initialize-environment.sh
docker-compose build
docker-compose up -d
```

### Service Recovery

```bash
# Restart new Go services
docker-compose restart control-actuator-go anomaly-detector benchmark-controller

# Force recreate if needed
docker-compose up -d --force-recreate control-actuator-go

# Reset control state
echo 'optimization_mode: balanced' > configs/control/optimization_mode.yaml
docker-compose restart control-actuator-go otelcol-main
```

### Rollback Procedures

```bash
# Revert to bash control actuator
docker-compose stop control-actuator-go
docker-compose up -d control-loop-actuator

# Disable anomaly detection
docker-compose stop anomaly-detector

# Use original collector config
# Edit docker-compose.yaml to use main.yaml instead of main-optimized.yaml
docker-compose up -d otelcol-main
```

## Getting Help

### Collecting Support Information

```bash
# Generate support bundle
./scripts/generate-support-bundle.sh

# Or manually:
echo "Phoenix-vNext Support Information" > support.txt
echo "=================================" >> support.txt
echo "Version: $(git describe --tags --always)" >> support.txt
echo "Date: $(date)" >> support.txt
echo -e "\nEnvironment Variables:" >> support.txt
grep -E "(TARGET|THRESHOLD|HYSTERESIS|NEW_RELIC)" .env >> support.txt
echo -e "\nService Status:" >> support.txt
docker-compose ps >> support.txt
echo -e "\nRecent Errors:" >> support.txt
docker-compose logs --tail=50 | grep -i error >> support.txt
```

### Common Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| `failed to query Prometheus` | Network/connectivity issue | Check Prometheus is running and accessible |
| `stability period not met` | Too frequent mode changes | Increase ADAPTIVE_CONTROLLER_STABILITY_SECONDS |
| `anomaly detection timeout` | Slow Prometheus queries | Optimize recording rules or increase timeout |
| `benchmark scenario failed` | Resource constraints | Increase memory limits or reduce load |
| `config file permission denied` | File permissions | Run `chmod 666 configs/control/optimization_mode.yaml` |

### Support Channels

- **GitHub Issues**: Bug reports and feature requests
- **Documentation**: Check docs/ directory for detailed guides
- **Community**: OpenTelemetry and Prometheus communities