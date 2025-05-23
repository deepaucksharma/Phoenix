#!/bin/bash

# Phoenix API Endpoints Verification Script
# Tests all documented API endpoints for availability and functionality

set -e

echo "🔌 Phoenix API Verification Started"
echo "==================================="

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

# Function to test API endpoint
test_api() {
    local name="$1"
    local method="$2"
    local url="$3"
    local data="$4"
    local expected_code="${5:-200}"
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "%{http_code}" -X "$method" -H "Content-Type: application/json" -d "$data" -m 5 "$url" 2>/dev/null || echo "000")
    else
        response=$(curl -s -w "%{http_code}" -X "$method" -m 5 "$url" 2>/dev/null || echo "000")
    fi
    
    code="${response: -3}"
    body="${response%???}"
    
    case $code in
        "200"|"201"|"204")
            test_result "$name" "PASS" "HTTP $code"
            if [ -n "$body" ] && [ "$body" != "000" ]; then
                echo "    Response preview: ${body:0:100}..."
            fi
            ;;
        "404")
            test_result "$name" "FAIL" "HTTP $code - Endpoint not implemented"
            ;;
        "405")
            test_result "$name" "FAIL" "HTTP $code - Method not allowed"
            ;;
        "500")
            test_result "$name" "FAIL" "HTTP $code - Server error"
            ;;
        "000")
            test_result "$name" "FAIL" "Connection failed/timeout"
            ;;
        *)
            test_result "$name" "WARN" "HTTP $code - Unexpected response"
            ;;
    esac
}

echo ""
echo "1. CONTROL ACTUATOR API"
echo "======================="

# Test documented port (8081) and actual port (8080)
for port in 8081 8080; do
    echo "Testing Control Actuator on port $port..."
    
    # Health endpoint (documented but likely missing)
    test_api "Health Check ($port)" "GET" "http://localhost:$port/health"
    
    # Metrics endpoint (should work)
    test_api "Metrics ($port)" "GET" "http://localhost:$port/metrics"
    
    # Mode control (documented but likely missing)
    test_api "Mode Control ($port)" "POST" "http://localhost:$port/mode" '{"mode": "aggressive"}'
    
    # Anomaly webhook (documented but likely missing)
    test_api "Anomaly Webhook ($port)" "POST" "http://localhost:$port/anomaly" '{"type": "cardinality_spike", "severity": "high"}'
done

echo ""
echo "2. ANOMALY DETECTOR API"
echo "======================="

# All these endpoints are documented but likely missing
test_api "Anomaly Health Check" "GET" "http://localhost:8082/health"
test_api "Active Alerts" "GET" "http://localhost:8082/alerts" 
test_api "Anomaly Metrics" "GET" "http://localhost:8082/metrics"

echo ""
echo "3. BENCHMARK CONTROLLER API"
echo "============================"

# All these endpoints are documented but likely missing
test_api "List Scenarios" "GET" "http://localhost:8083/benchmark/scenarios"
test_api "Run Benchmark" "POST" "http://localhost:8083/benchmark/run" '{"scenario": "baseline_steady_state"}'
test_api "Benchmark Results" "GET" "http://localhost:8083/benchmark/results"
test_api "Validate SLOs" "GET" "http://localhost:8083/benchmark/validate"

echo ""
echo "4. OPENTELEMETRY COLLECTOR APIs"
echo "==============================="

# Main collector endpoints (fix health path)
test_api "Main Collector Health" "GET" "http://localhost:13133/health"
test_api "Main Collector Metrics" "GET" "http://localhost:8888/metrics"

# Optimized pipeline  
test_api "Optimized Pipeline Metrics" "GET" "http://localhost:8889/metrics"

# Experimental pipeline
test_api "Experimental Pipeline Metrics" "GET" "http://localhost:8890/metrics"

# Observer collector (fix health path)
test_api "Observer Health" "GET" "http://localhost:13134/health"
test_api "Observer Metrics" "GET" "http://localhost:9888/metrics"

echo ""
echo "5. DEBUG ENDPOINTS"
echo "=================="

# pprof endpoint
test_api "pprof Debug" "GET" "http://localhost:1777/debug/pprof/"

# zpages endpoint  
test_api "zpages ServiceZ" "GET" "http://localhost:55679/debug/servicez"
test_api "zpages TracezZ" "GET" "http://localhost:55679/debug/tracez"

echo ""
echo "6. MONITORING STACK APIs"
echo "========================"

# Prometheus API
test_api "Prometheus Query API" "GET" "http://localhost:9090/api/v1/query?query=up"
test_api "Prometheus Targets" "GET" "http://localhost:9090/api/v1/targets"
test_api "Prometheus Rules" "GET" "http://localhost:9090/api/v1/rules"

