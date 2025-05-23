#!/bin/bash

# Phoenix Service Verification Script
# Tests service availability and basic connectivity

set -e

echo "🔍 Phoenix Service Verification Started"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Test result function
test_result() {
    local test_name="$1"
    local status="$2"
    local message="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    case $status in
        "PASS")
            echo -e "✅ ${GREEN}PASS${NC} - $test_name: $message"
            PASSED_TESTS=$((PASSED_TESTS + 1))
            ;;
        "FAIL") 
            echo -e "❌ ${RED}FAIL${NC} - $test_name: $message"
            FAILED_TESTS=$((FAILED_TESTS + 1))
            ;;
        "WARN")
            echo -e "⚠️  ${YELLOW}WARN${NC} - $test_name: $message"
            ;;
        "INFO")
            echo -e "ℹ️  ${BLUE}INFO${NC} - $test_name: $message"
            ;;
    esac
}

echo ""
echo "1. DOCKER SERVICES STATUS"
echo "========================="

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    test_result "Docker Compose" "FAIL" "docker-compose not found"
    exit 1
fi

# Check docker services
echo "Checking docker services..."
if docker-compose ps &> /dev/null; then
    test_result "Docker Compose" "PASS" "Command available"
    
    # Get service status
    services=$(docker-compose ps --services)
    echo "Found services: $services"
    
    for service in $services; do
        status=$(docker-compose ps -q $service | xargs docker inspect --format='{{.State.Status}}' 2>/dev/null || echo "not_found")
        
        case $status in
            "running")
                test_result "Service: $service" "PASS" "Running"
                ;;
            "exited")
                test_result "Service: $service" "FAIL" "Exited"
                ;;
            "not_found")
                test_result "Service: $service" "FAIL" "Not found"
                ;;
            *)
                test_result "Service: $service" "WARN" "Status: $status"
                ;;
        esac
    done
else
    test_result "Docker Services" "FAIL" "Cannot connect to docker daemon or no compose file"
fi

echo ""
echo "2. SERVICE ENDPOINTS"
echo "==================="

# Function to test HTTP endpoint
test_endpoint() {
    local name="$1"
    local url="$2"
    local expected_code="${3:-200}"
    
    if curl -s -f -m 5 "$url" >/dev/null 2>&1; then
        test_result "Endpoint: $name" "PASS" "$url responded"
    else
        # Get actual response code
        code=$(curl -s -o /dev/null -w "%{http_code}" -m 5 "$url" 2>/dev/null || echo "000")
        test_result "Endpoint: $name" "FAIL" "$url returned HTTP $code"
    fi
}

# Test main collector health (needs /health path)
test_endpoint "Main Collector Health" "http://localhost:13133/health"

# Test observer health (needs /health path)
test_endpoint "Observer Health" "http://localhost:13134/health"

# Test control actuator (both documented and actual ports)
test_endpoint "Control Actuator (8081)" "http://localhost:8081/metrics"
test_endpoint "Control Actuator (8080)" "http://localhost:8080/metrics"

# Test Prometheus
test_endpoint "Prometheus" "http://localhost:9090"

# Test Grafana
test_endpoint "Grafana" "http://localhost:3000"

echo ""
echo "3. SERVICE HEALTH DETAILS"
echo "========================="

# Get more detailed health info
echo "Detailed service inspection..."

# Check main collector metrics endpoint
if curl -s -f -m 5 "http://localhost:8888/metrics" >/dev/null 2>&1; then
    metrics_count=$(curl -s -m 5 "http://localhost:8888/metrics" | grep -c "^otelcol_" || echo "0")
    test_result "Main Collector Metrics" "PASS" "$metrics_count OTEL metrics available"
else
    test_result "Main Collector Metrics" "FAIL" "Metrics endpoint not responding"
fi

# Check observer metrics
if curl -s -f -m 5 "http://localhost:9888/metrics" >/dev/null 2>&1; then
    observer_metrics=$(curl -s -m 5 "http://localhost:9888/metrics" | grep -c "phoenix_observer" || echo "0")
    test_result "Observer Metrics" "PASS" "$observer_metrics observer metrics available"
else
    test_result "Observer Metrics" "FAIL" "Observer metrics endpoint not responding"
fi

# Check Prometheus targets with JSON safety
if curl -s -f -m 5 "http://localhost:9090/api/v1/targets" >/dev/null 2>&1; then
    response=$(curl -s -m 5 "http://localhost:9090/api/v1/targets" 2>/dev/null || echo "")
    if echo "$response" | jq . >/dev/null 2>&1; then
        targets=$(echo "$response" | jq -r '.data.activeTargets | length' 2>/dev/null || echo "unknown")
        test_result "Prometheus Targets" "PASS" "$targets active targets"
    else
        test_result "Prometheus Targets" "WARN" "Response not valid JSON"
    fi
else
    test_result "Prometheus Targets" "FAIL" "Cannot query Prometheus targets"
fi

echo ""
echo "4. RESOURCE USAGE"
echo "================="

# Check docker resource usage
if command -v docker &> /dev/null; then
    echo "Current resource usage:"
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" | head -10
    
    # Check main collector memory (should be under 1024MB) - use flexible container name matching
    main_memory=$(docker stats --no-stream --format "{{.Name}} {{.MemUsage}}" | grep -E "(otelcol-main|collector-main)" | head -1 | cut -d' ' -f2 | cut -d'/' -f1 | sed 's/MiB//' | sed 's/GiB/*1024/' | bc 2>/dev/null || echo "0")
    
    if [ "$main_memory" != "0" ]; then
        if (( $(echo "$main_memory < 1024" | bc -l) )); then
            test_result "Main Collector Memory" "PASS" "${main_memory}MB (under 1024MB limit)"
        else
            test_result "Main Collector Memory" "WARN" "${main_memory}MB (over 1024MB limit)"
        fi
    else
        test_result "Main Collector Memory" "INFO" "Could not determine memory usage"
    fi
fi

echo ""
echo "SUMMARY"
echo "======="
echo "Total Tests: $TOTAL_TESTS"
echo "Passed: $PASSED_TESTS"
echo "Failed: $FAILED_TESTS"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✅ All service tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ $FAILED_TESTS test(s) failed${NC}"
    exit 1
fi