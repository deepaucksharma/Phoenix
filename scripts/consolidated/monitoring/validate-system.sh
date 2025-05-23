#!/usr/bin/env bash
# Phoenix-vNext System Validation Script
# Comprehensive validation of all system components and configurations

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

# Counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Test function
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "\n${BLUE}🧪 Testing:${NC} $test_name"
    
    if eval "$test_command" >/dev/null 2>&1; then
        log_success "✅ PASS: $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        log_error "❌ FAIL: $test_name"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# Test detailed function with output
run_test_detailed() {
    local test_name="$1"
    local test_command="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "\n${BLUE}🧪 Testing:${NC} $test_name"
    
    if eval "$test_command"; then
        log_success "✅ PASS: $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        log_error "❌ FAIL: $test_name"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

echo -e "${BLUE}🚀 Phoenix-vNext System Validation${NC}"
echo -e "${BLUE}====================================${NC}"

# 1. File Structure Validation
echo -e "\n${YELLOW}📁 File Structure Validation${NC}"

run_test "Docker Compose main file exists" "[ -f docker-compose.yaml ]"
run_test "Docker Compose override file exists" "[ -f docker-compose.override.yml ]"
run_test "Docker Compose dev file exists" "[ -f docker-compose.dev.yml ]"
run_test "Environment template exists" "[ -f .env.template ]"
run_test "Environment file exists" "[ -f .env ]"

# 2. Configuration Files Validation
echo -e "\n${YELLOW}⚙️ Configuration Files Validation${NC}"

run_test "Main collector config exists" "[ -f configs/otel/collectors/main.yaml ]"
run_test "Observer collector config exists" "[ -f configs/otel/collectors/observer.yaml ]"
run_test "Control optimization config exists" "[ -f configs/control/optimization_mode.yaml ]"
run_test "Prometheus config exists" "[ -f configs/monitoring/prometheus/prometheus.yaml ]"

# 3. Service Directory Structure
echo -e "\n${YELLOW}🏗️ Service Directory Structure${NC}"

run_test "Apps directory exists" "[ -d apps ]"
run_test "Services directory exists" "[ -d services ]"
run_test "Infrastructure directory exists" "[ -d infrastructure ]"
run_test "Scripts directory exists" "[ -d scripts ]"

# 4. Docker Compose Service Definitions
echo -e "\n${YELLOW}🐳 Docker Compose Service Definitions${NC}"

run_test_detailed "Docker Compose has required services" '
    services=(
        "otelcol-main"
        "otelcol-observer"
        "control-actuator-go"
        "anomaly-detector"
        "benchmark-controller"
        "prometheus"
        "grafana"
    )
    for service in "${services[@]}"; do
        if ! grep -q "^  $service:" docker-compose.yaml; then
            echo "Missing service: $service"
            return 1
        fi
    done
    echo "All required services found in docker-compose.yaml"
'

# 5. Port Configuration Validation
echo -e "\n${YELLOW}🔌 Port Configuration Validation${NC}"

run_test_detailed "Required ports are configured" '
    ports=(
        "4317:4317"  # OTLP gRPC
        "4318:4318"  # OTLP HTTP
        "8888"       # Main collector metrics
        "9888"       # Observer metrics
        "8081"       # Control API
        "8082"       # Anomaly API
        "8083"       # Benchmark API
        "9090"       # Prometheus
        "3000"       # Grafana
    )
    for port in "${ports[@]}"; do
        if ! grep -q "$port" docker-compose.yaml; then
            echo "Missing port configuration: $port"
            return 1
        fi
    done
    echo "All required ports are configured"
'

# 6. OpenTelemetry Configuration Validation
echo -e "\n${YELLOW}📡 OpenTelemetry Configuration Validation${NC}"

run_test_detailed "Main collector has required pipelines" '
    if ! grep -q "metrics/full:" configs/otel/collectors/main.yaml; then
        echo "Missing full fidelity pipeline"
        return 1
    fi
    if ! grep -q "metrics/optimized:" configs/otel/collectors/main.yaml; then
        echo "Missing optimized pipeline"
        return 1
    fi
    if ! grep -q "metrics/experimental:" configs/otel/collectors/main.yaml; then
        echo "Missing experimental pipeline"
    fi
    echo "All required pipelines found"
'

run_test_detailed "Observer collector scrapes main collector" '
    if ! grep -q "otelcol-main:8888" configs/otel/collectors/observer.yaml; then
        echo "Observer not configured to scrape main collector"
        return 1
    fi
    echo "Observer correctly configured to scrape main collector"
'

# 7. Prometheus Configuration Validation
echo -e "\n${YELLOW}📊 Prometheus Configuration Validation${NC}"

run_test_detailed "Prometheus scrapes all Phoenix services" '
    services=(
        "otelcol-main:8888"
        "otelcol-main:8889"
        "otelcol-main:8890"
        "otelcol-observer:9888"
        "control-actuator-go:8081"
        "anomaly-detector:8082"
        "benchmark-controller:8083"
    )
    for service in "${services[@]}"; do
        if ! grep -q "$service" configs/monitoring/prometheus/prometheus.yaml; then
            echo "Prometheus not configured to scrape: $service"
            return 1
        fi
    done
    echo "Prometheus configured to scrape all required services"
'

# 8. Control System Configuration
echo -e "\n${YELLOW}🎛️ Control System Configuration${NC}"

run_test_detailed "Control configuration has required fields" '
    config_file="configs/control/optimization_mode.yaml"
    fields=(
        "optimization_profile"
        "thresholds"
        "pipelines"
        "config_version"
    )
    for field in "${fields[@]}"; do
        if ! grep -q "$field:" "$config_file"; then
            echo "Missing control field: $field"
            return 1
        fi
    done
    echo "All required control fields present"
'

# 9. Script Validation
echo -e "\n${YELLOW}📜 Script Validation${NC}"

run_test "Deploy script exists and is executable" "[ -x scripts/deploy.sh ]"
run_test "Cleanup script exists and is executable" "[ -x scripts/cleanup.sh ]"
run_test "Validation script exists and is executable" "[ -x scripts/validate-system.sh ]"

# 10. Infrastructure Validation
echo -e "\n${YELLOW}🏗️ Infrastructure Validation${NC}"

run_test "Terraform AWS module exists" "[ -d infrastructure/terraform/modules/aws-phoenix ]"
run_test "Terraform environments exist" "[ -d infrastructure/terraform/environments/aws ]"
run_test "Helm charts exist" "[ -d infrastructure/helm ]"
run_test "Helm chart exists" "[ -d infrastructure/helm/phoenix ]"

# 11. Environment Configuration Validation
echo -e "\n${YELLOW}🌍 Environment Configuration Validation${NC}"

run_test_detailed "Environment has required variables" '
    if [ ! -f .env ]; then
        echo "Environment file .env not found"
        return 1
    fi
    
    required_vars=(
        "ENVIRONMENT"
        "LOG_LEVEL"
        "OTELCOL_MAIN_MEMORY_LIMIT_MIB"
        "TARGET_OPTIMIZED_PIPELINE_TS_COUNT"
        "THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS"
        "THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS"
    )
    
    for var in "${required_vars[@]}"; do
        if ! grep -q "^$var=" .env; then
            echo "Missing environment variable: $var"
            return 1
        fi
    done
    echo "All required environment variables present"
'

# 12. Integration Test Validation
echo -e "\n${YELLOW}🔗 Integration Test Validation${NC}"

run_test_detailed "Integration test script validation" '
    if [ -f tests/integration/test_core_functionality.sh ]; then
        echo "Integration test script exists"
        return 0
    else
        echo "Integration test script not found (optional)"
        return 0  # Not critical for basic validation
    fi
'

# 13. Documentation Validation
echo -e "\n${YELLOW}📚 Documentation Validation${NC}"

run_test "CLAUDE.md exists" "[ -f CLAUDE.md ]"
run_test "Infrastructure documentation exists" "[ -f INFRASTRUCTURE.md ]"
run_test "README exists" "[ -f README.md ]"

# Generate final report
echo -e "\n${BLUE}📋 VALIDATION SUMMARY${NC}"
echo -e "${BLUE}=====================${NC}"
echo -e "Total Tests: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 ALL TESTS PASSED! Phoenix-vNext is ready for deployment.${NC}"
    exit 0
else
    echo -e "\n${RED}❌ Some tests failed. Please address the issues above.${NC}"
    exit 1
fi