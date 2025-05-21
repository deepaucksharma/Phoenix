#!/bin/bash
# filepath: /Users/deepaksharma/Desktop/src_main/Phoenix-vNext/phoenix-bench/monitor-pipelines.sh
#
# Phoenix-vNext 5-Pipeline Monitoring Script
# This script provides visibility into active series counts and mode transitions

set -e

# Check if required tools are available
command -v watch >/dev/null 2>&1 || { echo "Error: watch is required but not installed."; exit 1; }
command -v curl >/dev/null 2>&1 || { echo "Error: curl is required but not installed."; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "Error: jq is required but not installed. Install with: brew install jq"; exit 1; }

# Header
echo "Phoenix-vNext 5-Pipeline Monitor"
echo "==============================================="
echo "This script monitors key metrics across all 5 pipelines"
echo ""

# Function to query Prometheus
query_prometheus() {
    local query=$1
    local result=$(curl -s "http://localhost:9090/api/v1/query?query=${query}" | jq -r '.data.result[] | .value[1]')
    echo $result
}

# Function to display pipeline status
show_pipeline_status() {
    clear
    echo "Phoenix-vNext 5-Pipeline Status ($(date))"
    echo "==============================================="
    
    # Get current mode
    local mode=$(curl -s "http://localhost:9090/api/v1/query?query=otel_resource_attributes%7Botel_resource_observability_mode%21%3D%22%22%7D" | \
                jq -r '.data.result[] | .metric.otel_resource_observability_mode // "unknown"')
    
    # Get config version
    local version=$(curl -s "http://localhost:9090/api/v1/query?query=otel_resource_attributes%7Botel_resource_opt_version%21%3D%22%22%7D" | \
                  jq -r '.data.result[] | .metric.otel_resource_opt_version // "unknown"')
    
    # Get optimization level
    local opt_level=$(curl -s "http://localhost:9090/api/v1/query?query=otel_resource_attributes%7Botel_resource_optimization_level%21%3D%22%22%7D" | \
                    jq -r '.data.result[] | .metric.otel_resource_optimization_level // "unknown"')
    
    # Get active series
    local full_ts=$(query_prometheus "phoenix_ts_active" || echo "N/A")
    local opt_ts=$(query_prometheus "phoenix_opt_ts_active" || echo "N/A")
    local ultra_ts=$(query_prometheus "phoenix_ultra_ts_active" || echo "N/A")
    local hybrid_ts=$(query_prometheus "phoenix_hybrid_ts_active" || echo "N/A")
    local exp_ts=$(query_prometheus "phoenix_exp_ts_active" || echo "N/A")
    
    # Display system status
    echo "Current Mode: ${mode:-"unknown"}"
    echo "Config Version: ${version:-"unknown"}"
    echo "Optimization Level: ${opt_level:-"unknown"}%"
    echo ""
    
    echo "Active Time Series by Pipeline:"
    echo "-------------------------------"
    echo "Full Pipeline:        ${full_ts:-"N/A"}"
    echo "Opt Pipeline:         ${opt_ts:-"N/A"}"
    echo "Ultra Pipeline:       ${ultra_ts:-"N/A"}"
    echo "Hybrid Pipeline:      ${hybrid_ts:-"N/A"}"
    echo "Experimental Pipeline: ${exp_ts:-"N/A"}"
    echo ""
    
    # Show thresholds
    echo "Mode Transition Thresholds:"
    echo "--------------------------"
    echo "Moderate: 0-300 series"
    echo "Caution:  301-350 series"
    echo "Warning:  351-450 series"
    echo "Ultra:    >450 series"
    echo ""
    
    echo "Press Ctrl+C to exit"
}

# Main loop
watch -n 2 -t "$(declare -f show_pipeline_status); show_pipeline_status"
