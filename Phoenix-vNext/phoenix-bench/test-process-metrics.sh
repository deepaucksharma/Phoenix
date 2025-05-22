#!/bin/bash

# Script to test and verify process metrics implementation
echo "===================================================="
echo "Phoenix-vNext Process Metrics Implementation Tester"
echo "===================================================="

# Function to check if a service is running
check_service() {
  local service=$1
  if docker-compose ps | grep -q "$service.*Up"; then
    echo "✅ Service $service is running"
    return 0
  else
    echo "❌ Service $service is not running"
    return 1
  fi
}

# Function to check if metrics are available in Prometheus
check_metrics() {
  local metric=$1
  local expected_result=$2
  
  echo -n "Checking for metric $metric... "
  result=$(curl -s "http://localhost:9090/api/v1/query?query=$metric" | grep -o '"resultType":"matrix"')
  
  if [ ! -z "$result" ]; then
    echo "✅ Found metric $metric"
    return 0
  else
    echo "❌ Metric $metric not found"
    return 1
  fi
}

# Check Docker services
echo -e "\n1. Checking Docker services:"
check_service "otelcol-main" || exit 1
check_service "otelcol-observer" || exit 1
check_service "prometheus" || exit 1
check_service "grafana" || exit 1

# Wait for metrics to be collected
echo -e "\nWaiting for metrics collection (30 seconds)..."
sleep 30

# Check process metrics
echo -e "\n2. Checking process metrics:"
check_metrics "phoenix_process_full_ts_active" "matrix" || exit 1
check_metrics "phoenix_process_opt_ts_active" "matrix" || exit 1
check_metrics "phoenix_process_ultra_ts_active" "matrix" || exit 1

# Check for process priority tags
echo -e "\n3. Checking process priority classification:"
check_metrics "process_memory_usage{phoenix_priority=\"critical\"}" "matrix"
check_metrics "process_memory_usage{phoenix_priority=\"high\"}" "matrix"
check_metrics "process_memory_usage{phoenix_priority=\"medium\"}" "matrix"
check_metrics "process_memory_usage{phoenix_priority=\"low\"}" "matrix"

# Check for rollup
echo -e "\n4. Checking rollup functionality:"
check_metrics "process_memory_usage{phoenix_rollup_target=\"phoenix.others.low\"}" "matrix"

# Check for Business Tier tagging
echo -e "\n5. Checking business tier tagging:"
check_metrics "process_memory_usage{phoenix_business_tier=\"data\"}" "matrix"
check_metrics "process_memory_usage{phoenix_business_tier=\"application\"}" "matrix"

# Calculate and display metrics reduction
echo -e "\n6. Calculating metrics reduction:"
full_count=$(curl -s "http://localhost:9090/api/v1/query?query=phoenix_process_full_ts_active" | grep -o '"value":\[[0-9.]*,"[0-9.]*"\]' | grep -o '[0-9.]*"' | tr -d '"')
opt_count=$(curl -s "http://localhost:9090/api/v1/query?query=phoenix_process_opt_ts_active" | grep -o '"value":\[[0-9.]*,"[0-9.]*"\]' | grep -o '[0-9.]*"' | tr -d '"')
ultra_count=$(curl -s "http://localhost:9090/api/v1/query?query=phoenix_process_ultra_ts_active" | grep -o '"value":\[[0-9.]*,"[0-9.]*"\]' | grep -o '[0-9.]*"' | tr -d '"')

if [ ! -z "$full_count" ] && [ ! -z "$opt_count" ] && [ ! -z "$ultra_count" ]; then
  opt_reduction=$(echo "scale=2; (($full_count - $opt_count) / $full_count) * 100" | bc)
  ultra_reduction=$(echo "scale=2; (($full_count - $ultra_count) / $full_count) * 100" | bc)
  
  echo "Full Process Metrics: $full_count time series"
  echo "Optimized Process Metrics: $opt_count time series ($opt_reduction% reduction)"
  echo "Ultra Process Metrics: $ultra_count time series ($ultra_reduction% reduction)"
else
  echo "❌ Could not calculate metrics reduction"
fi

echo -e "\n===================================================="
echo "Process Metrics Implementation Test Complete!"
echo "===================================================="
