#!/bin/bash
# filepath: /Users/deepaksharma/Desktop/src_main/Phoenix-vNext/phoenix-bench/override-control-mode.sh
#
# Phoenix-vNext Control Mode Override Script
# This script allows manual overriding of the control mode

set -e

# Define constants
CONTROL_FILE="./phoenix-bench/configs/control_signals/opt_mode.yaml"
VALID_MODES=("moderate" "ultra" "adaptive")
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
CORRELATION_ID="manual-override-$(date +%s)"

# Show usage
show_usage() {
    echo "Phoenix-vNext Control Mode Override"
    echo "Usage: $0 [mode]"
    echo ""
    echo "Available modes:"
    echo "  moderate - Default optimization level with basic filtering"
    echo "  ultra    - Aggressive optimization with heavy filtering"
    echo "  adaptive - Dynamic optimization based on workload"
    echo ""
    echo "Example: $0 ultra"
    exit 1
}

# Check arguments
if [ "$#" -ne 1 ] || [[ ! " ${VALID_MODES[*]} " =~ " $1 " ]]; then
    show_usage
fi

# The mode to set
NEW_MODE=$1

# Read the current file to get the current values
echo "Reading current control file..."
if [ ! -f "$CONTROL_FILE" ]; then
    echo "Error: Control file not found at $CONTROL_FILE"
    echo "Make sure you are running this script from the Phoenix-vNext root directory."
    exit 1
fi

# Get the current configuration version
CURRENT_VERSION=$(grep "config_version:" "$CONTROL_FILE" | cut -d':' -f2 | tr -d ' ')
if [ -z "$CURRENT_VERSION" ]; then
    CURRENT_VERSION=0
fi

# Increment the version
NEW_VERSION=$((CURRENT_VERSION + 1))

# Determine optimization level based on mode
case $NEW_MODE in
    moderate)
        OPT_LEVEL=0
        ;;
    adaptive)
        OPT_LEVEL=50
        ;;
    ultra)
        OPT_LEVEL=100
        ;;
    *)
        OPT_LEVEL=0
        ;;
esac

# Create a backup of the original file
cp "$CONTROL_FILE" "${CONTROL_FILE}.bak"

# Create the new control file
echo "Creating new control file with mode: $NEW_MODE (version: $NEW_VERSION)..."
cat > "$CONTROL_FILE" << EOF
# Optimization mode control file - MANUALLY OVERRIDDEN
# This file is read by otelcol-main and written by otelcol-observer
# Changes trigger pipeline configuration updates

# Mode configuration - determines active pipeline optimization level
mode: "$NEW_MODE"

# Metadata for tracking and coordination
last_updated: "$TIMESTAMP"
reason: "manual_override"
ts_count: 0
config_version: $NEW_VERSION
correlation_id: "$CORRELATION_ID"

# Fine-grained optimization level (0-100)
# 0-25: moderate, 26-75: adaptive, 76-100: ultra
optimization_level: $OPT_LEVEL

# Current thresholds (graduated levels for improved stability)
thresholds:
  moderate: 300.0
  caution: 350.0
  warning: 400.0
  ultra: 450.0

# State transition information for debugging and audit
state:
  previous_mode: "manual_override"
  transition_timestamp: "$TIMESTAMP"
  transition_duration_seconds: 0
  stability_period_seconds: 300  # Don't change mode again for 5 minutes
EOF

echo "Mode successfully changed to: $NEW_MODE"
echo "New configuration version: $NEW_VERSION"
echo "Control file updated. The main collector should detect this change within 10 seconds."
echo "Monitor the Grafana dashboard to observe the change taking effect."
