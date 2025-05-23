#!/bin/bash
# Phoenix Setup Validation Script
# Validates symlinks, configurations, and service endpoints

set -euo pipefail

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo "=== Phoenix Setup Validation ==="
echo

# Function to check file/symlink
check_path() {
    local path=$1
    local description=$2
    if [ -e "$path" ]; then
        if [ -L "$path" ]; then
            local target=$(readlink "$path")
            echo -e "${GREEN}✓${NC} $description (symlink -> $target)"
        else
            echo -e "${GREEN}✓${NC} $description"
        fi
    else
        echo -e "${RED}✗${NC} $description"
        return 1
    fi
}

# Validate symlinks
echo "Checking symlinks..."
check_path "scripts/deploy.sh" "Deploy script symlink"
check_path "tests/integration/test_core_functionality.sh" "Integration test symlink"
check_path "tools/scripts/initialize-environment.sh" "Initialize environment symlink"
echo

# Validate configurations
echo "Checking configurations..."
check_path "configs/otel/collectors/main.yaml" "Main collector config"
check_path "configs/otel/collectors/observer.yaml" "Observer collector config"
check_path "configs/monitoring/prometheus/prometheus.yaml" "Prometheus config"
check_path "configs/monitoring/prometheus/rules/phoenix_rules.yml" "Prometheus rules"
check_path "configs/monitoring/prometheus/rules/phoenix_documented_metrics.yml" "Documented metrics"
check_path "configs/control/optimization_mode.yaml" "Control mode config"
echo

# Validate docker-compose services
echo "Checking Docker services..."
if command -v docker-compose &> /dev/null; then
    services=$(docker-compose ps --services 2>/dev/null | wc -l)
    echo -e "${GREEN}✓${NC} Docker Compose available ($services services defined)"
else
    echo -e "${YELLOW}!${NC} Docker Compose not found"
fi
echo

# Validate environment file
echo "Checking environment..."
if [ -f ".env" ]; then
    echo -e "${GREEN}✓${NC} .env file exists"
    # Check for critical variables
    for var in "TARGET_OPTIMIZED_PIPELINE_TS_COUNT" "OTELCOL_MAIN_MEMORY_LIMIT_MIB" "NEW_RELIC_LICENSE_KEY"; do
        if grep -q "^$var=" .env; then
            echo -e "  ${GREEN}✓${NC} $var defined"
        else
            echo -e "  ${YELLOW}!${NC} $var not defined"
        fi
    done
else
    echo -e "${RED}✗${NC} .env file missing"
fi
echo

# Validate data directories
echo "Checking data directories..."
for dir in "data/prometheus" "data/grafana" "data/control-signals"; do
    check_path "$dir" "Directory: $dir"
done
echo

# Summary
echo "=== Validation Complete ==="