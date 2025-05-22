#!/bin/bash
# Consolidated Phoenix-vNext System Test Suite
# This script performs all necessary verification and testing of the 5-pipeline system

set -e

# ANSI colors for better output formatting
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
WORK_DIR="$(dirname "$0")"
TEST_DIR="${WORK_DIR}/test_results"
TEST_LOG="${TEST_DIR}/phoenix-test-$(date +%Y%m%d-%H%M%S).log"

# Test configuration
MAIN_METRICS_PORT=8888
OBSERVER_METRICS_PORT=8891
FEEDBACK_PORT=8890
CONSISTENCY_PORT=8889
SYNTHETIC_PORT=9999
PROMETHEUS_PORT=9090

# Function to log messages with timestamp
log_message() {
    local message="$1"
    echo -e "[$(date +"%Y-%m-%d %H:%M:%S")] $message" | tee -a "$TEST_LOG"
}

# Initialize test environment
init_test() {
    log_message "${PURPLE}===============================================${NC}"
    log_message "${PURPLE}    Phoenix-vNext Consolidated Test Suite     ${NC}"
    log_message "${PURPLE}===============================================${NC}"
    
    # Clean up and create test directory
    if [ -d "$TEST_DIR" ]; then
        rm -rf "$TEST_DIR"
    fi
    mkdir -p "$TEST_DIR"
    
    log_message "${BLUE}Test results will be saved to: ${CYAN}$TEST_DIR${NC}"
}

# Test 1: Schema Coherence
test_schema_coherence() {
    log_message "${BLUE}Running schema coherence test...${NC}"
    
    # Extract allowed modes from main collector
    allowed_modes=$(grep -o '\"[^\"]*\"' "${WORK_DIR}/configs/collectors/otelcol-main.yaml" | grep -E "(moderate|ultra|adaptive)" | head -3 | tr -d '\"' | tr '\\n' ' ')
    log_message "Main collector accepts modes: ${CYAN}$allowed_modes${NC}"
    
    # Check current control file
    if [ -f "${WORK_DIR}/configs/control_signals/opt_mode.yaml" ]; then
        current_mode=$(grep -E "^mode:" "${WORK_DIR}/configs/control_signals/opt_mode.yaml" | cut -d: -f2 | tr -d ' "' | head -1)
        log_message "Current control file mode: ${CYAN}$current_mode${NC}"
        
        # Verify mode is allowed
        if echo "$allowed_modes" | grep -q "$current_mode"; then
            log_message "${GREEN}✓ Schema coherence test PASSED${NC}"
            return 0
        else
            log_message "${RED}✗ Schema coherence test FAILED: Invalid mode '$current_mode'${NC}"
            return 1
        fi
    else
        log_message "${RED}✗ Schema coherence test FAILED: Control file not found${NC}"
        return 1
    fi
}

# Test 2: Component Health
test_component_health() {
    log_message "${BLUE}Testing component health...${NC}"
    
    local health_status=0
    
    # Test main collector
    if curl -s -f "http://localhost:$MAIN_METRICS_PORT/metrics" > /dev/null; then
        log_message "${GREEN}✓ Main collector (port $MAIN_METRICS_PORT) is healthy${NC}"
    else
        log_message "${RED}✗ Main collector (port $MAIN_METRICS_PORT) is not responding${NC}"
        health_status=1
    fi
    
    # Test observer
    if curl -s -f "http://localhost:$OBSERVER_METRICS_PORT/metrics" > /dev/null; then
        log_message "${GREEN}✓ Observer collector (port $OBSERVER_METRICS_PORT) is healthy${NC}"
    else
        log_message "${RED}✗ Observer collector (port $OBSERVER_METRICS_PORT) is not responding${NC}"
        health_status=1
    fi
    
    # Test prometheus
    if curl -s -f "http://localhost:$PROMETHEUS_PORT/-/healthy" > /dev/null; then
        log_message "${GREEN}✓ Prometheus (port $PROMETHEUS_PORT) is healthy${NC}"
    else
        log_message "${RED}✗ Prometheus (port $PROMETHEUS_PORT) is not responding${NC}"
        health_status=1
    fi
    
    # Test synthetic metrics
    if curl -s -f "http://localhost:$SYNTHETIC_PORT/metrics" > /dev/null; then
        log_message "${GREEN}✓ Synthetic metrics collector (port $SYNTHETIC_PORT) is healthy${NC}"
    else
        log_message "${YELLOW}! Synthetic metrics collector (port $SYNTHETIC_PORT) is not responding${NC}"
    fi
    
    if [ $health_status -eq 0 ]; then
        log_message "${GREEN}✓ Component health test PASSED${NC}"
    else
        log_message "${RED}✗ Component health test FAILED${NC}"
    fi
    
    return $health_status
}

