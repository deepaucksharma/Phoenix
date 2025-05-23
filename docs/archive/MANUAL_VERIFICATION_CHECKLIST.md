# Phoenix Manual Verification Checklist

Based on the implementation gaps analysis, this document provides a comprehensive manual testing checklist to verify the actual state of the Phoenix system versus documentation claims.

## Quick Reference

- **🔴 BROKEN**: Feature doesn't work as documented
- **🟡 PARTIAL**: Feature partially works or has limitations  
- **🟢 WORKING**: Feature works as documented
- **❓ UNTESTED**: Needs verification

## Pre-Test Setup

```bash
# Initialize environment
./scripts/initialize-environment.sh

# Start core services
./run-phoenix.sh

# Wait for services to start (30 seconds)
sleep 30

# Verify basic connectivity
docker-compose ps
```

## 1. Service Availability Tests

### 1.1 Docker Services Status
```bash
# Check all services are running
docker-compose ps

# Expected services:
# - otelcol-main (healthy)
# - otelcol-observer (healthy) 
# - control-actuator-go (running)
# - anomaly-detector (running)
# - prometheus (running)
# - grafana (running)
```
**Status**: ❓ UNTESTED  
**Notes**: 

### 1.2 Service Endpoints Connectivity
```bash
# Main collector health (documented: 13133)
curl -f http://localhost:13133 || echo "FAILED"

# Observer health (documented: 13134)  
curl -f http://localhost:13134 || echo "FAILED"

# Control actuator (documented: 8081, actual: 8080)
curl -f http://localhost:8081/metrics || echo "FAILED on 8081"
curl -f http://localhost:8080/metrics || echo "FAILED on 8080"

# Prometheus
curl -f http://localhost:9090 || echo "FAILED"

# Grafana
curl -f http://localhost:3000 || echo "FAILED"
```
**Status**: ❓ UNTESTED  
**Notes**: 

## 2. API Endpoints Verification

### 2.1 Control Actuator API
```bash
# Health endpoint (documented but missing)
curl -f http://localhost:8081/health
curl -f http://localhost:8080/health
echo "Expected: Working health check"

# Metrics endpoint (should work)
curl -s http://localhost:8081/metrics | jq . || echo "JSON parse failed"
curl -s http://localhost:8080/metrics | jq . || echo "JSON parse failed" 

# Mode control endpoint (documented but missing)
curl -X POST http://localhost:8081/mode \
  -H "Content-Type: application/json" \
  -d '{"mode": "aggressive"}' || echo "FAILED"

# Anomaly webhook (documented but missing)
curl -X POST http://localhost:8081/anomaly \
  -H "Content-Type: application/json" \
  -d '{"type": "cardinality_spike", "severity": "high"}' || echo "FAILED"
```
**Status**: ❓ UNTESTED  
**Notes**: 

### 2.2 Anomaly Detector API
```bash
# Health check (documented but missing)
curl -f http://localhost:8082/health || echo "FAILED"

# Active alerts (documented but missing)
curl -f http://localhost:8082/alerts || echo "FAILED"

# Metrics endpoint (documented but missing)
curl -f http://localhost:8082/metrics || echo "FAILED"
```
**Status**: ❓ UNTESTED  
**Notes**: 

### 2.3 Benchmark Controller API
```bash
# List scenarios (documented but missing)
curl -f http://localhost:8083/benchmark/scenarios || echo "FAILED"

# Run benchmark (documented but missing)
curl -X POST http://localhost:8083/benchmark/run \
  -H "Content-Type: application/json" \
  -d '{"scenario": "baseline_steady_state"}' || echo "FAILED"

# Check results (documented but missing)
curl -f http://localhost:8083/benchmark/results || echo "FAILED"
```
**Status**: ❓ UNTESTED  
**Notes**: 

## 3. Configuration Verification

### 3.1 File Structure Check
```bash
# Check for missing directories/files
echo "=== Configuration Structure ==="

# Missing processors directory
ls -la configs/otel/processors/ 2>/dev/null || echo "❌ configs/otel/processors/ MISSING"

# Control template location
ls -la configs/control/optimization_mode_template.yaml 2>/dev/null || echo "❌ Template in wrong location"
ls -la configs/templates/control/optimization_mode_template.yaml 2>/dev/null || echo "✅ Template found in templates/"

# Grafana dashboards
ls -la configs/monitoring/grafana/dashboards/*.json 2>/dev/null || echo "❌ No Grafana dashboards"
ls -la archive/_cleanup_2025_05_24/monitoring/grafana/dashboards/*.json 2>/dev/null || echo "ℹ️ Dashboards only in archive"
```
**Status**: ❓ UNTESTED  
**Notes**: 

