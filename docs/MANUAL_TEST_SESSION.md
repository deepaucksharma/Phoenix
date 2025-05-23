# Phoenix Manual Testing Session

**Date**: 2025-05-24  
**Purpose**: Full manual testing round after cleanup and consolidation  
**Tester**: Automated testing with manual verification  

## Test Session Log

### Phase 1: System Initialization ⏳
**Time**: Starting...  
**Status**: In Progress  

#### 1.1 Environment Check
```bash
# Check working directory
pwd
# Expected: /Users/deepaksharma/Desktop/src/Phoenix

# Check git status
git status --porcelain
# Expected: Clean working directory or known changes

# Check docker status
docker info > /dev/null && echo "Docker OK" || echo "Docker FAILED"
```

#### 1.2 Initialize Environment
```bash
# Test consolidated initialization script
./scripts/consolidated/phoenix-scripts.sh init
# Expected: Environment setup completes successfully

# Alternative direct call
./scripts/consolidated/core/initialize-environment.sh
# Expected: Should work identically

# Test symbolic link
./tools/scripts/initialize-environment.sh
# Expected: Should work via symbolic link
```

#### 1.3 System Startup
```bash
# Clean start
./scripts/consolidated/phoenix-scripts.sh clean
./scripts/consolidated/phoenix-scripts.sh start

# Alternative: Direct run-phoenix.sh
./run-phoenix.sh

# Monitor startup
docker-compose ps
docker-compose logs --tail=10
```

**Results**: 
- [ ] Environment initialization successful
- [ ] System startup successful  
- [ ] All services healthy
- [ ] No error messages in logs

---

### Phase 2: Automated Verification ⏳
**Time**: Pending...  
**Status**: Waiting for Phase 1  

#### 2.1 Full Verification Suite
```bash
# Run complete verification
./scripts/consolidated/phoenix-scripts.sh test

# Individual verification tests
./scripts/consolidated/testing/verify-services.sh
./scripts/consolidated/testing/verify-apis.sh
./scripts/consolidated/testing/verify-configs.sh
```

#### 2.2 Master Script Testing
```bash
# Test help system
./scripts/consolidated/phoenix-scripts.sh help

# Test category listing
./scripts/consolidated/phoenix-scripts.sh list
./scripts/consolidated/phoenix-scripts.sh list testing

# Test quick commands
./scripts/consolidated/phoenix-scripts.sh health
```

**Results**:
- [ ] Service verification passed
- [ ] API verification results documented
- [ ] Config verification passed
- [ ] Master script help working
- [ ] All quick commands functional

---

### Phase 3: Manual Verification Checklist ⏳
**Time**: Pending...  
**Status**: Waiting for Phase 2  

#### 3.1 Service Availability
Manual check of each documented endpoint:

**Main Collector**:
```bash
curl -f http://localhost:13133/health
curl -s http://localhost:8888/metrics | head -5
```
- [ ] Health endpoint responds
- [ ] Metrics endpoint available
- [ ] Metrics contain OTEL data

**Observer Collector**:
```bash
curl -f http://localhost:13134/health
curl -s http://localhost:9888/metrics | head -5
```
- [ ] Health endpoint responds
- [ ] Observer metrics available
- [ ] KPI metrics present

**Control Actuator**:
```bash
curl -f http://localhost:8081/health
curl -s http://localhost:8081/metrics | jq .
```
- [ ] Health endpoint responds
- [ ] Metrics in JSON format
- [ ] Control state information present

**Monitoring Stack**:
```bash
curl -f http://localhost:9090/-/healthy
curl -f http://localhost:3000/api/health
```
- [ ] Prometheus healthy
- [ ] Grafana accessible

#### 3.2 Data Flow Verification
```bash
# Check metric ingestion
curl -s http://localhost:8888/metrics | grep -c "otelcol_receiver_accepted"

# Check pipeline differentiation
curl -s http://localhost:8888/metrics | grep "pipeline_full_fidelity"
curl -s http://localhost:8889/metrics | grep "pipeline_optimised" 
curl -s http://localhost:8890/metrics | grep "pipeline_experimental"

# Check control file updates
cat configs/control/optimization_mode.yaml
sleep 70  # Wait for control loop
cat configs/control/optimization_mode.yaml
```
- [ ] Metrics being ingested
- [ ] Pipeline differentiation working
- [ ] Control file being updated

---

### Phase 4: API Testing ⏳
**Time**: Pending...  
**Status**: Waiting for Phase 3  

#### 4.1 Health Endpoints
Test all documented health endpoints:
```bash
# Main services
curl -w "Status: %{http_code}\n" http://localhost:13133/health
curl -w "Status: %{http_code}\n" http://localhost:13134/health
curl -w "Status: %{http_code}\n" http://localhost:8081/health
curl -w "Status: %{http_code}\n" http://localhost:8082/health
curl -w "Status: %{http_code}\n" http://localhost:8083/health

# Monitoring
curl -w "Status: %{http_code}\n" http://localhost:9090/-/healthy
curl -w "Status: %{http_code}\n" http://localhost:3000/api/health
```

