# Phoenix-vNext Troubleshooting Guide

## Common Issues and Solutions

### Service Startup Issues

#### Services Won't Start

**Symptoms:**
- `docker-compose up` fails
- Services exit immediately
- Port binding errors

**Diagnosis:**
```bash
# Check service status
docker-compose ps

# View service logs
docker-compose logs otelcol-main
docker-compose logs otelcol-observer

# Check port availability
netstat -tulpn | grep -E ":(3000|4318|8888|9090|13133)"
```

**Solutions:**

1. **Port conflicts:**
```bash
# Find processes using required ports
sudo lsof -i :3000
sudo lsof -i :4318

# Kill conflicting processes
sudo kill -9 <PID>

# Or modify port mappings in docker-compose.yaml
```

2. **Missing environment file:**
```bash
# Initialize environment
./scripts/initialize-environment.sh

# Verify .env exists
ls -la .env
```

3. **Docker daemon issues:**
```bash
# Restart Docker Desktop
sudo systemctl restart docker

# Clean Docker state
docker system prune -f
docker volume prune -f
```

#### Configuration Validation Errors

**Symptoms:**
- Collectors fail to start with config errors
- Invalid YAML syntax errors

**Diagnosis:**
```bash
# Validate YAML syntax
yamllint configs/otel/collectors/main.yaml

# Test configuration
docker run --rm -v $(pwd)/configs/otel/collectors:/configs \
  otel/opentelemetry-collector-contrib:0.103.1 \
  --config=/configs/main.yaml --dry-run
```

**Solutions:**

1. **YAML syntax errors:**
```bash
# Fix indentation (use 2 spaces)
# Remove tabs
# Check for missing colons
```

2. **Missing processors:**
```bash
# Ensure all referenced processors are defined
grep -r "processors:" configs/otel/collectors/main.yaml
```

3. **Environment variable issues:**
```bash
# Check environment variables are set
docker-compose config
```

### Data Flow Issues

#### No Metrics Appearing

**Symptoms:**
- Grafana dashboards show no data
- Prometheus has no targets
- Empty metrics endpoints

**Diagnosis:**
```bash
# Check collector health
curl http://localhost:13133
curl http://localhost:13134

# Check metrics endpoints
curl http://localhost:8888/metrics | head -20
curl http://localhost:8889/metrics | head -20
curl http://localhost:8890/metrics | head -20

# Check Prometheus targets
curl http://localhost:9090/api/v1/targets
```

**Solutions:**

1. **Hostmetrics not collecting:**
```bash
# Check host filesystem mounts
docker-compose exec otelcol-main ls -la /hostfs/proc
docker-compose exec otelcol-main ls -la /hostfs/sys

# Verify process metrics are being generated
curl http://localhost:8888/metrics | grep process_
```

2. **Synthetic generator not running:**
```bash
# Check synthetic generator status
docker-compose logs synthetic-metrics-generator

# Verify OTLP endpoint connectivity
docker-compose exec synthetic-metrics-generator curl http://otelcol-main:4318
```

3. **Pipeline filtering too aggressive:**
```bash
# Check pipeline outputs
for port in 8888 8889 8890; do
  echo "Pipeline on port $port:"
  curl -s http://localhost:$port/metrics | grep -v "^#" | wc -l
done

# Temporarily disable filters for debugging
# Comment out filter processors in main.yaml
```

#### Missing Pipeline Data

**Symptoms:**
- Only one pipeline showing data
- Cardinality estimates at zero
- Control system not switching

**Diagnosis:**
```bash
# Check pipeline-specific metrics
curl http://localhost:8888/metrics | grep phoenix_full_output
curl http://localhost:8889/metrics | grep phoenix_opt_output  
curl http://localhost:8890/metrics | grep phoenix_exp_output

# Check routing connector
docker-compose logs otelcol-main | grep routing
```

**Solutions:**

1. **Routing connector issues:**
```bash
# Verify routing configuration in main.yaml
# Check that all pipelines are listed in routing table
```

