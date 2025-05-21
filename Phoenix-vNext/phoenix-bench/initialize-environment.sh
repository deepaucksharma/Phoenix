#!/bin/bash
# filepath: /Users/deepaksharma/Desktop/src_main/Phoenix-vNext/phoenix-bench/initialize-environment.sh
#
# Phoenix-vNext 5-Pipeline Initialization Script
# This script prepares the required directories and initial files for the 5-pipeline architecture

set -e  # Exit immediately if a command exits with non-zero status
echo "Initializing Phoenix-vNext 5-Pipeline Environment..."

# Directory setup
echo "Setting up directory structure..."
mkdir -p data/otelcol_main
mkdir -p data/prometheus
mkdir -p data/grafana

# Control signals directory
mkdir -p configs/control_signals

# Create the initial control signal file from template
echo "Creating initial control signal file..."
cat > configs/control_signals/opt_mode.yaml << EOF
# Optimization mode control file with enhanced structure
# This file is read by otelcol-main and written by otelcol-observer
# Changes trigger pipeline configuration updates

# Mode configuration - determines active pipeline optimization level
mode: "moderate"

# Metadata for tracking and coordination
last_updated: "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
reason: "initial_deployment"
ts_count: 0
config_version: 1
correlation_id: "init-phoenix-vnext-$(date +%s)"

# Fine-grained optimization level (0-100)
# 0-25: moderate, 26-75: adaptive, 76-100: ultra
optimization_level: 0

# Current thresholds (graduated levels for improved stability)
thresholds:
  moderate: 300.0
  caution: 350.0
  warning: 400.0
  ultra: 450.0

# State transition information for debugging and audit
state:
  previous_mode: "initial"
  transition_timestamp: "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  transition_duration_seconds: 0
  stability_period_seconds: 300  # Don't change mode again for 5 minutes
EOF

# Set permissions
chmod 644 configs/control_signals/opt_mode.yaml

# Verify that required files exist
echo "Verifying configuration files..."
required_files=(
    "configs/collectors/otelcol-main.yaml"
    "configs/collectors/otelcol-observer.yaml"
    "configs/monitoring/prometheus.yaml"
    "configs/monitoring/grafana-datasource.yaml"
    "configs/monitoring/grafana-dashboards.yaml"
    "configs/dashboards/phoenix-5pipeline-dashboard.json"
    "docker-compose.yaml"
)

for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        echo "ERROR: Required file not found: $file"
        exit 1
    fi
done

echo "Checking environment variables..."
echo "Required environment variables for New Relic export:"
echo "  - NR_FULL_KEY"
echo "  - NR_OPT_KEY"
echo "  - NR_ULTRA_KEY"
echo "  - NR_EXP_KEY"
echo "  - NR_HYBRID_KEY"
echo ""
echo "Optional environment variables:"
echo "  - BENCHMARK_ID (default: phoenix-vnext)"
echo "  - DEPLOYMENT_ENV (default: development)"
echo "  - CORRELATION_ID (default: startup-default)"
echo "  - THRESHOLD_MODERATE (default: 300.0)"
echo "  - THRESHOLD_CAUTION (default: 350.0)"
echo "  - THRESHOLD_WARNING (default: 400.0)"
echo "  - THRESHOLD_ULTRA (default: 450.0)"

echo ""
echo "Environment initialized successfully!"
echo "To start the stack, run: cd phoenix-bench && docker compose up -d"
echo "Access Grafana at http://localhost:3000 (admin/admin)"
echo "Access Prometheus at http://localhost:9090"
