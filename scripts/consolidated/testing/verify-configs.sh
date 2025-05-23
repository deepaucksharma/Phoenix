#!/bin/bash

# Phoenix Configuration Verification Script
# Tests configuration files, structure, and runtime configuration

set -e

echo "⚙️  Phoenix Configuration Verification Started"
echo "=============================================="

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

# Function to check file exists
check_file() {
    local name="$1"
    local path="$2"
    local required="${3:-true}"
    
    if [ -f "$path" ]; then
        test_result "File: $name" "PASS" "$path exists"
        return 0
    elif [ -d "$path" ]; then
        test_result "File: $name" "WARN" "$path is a directory, not a file"
        return 1
    else
        if [ "$required" = "true" ]; then
            test_result "File: $name" "FAIL" "$path not found"
        else
            test_result "File: $name" "INFO" "$path not found (optional)"
        fi
        return 1
    fi
}

# Function to check directory exists
check_directory() {
    local name="$1"
    local path="$2"
    local required="${3:-true}"
    
    if [ -d "$path" ]; then
        file_count=$(ls -1 "$path" 2>/dev/null | wc -l)
        test_result "Directory: $name" "PASS" "$path exists with $file_count files"
        return 0
    else
        if [ "$required" = "true" ]; then
            test_result "Directory: $name" "FAIL" "$path not found"
        else
            test_result "Directory: $name" "INFO" "$path not found (optional)"
        fi
        return 1
    fi
}

# Function to validate YAML
validate_yaml() {
    local name="$1"
    local path="$2"
    
    if [ ! -f "$path" ]; then
        test_result "YAML: $name" "FAIL" "$path not found"
        return 1
    fi
    
    if command -v yq &> /dev/null; then
        if yq eval '.' "$path" >/dev/null 2>&1; then
            test_result "YAML: $name" "PASS" "Valid YAML syntax"
        else
            test_result "YAML: $name" "FAIL" "Invalid YAML syntax"
        fi
    elif python3 -c "import yaml" 2>/dev/null; then
        if python3 -c "import yaml; yaml.safe_load(open('$path'))" >/dev/null 2>&1; then
            test_result "YAML: $name" "PASS" "Valid YAML syntax"
        else
            test_result "YAML: $name" "FAIL" "Invalid YAML syntax"
        fi
    else
        test_result "YAML: $name" "INFO" "Cannot validate YAML (yq/python3+yaml not available)"
    fi
}

echo ""
echo "1. CONFIGURATION DIRECTORY STRUCTURE"
echo "===================================="

# Check main config directories
check_directory "Main configs" "configs"
check_directory "OTEL configs" "configs/otel"
check_directory "OTEL collectors" "configs/otel/collectors"
check_directory "OTEL exporters" "configs/otel/exporters"
check_directory "Control configs" "configs/control" 
check_directory "Monitoring configs" "configs/monitoring"
check_directory "Templates" "configs/templates"

# Check for missing/documented directories
check_directory "OTEL processors (documented)" "configs/otel/processors" "false"
check_directory "Grafana dashboards" "configs/monitoring/grafana/dashboards" "false"

echo ""
echo "2. CORE CONFIGURATION FILES"
echo "==========================="

# OTEL Collector configurations
check_file "Main collector config" "configs/otel/collectors/main.yaml"
check_file "Observer collector config" "configs/otel/collectors/observer.yaml"

# Check for documented but potentially missing files
check_file "Common processors (documented)" "configs/otel/processors/common_intake_processors.yaml" "false"
check_file "Main optimized config" "configs/otel/collectors/main-optimized.yaml" "false"

# Control system files
check_file "Optimization mode" "configs/control/optimization_mode.yaml"
check_file "Control template (templates)" "configs/templates/control/optimization_mode_template.yaml" "false"
check_file "Control template (documented location)" "configs/control/optimization_mode_template.yaml" "false"

# Monitoring configurations
check_file "Prometheus config" "configs/monitoring/prometheus/prometheus.yaml"
check_file "Grafana datasource" "configs/monitoring/grafana/grafana-datasource.yaml" "false"

echo ""
echo "3. YAML VALIDATION"
echo "=================="

# Validate key YAML files
if [ -f "configs/otel/collectors/main.yaml" ]; then
    validate_yaml "Main collector" "configs/otel/collectors/main.yaml"
fi

if [ -f "configs/otel/collectors/observer.yaml" ]; then
    validate_yaml "Observer collector" "configs/otel/collectors/observer.yaml"
fi

if [ -f "configs/control/optimization_mode.yaml" ]; then
    validate_yaml "Optimization mode" "configs/control/optimization_mode.yaml"