2. **Pipeline filters blocking data:**
```bash
# Check filter expressions
# Ensure environment variables are set correctly
echo $ENABLE_NR_EXPORT_FULL
echo $ENABLE_NR_EXPORT_OPTIMISED
echo $ENABLE_NR_EXPORT_EXPERIMENTAL
```

### Control System Issues

#### Control System Not Switching Profiles

**Symptoms:**
- `optimization_mode.yaml` never changes
- Control actuator script errors
- Thresholds not being respected

**Diagnosis:**
```bash
# Check control actuator logs
docker-compose logs control-loop-actuator

# Check control file permissions
ls -la configs/control/optimization_mode.yaml

# Verify Prometheus connectivity
docker-compose exec control-loop-actuator curl http://prometheus:9090/-/healthy
```

**Solutions:**

1. **Prometheus query issues:**
```bash
# Test queries manually
curl "http://localhost:9090/api/v1/query?query=phoenix_pipeline_output_cardinality_estimate"

# Check metric names
curl http://localhost:9090/api/v1/label/__name__/values | grep phoenix
```

2. **Script permissions:**
```bash
# Fix script permissions
chmod +x apps/control-actuator/update-control-file.sh

# Check volume mounts
docker-compose exec control-loop-actuator ls -la /app/control_signals/
```

3. **Threshold configuration:**
```bash
# Verify threshold environment variables
docker-compose exec control-loop-actuator env | grep THRESHOLD
```

#### Control System Oscillating

**Symptoms:**
- Rapid switching between profiles
- Control file changing frequently
- Unstable cardinality readings

**Solutions:**

1. **Add hysteresis:**
```bash
# Increase threshold gaps
export THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS=15000
export THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS=30000  # Larger gap

# Add minimum switch intervals
# Modify control script to enforce cooldown periods
```

2. **Smooth cardinality estimates:**
```bash
# Use time-averaged queries in control script
# Replace instant queries with rate() functions
```

## Control Loop Issues

### Optimization Drift {#optimization-drift}

**Symptom**: The `PhoenixOptimizationDrift` alert is triggered, indicating that the cost reduction ratio has fallen below the expected threshold (40%).

**Possible Causes**:
- Changes in telemetry patterns causing the optimization algorithms to be less effective
- Misconfiguration of processing pipelines
- Transient spikes in telemetry cardinality

**Resolution Steps**:
1. Check recent changes in data patterns by examining the `phoenix:cardinality_delta_percent` metric
2. Verify the control loop logs using: `docker logs phoenix-control-actuator`
3. Check control file settings: `cat /path/to/configs/control/optimization_mode.yaml`
4. Consider adjusting thresholds in `.env` if the data patterns have permanently changed:
   ```
   THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS=15000
   THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS=25000
   ```
5. Restart the control actuator: `docker restart phoenix-control-actuator`

### Controller Failure {#controller-failure}

**Symptom**: The `PhoenixControllerFailure` alert is triggered, indicating that the controller hasn't updated metrics recently.

**Possible Causes**:
- Controller container crashed or is unresponsive
- Network issues preventing controller from reaching Prometheus
- Misconfigured controller endpoints

**Resolution Steps**:
1. Check if controller is running: `docker ps | grep phoenix-control-actuator`
2. View controller logs: `docker logs phoenix-control-actuator`
3. Check if controller can reach Prometheus:
   ```bash
   docker exec phoenix-control-actuator curl -s http://prometheus:9090/api/v1/query?query=up
   ```
4. Verify environment variables: `docker exec phoenix-control-actuator env | grep PROM`
5. Restart the controller if necessary: `docker restart phoenix-control-actuator`

### Configuration Write Failures {#config-write-failures}

**Symptom**: The `PhoenixControllerConfigurationFailure` alert is triggered, indicating that the controller failed to update configuration files.

**Possible Causes**:
- Permission issues with the config directory
- Disk space issues
- File locking conflicts

