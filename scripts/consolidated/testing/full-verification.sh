#!/bin/bash

# Phoenix Full System Verification Script
# Runs all verification tests and generates comprehensive report

set -e

echo "🚀 Phoenix Full System Verification"
echo "==================================="
echo "Starting comprehensive system verification..."
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Timestamp for reporting
TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")
REPORT_DIR="test-reports"
REPORT_FILE="$REPORT_DIR/verification_report_$TIMESTAMP.md"

# Create report directory
mkdir -p "$REPORT_DIR"

# Initialize report
cat > "$REPORT_FILE" << EOF
# Phoenix System Verification Report

**Date**: $(date)  
**Version**: $(git rev-parse --short HEAD 2>/dev/null || echo "unknown")  
**Branch**: $(git branch --show-current 2>/dev/null || echo "unknown")  
**Tester**: $USER  

## Executive Summary

This report contains the results of a comprehensive verification of the Phoenix Cardinality Optimization System.

EOF

# Function to run test and capture results
run_test_script() {
    local script_name="$1"
    local script_path="docs/scripts/$script_name"
    local section_title="$2"
    
    echo -e "${BLUE}Running $section_title...${NC}"
    
    if [ -f "$script_path" ]; then
        # Make script executable
        chmod +x "$script_path"
        
        # Run script and capture output
        echo "## $section_title" >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
        echo "\`\`\`" >> "$REPORT_FILE"
        
        if ./"$script_path" >> "$REPORT_FILE" 2>&1; then
            echo -e "✅ ${GREEN}$section_title completed successfully${NC}"
            local result="PASS"
        else
            echo -e "❌ ${RED}$section_title completed with failures${NC}"
            local result="FAIL"
        fi
        
        echo "\`\`\`" >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
        echo "**Result**: $result" >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
        
        return $?
    else
        echo -e "⚠️  ${YELLOW}Script $script_path not found, skipping...${NC}"
        echo "## $section_title" >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
        echo "**Result**: SKIPPED - Script not found" >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
        return 1
    fi
}

# Check prerequisites
echo -e "${BLUE}Checking prerequisites...${NC}"

# Check if we're in the Phoenix directory
if [ ! -f "CLAUDE.md" ] || [ ! -f "docker-compose.yaml" ]; then
    echo -e "${RED}❌ Not in Phoenix project directory${NC}"
    echo "Please run this script from the Phoenix project root directory"
    exit 1
fi

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running${NC}"
    echo "Please start Docker and try again"
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose >/dev/null 2>&1; then
    echo -e "${RED}❌ docker-compose not found${NC}"
    echo "Please install docker-compose and try again"
    exit 1
fi

echo -e "${GREEN}✅ Prerequisites check passed${NC}"
echo ""

# Initialize environment if needed
echo -e "${BLUE}Initializing environment...${NC}"
if [ -f "scripts/initialize-environment.sh" ]; then
    chmod +x scripts/initialize-environment.sh
    ./scripts/initialize-environment.sh
    echo -e "${GREEN}✅ Environment initialized${NC}"
else
    echo -e "${YELLOW}⚠️  Environment initialization script not found${NC}"
fi

# Start Phoenix system
echo -e "${BLUE}Starting Phoenix system...${NC}"
if [ -f "run-phoenix.sh" ]; then
    chmod +x run-phoenix.sh
    ./run-phoenix.sh
    echo -e "${GREEN}✅ Phoenix system started${NC}"
else
    echo "Starting with docker-compose..."
    docker-compose up -d
    echo -e "${GREEN}✅ Services started with docker-compose${NC}"
fi

# Wait for services to start
echo -e "${BLUE}Waiting for services to start (30 seconds)...${NC}"
sleep 30

# Run verification tests
echo ""
echo -e "${BLUE}Starting verification tests...${NC}"

# Track overall results
TOTAL_CATEGORIES=0
PASSED_CATEGORIES=0

# 1. Service Availability Tests
if run_test_script "verify-services.sh" "Service Availability Tests"; then
    PASSED_CATEGORIES=$((PASSED_CATEGORIES + 1))
fi
TOTAL_CATEGORIES=$((TOTAL_CATEGORIES + 1))