### 3.2 Control File Updates
```bash
# Monitor control file changes
echo "=== Control File Monitoring ==="

# Initial state
echo "Initial optimization mode:"
cat configs/control/optimization_mode.yaml

# Wait and check for updates (should happen every 60s)
echo "Waiting 90 seconds for control updates..."
sleep 90

echo "Updated optimization mode:"
cat configs/control/optimization_mode.yaml

# Check if version/correlation_id changed
echo "Look for version increments and correlation_id changes"
```
**Status**: ❓ UNTESTED  
**Notes**: 

## 4. Metrics and Monitoring

### 4.1 Prometheus Recording Rules
```bash
# Check if documented recording rules exist
echo "=== Recording Rules Check ==="

# Query documented rules (with colons)
curl -s "http://localhost:9090/api/v1/query?query=phoenix:signal_preservation_score" | jq '.data.result | length'
curl -s "http://localhost:9090/api/v1/query?query=phoenix:cardinality_efficiency_ratio" | jq '.data.result | length'

# Query actual rules (with underscores) 
curl -s "http://localhost:9090/api/v1/query?query=phoenix_signal_preservation_score" | jq '.data.result | length'
curl -s "http://localhost:9090/api/v1/query?query=phoenix_cardinality_efficiency_ratio" | jq '.data.result | length'

echo "Non-zero results indicate working rules"
```
**Status**: ❓ UNTESTED  
**Notes**: 

### 4.2 Pipeline Metrics
```bash
# Check main collector metrics
echo "=== Pipeline Metrics ==="

# Main collector (port 8888)
curl -s http://localhost:8888/metrics | grep -c "otelcol_processor" || echo "No processor metrics"

# Optimized pipeline (port 8889) 
curl -s http://localhost:8889/metrics | grep -c "otelcol_processor" || echo "No optimized metrics"

# Experimental pipeline (port 8890)
curl -s http://localhost:8890/metrics | grep -c "otelcol_processor" || echo "No experimental metrics"

# Observer metrics (port 9888)
curl -s http://localhost:9888/metrics | grep -c "phoenix_observer" || echo "No observer metrics"
```
**Status**: ❓ UNTESTED  
**Notes**: 

### 4.3 Cardinality Estimates
```bash
# Query cardinality estimates (key control metric)
echo "=== Cardinality Monitoring ==="

curl -s "http://localhost:9090/api/v1/query?query=phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate" \
  | jq '.data.result[] | {pipeline: .metric.pipeline, cardinality: .value[1]}'

echo "Should show 3 pipelines with cardinality estimates"
```
**Status**: ❓ UNTESTED  
**Notes**: 

## 5. Load Generation and Control

### 5.1 Synthetic Generator
```bash
# Check if synthetic generator is running and generating metrics
echo "=== Synthetic Load Generation ==="

# Check service status
docker-compose ps synthetic-metrics-generator

# Check if metrics are being received
curl -s http://localhost:8888/metrics | grep -c "received" || echo "No receive metrics"

# Monitor metric ingestion rate
echo "Checking ingestion rate..."
curl -s "http://localhost:9090/api/v1/query?query=rate(otelcol_receiver_accepted_metric_points_total[1m])" \
  | jq '.data.result[] | .value[1]' | head -1
```
**Status**: ❓ UNTESTED  
**Notes**: 

### 5.2 Control Loop Operation
```bash
# Verify control loop is functioning
echo "=== Control Loop Verification ==="

# Check control actuator logs for decisions
docker-compose logs --tail=20 control-actuator-go | grep -i "switching\|mode\|threshold"

# Monitor for mode changes by checking file timestamps
stat -c %Y configs/control/optimization_mode.yaml
sleep 70  # Wait longer than control interval (60s)
stat -c %Y configs/control/optimization_mode.yaml

echo "Timestamps should differ if control loop is active"
```
**Status**: ❓ UNTESTED  
**Notes**: 

## 6. Development Commands