**Resolution Steps**:
1. Check container permissions: `docker exec -it phoenix-control-actuator ls -la /app/control_signals/`
2. Check disk space: `docker exec -it phoenix-control-actuator df -h`
3. Check for lock files: `docker exec -it phoenix-control-actuator ls -la /tmp/phoenix_control_lock*`
4. Verify template file exists: `docker exec -it phoenix-control-actuator ls -la /app/optimization_mode_template.yaml`
5. Delete lock file if necessary (caution): `docker exec -it phoenix-control-actuator rm /tmp/phoenix_control_lock`
6. Restart the controller: `docker restart phoenix-control-actuator`

### Control Loop Oscillation

**Symptom**: The optimization profile switches back and forth between "conservative" and "balanced" or other profiles frequently.

**Possible Causes**:
- Metric values hover near threshold boundaries
- Insufficient hysteresis in the controller logic
- Stability period too short

**Resolution Steps**:
1. Check the recent history of optimization profile changes:
   ```bash
   docker exec -it phoenix-control-actuator grep "Profile changing" /var/log/phoenix-actuator.log
   ```
2. Adjust the hysteresis factor (default is 0.1 or 10%):
   ```bash
   docker exec -it phoenix-control-actuator export HYSTERESIS_FACTOR=0.15
   ```
3. Increase the stability period:
   ```bash
   docker exec -it phoenix-control-actuator export ADAPTIVE_CONTROLLER_STABILITY_SECONDS=300
   ```

## Pipeline Issues

#### No Data in Pipeline

**Symptoms:**
- Metrics not appearing in Grafana
- Empty Prometheus targets
- No data from collector endpoints

**Diagnosis:**
```bash
# Check collector logs
docker-compose logs otelcol-main | grep "error\|failed\|panic"

# Test direct endpoint access
curl -v http://localhost:8888/metrics
curl -v http://localhost:8889/metrics
curl -v http://localhost:8890/metrics

# Check Prometheus configuration
curl -v http://localhost:9090/api/v1/config
```

**Solutions:**

1. **Collector misconfiguration:**
```bash
# Review collector configuration files
cat configs/otel/collectors/main.yaml

# Validate YAML syntax
yamllint configs/otel/collectors/main.yaml

# Test configuration with collector image
docker run --rm -v $(pwd)/configs/otel/collectors:/configs \
  otel/opentelemetry-collector-contrib:0.103.1 \
  --config=/configs/main.yaml --dry-run
```

2. **Network issues:**
```bash
# Check Docker network
docker network ls
docker network inspect phoenix-vnext_default

# Test connectivity between containers
docker-compose exec otelcol-main ping otelcol-observer
docker-compose exec otelcol-observer ping otelcol-main
```

3. **Prometheus scrape configuration:**
```bash
# Check scrape configs in Prometheus
curl -s http://localhost:9090/api/v1/config | jq '.scrape_configs'

# Verify target endpoints are correct
curl -s http://localhost:9090/api/v1/targets | jq '.data.active'
```

4. **Data format issues:**
```bash
# Check for unexpected metric names or labels
curl -s http://localhost:8888/metrics | head -20
curl -s http://localhost:8889/metrics | head -20
curl -s http://localhost:8890/metrics | head -20
```

5. **Restart affected services:**
```bash
# Restart collectors
docker-compose restart otelcol-main otelcol-observer

# Restart Prometheus
docker-compose restart prometheus
```

#### Unexpected Data in Pipeline

**Symptoms:**
- Cardinality estimates are too high or too low
- Metrics with unexpected labels or values
- Alerts for high cardinality or data spikes

**Diagnosis:**
```bash
# Check recent changes in telemetry data
curl -s http://localhost:8888/metrics | grep phoenix_cardinality
curl -s http://localhost:8889/metrics | grep phoenix_cardinality
curl -s http://localhost:8890/metrics | grep phoenix_cardinality

# Review collector logs for errors or warnings
docker-compose logs otelcol-main | grep "error\|warn"

# Check for recent changes in configuration
git diff HEAD~1 HEAD -- configs/otel/collectors
```

**Solutions:**

1. **Adjust cardinality thresholds:**
```bash
# Update thresholds in .env
THRESHOLD_CARDINALITY_ESTIMATE=10000

# Restart affected services
docker-compose restart otelcol-main otelcol-observer
```