# Grafana API
test_api "Grafana Health" "GET" "http://localhost:3000/api/health"

echo ""
echo "7. FUNCTIONAL API TESTS"
echo "======================="

# Test if control actuator metrics contain expected data
echo "Testing Control Actuator metrics content..."
for port in 8080 8081; do
    if curl -s -f -m 5 "http://localhost:$port/metrics" >/dev/null 2>&1; then
        metrics_response=$(curl -s -m 5 "http://localhost:$port/metrics" 2>/dev/null || echo "")
        
        if echo "$metrics_response" | grep -q "current_mode\|optimization_mode"; then
            test_result "Control Metrics Content ($port)" "PASS" "Contains mode information"
        elif echo "$metrics_response" | jq . >/dev/null 2>&1; then
            test_result "Control Metrics Content ($port)" "WARN" "Valid JSON but missing expected fields"
        else
            test_result "Control Metrics Content ($port)" "FAIL" "Invalid response format"
        fi
        break
    fi
done

# Test Prometheus for Phoenix-specific metrics with JSON safety
echo "Testing Prometheus for Phoenix metrics..."
prom_response=$(curl -s -m 5 "http://localhost:9090/api/v1/label/__name__/values" 2>/dev/null || echo "")
if echo "$prom_response" | jq . >/dev/null 2>&1; then
    phoenix_metrics=$(echo "$prom_response" | jq -r '.data[]' 2>/dev/null | grep -c "phoenix" || echo "0")
    if [ "$phoenix_metrics" -gt 0 ]; then
        test_result "Phoenix Metrics in Prometheus" "PASS" "$phoenix_metrics phoenix metrics found"
    else
        test_result "Phoenix Metrics in Prometheus" "FAIL" "No phoenix metrics found"
    fi
else
    test_result "Phoenix Metrics in Prometheus" "FAIL" "Prometheus not responding or invalid JSON"
fi

# Test Observer KPI metrics specifically with JSON safety
echo "Testing Observer KPI metrics..."
kpi_query="phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate"
kpi_response=$(curl -s -m 5 "http://localhost:9090/api/v1/query?query=$kpi_query" 2>/dev/null || echo "")

if echo "$kpi_response" | jq . >/dev/null 2>&1; then
    if echo "$kpi_response" | jq -e '.data.result | length > 0' >/dev/null 2>&1; then
        result_count=$(echo "$kpi_response" | jq -r '.data.result | length')
        test_result "Observer KPI Metrics" "PASS" "$result_count cardinality estimates available"
    else
        test_result "Observer KPI Metrics" "FAIL" "No cardinality estimates found"
    fi
else
    test_result "Observer KPI Metrics" "FAIL" "Prometheus not responding or invalid JSON"
fi

echo ""
echo "8. DATA FLOW VERIFICATION"
echo "========================="

# Check if metrics are flowing from generator to collector
echo "Checking metrics flow..."

# Check receiver metrics in main collector
receiver_metrics=$(curl -s -m 5 "http://localhost:8888/metrics" 2>/dev/null | grep -c "otelcol_receiver_accepted" || echo "0")
if [ "$receiver_metrics" -gt 0 ]; then
    test_result "Metrics Ingestion" "PASS" "Receiver metrics found"
else
    test_result "Metrics Ingestion" "FAIL" "No receiver metrics found"
fi

# Check processor metrics
processor_metrics=$(curl -s -m 5 "http://localhost:8888/metrics" 2>/dev/null | grep -c "otelcol_processor" || echo "0") 
if [ "$processor_metrics" -gt 0 ]; then
    test_result "Metrics Processing" "PASS" "Processor metrics found"
else
    test_result "Metrics Processing" "FAIL" "No processor metrics found"
fi

# Check exporter metrics
exporter_metrics=$(curl -s -m 5 "http://localhost:8888/metrics" 2>/dev/null | grep -c "otelcol_exporter" || echo "0")
if [ "$exporter_metrics" -gt 0 ]; then
    test_result "Metrics Export" "PASS" "Exporter metrics found"
else
    test_result "Metrics Export" "FAIL" "No exporter metrics found"
fi

echo ""
echo "SUMMARY"
echo "======="
echo "Total Tests: $TOTAL_TESTS"
echo "Passed: $PASSED_TESTS"
echo "Failed: $FAILED_TESTS"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✅ All API tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ $FAILED_TESTS test(s) failed${NC}"
    echo ""
    echo "Common issues:"
    echo "- Control actuator port mismatch (8080 vs 8081)"
    echo "- Missing API endpoint implementations"
    echo "- Services not fully started"
    echo ""
    echo "Recommendations:"
    echo "1. Check service logs: docker-compose logs [service-name]"
    echo "2. Verify services are running: docker-compose ps"
    echo "3. Check network connectivity: docker network ls"
    exit 1
fi