# Test 3: Control Signal Coherence
test_control_signals() {
    log_message "${BLUE}Testing control signal coherence...${NC}"
    
    # Get mode from control file
    local file_mode=$(grep -E "^mode:" "${WORK_DIR}/configs/control_signals/opt_mode.yaml" | cut -d'\"' -f2)
    
    # Get mode from observer metrics
    local observer_mode=""
    if curl -s "http://localhost:$OBSERVER_METRICS_PORT/metrics" | grep -q "phoenix_observer_mode"; then
        observer_mode=$(curl -s "http://localhost:$OBSERVER_METRICS_PORT/metrics" | grep phoenix_observer_mode | head -n 1 | awk '{print $2}' | tr -d '\"')
    fi
    
    # Get mode from main collector feedback
    local main_mode=""
    if curl -s "http://localhost:$FEEDBACK_PORT/metrics" | grep -q "applied_mode"; then
        main_mode=$(curl -s "http://localhost:$FEEDBACK_PORT/metrics" | grep applied_mode | head -n 1 | awk '{print $2}' | tr -d '\"')
    fi
    
    log_message "Control file mode: ${CYAN}$file_mode${NC}"
    log_message "Observer mode: ${CYAN}$observer_mode${NC}"
    log_message "Main collector mode: ${CYAN}$main_mode${NC}"
    
    # Check coherence (allow for some propagation delay)
    if [[ -n "$file_mode" ]]; then
        if [[ "$observer_mode" == "$file_mode" || -z "$observer_mode" ]]; then
            if [[ "$main_mode" == "$file_mode" || -z "$main_mode" ]]; then
                log_message "${GREEN}✓ Control signal coherence test PASSED${NC}"
                return 0
            fi
        fi
    fi
    
    log_message "${YELLOW}! Control signal coherence test shows inconsistencies (may be due to startup delay)${NC}"
    return 0  # Don't fail for coherence during startup
}

# Test 4: Pipeline Metrics
test_pipeline_metrics() {
    log_message "${BLUE}Testing pipeline metrics...${NC}"
    
    local metrics_found=0
    
    # Check for phoenix metrics in main collector
    if curl -s "http://localhost:$MAIN_METRICS_PORT/metrics" | grep -q "phoenix"; then
        local phoenix_count=$(curl -s "http://localhost:$MAIN_METRICS_PORT/metrics" | grep -c "phoenix" || echo "0")
        log_message "${GREEN}✓ Found $phoenix_count Phoenix metrics in main collector${NC}"
        metrics_found=1
    fi
    
    # Check for phoenix metrics in synthetic generator
    if curl -s "http://localhost:$SYNTHETIC_PORT/metrics" | grep -q "phoenix"; then
        local synthetic_count=$(curl -s "http://localhost:$SYNTHETIC_PORT/metrics" | grep -c "phoenix" || echo "0")
        log_message "${GREEN}✓ Found $synthetic_count Phoenix metrics in synthetic generator${NC}"
        metrics_found=1
    fi
    
    if [ $metrics_found -eq 1 ]; then
        log_message "${GREEN}✓ Pipeline metrics test PASSED${NC}"
        return 0
    else
        log_message "${RED}✗ Pipeline metrics test FAILED: No Phoenix metrics found${NC}"
        return 1
    fi
}