2. **Refine metric filters:**
```bash
# Edit processor filters in main.yaml
# Exclude unnecessary labels or metrics
```

3. **Review and update documentation:**
```bash
# Document any changes in data patterns or processing
# Update runbooks and troubleshooting guides
```

## Performance Issues

#### High Memory Usage

**Symptoms:**
- OOM kills
- Slow response times
- High container memory usage

**Diagnosis:**
```bash
# Monitor container memory
docker stats --format "table {{.Name}}\t{{.MemUsage}}\t{{.MemPerc}}"

# Check collector memory metrics
curl http://localhost:8888/metrics | grep otelcol_process_memory

# Check for memory leaks
curl http://localhost:1777/debug/pprof/heap > heap.prof
```

**Solutions:**

1. **Increase memory limits:**
```bash
# Adjust memory limits in .env
OTELCOL_MAIN_MEMORY_LIMIT_MIB=2048
OTELCOL_OBSERVER_MEMORY_LIMIT_MIB=512

# Restart services
docker-compose restart otelcol-main otelcol-observer
```

2. **Optimize processing:**
```bash
# Reduce batch sizes
send_batch_size: 4096  # Reduce from 8192

# Increase timeout to reduce frequent batching
timeout: 30s  # Increase from 10s

# Enable memory ballast
OTEL_MAIN_MEMBALLAST_MIB_ENV=512
```

3. **Reduce cardinality:**
```bash
# Lower process count
SYNTHETIC_PROCESS_COUNT_PER_HOST=100  # Reduce from 250

# More aggressive filtering
# Modify processor filters to be more selective
```

#### High CPU Usage

**Symptoms:**
- High CPU utilization
- Slow metrics processing
- Timeouts and backpressure

**Diagnosis:**
```bash
# Monitor CPU usage
docker stats --format "table {{.Name}}\t{{.CPUPerc}}"

# Check processing rates
curl http://localhost:8888/metrics | grep otelcol_processor_batch_batch_send_size_sum
```

**Solutions:**

1. **Optimize Go runtime:**
```bash
# Adjust GOMAXPROCS
OTELCOL_MAIN_GOMAXPROCS=2  # Match available cores

# Set memory limit
GOMEMLIMIT=1GiB
```

2. **Reduce processing overhead:**
```bash
# Simplify transform operations
# Use more efficient attribute operations
# Reduce regex complexity in filters
```

### Network and Connectivity Issues

#### New Relic Export Failures

**Symptoms:**
- Export errors in logs
- High retry counts
- Missing data in New Relic

**Diagnosis:**
```bash
# Check export logs
docker-compose logs otelcol-main | grep newrelic

# Test connectivity
docker-compose exec otelcol-main curl -I https://otlp.nr-data.net:4318/v1/metrics
```

**Solutions:**

1. **Authentication issues:**
```bash
# Verify API keys
echo $NEW_RELIC_LICENSE_KEY_FULL
echo $NEW_RELIC_LICENSE_KEY_OPTIMISED
echo $NEW_RELIC_LICENSE_KEY_EXPERIMENTAL

# Test with curl
curl -X POST https://otlp.nr-data.net:4318/v1/metrics \
  -H "api-key: $NEW_RELIC_LICENSE_KEY_FULL" \
  -H "Content-Type: application/json" \
  -d '{}'
```

2. **Network connectivity:**
```bash
# Check DNS resolution
docker-compose exec otelcol-main nslookup otlp.nr-data.net

# Check firewall rules
# Ensure outbound HTTPS (443) is allowed
```

#### Prometheus Scraping Issues

**Symptoms:**
- Missing targets in Prometheus
- Scrape failures
- Stale data

**Diagnosis:**
```bash
# Check Prometheus targets
curl http://localhost:9090/api/v1/targets

# Check scrape configuration
curl http://localhost:9090/api/v1/status/config
```

**Solutions:**