### 6.1 Makefile Commands
```bash
# Test documented Makefile commands
echo "=== Makefile Commands ==="

# Basic commands (should work)
make help 2>/dev/null || echo "❌ make help FAILED"
make test 2>/dev/null || echo "❌ make test FAILED"

# Missing commands (expected to fail)
make monitor 2>/dev/null || echo "❌ make monitor MISSING (expected)"
make collector-logs 2>/dev/null || echo "❌ make collector-logs MISSING (expected)"
make validate-config 2>/dev/null || echo "❌ make validate-config MISSING (expected)"
make docs-serve 2>/dev/null || echo "❌ make docs-serve MISSING (expected)"
```
**Status**: ❓ UNTESTED  
**Notes**: 

### 6.2 Environment Variables
```bash
# Check if documented environment variables are used
echo "=== Environment Variables Usage ==="

# HYSTERESIS_FACTOR (documented but not used)
grep -r "HYSTERESIS_FACTOR" apps/control-actuator-go/ || echo "❌ HYSTERESIS_FACTOR not used in code"

# Control thresholds
grep -r "TARGET_OPTIMIZED_PIPELINE_TS_COUNT" apps/control-actuator-go/ || echo "❌ TARGET not used"
grep -r "THRESHOLD_OPTIMIZATION" apps/control-actuator-go/ || echo "❌ THRESHOLD not used"

echo "Check .env file:"
cat .env | grep -E "(HYSTERESIS|THRESHOLD|TARGET)"
```
**Status**: ❓ UNTESTED  
**Notes**: 

## 7. Integration Tests

### 7.1 End-to-End Flow
```bash
# Test complete data flow
echo "=== End-to-End Flow Test ==="

# 1. Generate some load
echo "Starting synthetic generator..."
docker-compose up -d synthetic-metrics-generator

# 2. Wait for metrics to flow through
sleep 30

# 3. Check metrics at each stage
echo "Checking metrics flow:"
echo "Generator -> Collector:"
curl -s http://localhost:8888/metrics | grep -c "otelcol_receiver_accepted_metric_points_total"

echo "Collector -> Prometheus:"
curl -s "http://localhost:9090/api/v1/query?query=up" | jq '.data.result | length'

echo "Observer -> Control:"
docker-compose logs --tail=5 control-actuator-go | grep -i "cardinality\|switching"
```
**Status**: ❓ UNTESTED  
**Notes**: 

### 7.2 Control Response Test
```bash
# Test control system response to load changes
echo "=== Control Response Test ==="

# Baseline cardinality
echo "Baseline cardinality:"
curl -s "http://localhost:9090/api/v1/query?query=phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate" \
  | jq '.data.result[0].value[1]' 2>/dev/null || echo "No cardinality data"

# Force high load (if generator supports it)
# This would require modifying generator config or using benchmark tool

echo "Monitor configs/control/optimization_mode.yaml for changes"
watch -n 10 "cat configs/control/optimization_mode.yaml | grep mode"
```
**Status**: ❓ UNTESTED  
**Notes**: 

## 8. Security and Performance

### 8.1 Resource Usage
```bash
# Check resource usage against documented limits
echo "=== Resource Usage Check ==="

# Memory usage (documented limit: 1024MB for main collector)
docker stats --no-stream --format "table {{.Name}}\t{{.MemUsage}}\t{{.CPUPerc}}" | grep otelcol-main

# Check if within documented limits
echo "Should be under 1024MB as per OTELCOL_MAIN_MEMORY_LIMIT_MIB"
```
**Status**: ❓ UNTESTED  
**Notes**: 

### 8.2 Debug Endpoints
```bash
# Test documented debug endpoints
echo "=== Debug Endpoints ==="

# pprof (documented: port 1777)
curl -f http://localhost:1777/debug/pprof/ || echo "❌ pprof endpoint FAILED"

# zpages (documented: port 55679)
curl -f http://localhost:55679/debug/servicez || echo "❌ zpages endpoint FAILED"
```
**Status**: ❓ UNTESTED  
**Notes**: 

## Test Execution Log

Date: _______________  
Tester: _______________  
Version/Commit: _______________  

### Summary Results
- Total Tests: ___
- Passing: ___
- Failing: ___  
- Partially Working: ___
- Untested: ___

### Critical Issues Found
1. 
2. 
3. 

### Recommendations
1. 
2. 
3. 

### Next Steps
1. 
2. 
3. 