fi

if [ -f "configs/monitoring/prometheus/prometheus.yaml" ]; then
    validate_yaml "Prometheus config" "configs/monitoring/prometheus/prometheus.yaml"
fi

if [ -f "docker-compose.yaml" ]; then
    validate_yaml "Docker Compose" "docker-compose.yaml"
fi

echo ""
echo "4. CONFIGURATION CONTENT VALIDATION"
echo "==================================="

# Check control file structure
if [ -f "configs/control/optimization_mode.yaml" ]; then
    echo "Checking optimization mode configuration..."
    
    if grep -q "mode:" "configs/control/optimization_mode.yaml"; then
        mode=$(grep "mode:" "configs/control/optimization_mode.yaml" | cut -d':' -f2 | xargs)
        case $mode in
            "conservative"|"balanced"|"aggressive")
                test_result "Control Mode Value" "PASS" "Valid mode: $mode"
                ;;
            *)
                test_result "Control Mode Value" "FAIL" "Invalid mode: $mode"
                ;;
        esac
    else
        test_result "Control Mode Field" "FAIL" "No 'mode' field found"
    fi
    
    if grep -q "config_version:" "configs/control/optimization_mode.yaml"; then
        test_result "Control Version Field" "PASS" "config_version field present"
    else
        test_result "Control Version Field" "FAIL" "config_version field missing"
    fi
    
    if grep -q "correlation_id:" "configs/control/optimization_mode.yaml"; then
        test_result "Control Correlation Field" "PASS" "correlation_id field present"
    else
        test_result "Control Correlation Field" "FAIL" "correlation_id field missing"
    fi
fi

# Check main collector for pipeline configuration
if [ -f "configs/otel/collectors/main.yaml" ]; then
    echo "Checking main collector pipelines..."
    
    if grep -q "pipeline_full_fidelity" "configs/otel/collectors/main.yaml"; then
        test_result "Full Fidelity Pipeline" "PASS" "Pipeline configuration found"
    else
        test_result "Full Fidelity Pipeline" "FAIL" "Pipeline configuration missing"
    fi
    
    if grep -q "pipeline_optimised" "configs/otel/collectors/main.yaml"; then
        test_result "Optimized Pipeline" "PASS" "Pipeline configuration found"
    else
        test_result "Optimized Pipeline" "FAIL" "Pipeline configuration missing"
    fi
    
    if grep -q "pipeline_experimental" "configs/otel/collectors/main.yaml"; then
        test_result "Experimental Pipeline" "PASS" "Pipeline configuration found"
    else
        test_result "Experimental Pipeline" "FAIL" "Pipeline configuration missing"
    fi
fi

echo ""
echo "5. ENVIRONMENT CONFIGURATION"
echo "============================"

# Check .env file
if [ -f ".env" ]; then
    test_result "Environment File" "PASS" ".env file exists"
    
    # Check for documented environment variables
    env_vars=(
        "TARGET_OPTIMIZED_PIPELINE_TS_COUNT"
        "THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS"
        "THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS"
        "HYSTERESIS_FACTOR"
        "OTELCOL_MAIN_MEMORY_LIMIT_MIB"
        "ADAPTIVE_CONTROLLER_INTERVAL_SECONDS"
        "NEW_RELIC_LICENSE_KEY"
    )
    
    for var in "${env_vars[@]}"; do
        if grep -q "^$var=" ".env"; then
            value=$(grep "^$var=" ".env" | cut -d'=' -f2)
            test_result "Env Var: $var" "PASS" "Set to: $value"
        else
            test_result "Env Var: $var" "FAIL" "Not found in .env"
        fi
    done
else
    test_result "Environment File" "FAIL" ".env file not found"
fi

echo ""
echo "6. RUNTIME CONFIGURATION"
echo "========================"

# Check if control file is being updated
if [ -f "configs/control/optimization_mode.yaml" ]; then
    echo "Checking control file timestamps..."
    
    initial_timestamp=$(stat -c %Y "configs/control/optimization_mode.yaml" 2>/dev/null || stat -f %m "configs/control/optimization_mode.yaml" 2>/dev/null || echo "0")
    
    if [ "$initial_timestamp" != "0" ]; then
        test_result "Control File Timestamp" "PASS" "File timestamp accessible"
        
        # Check if file has been modified recently (within last 5 minutes)
        current_time=$(date +%s)
        age=$((current_time - initial_timestamp))
        
        if [ $age -lt 300 ]; then  # 5 minutes
            test_result "Control File Freshness" "PASS" "Modified ${age}s ago (recent)"
        else
            test_result "Control File Freshness" "WARN" "Modified ${age}s ago (may be stale)"
        fi
    else
        test_result "Control File Timestamp" "FAIL" "Cannot read file timestamp"
    fi