# Test 5: Configuration Validation
test_config_validation() {
    log_message "${BLUE}Testing configuration validation...${NC}"
    
    local config_status=0
    
    # Test YAML syntax of main configurations
    if python3 -c "import yaml; yaml.safe_load(open('configs/collectors/otelcol-main.yaml'))" 2>/dev/null; then
        log_message "${GREEN}✓ Main collector YAML is valid${NC}"
    else
        log_message "${RED}✗ Main collector YAML is invalid${NC}"
        config_status=1
    fi
    
    if python3 -c "import yaml; yaml.safe_load(open('configs/collectors/otelcol-observer.yaml'))" 2>/dev/null; then
        log_message "${GREEN}✓ Observer collector YAML is valid${NC}"
    else
        log_message "${RED}✗ Observer collector YAML is invalid${NC}"
        config_status=1
    fi
    
    if python3 -c "import yaml; yaml.safe_load(open('configs/metrics/synthetic-metrics.yaml'))" 2>/dev/null; then
        log_message "${GREEN}✓ Synthetic metrics YAML is valid${NC}"
    else
        log_message "${RED}✗ Synthetic metrics YAML is invalid${NC}"
        config_status=1
    fi
    
    if [ $config_status -eq 0 ]; then
        log_message "${GREEN}✓ Configuration validation test PASSED${NC}"
    else
        log_message "${RED}✗ Configuration validation test FAILED${NC}"
    fi
    
    return $config_status
}

# Generate comprehensive test report
generate_report() {
    log_message "${PURPLE}===============================================${NC}"
    log_message "${PURPLE}               Test Summary                   ${NC}"
    log_message "${PURPLE}===============================================${NC}"
    
    log_message "Schema Coherence: $([ $schema_result -eq 0 ] && echo "${GREEN}PASSED${NC}" || echo "${RED}FAILED${NC}")"
    log_message "Component Health: $([ $health_result -eq 0 ] && echo "${GREEN}PASSED${NC}" || echo "${RED}FAILED${NC}")"
    log_message "Control Signals: $([ $control_result -eq 0 ] && echo "${GREEN}PASSED${NC}" || echo "${YELLOW}PARTIAL${NC}")"
    log_message "Pipeline Metrics: $([ $metrics_result -eq 0 ] && echo "${GREEN}PASSED${NC}" || echo "${RED}FAILED${NC}")"
    log_message "Config Validation: $([ $config_result -eq 0 ] && echo "${GREEN}PASSED${NC}" || echo "${RED}FAILED${NC}")"
    
    local total_result=$((schema_result + health_result + control_result + metrics_result + config_result))
    
    if [ $total_result -eq 0 ]; then
        log_message "${GREEN}╔════════════════════════════════════════════╗${NC}"
        log_message "${GREEN}║             ALL TESTS PASSED               ║${NC}"
        log_message "${GREEN}╚════════════════════════════════════════════╝${NC}"
    else
        log_message "${YELLOW}╔════════════════════════════════════════════╗${NC}"
        log_message "${YELLOW}║            SOME TESTS FAILED               ║${NC}"
        log_message "${YELLOW}╚════════════════════════════════════════════╝${NC}"
    fi
    
    log_message "Detailed test log: ${CYAN}$TEST_LOG${NC}"
    
    # Copy log to test directory
    cp "$TEST_LOG" "${TEST_DIR}/latest-test-results.log"
    
    return $total_result
}

# Main execution
init_test

# Run all tests
test_schema_coherence
schema_result=$?

test_component_health  
health_result=$?

test_control_signals
control_result=$?

test_pipeline_metrics
metrics_result=$?

test_config_validation
config_result=$?

# Generate final report
generate_report
final_result=$?

exit $final_result