# 2. API Endpoints Tests
if run_test_script "verify-apis.sh" "API Endpoints Tests"; then
    PASSED_CATEGORIES=$((PASSED_CATEGORIES + 1))
fi
TOTAL_CATEGORIES=$((TOTAL_CATEGORIES + 1))

# 3. Configuration Tests
if run_test_script "verify-configs.sh" "Configuration Tests"; then
    PASSED_CATEGORIES=$((PASSED_CATEGORIES + 1))
fi
TOTAL_CATEGORIES=$((TOTAL_CATEGORIES + 1))

# 4. Additional manual checks
echo -e "${BLUE}Running additional verification checks...${NC}"

echo "## Additional Verification Checks" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Check system resource usage
echo "### System Resource Usage" >> "$REPORT_FILE"
echo "\`\`\`" >> "$REPORT_FILE"
docker stats --no-stream >> "$REPORT_FILE" 2>&1
echo "\`\`\`" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Check service logs for errors
echo "### Recent Service Logs (Last 20 lines)" >> "$REPORT_FILE"
for service in otelcol-main otelcol-observer control-actuator-go anomaly-detector prometheus grafana; do
    echo "#### $service logs:" >> "$REPORT_FILE"
    echo "\`\`\`" >> "$REPORT_FILE"
    docker-compose logs --tail=20 "$service" >> "$REPORT_FILE" 2>&1 || echo "Service $service not found" >> "$REPORT_FILE"
    echo "\`\`\`" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
done

# Generate summary
echo ""
echo -e "${BLUE}Generating verification summary...${NC}"

# Add summary to report
cat >> "$REPORT_FILE" << EOF

## Summary

**Test Categories**: $TOTAL_CATEGORIES  
**Passed Categories**: $PASSED_CATEGORIES  
**Failed Categories**: $((TOTAL_CATEGORIES - PASSED_CATEGORIES))  
**Success Rate**: $(( PASSED_CATEGORIES * 100 / TOTAL_CATEGORIES ))%

EOF

# Add recommendations based on results
if [ $PASSED_CATEGORIES -eq $TOTAL_CATEGORIES ]; then
    cat >> "$REPORT_FILE" << EOF
### Overall Result: ✅ SYSTEM HEALTHY

All verification categories passed successfully. The Phoenix system appears to be functioning correctly.

### Next Steps:
1. Run performance tests if needed
2. Validate specific use cases
3. Monitor system over time

EOF
    echo -e "${GREEN}🎉 All verification tests passed! System is healthy.${NC}"
else
    cat >> "$REPORT_FILE" << EOF
### Overall Result: ⚠️ ISSUES FOUND

Some verification categories failed. Review the detailed results above for specific issues.

### Recommended Actions:
1. Review failed test categories
2. Check service logs for errors
3. Verify configuration files
4. Ensure all services are properly started
5. Check network connectivity between services

### Common Issues:
- Service path mismatches in docker-compose.yaml
- Missing API endpoint implementations
- Configuration files in wrong locations
- Port conflicts or misalignments
- Missing environment variables

EOF
    echo -e "${YELLOW}⚠️  Some tests failed. Check the report for details.${NC}"
fi

# Display report location
echo ""
echo -e "${BLUE}📄 Verification report generated:${NC}"
echo "   $REPORT_FILE"
echo ""
echo -e "${BLUE}📊 Quick view of report:${NC}"
tail -20 "$REPORT_FILE"

echo ""
echo -e "${BLUE}🔍 To view full report:${NC}"
echo "   cat $REPORT_FILE"
echo "   # or"
echo "   code $REPORT_FILE"

# Optional: Open report in browser if markdown viewer available
if command -v mdless >/dev/null 2>&1; then
    echo ""
    echo -e "${BLUE}📖 View formatted report:${NC}"
    echo "   mdless $REPORT_FILE"
fi

# Final status
echo ""
if [ $PASSED_CATEGORIES -eq $TOTAL_CATEGORIES ]; then
    echo -e "${GREEN}✅ Phoenix verification completed successfully!${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠️  Phoenix verification completed with issues.${NC}"
    echo -e "Check the report at: ${BLUE}$REPORT_FILE${NC}"
    exit 1
fi