#### 4.2 Metrics Endpoints
```bash
# Collector metrics
curl -s http://localhost:8888/metrics | wc -l
curl -s http://localhost:8889/metrics | wc -l
curl -s http://localhost:8890/metrics | wc -l
curl -s http://localhost:9888/metrics | wc -l

# Control metrics
curl -s http://localhost:8081/metrics | jq . | head -20
```

#### 4.3 Prometheus Queries
```bash
# Basic connectivity
curl -s "http://localhost:9090/api/v1/query?query=up" | jq '.data.result | length'

# Phoenix-specific metrics
curl -s "http://localhost:9090/api/v1/label/__name__/values" | jq '.data[]' | grep phoenix | wc -l

# Cardinality estimates
curl -s "http://localhost:9090/api/v1/query?query=phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate" | jq '.data.result'
```

**Results**:
- [ ] All health endpoints responding
- [ ] Metrics endpoints accessible
- [ ] Prometheus queries working
- [ ] Phoenix metrics present

---

### Phase 5: Script Functionality ⏳
**Time**: Pending...  
**Status**: Waiting for Phase 4  

#### 5.1 Consolidated Scripts
Test each category:
```bash
# Core operations
./scripts/consolidated/phoenix-scripts.sh core run-phoenix.sh --help
./scripts/consolidated/phoenix-scripts.sh core initialize-environment.sh --help

# Testing
./scripts/consolidated/phoenix-scripts.sh testing verify-services.sh
./scripts/consolidated/phoenix-scripts.sh testing verify-apis.sh

# Monitoring  
./scripts/consolidated/phoenix-scripts.sh monitoring health_check_aggregator.sh
./scripts/consolidated/phoenix-scripts.sh monitoring validate-system.sh

# Maintenance
./scripts/consolidated/phoenix-scripts.sh maintenance cleanup.sh --help
```

#### 5.2 Backward Compatibility
```bash
# Test symbolic links
./tools/scripts/initialize-environment.sh --help
./tests/integration/test_core_functionality.sh --help

# Test original locations (should work via symlinks)
ls -la tools/scripts/
ls -la tests/integration/
```

#### 5.3 Documentation Access
```bash
# Test documentation scripts
./scripts/consolidated/utils/show-docs.sh
./scripts/consolidated/utils/project-summary.sh
```

**Results**:
- [ ] All consolidated scripts working
- [ ] Symbolic links functional
- [ ] Backward compatibility maintained
- [ ] Documentation scripts accessible

---

### Phase 6: Integration Testing ⏳
**Time**: Pending...  
**Status**: Waiting for Phase 5  

#### 6.1 End-to-End Flow
```bash
# Start synthetic generator
docker-compose up -d synthetic-metrics-generator

# Wait for metrics flow
sleep 30

# Verify metric flow
echo "=== Metric Ingestion ==="
curl -s http://localhost:8888/metrics | grep otelcol_receiver_accepted_metric_points_total

echo "=== Pipeline Processing ==="
curl -s http://localhost:8888/metrics | grep otelcol_processor

echo "=== Control Loop ==="
docker-compose logs --tail=5 control-actuator-go

echo "=== Cardinality Monitoring ==="
curl -s "http://localhost:9090/api/v1/query?query=phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate"
```

#### 6.2 Control System Response
```bash
# Monitor control file
echo "Initial state:"
cat configs/control/optimization_mode.yaml

# Monitor for changes
for i in {1..3}; do
    echo "Check $i (after ${i}min):"
    sleep 60
    cat configs/control/optimization_mode.yaml
    echo "---"
done
```

#### 6.3 Resource Usage
```bash
# Check resource consumption
docker stats --no-stream

# Check against limits
echo "Memory limits check:"
docker stats --no-stream --format "table {{.Name}}\t{{.MemUsage}}" | grep -E "(collector|actuator|detector)"
```

**Results**:
- [ ] End-to-end flow working
- [ ] Control system responding
- [ ] Resource usage within limits
- [ ] No memory leaks observed

---

## Test Results Summary

### Overall Status: 🔄 IN PROGRESS

### Phase Completion:
- [⏳] Phase 1: System Initialization
- [⏳] Phase 2: Automated Verification  
- [⏳] Phase 3: Manual Verification Checklist
- [⏳] Phase 4: API Testing
- [⏳] Phase 5: Script Functionality
- [⏳] Phase 6: Integration Testing

### Critical Issues Found:
- None yet identified

### Non-Critical Issues Found:
- None yet identified

### Performance Observations:
- To be documented during testing

### Recommendations:
- To be provided after completion

---

**Next Step**: Execute Phase 1 testing...