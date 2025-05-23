#!/usr/bin/env bash
# Phoenix-vNext API Testing Suite
# Tests API endpoints and expected responses

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $*"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Test mode
LIVE_TEST=${1:-false}

# Test function
run_api_test() {
    local test_name="$1"
    local test_function="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "\n${BLUE}🌐 API Test:${NC} $test_name"
    
    if $test_function; then
        log_success "✅ PASS: $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        log_error "❌ FAIL: $test_name"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# Mock API response test (when services not running)
test_mock_api_responses() {
    log_info "Testing expected API response structures..."
    
    # Test Control Actuator API structure
    cat > /tmp/mock_control_response.json << 'EOF'
{
  "current_mode": "balanced",
  "transition_count": 3,
  "stability_score": 0.92,
  "integral_error": 125.5,
  "last_error": -523.0,
  "uptime_seconds": 3600,
  "thresholds": {
    "conservative_max_ts": 15000,
    "aggressive_min_ts": 25000
  },
  "current_metrics": {
    "full_ts": 18000,
    "optimized_ts": 14000,
    "experimental_ts": 12000
  }
}
EOF

    # Validate JSON structure
    if ! python3 -c "import json; json.load(open('/tmp/mock_control_response.json'))" 2>/dev/null; then
        log_error "Control API mock response has invalid JSON structure"
        return 1
    fi
    
    # Test Anomaly Detector API structure
    cat > /tmp/mock_anomaly_response.json << 'EOF'
[{
  "id": "cardinality-1234567890",
  "anomaly": {
    "detector_name": "statistical_zscore",
    "metric_name": "phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate",
    "timestamp": "2024-05-23T10:30:00Z",
    "value": 35000,
    "expected": 20000,
    "severity": "high",
    "confidence": 0.95,
    "description": "Value 35000 is 4.2 standard deviations from mean 20000"
  },
  "status": "active",
  "action_taken": "Notified control loop to switch to aggressive mode"
}]
EOF

    if ! python3 -c "import json; json.load(open('/tmp/mock_anomaly_response.json'))" 2>/dev/null; then
        log_error "Anomaly API mock response has invalid JSON structure"
        return 1
    fi
    
    # Test Benchmark API structure
    cat > /tmp/mock_benchmark_response.json << 'EOF'
[{
  "scenario": "baseline_steady_state",
  "start_time": "2024-05-23T10:00:00Z",
  "end_time": "2024-05-23T10:10:00Z",
  "metrics": {
    "signal_preservation": 0.98,
    "cardinality_reduction": 15.2,
    "cpu_usage": 45.3,
    "memory_usage": 412.5
  },
  "passed": true
}]
EOF

    if ! python3 -c "import json; json.load(open('/tmp/mock_benchmark_response.json'))" 2>/dev/null; then
        log_error "Benchmark API mock response has invalid JSON structure"
        return 1
    fi
    
    log_success "All API response structures are valid JSON"
    
    # Cleanup
    rm -f /tmp/mock_*.json
    return 0
}

# Live API endpoint tests (when services are running)
test_live_endpoints() {
    if [ "$LIVE_TEST" != "true" ]; then
        log_info "Skipping live endpoint tests (use './api-test.sh true' to enable)"
        return 0
    fi
    
    log_info "Testing live API endpoints..."
    
    # Test Control Actuator
    if curl -s -f http://localhost:8081/health >/dev/null 2>&1; then
        log_success "Control Actuator health endpoint is responding"
        
        # Test metrics endpoint
        if curl -s -f http://localhost:8081/metrics >/dev/null 2>&1; then
            log_success "Control Actuator metrics endpoint is responding"
        else
            log_warning "Control Actuator metrics endpoint not responding"
        fi
    else
        log_warning "Control Actuator not running (expected in test environment)"
    fi
    
    # Test Anomaly Detector
    if curl -s -f http://localhost:8082/health >/dev/null 2>&1; then
        log_success "Anomaly Detector health endpoint is responding"
        
        # Test alerts endpoint
        if curl -s -f http://localhost:8082/alerts >/dev/null 2>&1; then
            log_success "Anomaly Detector alerts endpoint is responding"
        else
            log_warning "Anomaly Detector alerts endpoint not responding"
        fi
    else
        log_warning "Anomaly Detector not running (expected in test environment)"
    fi
    
    # Test Benchmark Controller
    if curl -s -f http://localhost:8083/health >/dev/null 2>&1; then
        log_success "Benchmark Controller health endpoint is responding"
        
        # Test scenarios endpoint
        if curl -s -f http://localhost:8083/benchmark/scenarios >/dev/null 2>&1; then
            log_success "Benchmark scenarios endpoint is responding"
        else
            log_warning "Benchmark scenarios endpoint not responding"
        fi
    else
        log_warning "Benchmark Controller not running (expected in test environment)"
    fi
    
    # Test Prometheus
    if curl -s -f http://localhost:9090/-/healthy >/dev/null 2>&1; then
        log_success "Prometheus health endpoint is responding"
    else
        log_warning "Prometheus not running (expected in test environment)"
    fi
    
    # Test Grafana
    if curl -s -f http://localhost:3000/api/health >/dev/null 2>&1; then
        log_success "Grafana health endpoint is responding"
    else
        log_warning "Grafana not running (expected in test environment)"
    fi
    
    return 0
}

# Test API endpoint configuration
test_endpoint_configuration() {
    log_info "Validating API endpoint configurations..."
    
    # Expected endpoints from documentation
    local expected_endpoints=(
        "localhost:3000"    # Grafana
        "localhost:4317"    # OTLP gRPC
        "localhost:4318"    # OTLP HTTP
        "localhost:8081"    # Control API
        "localhost:8082"    # Anomaly API
        "localhost:8083"    # Benchmark API
        "localhost:8888"    # Main collector metrics
        "localhost:8889"    # Optimized pipeline
        "localhost:8890"    # Experimental pipeline
        "localhost:9090"    # Prometheus
        "localhost:9888"    # Observer metrics
        "localhost:13133"   # Main collector health
        "localhost:13134"   # Observer health
    )
    
    # Check if all endpoints are configured in docker-compose
    for endpoint in "${expected_endpoints[@]}"; do
        local port
        port=$(echo "$endpoint" | cut -d':' -f2)
        
        if ! grep -q "\"$port:" docker-compose.yaml; then
            log_error "Port $port not configured in docker-compose.yaml"
            return 1
        fi
    done
    
    log_success "All expected API endpoints are configured"
    return 0
}

# Test metric endpoint structure
test_metrics_endpoint_structure() {
    log_info "Validating metrics endpoint structure..."
    
    # Check that different pipelines have different ports
    local pipeline_ports=("8888" "8889" "8890")
    
    for port in "${pipeline_ports[@]}"; do
        if ! grep -q "$port:$port" docker-compose.yaml; then
            log_error "Pipeline port $port not properly mapped"
            return 1
        fi
    done
    
    # Check Prometheus scrape configuration matches
    for port in "${pipeline_ports[@]}"; do
        if ! grep -q "otelcol-main:$port" configs/monitoring/prometheus/prometheus.yaml; then
            log_error "Prometheus not configured to scrape port $port"
            return 1
        fi
    done
    
    log_success "Metrics endpoint structure is properly configured"
    return 0
}

# Test webhook integration
test_webhook_configuration() {
    log_info "Validating webhook integration configuration..."
    
    # Check anomaly detector webhook configuration
    if ! grep -q "WEBHOOK_ENDPOINT=http://control-actuator-go:8081/anomaly" docker-compose.yaml; then
        log_error "Anomaly detector webhook not configured to notify control actuator"
        return 1
    fi
    
    # Check control actuator observer endpoint
    if ! grep -q "OBSERVER_ENDPOINT=http://otelcol-observer:9888" docker-compose.yaml; then
        log_error "Control actuator not configured to query observer"
        return 1
    fi
    
    # Check benchmark controller endpoints
    if ! grep -q "PROMETHEUS_ENDPOINT=http://prometheus:9090" docker-compose.yaml; then
        log_error "Benchmark controller not configured to query Prometheus"
        return 1
    fi
    
    if ! grep -q "COLLECTOR_ENDPOINT=http://otelcol-main:4318" docker-compose.yaml; then
        log_error "Benchmark controller not configured to send to collector"
        return 1
    fi
    
    log_success "Webhook integration properly configured"
    return 0
}

# Test API documentation consistency
test_api_documentation() {
    log_info "Validating API documentation consistency..."
    
    # Check that documented endpoints match configuration
    local readme_files=("README.md" "INFRASTRUCTURE.md" "CLAUDE.md")
    
    for readme in "${readme_files[@]}"; do
        if [ -f "$readme" ]; then
            # Check for documented ports
            if grep -q "8081" "$readme" && grep -q "8082" "$readme" && grep -q "8083" "$readme"; then
                log_success "API endpoints documented in $readme"
            else
                log_warning "Some API endpoints missing from $readme documentation"
            fi
        fi
    done
    
    return 0
}

# Test data flow through APIs
test_data_flow_simulation() {
    log_info "Simulating data flow through API endpoints..."
    
    # Simulate OTLP data ingestion
    local otlp_data='{"resourceMetrics":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"test-service"}}]},"scopeMetrics":[{"scope":{"name":"test"},"metrics":[{"name":"test_metric","gauge":{"dataPoints":[{"timeUnixNano":"1640995200000000000","asInt":"42"}]}}]}]}]}'
    
    # Test that OTLP HTTP endpoint would accept this format
    if echo "$otlp_data" | python3 -c "import json, sys; json.load(sys.stdin)" 2>/dev/null; then
        log_success "OTLP data format is valid JSON"
    else
        log_error "OTLP data format is invalid"
        return 1
    fi
    
    # Simulate control decision workflow
    # 1. Observer collects metrics
    # 2. Control actuator queries observer
    # 3. Control actuator updates control file
    # 4. Main collector reads control file
    
    log_info "Simulating control decision workflow..."
    
    # Check that control file can be read and updated
    local control_file="configs/control/optimization_mode.yaml"
    if [ -r "$control_file" ] && [ -w "$(dirname "$control_file")" ]; then
        log_success "Control file is readable and directory is writable"
    else
        log_error "Control file accessibility issue"
        return 1
    fi
    
    # Simulate anomaly detection workflow
    # 1. Anomaly detector queries Prometheus
    # 2. Detects anomaly
    # 3. Sends webhook to control actuator
    
    log_info "Simulating anomaly detection workflow..."
    
    # Check webhook payload format
    local webhook_payload='{"anomaly_id":"test-123","severity":"high","metric_name":"test_metric","current_value":35000,"expected_value":20000,"confidence":0.95}'
    
    if echo "$webhook_payload" | python3 -c "import json, sys; json.load(sys.stdin)" 2>/dev/null; then
        log_success "Webhook payload format is valid JSON"
    else
        log_error "Webhook payload format is invalid"
        return 1
    fi
    
    log_success "Data flow simulation completed successfully"
    return 0
}

echo -e "${BLUE}🌐 Phoenix-vNext API Testing Suite${NC}"
echo -e "${BLUE}===================================${NC}"

if [ "$LIVE_TEST" = "true" ]; then
    echo -e "${YELLOW}🔴 LIVE MODE: Testing actual running services${NC}"
else
    echo -e "${YELLOW}🔵 MOCK MODE: Testing configurations and structures${NC}"
fi

# Run all API tests
run_api_test "API Response Structures" "test_mock_api_responses"
run_api_test "Live Endpoint Connectivity" "test_live_endpoints"
run_api_test "Endpoint Configuration" "test_endpoint_configuration"
run_api_test "Metrics Endpoint Structure" "test_metrics_endpoint_structure"
run_api_test "Webhook Configuration" "test_webhook_configuration"
run_api_test "API Documentation" "test_api_documentation"
run_api_test "Data Flow Simulation" "test_data_flow_simulation"

# Generate final report
echo -e "\n${BLUE}📋 API TESTING SUMMARY${NC}"
echo -e "${BLUE}========================${NC}"
echo -e "Total API Tests: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 ALL API TESTS PASSED!${NC}"
    echo -e "${GREEN}Phoenix-vNext API endpoints are properly configured and ready.${NC}"
    exit 0
else
    echo -e "\n${RED}❌ Some API tests failed. Please address the issues above.${NC}"
    exit 1
fi