1. **Service discovery issues:**
```bash
# Verify service connectivity
docker-compose exec prometheus curl http://otelcol-main:8888/metrics
docker-compose exec prometheus curl http://otelcol-observer:9888/metrics

# Check network connectivity
docker network ls
docker network inspect phoenix-vnext_default
```

## Diagnostic Commands

#### Health Check Commands

```bash
# Overall system health
./scripts/health-check.sh

# Service-specific health
curl http://localhost:13133  # Main collector
curl http://localhost:13134  # Observer
curl http://localhost:9090/-/healthy  # Prometheus
curl http://localhost:3000/api/health  # Grafana
```

#### Data Validation Commands

```bash
# Verify data flow
./scripts/validate-data-flow.sh

# Check cardinality across pipelines
for port in 8888 8889 8890; do
  echo "Pipeline $port cardinality:"
  curl -s http://localhost:$port/metrics | grep -v "^#" | wc -l
done

# Control system validation
./scripts/validate-control-system.sh
```

#### Performance Monitoring

```bash
# Resource usage
docker stats --no-stream

# Metrics rates
curl "http://localhost:9090/api/v1/query?query=rate(otelcol_processor_batch_batch_send_size_sum[5m])"

# Error rates  
curl "http://localhost:9090/api/v1/query?query=rate(otelcol_exporter_send_failed_metric_points_total[5m])"
```

## Emergency Procedures

### Complete System Reset

```bash
# Stop all services
docker-compose down

# Remove all data
docker-compose down -v
rm -rf data/

# Clean Docker state
docker system prune -f

# Reinitialize
./scripts/initialize-environment.sh
docker-compose up -d
```

### Backup Current State

```bash
# Create backup
mkdir -p backups/emergency_$(date +%Y%m%d_%H%M%S)

# Backup data volumes
docker run --rm -v phoenix-vnext_prometheus_data:/data -v $(pwd)/backups/emergency_$(date +%Y%m%d_%H%M%S):/backup alpine tar czf /backup/prometheus.tar.gz -C /data .

# Backup configuration
cp -r configs/ backups/emergency_$(date +%Y%m%d_%H%M%S)/
cp .env backups/emergency_$(date +%Y%m%d_%H%M%S)/
```

### Service Recovery

```bash
# Restart individual services
docker-compose restart otelcol-main
docker-compose restart otelcol-observer
docker-compose restart control-loop-actuator

# Force recreate if needed
docker-compose up -d --force-recreate otelcol-main
```

## Log Analysis

### Structured Log Parsing

```bash
# Parse collector logs
docker-compose logs otelcol-main | jq 'select(.level == "error")'

# Filter by component
docker-compose logs otelcol-main | jq 'select(.caller | contains("processor"))'

# Time-based filtering
docker-compose logs --since "30m" otelcol-main
```

### Common Log Patterns

#### Successful Processing
```
{"level":"info","ts":"...","caller":"...","msg":"Started pipeline","pipeline":"metrics/pipeline_full_fidelity"}
```

#### Configuration Errors
```
{"level":"error","ts":"...","caller":"...","msg":"Failed to load config","error":"yaml: line 123: ..."}
```

#### Memory Pressure
```
{"level":"warn","ts":"...","caller":"...","msg":"Memory usage is above limit","current":"1.2GB","limit":"1GB"}
```

## Getting Help

### Collecting Debug Information

```bash
# Generate debug bundle
./scripts/collect-debug-info.sh

# Manual collection
mkdir debug_$(date +%Y%m%d_%H%M%S)
docker-compose logs > debug_*/docker-compose.logs
curl http://localhost:13133 > debug_*/main-collector-health.json
curl http://localhost:9090/api/v1/targets > debug_*/prometheus-targets.json
cp configs/control/optimization_mode.yaml debug_*/
cp .env debug_*/
```

### Community Resources

- **GitHub Issues**: Report bugs and request features
- **OpenTelemetry Community**: General OTel questions
- **Prometheus Community**: Prometheus-specific issues

### Professional Support

For production deployments, consider:
- OpenTelemetry vendor support
- Prometheus enterprise solutions
- Custom consulting services