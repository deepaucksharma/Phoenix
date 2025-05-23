#!/bin/bash

# Phoenix Health Check Aggregator
# Comprehensive health check for all Phoenix services

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Service endpoints
declare -A SERVICES=(
    ["Main Collector"]="http://localhost:13133/health"
    ["Observer Collector"]="http://localhost:13134/health"
    ["Control Actuator"]="http://localhost:8081/metrics"
    ["Anomaly Detector"]="http://localhost:8082/health"
    ["Benchmark Controller"]="http://localhost:8083/health"
    ["Prometheus"]="http://localhost:9090/-/healthy"
    ["Grafana"]="http://localhost:3000/api/health"
)

# Check individual service
check_service() {
    local name=$1
    local url=$2
    local status
    local response_time
    
    start_time=$(date +%s%N)
    
    if response=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 "$url" 2>/dev/null); then
        end_time=$(date +%s%N)
        response_time=$(( (end_time - start_time) / 1000000 ))
        
        if [[ "$response" == "200" ]]; then
            echo -e "${GREEN}✓${NC} $name: UP (${response_time}ms)"
            return 0
        else
            echo -e "${YELLOW}⚠${NC} $name: DEGRADED (HTTP $response, ${response_time}ms)"
            return 1
        fi
    else
        echo -e "${RED}✗${NC} $name: DOWN"
        return 1
    fi
}

# Check pipeline metrics
check_pipelines() {
    echo -e "\n=== Pipeline Health ==="
    
    for port in 8888 8889 8890; do
        pipeline_name=""
        case $port in
            8888) pipeline_name="Full Fidelity" ;;
            8889) pipeline_name="Optimized" ;;
            8890) pipeline_name="Experimental" ;;
        esac
        
        if metrics=$(curl -s "http://localhost:$port/metrics" 2>/dev/null | grep -c "^[^#]"); then
            echo -e "${GREEN}✓${NC} $pipeline_name Pipeline: $metrics metrics"
        else
            echo -e "${RED}✗${NC} $pipeline_name Pipeline: No metrics"
        fi
    done
}

# Check control system state
check_control_system() {
    echo -e "\n=== Control System State ==="
    
    if control_state=$(curl -s http://localhost:8081/metrics 2>/dev/null); then
        mode=$(echo "$control_state" | jq -r '.current_mode // "unknown"')
        stability=$(echo "$control_state" | jq -r '.stability_score // 0')
        transitions=$(echo "$control_state" | jq -r '.transition_count // 0')
        
        echo "Current Mode: $mode"
        echo "Stability Score: $stability"
        echo "Mode Transitions: $transitions"
        
        if (( $(echo "$stability < 0.5" | bc -l) )); then
            echo -e "${YELLOW}⚠ Warning: Low stability score${NC}"
        fi
    else
        echo -e "${RED}✗ Control system unavailable${NC}"
    fi
}

# Check resource usage
check_resources() {
    echo -e "\n=== Resource Usage ==="
    
    if command -v docker &> /dev/null; then
        docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" | grep -E "(otelcol|control|anomaly|benchmark)" || true
    else
        echo "Docker not available - skipping resource checks"
    fi
}

# Check recent anomalies
check_anomalies() {
    echo -e "\n=== Recent Anomalies ==="
    
    if anomalies=$(curl -s http://localhost:8082/alerts 2>/dev/null); then
        count=$(echo "$anomalies" | jq '. | length')
        critical=$(echo "$anomalies" | jq '[.[] | select(.anomaly.severity == "critical")] | length')
        
        echo "Total Alerts: $count"
        echo "Critical Alerts: $critical"
        
        if [[ $critical -gt 0 ]]; then
            echo -e "${RED}⚠ Critical anomalies detected!${NC}"
            echo "$anomalies" | jq -r '.[] | select(.anomaly.severity == "critical") | "\(.anomaly.metric_name): \(.anomaly.description)"'
        fi
    else
        echo "Anomaly detector unavailable"
    fi
}

# Generate health report
generate_report() {
    local timestamp=$(date -u +"%Y-%m-%d %H:%M:%S UTC")
    local healthy=0
    local total=${#SERVICES[@]}
    
    echo "Phoenix Health Check Report"
    echo "=========================="
    echo "Timestamp: $timestamp"
    echo ""
    echo "=== Service Status ==="
    
    for service in "${!SERVICES[@]}"; do
        if check_service "$service" "${SERVICES[$service]}"; then
            ((healthy++))
        fi
    done
    
    check_pipelines
    check_control_system
    check_anomalies
    check_resources
    
    echo -e "\n=== Summary ==="
    echo "Services: $healthy/$total healthy"
    
    if [[ $healthy -eq $total ]]; then
        echo -e "${GREEN}Overall Status: HEALTHY${NC}"
        exit 0
    elif [[ $healthy -gt $((total / 2)) ]]; then
        echo -e "${YELLOW}Overall Status: DEGRADED${NC}"
        exit 1
    else
        echo -e "${RED}Overall Status: CRITICAL${NC}"
        exit 2
    fi
}

# Main execution
main() {
    case "${1:-report}" in
        report)
            generate_report
            ;;
        json)
            # JSON output for automation
            generate_report | jq -Rs '{report: .}'
            ;;
        nagios)
            # Nagios-compatible output
            if generate_report > /dev/null 2>&1; then
                echo "OK - All services healthy"
                exit 0
            else
                echo "CRITICAL - Services degraded"
                exit 2
            fi
            ;;
        *)
            echo "Usage: $0 [report|json|nagios]"
            exit 1
            ;;
    esac
}

main "$@"