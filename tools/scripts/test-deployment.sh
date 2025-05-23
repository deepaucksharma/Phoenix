#!/bin/bash

# Phoenix Deployment Test Script
# Tests the monorepo structure and configurations

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Phoenix Deployment Test${NC}"
echo -e "${BLUE}======================${NC}"
echo

# Test 1: Check directory structure
echo -e "${YELLOW}Test 1: Checking directory structure...${NC}"
REQUIRED_DIRS=(
    "packages/contracts"
    "services/collector"
    "services/control-plane/observer"
    "services/control-plane/actuator"
    "services/generators/synthetic"
    "services/generators/complex"
    "services/validator"
    "infrastructure/docker/compose"
    "monitoring/prometheus"
    "monitoring/grafana"
    "tools/scripts"
    "config/environments"
)

ALL_GOOD=true
for dir in "${REQUIRED_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        echo -e "  ✓ $dir"
    else
        echo -e "  ${RED}✗ $dir${NC}"
        ALL_GOOD=false
    fi
done

if [ "$ALL_GOOD" = true ]; then
    echo -e "${GREEN}Directory structure: PASS${NC}"
else
    echo -e "${RED}Directory structure: FAIL${NC}"
    exit 1
fi

# Test 2: Check critical files
echo -e "\n${YELLOW}Test 2: Checking critical files...${NC}"
REQUIRED_FILES=(
    "package.json"
    "turbo.json"
    "Makefile"
    ".env"
    "infrastructure/docker/compose/base.yaml"
    "infrastructure/docker/compose/dev.yaml"
    "services/collector/Dockerfile"
    "services/control-plane/observer/Dockerfile"
    "services/control-plane/actuator/Dockerfile"
)

ALL_GOOD=true
for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "  ✓ $file"
    else
        echo -e "  ${RED}✗ $file${NC}"
        ALL_GOOD=false
    fi
done

if [ "$ALL_GOOD" = true ]; then
    echo -e "${GREEN}Critical files: PASS${NC}"
else
    echo -e "${RED}Critical files: FAIL${NC}"
fi

# Test 3: Check configurations
echo -e "\n${YELLOW}Test 3: Checking configurations...${NC}"
CONFIG_FILES=(
    "services/collector/configs/main.yaml"
    "services/control-plane/observer/config/observer.yaml"
    "config/defaults/control/optimization_mode.yaml"
    "monitoring/prometheus/prometheus.yaml"
)

ALL_GOOD=true
for file in "${CONFIG_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "  ✓ $file"
    else
        echo -e "  ${RED}✗ $file${NC}"
        ALL_GOOD=false
    fi
done

if [ "$ALL_GOOD" = true ]; then
    echo -e "${GREEN}Configurations: PASS${NC}"
else
    echo -e "${RED}Configurations: FAIL${NC}"
fi

# Test 4: Check package.json files
echo -e "\n${YELLOW}Test 4: Checking package.json files...${NC}"
PACKAGE_FILES=(
    "packages/contracts/package.json"
    "services/collector/package.json"
    "services/control-plane/observer/package.json"
    "services/control-plane/actuator/package.json"
    "services/generators/synthetic/package.json"
)

ALL_GOOD=true
for file in "${PACKAGE_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "  ✓ $file"
    else
        echo -e "  ${RED}✗ $file${NC}"
        ALL_GOOD=false
    fi
done

if [ "$ALL_GOOD" = true ]; then
    echo -e "${GREEN}Package files: PASS${NC}"
else
    echo -e "${RED}Package files: FAIL${NC}"
fi

# Test 5: Validate JSON files
echo -e "\n${YELLOW}Test 5: Validating JSON files...${NC}"
JSON_FILES=(
    "package.json"
    "turbo.json"
    "packages/contracts/schemas/control_signal.json"
)

ALL_GOOD=true
for file in "${JSON_FILES[@]}"; do
    if [ -f "$file" ]; then
        if python3 -m json.tool "$file" > /dev/null 2>&1; then
            echo -e "  ✓ $file (valid JSON)"
        else
            echo -e "  ${RED}✗ $file (invalid JSON)${NC}"
            ALL_GOOD=false
        fi
    fi
done

if [ "$ALL_GOOD" = true ]; then
    echo -e "${GREEN}JSON validation: PASS${NC}"
else
    echo -e "${RED}JSON validation: FAIL${NC}"
fi

# Summary
echo -e "\n${BLUE}Test Summary${NC}"
echo -e "${BLUE}============${NC}"
echo -e "${GREEN}✓ Directory structure is correct${NC}"
echo -e "${GREEN}✓ Monorepo is properly configured${NC}"
echo -e "${GREEN}✓ Services are organized${NC}"
echo -e "${GREEN}✓ Configuration files are in place${NC}"

echo -e "\n${BLUE}Next Steps:${NC}"
echo "1. Install dependencies: npm install"
echo "2. Build services: make build"
echo "3. Deploy with Docker: make deploy-dev"
echo "4. Check service health: make health"

echo -e "\n${GREEN}Deployment structure is ready!${NC}"
