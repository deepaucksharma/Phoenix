#!/bin/bash

# Script to verify the functionality of the new processors
# This script builds Phoenix, runs tests, and verifies basic functionality

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Starting verification of new processors...${NC}"

# Step 1: Build Phoenix
echo -e "${YELLOW}Building Phoenix...${NC}"
make build || { echo -e "${RED}Build failed!${NC}"; exit 1; }
echo -e "${GREEN}Build successful!${NC}"

# Step 2: Run unit tests for the new processors
echo -e "${YELLOW}Running unit tests for timeseries_estimator...${NC}"
go test -v ./test/processors/timeseries_estimator/... || { echo -e "${RED}Timeseries estimator tests failed!${NC}"; exit 1; }
echo -e "${GREEN}Timeseries estimator tests passed!${NC}"

echo -e "${YELLOW}Running unit tests for cpu_histogram_converter...${NC}"
go test -v ./test/processors/cpu_histogram_converter/... || { echo -e "${RED}CPU histogram converter tests failed!${NC}"; exit 1; }
echo -e "${GREEN}CPU histogram converter tests passed!${NC}"

# Step 3: Run benchmarks
echo -e "${YELLOW}Running benchmarks for timeseries_estimator...${NC}"
go test -v ./test/benchmarks/processors/timeseries_estimator_benchmark_test.go -bench=. -benchtime=1s || { echo -e "${RED}Timeseries estimator benchmarks failed!${NC}"; exit 1; }
echo -e "${GREEN}Timeseries estimator benchmarks completed!${NC}"

echo -e "${YELLOW}Running benchmarks for cpu_histogram_converter...${NC}"
go test -v ./test/benchmarks/processors/cpu_histogram_converter_benchmark_test.go -bench=. -benchtime=1s || { echo -e "${RED}CPU histogram converter benchmarks failed!${NC}"; exit 1; }
echo -e "${GREEN}CPU histogram converter benchmarks completed!${NC}"

# Step 4: Verify state persistence
echo -e "${YELLOW}Verifying state persistence...${NC}"

# Create temporary directory for state
TMP_DIR=$(mktemp -d)
STATE_FILE="$TMP_DIR/cpu_state.json"

# Run a basic test with state persistence
echo "Running test with state persistence at $STATE_FILE"
CONFIG=$(cat <<EOF
processors:
  cpu_histogram_converter:
    enabled: true
    input_metric_name: "process.cpu.time"
    output_metric_name: "process.cpu.utilization.histogram"
    state_storage_path: "$STATE_FILE"
    state_flush_interval_seconds: 1
EOF
)

# Write test config
echo "$CONFIG" > $TMP_DIR/test_config.yaml

# Start Phoenix with test config for a few seconds
echo "Starting Phoenix with test config..."
./bin/sa-omf-otelcol --config=$TMP_DIR/test_config.yaml &
PID=$!
sleep 5
kill $PID

# Check if state file was created
if [ -f "$STATE_FILE" ]; then
  echo -e "${GREEN}State persistence verified - state file was created!${NC}"
else
  echo -e "${RED}State persistence verification failed - state file was not created!${NC}"
  exit 1
fi

# Clean up
rm -rf $TMP_DIR

# Step 5: Print verification summary
echo -e "\n${GREEN}============================================${NC}"
echo -e "${GREEN}New processors verification completed successfully!${NC}"
echo -e "${GREEN}============================================${NC}"
echo -e "${YELLOW}Verification summary:${NC}"
echo -e "✅ Build successful"
echo -e "✅ Unit tests passed"
echo -e "✅ Benchmarks completed"
echo -e "✅ State persistence verified"
echo -e "\n${YELLOW}Next steps:${NC}"
echo -e "1. Create integration tests with New Relic endpoint"
echo -e "2. Add dynamic attribute stripping based on cardinality"
echo -e "3. Deploy to testing environment"
echo -e "${GREEN}============================================${NC}"