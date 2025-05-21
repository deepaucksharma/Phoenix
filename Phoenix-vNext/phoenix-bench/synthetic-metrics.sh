#!/bin/bash

# Generate synthetic metrics using curl to push to Prometheus
# This script simulates the five pipelines with their metrics

while true; do
  # Pipeline values
  curl -X POST http://localhost:9999/metrics \
    -H "Content-Type: text/plain" \
    --data-binary '
      # Pipeline metrics
      phoenix_pipeline_full 3.0
      phoenix_pipeline_opt_moderate 1.0
      phoenix_pipeline_opt_ultra 0.5
      phoenix_pipeline_exp 1.35
      phoenix_pipeline_hybrid 0.8
      
      # Timeseries counts
      phoenix_ts_count_full 410.0
      phoenix_ts_count_opt_moderate 180.0
      phoenix_ts_count_opt_ultra 75.0
      phoenix_ts_count_exp 205.0
      phoenix_ts_count_hybrid 163.0
      
      # Signal quality metrics
      phoenix_quality_full 100.0
      phoenix_quality_opt_moderate 92.0
      phoenix_quality_opt_ultra 85.0
      phoenix_quality_exp 88.0
      phoenix_quality_hybrid 90.0
      
      # Cost reduction percentages
      phoenix_cost_reduction_opt_moderate 56.0
      phoenix_cost_reduction_opt_ultra 82.0
      phoenix_cost_reduction_exp 50.0
      phoenix_cost_reduction_hybrid 60.0
      
      # Thresholds
      phoenix_threshold_moderate 300.0
      phoenix_threshold_ultra 450.0
      
      # Current optimization mode (0=moderate, 1=ultra)
      phoenix_opt_mode 0.0
    '
    
  # Sleep for 5 seconds
  sleep 5
done