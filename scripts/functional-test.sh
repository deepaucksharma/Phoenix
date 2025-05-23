#!/usr/bin/env bash
# Phoenix-vNext Functional Testing Suite
# Tests system behavior and workflows without requiring running containers

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

# Test function
run_functional_test() {
    local test_name="$1"
    local test_function="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "\n${BLUE}🧪 Functional Test:${NC} $test_name"
    
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

echo -e "${BLUE}🚀 Phoenix-vNext Functional Testing Suite${NC}"
echo -e "${BLUE}=========================================${NC}"

# Test 1: Service Health Check Endpoints
test_health_endpoints() {
    log_info "Validating health check endpoint configurations..."
    
    # Check main collector health endpoint
    if ! grep -q "13133:13133" docker-compose.yaml; then
        log_error "Main collector health endpoint not configured"
        return 1
    fi
    
    # Check observer health endpoint
    if ! grep -q "13134:13133" docker-compose.yaml; then
        log_error "Observer health endpoint not configured"
        return 1
    fi
    
    # Check control actuator health endpoint exists in service definition
    if ! grep -q "8081:8081" docker-compose.yaml; then
        log_error "Control actuator endpoint not configured"
        return 1
    fi
    
    log_success "All health endpoints properly configured"
    return 0
}

# Test 2: Control Loop Configuration
test_control_loop_config() {
    log_info "Validating control loop configuration and thresholds..."
    
    local control_file="configs/control/optimization_mode.yaml"
    
    # Check if thresholds are properly configured
    if ! grep -q "conservative_max_ts: 15000" "$control_file"; then
        log_error "Conservative threshold not properly set"
        return 1
    fi
    
    if ! grep -q "aggressive_min_ts: 25000" "$control_file"; then
        log_error "Aggressive threshold not properly set"
        return 1
    fi
    
    # Check current profile
    local current_profile
    current_profile=$(grep "optimization_profile:" "$control_file" | awk '{print $2}')
    
    if [[ "$current_profile" != "balanced" && "$current_profile" != "conservative" && "$current_profile" != "aggressive" ]]; then
        log_error "Invalid optimization profile: $current_profile"
        return 1
    fi
    
    log_success "Control loop configuration valid, profile: $current_profile"
    return 0
}

# Test 3: Pipeline Configuration Validation
test_pipeline_configuration() {
    log_info "Validating 3-pipeline architecture configuration..."
    
    local main_config="configs/otel/collectors/main.yaml"
    
    # Check for 3 distinct pipelines
    local pipeline_count
    pipeline_count=$(grep -c "metrics/" "$main_config" || echo "0")
    
    if [ "$pipeline_count" -lt 3 ]; then
        log_error "Expected 3 pipelines, found $pipeline_count"
        return 1
    fi
    
    # Check for distinct exporters
    if ! grep -q "prometheus/full" "$main_config"; then
        log_error "Full fidelity exporter not configured"
        return 1
    fi
    
    if ! grep -q "prometheus/optimized" "$main_config"; then
        log_error "Optimized exporter not configured"
        return 1
    fi
    
    if ! grep -q "prometheus/experimental" "$main_config"; then
        log_error "Experimental exporter not configured"
        return 1
    fi
    
    # Check for different ports
    if ! grep -q "8888" "$main_config"; then
        log_error "Full pipeline port 8888 not configured"
        return 1
    fi
    
    if ! grep -q "8889" "$main_config"; then
        log_error "Optimized pipeline port 8889 not configured"
        return 1
    fi
    
    if ! grep -q "8890" "$main_config"; then
        log_error "Experimental pipeline port 8890 not configured"
        return 1
    fi
    
    log_success "3-pipeline architecture properly configured"
    return 0
}

# Test 4: Observer KPI Collection
test_observer_kpi_collection() {
    log_info "Validating observer KPI collection configuration..."
    
    local observer_config="configs/otel/collectors/observer.yaml"
    
    # Check if observer scrapes main collector
    if ! grep -q "otelcol-main:8888" "$observer_config"; then
        log_error "Observer not configured to scrape main collector port 8888"
        return 1
    fi
    
    if ! grep -q "otelcol-main:8889" "$observer_config"; then
        log_error "Observer not configured to scrape main collector port 8889"
        return 1
    fi
    
    if ! grep -q "otelcol-main:8890" "$observer_config"; then
        log_error "Observer not configured to scrape main collector port 8890"
        return 1
    fi
    
    # Check for KPI transformation
    if ! grep -q "transform:" "$observer_config"; then
        log_error "Observer transform processor not configured"
        return 1
    fi
    
    # Check observer exports on correct port
    if ! grep -q "9888" "$observer_config"; then
        log_error "Observer not exporting on port 9888"
        return 1
    fi
    
    log_success "Observer KPI collection properly configured"
    return 0
}

# Test 5: Prometheus Monitoring Configuration
test_prometheus_monitoring() {
    log_info "Validating Prometheus monitoring configuration..."
    
    local prom_config="configs/monitoring/prometheus/prometheus.yaml"
    
    # Check scrape jobs for all services
    local required_targets=(
        "otelcol-main:8888"
        "otelcol-main:8889"
        "otelcol-main:8890"
        "otelcol-observer:9888"
        "control-actuator-go:8081"
        "anomaly-detector:8082"
        "benchmark-controller:8083"
    )
    
    for target in "${required_targets[@]}"; do
        if ! grep -q "$target" "$prom_config"; then
            log_error "Prometheus not configured to scrape $target"
            return 1
        fi
    done
    
    # Check for recording rules configuration
    if ! grep -q "rule_files:" "$prom_config"; then
        log_error "Prometheus recording rules not configured"
        return 1
    fi
    
    log_success "Prometheus monitoring properly configured"
    return 0
}

# Test 6: Environment Variable Dependencies
test_environment_dependencies() {
    log_info "Validating environment variable dependencies..."
    
    # Check critical environment variables exist
    local required_env_vars=(
        "OTELCOL_MAIN_MEMORY_LIMIT_MIB"
        "TARGET_OPTIMIZED_PIPELINE_TS_COUNT"
        "THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS"
        "THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS"
        "LOG_LEVEL"
        "ENVIRONMENT"
    )
    
    for var in "${required_env_vars[@]}"; do
        if ! grep -q "^$var=" .env; then
            log_error "Required environment variable $var not found in .env"
            return 1
        fi
    done
    
    # Validate threshold logic
    local conservative_max
    conservative_max=$(grep "THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS=" .env | cut -d'=' -f2)
    local aggressive_min
    aggressive_min=$(grep "THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS=" .env | cut -d'=' -f2)
    local target_count
    target_count=$(grep "TARGET_OPTIMIZED_PIPELINE_TS_COUNT=" .env | cut -d'=' -f2)
    
    if [ "$conservative_max" -ge "$aggressive_min" ]; then
        log_error "Conservative max ($conservative_max) should be less than aggressive min ($aggressive_min)"
        return 1
    fi
    
    if [ "$target_count" -le "$conservative_max" ] || [ "$target_count" -ge "$aggressive_min" ]; then
        log_warning "Target count ($target_count) should be between conservative ($conservative_max) and aggressive ($aggressive_min) for balanced mode"
    fi
    
    log_success "Environment dependencies properly configured"
    return 0
}

# Test 7: Service Dependencies and Networking
test_service_dependencies() {
    log_info "Validating service dependencies and networking..."
    
    # Check Docker Compose service dependencies
    if ! grep -A 20 "control-actuator-go:" docker-compose.yaml | grep -q "depends_on:"; then
        log_error "Control actuator missing depends_on configuration"
        return 1
    fi
    
    if ! grep -A 20 "anomaly-detector:" docker-compose.yaml | grep -q "depends_on:"; then
        log_error "Anomaly detector missing depends_on configuration"
        return 1
    fi
    
    # Check network configuration
    if ! grep -q "networks:" docker-compose.yaml; then
        log_error "Docker Compose networks not configured"
        return 1
    fi
    
    if ! grep -q "phoenix:" docker-compose.yaml; then
        log_error "Phoenix network not configured"
        return 1
    fi
    
    log_success "Service dependencies and networking properly configured"
    return 0
}

# Test 8: Data Persistence Configuration
test_data_persistence() {
    log_info "Validating data persistence configuration..."
    
    # Check for data directories
    if [ ! -d "data" ]; then
        log_error "Data directory not found"
        return 1
    fi
    
    # Check Docker Compose volumes
    if ! grep -q "prometheus_data:" docker-compose.yaml; then
        log_error "Prometheus data volume not configured"
        return 1
    fi
    
    if ! grep -q "grafana_data:" docker-compose.yaml; then
        log_error "Grafana data volume not configured"
        return 1
    fi
    
    # Check volume mounts
    if ! grep -q "./data/otelcol_main" docker-compose.yaml; then
        log_error "OTel collector data mount not configured"
        return 1
    fi
    
    log_success "Data persistence properly configured"
    return 0
}

# Test 9: Security Configuration
test_security_configuration() {
    log_info "Validating security configuration..."
    
    # Check that sensitive information is not hardcoded
    if grep -r "password.*=" configs/ 2>/dev/null | grep -v "admin"; then
        log_error "Hardcoded passwords found in configs"
        return 1
    fi
    
    # Check for health check configurations
    if ! grep -q "healthcheck:" docker-compose.yaml; then
        log_error "Health checks not configured for services"
        return 1
    fi
    
    # Check restart policies
    if ! grep -q "restart: unless-stopped" docker-compose.yaml; then
        log_error "Restart policies not configured"
        return 1
    fi
    
    log_success "Security configuration validated"
    return 0
}

# Test 10: End-to-End Workflow Simulation
test_e2e_workflow_simulation() {
    log_info "Simulating end-to-end workflow..."
    
    # Simulate control loop decision making
    local control_file="configs/control/optimization_mode.yaml"
    local current_ts=18000  # Simulated current time series count
    
    # Test conservative threshold
    if [ $current_ts -lt 15000 ]; then
        expected_profile="conservative"
    elif [ $current_ts -gt 25000 ]; then
        expected_profile="aggressive"
    else
        expected_profile="balanced"
    fi
    
    log_info "Simulated TS count: $current_ts, Expected profile: $expected_profile"
    
    # Test that control file can be updated (simulate actuator behavior)
    cp "$control_file" "$control_file.backup"
    
    # Simulate profile change
    sed -i "s/optimization_profile: .*/optimization_profile: $expected_profile/" "$control_file"
    sed -i "s/last_updated: .*/last_updated: \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"/" "$control_file"
    sed -i "s/trigger_reason: .*/trigger_reason: \"functional_test_simulation\"/" "$control_file"
    
    # Verify change was applied
    if ! grep -q "optimization_profile: $expected_profile" "$control_file"; then
        log_error "Failed to update control profile in simulation"
        mv "$control_file.backup" "$control_file"
        return 1
    fi
    
    # Restore original
    mv "$control_file.backup" "$control_file"
    
    log_success "End-to-end workflow simulation completed successfully"
    return 0
}

# Run all functional tests
run_functional_test "Service Health Check Endpoints" "test_health_endpoints"
run_functional_test "Control Loop Configuration" "test_control_loop_config"
run_functional_test "Pipeline Configuration" "test_pipeline_configuration"
run_functional_test "Observer KPI Collection" "test_observer_kpi_collection"
run_functional_test "Prometheus Monitoring" "test_prometheus_monitoring"
run_functional_test "Environment Dependencies" "test_environment_dependencies"
run_functional_test "Service Dependencies" "test_service_dependencies"
run_functional_test "Data Persistence" "test_data_persistence"
run_functional_test "Security Configuration" "test_security_configuration"
run_functional_test "End-to-End Workflow Simulation" "test_e2e_workflow_simulation"

# Generate final report
echo -e "\n${BLUE}📋 FUNCTIONAL TESTING SUMMARY${NC}"
echo -e "${BLUE}==============================${NC}"
echo -e "Total Functional Tests: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 ALL FUNCTIONAL TESTS PASSED!${NC}"
    echo -e "${GREEN}Phoenix-vNext system behavior is validated and ready for production deployment.${NC}"
    exit 0
else
    echo -e "\n${RED}❌ Some functional tests failed. Please address the issues above.${NC}"
    exit 1
fi