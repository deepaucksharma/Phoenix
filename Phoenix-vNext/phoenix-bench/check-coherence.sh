#!/bin/bash
# Phoenix-vNext System Coherence Monitoring Script
# This script checks the health and coherence of the entire Phoenix-vNext system

# Color constants for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Phoenix-vNext System Coherence Check ===${NC}"
echo "$(date)"
echo

# Function to check component health
check_component() {
    local component=$1
    local endpoint=$2
    
    echo -e "${BLUE}Checking $component at $endpoint...${NC}"
    
    # Try curl with a timeout
    if curl -s --max-time 5 "$endpoint" > /dev/null; then
        echo -e "${GREEN}✓ $component is UP${NC}"
        return 0
    else
        echo -e "${RED}✗ $component is DOWN or unreachable${NC}"
        return 1
    fi
}

# Function to check control signal coherence
check_control_signals() {
    echo -e "${BLUE}Checking control signal coherence...${NC}"
    
    # Get active mode from observer
    observer_mode=$(curl -s http://localhost:9889/metrics | grep phoenix_opt_mode | head -n 1 | awk '{print $2}')
    
    # Get applied mode from main collector
    main_mode=$(curl -s http://localhost:8890/metrics | grep applied_mode | head -n 1 | awk '{print $2}')
    
    if [ "$observer_mode" = "$main_mode" ]; then
        echo -e "${GREEN}✓ Mode settings coherent: $observer_mode${NC}"
        return 0
    else
        echo -e "${RED}✗ Mode mismatch: Observer=$observer_mode, Main=$main_mode${NC}"
        return 1
    fi
}

# Function to check metric family coherence
check_metric_families() {
    echo -e "${BLUE}Checking metric family coherence...${NC}"
    
    # Check if related metrics are flowing through consistent pipelines
    consistency=$(curl -s http://localhost:8889/metrics | grep phoenix_pipeline_coherence | wc -l)
    
    if [ $consistency -gt 0 ]; then
        echo -e "${GREEN}✓ Metric families are coherent ($consistency checks passed)${NC}"
        return 0
    else
        echo -e "${RED}✗ Metric family coherence issues detected${NC}"
        return 1
    fi
}

# Check configuration files
check_config_files() {
    echo -e "${BLUE}Checking configuration files...${NC}"
    
    issues=0
    
    # Check main collector config
    if [ ! -f "/etc/otelcol/otelcol-main.yaml" ]; then
        echo -e "${RED}✗ Main collector config not found${NC}"
        issues=$((issues+1))
    fi
    
    # Check observer config
    if [ ! -f "/etc/otelcol/otelcol-observer.yaml" ]; then
        echo -e "${RED}✗ Observer config not found${NC}"
        issues=$((issues+1)) 
    fi
    
    # Check control signal file
    control_file="/etc/otelcol/control_signals/opt_mode.yaml"
    if [ ! -f "$control_file" ]; then
        echo -e "${RED}✗ Control signal file not found${NC}"
        issues=$((issues+1))
    else
        # Check for required fields in control file
        if ! grep -q "mode:" "$control_file" || ! grep -q "correlation_id:" "$control_file"; then
            echo -e "${YELLOW}⚠ Control file missing required fields${NC}"
            issues=$((issues+1))
        fi
    fi
    
    if [ $issues -eq 0 ]; then
        echo -e "${GREEN}✓ All configuration files are valid${NC}"
        return 0
    else
        echo -e "${RED}✗ Configuration file issues: $issues${NC}"
        return 1
    fi
}

# Check for pipeline switching stability
check_stability() {
    echo -e "${BLUE}Checking pipeline switching stability...${NC}"
    
    # Count mode switches in the last hour
    switches=$(curl -s 'http://localhost:9090/api/v1/query' --data-urlencode 'query=count_over_time(delta(phoenix_opt_mode[1h]))' | grep -o "value.*" | grep -o "[0-9]\+\.[0-9]\+")
    
    # Round to nearest integer
    switches=$(printf "%.0f" "$switches")
    
    if [ "$switches" -lt 5 ]; then
        echo -e "${GREEN}✓ Pipeline switching stable ($switches switches in last hour)${NC}"
        return 0
    else
        echo -e "${YELLOW}⚠ Frequent pipeline switching detected ($switches switches in last hour)${NC}"
        return 1
    fi
}

# Run all checks
run_all_checks() {
    failures=0
    
    echo "Checking core components..."
    check_component "Main Collector" "http://localhost:8888/metrics" || failures=$((failures+1))
    check_component "Observer Collector" "http://localhost:9889/metrics" || failures=$((failures+1))
    check_component "Prometheus" "http://localhost:9090/-/healthy" || failures=$((failures+1))
    check_component "Grafana" "http://localhost:3000/api/health" || failures=$((failures+1))
    
    echo
    check_control_signals || failures=$((failures+1))
    echo
    check_metric_families || failures=$((failures+1))
    echo
    check_config_files || failures=$((failures+1))
    echo
    check_stability || failures=$((failures+1))
    
    echo
    if [ $failures -eq 0 ]; then
        echo -e "${GREEN}✓ All coherence checks passed${NC}"
    else
        echo -e "${RED}✗ $failures coherence checks failed${NC}"
    fi
    
    return $failures
}

# Main execution
run_all_checks

# Add a recommendation based on the state
echo 
echo -e "${BLUE}=== System Recommendations ===${NC}"
if [ $? -eq 0 ]; then
    echo -e "${GREEN}System is operating coherently. No action needed.${NC}"
else
    echo -e "${YELLOW}Coherence issues detected. Recommended actions:${NC}"
    echo "1. Check for version mismatches between components"
    echo "2. Inspect the correlation IDs in /etc/otelcol/control_signals/opt_mode.yaml" 
    echo "3. Restart components in this order: observer → main → synthetic-metrics"
    echo "4. Check logs for pipeline selection errors or mode transition failures"
fi

echo
echo -e "${BLUE}==== End of coherence check ====${NC}"
