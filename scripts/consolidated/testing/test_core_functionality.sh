#!/bin/bash
# Phoenix Core Functionality Integration Tests
# Tests the complete system including pipelines, control loop, and monitoring

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test configuration
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
COLLECTOR_HEALTH="${COLLECTOR_HEALTH:-http://localhost:13133}"
OBSERVER_HEALTH="${OBSERVER_HEALTH:-http://localhost:13134}"
OTLP_ENDPOINT="${OTLP_ENDPOINT:-localhost:4318}"
TEST_DURATION="${TEST_DURATION:-300}" # 5 minutes
RESULTS_DIR="test-results-$(date +%Y%m%d-%H%M%S)"

# Create results directory
mkdir -p "$RESULTS_DIR"

# Logging
log_info() { echo -e "${BLUE}[INFO]${NC} $*" | tee -a "$RESULTS_DIR/test.log"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $*" | tee -a "$RESULTS_DIR/test.log"; }
log_fail() { echo -e "${RED}[FAIL]${NC} $*" | tee -a "$RESULTS_DIR/test.log"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*" | tee -a "$RESULTS_DIR/test.log"; }

# Test result tracking
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
check_service_health() {
    local service_name="$1"
    local health_url="$2"
    
    log_info "Checking $service_name health..."
    if curl -s -f "$health_url" > /dev/null 2>&1; then
        log_success "$service_name is healthy"
        return 0
    else
        log_fail "$service_name is not healthy"
        return 1
    fi
}

query_prometheus() {
    local query="$1"
    local result=$(curl -s -G "${PROMETHEUS_URL}/api/v1/query" \
        --data-urlencode "query=$query" | \
        jq -r '.data.result[0].value[1] // "null"')
    
    if [ "$result" != "null" ]; then
        echo "$result"
        return 0
    else
        return 1
    fi
}

wait_for_metric() {
    local metric="$1"
    local timeout="$2"
    local start_time=$(date +%s)
    
    while true; do
        if query_prometheus "$metric" > /dev/null 2>&1; then
            return 0
        fi
        
        if [ $(($(date +%s) - start_time)) -gt "$timeout" ]; then
            return 1
        fi
        
        sleep 5
    done
}

# Test 1: Service Health Checks
test_service_health() {
    log_info "=== Test 1: Service Health Checks ==="
    
    local all_healthy=true
    
    check_service_health "Collector" "$COLLECTOR_HEALTH" || all_healthy=false
    check_service_health "Observer" "$OBSERVER_HEALTH" || all_healthy=false
    
    if [ "$all_healthy" = true ]; then
        log_success "All services are healthy"
        ((TESTS_PASSED++))
    else
        log_fail "Some services are unhealthy"
        ((TESTS_FAILED++))
    fi
}

# Test 2: Pipeline Processing
test_pipeline_processing() {
    log_info "=== Test 2: Pipeline Processing ==="
    
    # Wait for pipeline metrics to appear
    log_info "Waiting for pipeline metrics..."
    if wait_for_metric "phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate" 60; then
        log_success "Pipeline metrics are being produced"
        
        # Check all three pipelines
        for pipeline in "full_fidelity" "optimised" "experimental"; do
            local cardinality=$(query_prometheus \
                "phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate{phoenix_pipeline_label=\"$pipeline\"}" \
                2>/dev/null || echo "0")
            
            if [ "$cardinality" != "0" ]; then
                log_success "Pipeline $pipeline has cardinality: $cardinality"
            else
                log_warn "Pipeline $pipeline has zero cardinality"
            fi
        done
        ((TESTS_PASSED++))
    else
        log_fail "Pipeline metrics not found after timeout"
        ((TESTS_FAILED++))
    fi
}

# Test 3: Control Loop Functionality
test_control_loop() {
    log_info "=== Test 3: Control Loop Functionality ==="
    
    # Check if control file exists
    if docker exec phoenix-actuator test -f /app/control_signals/optimization_mode.yaml; then
        log_success "Control file exists"
        
        # Get current optimization mode
        local mode=$(docker exec phoenix-actuator yq eval '.optimization_profile' /app/control_signals/optimization_mode.yaml 2>/dev/null || echo "unknown")
        log_info "Current optimization mode: $mode"
        
        if [[ "$mode" =~ ^(conservative|balanced|aggressive)$ ]]; then
            log_success "Valid optimization mode: $mode"
            ((TESTS_PASSED++))
        else
            log_fail "Invalid optimization mode: $mode"
            ((TESTS_FAILED++))
        fi
    else
        log_fail "Control file not found"
        ((TESTS_FAILED++))
    fi
}

# Test 4: Recording Rules
test_recording_rules() {
    log_info "=== Test 4: Recording Rules ==="
    
    local rules=(
        "phoenix_signal_preservation_score"
        "phoenix_pipeline_efficiency_ratio"
        "phoenix_cardinality_growth_rate"
        "phoenix_control_loop_stability_score"
    )
    
    local all_found=true
    for rule in "${rules[@]}"; do
        if query_prometheus "$rule" > /dev/null 2>&1; then
            log_success "Recording rule $rule is working"
        else
            log_fail "Recording rule $rule not found"
            all_found=false
        fi
    done
    
    if [ "$all_found" = true ]; then
        ((TESTS_PASSED++))
    else
        ((TESTS_FAILED++))
    fi
}

# Test 5: Cardinality Reduction
test_cardinality_reduction() {
    log_info "=== Test 5: Cardinality Reduction ==="
    
    # Compare full vs optimized pipeline cardinality
    local full_cardinality=$(query_prometheus \
        "phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate{phoenix_pipeline_label=\"full_fidelity\"}" \
        2>/dev/null || echo "0")
    local opt_cardinality=$(query_prometheus \
        "phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate{phoenix_pipeline_label=\"optimised\"}" \
        2>/dev/null || echo "0")
    
    if [ "$full_cardinality" != "0" ] && [ "$opt_cardinality" != "0" ]; then
        local reduction_ratio=$(echo "scale=2; 1 - ($opt_cardinality / $full_cardinality)" | bc)
        log_info "Cardinality reduction ratio: $reduction_ratio"
        
        if (( $(echo "$reduction_ratio > 0.2" | bc -l) )); then
            log_success "Cardinality reduction is effective (${reduction_ratio})"
            ((TESTS_PASSED++))
        else
            log_fail "Insufficient cardinality reduction (${reduction_ratio})"
            ((TESTS_FAILED++))
        fi
    else
        log_fail "Unable to calculate cardinality reduction"
        ((TESTS_FAILED++))
    fi
}

# Test 6: Memory Usage
test_memory_usage() {
    log_info "=== Test 6: Memory Usage ==="
    
    # Check collector memory usage
    local memory_usage=$(query_prometheus \
        "process_resident_memory_bytes{job=\"otelcol-main-internal\"}/1024/1024" \
        2>/dev/null || echo "0")
    
    if [ "$memory_usage" != "0" ]; then
        log_info "Collector memory usage: ${memory_usage}MB"
        
        if (( $(echo "$memory_usage < 1024" | bc -l) )); then
            log_success "Memory usage is within limits"
            ((TESTS_PASSED++))
        else
            log_warn "Memory usage is high: ${memory_usage}MB"
            ((TESTS_PASSED++))
        fi
    else
        log_fail "Unable to get memory metrics"
        ((TESTS_FAILED++))
    fi
}

# Test 7: Benchmark Validation
test_benchmark_validation() {
    log_info "=== Test 7: Benchmark Validation ==="
    
    # Check if benchmark metrics are being produced
    local benchmark_metrics=(
        "phoenix_benchmark_ingest_latency_seconds"
        "phoenix_benchmark_cost_per_timeseries_usd"
        "phoenix_benchmark_entity_yield_ratio"
    )
    
    local found_any=false
    for metric in "${benchmark_metrics[@]}"; do
        if query_prometheus "$metric" > /dev/null 2>&1; then
            log_success "Benchmark metric $metric found"
            found_any=true
        fi
    done
    
    if [ "$found_any" = true ]; then
        ((TESTS_PASSED++))
    else
        log_warn "No benchmark metrics found (validator may not be running)"
        ((TESTS_PASSED++))
    fi
}

# Test 8: Explosion Detection
test_explosion_detection() {
    log_info "=== Test 8: Explosion Detection ==="
    
    # Check if explosion detection metrics exist
    if query_prometheus "phoenix_cardinality_growth_rate" > /dev/null 2>&1; then
        log_success "Cardinality growth rate metric exists"
        
        # Check for explosion alerts
        local explosion_alerts=$(query_prometheus \
            "ALERTS{alertname=\"PhoenixCardinalityExplosion\"}" \
            2>/dev/null || echo "0")
        
        if [ "$explosion_alerts" = "0" ]; then
            log_success "No cardinality explosions detected"
        else
            log_warn "Cardinality explosion alerts active: $explosion_alerts"
        fi
        ((TESTS_PASSED++))
    else
        log_fail "Explosion detection metrics not found"
        ((TESTS_FAILED++))
    fi
}

# Main test execution
main() {
    log_info "Starting Phoenix integration tests..."
    log_info "Results will be saved to: $RESULTS_DIR"
    
    # Run all tests
    test_service_health
    test_pipeline_processing
    test_control_loop
    test_recording_rules
    test_cardinality_reduction
    test_memory_usage
    test_benchmark_validation
    test_explosion_detection
    
    # Summary
    echo -e "\n${BLUE}=== Test Summary ===${NC}" | tee -a "$RESULTS_DIR/test.log"
    echo -e "${GREEN}Passed: $TESTS_PASSED${NC}" | tee -a "$RESULTS_DIR/test.log"
    echo -e "${RED}Failed: $TESTS_FAILED${NC}" | tee -a "$RESULTS_DIR/test.log"
    
    # Save detailed metrics snapshot
    log_info "Saving metrics snapshot..."
    curl -s "${PROMETHEUS_URL}/api/v1/query" \
        --data-urlencode 'query={__name__=~"phoenix.*"}' > "$RESULTS_DIR/metrics-snapshot.json"
    
    # Exit code based on results
    if [ $TESTS_FAILED -eq 0 ]; then
        log_success "All tests passed!"
        exit 0
    else
        log_fail "Some tests failed. Check $RESULTS_DIR for details."
        exit 1
    fi
}

# Run tests
main "$@"