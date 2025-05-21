#!/bin/bash
# Phoenix-vNext synthetic metrics generator
# This script generates synthetic metrics directly to Prometheus through remote write
# to ensure all required metrics for the dashboard are available.

# Configuration 
PROMETHEUS_URL=${PROMETHEUS_URL:-http://localhost:9090}
METRICS_INTERVAL=${METRICS_INTERVAL:-10} # seconds

echo "Starting Phoenix-vNext synthetic metrics generator..."
echo "Sending metrics to Prometheus at $PROMETHEUS_URL every $METRICS_INTERVAL seconds"

# Timestamp helper
get_timestamp() {
  date +%s000 # Milliseconds since epoch
}

# Basic metrics initialization
ts_count_full=250
ts_count_opt_moderate=150
ts_count_opt_ultra=50
ts_count_exp=100
ts_count_hybrid=125

quality_full=100
quality_opt_moderate=95
quality_opt_ultra=85
quality_exp=90
quality_hybrid=93

cost_red_moderate=40
cost_red_ultra=80
cost_red_exp=60
cost_red_hybrid=50

opt_mode=0 # 0=moderate, 1=ultra
threshold_moderate=300
threshold_ultra=450

# Run continuously
while true; do
  timestamp=$(get_timestamp)

  # Generate Phoenix metrics payload
  cat <<EOF > /tmp/metrics_payload.txt
# TYPE phoenix_pipeline_full gauge
phoenix_pipeline_full $ts_count_full $timestamp

# TYPE phoenix_pipeline_opt_moderate gauge
phoenix_pipeline_opt_moderate $ts_count_opt_moderate $timestamp

# TYPE phoenix_pipeline_opt_ultra gauge
phoenix_pipeline_opt_ultra $ts_count_opt_ultra $timestamp

# TYPE phoenix_pipeline_exp gauge
phoenix_pipeline_exp $ts_count_exp $timestamp

# TYPE phoenix_pipeline_hybrid gauge
phoenix_pipeline_hybrid $ts_count_hybrid $timestamp

# TYPE phoenix_system_cpu_time_seconds_total gauge
phoenix_system_cpu_time_seconds_total{opt_mode="$opt_mode",threshold_moderate="$threshold_moderate",threshold_ultra="$threshold_ultra",ts_count_full="$ts_count_full",ts_count_opt_moderate="$ts_count_opt_moderate",ts_count_opt_ultra="$ts_count_opt_ultra",ts_count_exp="$ts_count_exp",ts_count_hybrid="$ts_count_hybrid",quality_full="$quality_full",quality_opt_moderate="$quality_opt_moderate",quality_opt_ultra="$quality_opt_ultra",quality_exp="$quality_exp",quality_hybrid="$quality_hybrid",cost_reduction_opt_moderate="$cost_red_moderate",cost_reduction_opt_ultra="$cost_red_ultra",cost_reduction_exp="$cost_red_exp",cost_reduction_hybrid="$cost_red_hybrid"} $ts_count_full $timestamp

# TYPE phoenix_opt_mode gauge
phoenix_opt_mode $opt_mode $timestamp
EOF

  # Send metrics to Prometheus
  curl -s -X POST "$PROMETHEUS_URL/api/v1/write" --data-binary "@/tmp/metrics_payload.txt"
  
  echo "Sent metrics at $(date)"
  
  # Random fluctuation in metrics for demo purposes
  if (( RANDOM % 10 > 7 )); then
    # Randomly adjust values slightly to create some variation
    ts_count_full=$((ts_count_full + (RANDOM % 50) - 25))
    
    # Enforce bounds
    [[ $ts_count_full -lt 200 ]] && ts_count_full=200
    [[ $ts_count_full -gt 500 ]] && ts_count_full=500
    
    # Update derived metrics
    ts_count_opt_moderate=$((ts_count_full * 60 / 100))
    ts_count_opt_ultra=$((ts_count_full * 20 / 100))
    ts_count_exp=$((ts_count_full * 40 / 100))
    ts_count_hybrid=$((ts_count_full * 50 / 100))
    
    # Update optimization mode based on threshold
    if [[ $ts_count_full -gt $threshold_ultra ]]; then
      opt_mode=1 # Ultra mode
    elif [[ $ts_count_full -lt $threshold_moderate ]]; then
      opt_mode=0 # Moderate mode
    fi
    
    echo "Updated metrics - ts_count: $ts_count_full, opt_mode: $opt_mode"
  fi
  
  # Sleep until next interval
  sleep $METRICS_INTERVAL
done
