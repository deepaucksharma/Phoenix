# Phoenix Documentation vs Implementation Gaps

This document identifies discrepancies between the documentation and actual implementation as of the validation performed on May 24, 2025.

## Critical Gaps

### 1. Service Path Mismatch
**Documentation**: Synthetic generator at `./apps/synthetic-generator`  
**Implementation**: Actually at `./services/generators/synthetic`  
**Impact**: Docker build failure - service won't start  
**Fix Required**: Update docker-compose.yaml build context

### 2. Control Actuator Port Mismatch
**Documentation**: Port 8081  
**Implementation**: Port 8080 (hardcoded in main.go:279)  
**Impact**: Connection failures when trying to access documented endpoints  
**Fix Required**: Either update code or docker-compose port mapping

### 3. Missing API Endpoints
**Documentation**: Lists multiple endpoints for control actuator  
**Implementation**: Only `/metrics` endpoint exists  
**Missing**:
- `/health` endpoint
- `/anomaly` webhook endpoint  
- `/mode` endpoint for manual control
**Impact**: Limited control and monitoring capabilities

### 4. Benchmark Controller API Missing
**Documentation**: Full REST API with scenarios, run, results endpoints  
**Implementation**: No HTTP server setup found in main.go  
**Impact**: Cannot run benchmarks via API as documented

## Configuration Gaps

### 1. Missing Configuration Files
- `configs/otel/processors/common_intake_processors.yaml` - directory doesn't exist
- `configs/control/optimization_mode_template.yaml` - in wrong location (templates/)
- Grafana dashboard JSON files - only exist in archive, not active location

### 2. Recording Rules Mismatch
**Documentation**: Lists rules like `phoenix:signal_preservation_score`  
**Implementation**: Rules use different metric names like `phoenix_signal_preservation_score`  
**Impact**: Queries in documentation won't work

## Missing Features

### 1. Anomaly Detector Endpoints
**Documentation**: `/alerts`, `/health`, `/metrics` endpoints  
**Implementation**: No HTTP server found in anomaly detector  
**Impact**: Cannot query anomalies via API

### 2. Health Check Endpoints
**Documentation**: Health checks on specific ports  
**Implementation**: Missing for most services except collectors  
**Impact**: Cannot properly monitor service health

### 3. Makefile Commands
**Documentation**: Lists many commands (monitor, logs, validate-config)  
**Implementation**: Basic Makefile with only build/test commands  
**Missing**:
- `make monitor`
- `make collector-logs`, `observer-logs`, etc.
- `make validate-config`
- `make docs-serve`

## Environment Variable Gaps

### 1. Hysteresis Factor
**Documentation**: `HYSTERESIS_FACTOR=0.1`  
**Implementation**: Not used in control actuator code  
**Impact**: No hysteresis prevention of oscillation

### 2. New Relic Configuration
**Documentation**: Multiple NR export flags  
**Implementation**: Not referenced in collector configs  
**Impact**: New Relic export may not work as documented

## Functional Gaps

### 1. PID Control Implementation
**Documentation**: Full PID algorithm with integral and derivative  
**Implementation**: Simple threshold-based switching, no true PID  
**Impact**: Less sophisticated control, potential oscillation

### 2. Stability Score
**Documentation**: Mentions stability score tracking  
**Implementation**: Basic implementation but not exposed via API  
**Impact**: Cannot monitor control stability

### 3. Webhook Integration
**Documentation**: Anomaly detector sends webhooks  
**Implementation**: No webhook sending code found  
**Impact**: No automated response to anomalies

## Quick Fixes Required

### Priority 1 (Breaking Issues)
1. Fix synthetic generator path in docker-compose.yaml
2. Align control actuator port (8080 vs 8081)
3. Create missing config directories

### Priority 2 (Functional Gaps)
1. Implement missing API endpoints
2. Add health checks to all services
3. Set up benchmark controller HTTP server

### Priority 3 (Enhancement)
1. Implement true PID control
2. Add webhook support
3. Complete Makefile commands

## Validation Commands

To verify these gaps:

```bash
# Check service paths
ls -la apps/synthetic-generator  # Will fail
ls -la services/generators/synthetic  # Exists

# Check ports
docker-compose config | grep -A5 control-actuator-go | grep 8081
grep ListenAndServe apps/control-actuator-go/main.go  # Shows :8080

# Check config files
ls configs/otel/processors/  # Directory doesn't exist
ls configs/monitoring/grafana/dashboards/*.json  # No files

# Check API endpoints
curl http://localhost:8081/health  # Will fail
curl http://localhost:8080/metrics  # Should work
```

## Recommendations

1. **Immediate**: Fix breaking issues that prevent system from running
2. **Short-term**: Implement missing APIs and health checks
3. **Medium-term**: Add missing features like PID control and webhooks
4. **Long-term**: Full feature parity with documentation

This validation was performed by comparing:
- Docker-compose.yaml service definitions
- Source code in apps/ and services/
- Configuration file locations
- Makefile contents
- API endpoint implementations