fi

# Check docker-compose configuration
if command -v docker-compose &> /dev/null; then
    if docker-compose config >/dev/null 2>&1; then
        test_result "Docker Compose Config" "PASS" "Configuration valid"
        
        # Check for service path issues
        if docker-compose config | grep -q "services/generators/synthetic"; then
            test_result "Synthetic Generator Path" "PASS" "Correct service path"
        elif docker-compose config | grep -q "apps/synthetic-generator"; then
            test_result "Synthetic Generator Path" "FAIL" "Using old/incorrect path"
        else
            test_result "Synthetic Generator Path" "WARN" "Service path unclear"
        fi
        
        # Check for port mappings
        if docker-compose config | grep -q "8081:808"; then
            test_result "Control Actuator Port" "WARN" "Port mapping may be misaligned"
        else
            test_result "Control Actuator Port" "INFO" "Port mapping not in standard format"
        fi
        
    else
        test_result "Docker Compose Config" "FAIL" "Configuration invalid"
    fi
fi

echo ""
echo "7. CONFIGURATION CONSISTENCY"
echo "============================"

# Check for consistency between documented and actual configurations
echo "Checking configuration consistency..."

# Check if Prometheus recording rules match documentation format
if [ -f "configs/monitoring/prometheus/rules/phoenix_rules.yml" ] || [ -f "configs/monitoring/prometheus/rules/phoenix_core_rules.yml" ]; then
    rules_file=""
    [ -f "configs/monitoring/prometheus/rules/phoenix_rules.yml" ] && rules_file="configs/monitoring/prometheus/rules/phoenix_rules.yml"
    [ -f "configs/monitoring/prometheus/rules/phoenix_core_rules.yml" ] && rules_file="configs/monitoring/prometheus/rules/phoenix_core_rules.yml"
    
    if [ -n "$rules_file" ]; then
        # Check for colon-format rules (documented)
        colon_rules=$(grep -c "phoenix:" "$rules_file" 2>/dev/null || echo "0")
        underscore_rules=$(grep -c "phoenix_" "$rules_file" 2>/dev/null || echo "0")
        
        if [ "$colon_rules" -gt 0 ]; then
            test_result "Recording Rules Format" "PASS" "$colon_rules rules use colon format (documented)"
        elif [ "$underscore_rules" -gt 0 ]; then
            test_result "Recording Rules Format" "WARN" "$underscore_rules rules use underscore format (undocumented)"
        else
            test_result "Recording Rules Format" "FAIL" "No phoenix rules found"
        fi
    fi
fi

# Check Grafana dashboard availability
if [ -d "configs/monitoring/grafana/dashboards" ]; then
    dashboard_count=$(ls -1 "configs/monitoring/grafana/dashboards"/*.json 2>/dev/null | wc -l)
    if [ "$dashboard_count" -gt 0 ]; then
        test_result "Grafana Dashboards" "PASS" "$dashboard_count dashboard(s) found"
    else
        test_result "Grafana Dashboards" "FAIL" "No dashboard JSON files found"
    fi
else
    # Check archive location
    if [ -d "archive/_cleanup_2025_05_24/monitoring/grafana/dashboards" ]; then
        archived_count=$(ls -1 "archive/_cleanup_2025_05_24/monitoring/grafana/dashboards"/*.json 2>/dev/null | wc -l)
        test_result "Grafana Dashboards" "WARN" "$archived_count dashboard(s) only in archive"
    else
        test_result "Grafana Dashboards" "FAIL" "No dashboards found anywhere"
    fi
fi

echo ""
echo "SUMMARY"
echo "======="
echo "Total Tests: $TOTAL_TESTS"
echo "Passed: $PASSED_TESTS"
echo "Failed: $FAILED_TESTS"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✅ All configuration tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ $FAILED_TESTS test(s) failed${NC}"
    echo ""
    echo "Common configuration issues found:"
    echo "- Missing configuration directories"
    echo "- Files in wrong locations (templates vs configs)"
    echo "- Service path mismatches in docker-compose"
    echo "- Environment variables not properly set"
    echo ""
    echo "Recommendations:"
    echo "1. Create missing directories: mkdir -p configs/otel/processors"
    echo "2. Move template files to correct locations"
    echo "3. Fix service paths in docker-compose.yaml"
    echo "4. Validate .env file against documentation"
    exit 1
fi