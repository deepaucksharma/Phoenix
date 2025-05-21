#!/bin/bash
# filepath: /Users/deepaksharma/Desktop/src_main/Phoenix-vNext/phoenix-bench/update-control-file.sh
# Dynamic control file updater for Phoenix-vNext
# This script pulls metrics from Prometheus and updates opt_mode.yaml
# with appropriate optimization settings

set -e

# Configuration
CONTROL_FILE="/Users/deepaksharma/Desktop/src_main/Phoenix-vNext/phoenix-bench/configs/control_signals/opt_mode.yaml"
PROMETHEUS_URL="http://localhost:9090"
QUERY_ACTIVE_SERIES="phoenix_opt_ts_active"

# Load thresholds from .env file if it exists
if [ -f "/Users/deepaksharma/Desktop/src_main/Phoenix-vNext/phoenix-bench/.env" ]; then
  source "/Users/deepaksharma/Desktop/src_main/Phoenix-vNext/phoenix-bench/.env"
fi

# Default thresholds if not set from .env
THRESHOLD_MODERATE=${THRESHOLD_MODERATE:-300.0}
THRESHOLD_CAUTION=${THRESHOLD_CAUTION:-350.0}
THRESHOLD_WARNING=${THRESHOLD_WARNING:-400.0}
THRESHOLD_ULTRA=${THRESHOLD_ULTRA:-450.0}

# Generate a correlation ID based on timestamp
CORRELATION_ID="phoenix-control-$(date +%s)"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Query Prometheus for current active series count
echo "Querying Prometheus at $PROMETHEUS_URL for $QUERY_ACTIVE_SERIES..."
TS_COUNT=$(curl -s -G "$PROMETHEUS_URL/api/v1/query" --data-urlencode "query=$QUERY_ACTIVE_SERIES" | jq -r '.data.result[0].value[1]')

# Default to 0 if query fails
if [[ -z "$TS_COUNT" || "$TS_COUNT" == "null" ]]; then
    echo "Failed to get current value, defaulting to 0"
    TS_COUNT=0
fi

# Read the current config version and mode from the existing file
if [ -f "$CONTROL_FILE" ]; then
    CURRENT_VERSION=$(grep "config_version:" "$CONTROL_FILE" | awk '{print $2}')
    CURRENT_MODE=$(grep "mode:" "$CONTROL_FILE" | awk '{print $2}' | tr -d '"')
    CURRENT_LEVEL=$(grep "optimization_level:" "$CONTROL_FILE" | awk '{print $2}')
else
    CURRENT_VERSION=0
    CURRENT_MODE="moderate"
    CURRENT_LEVEL=0
fi

# Increment the config version
NEW_VERSION=$((CURRENT_VERSION + 1))

# Determine the new mode and optimization level based on the metrics
if (( $(echo "$TS_COUNT > $THRESHOLD_ULTRA" | bc -l) )); then
    NEW_MODE="ultra"
    NEW_LEVEL=90  # High in the ultra range
    REASON="high_cardinality_emergency"
    
elif (( $(echo "$TS_COUNT > $THRESHOLD_WARNING" | bc -l) )); then
    NEW_MODE="adaptive"
    NEW_LEVEL=50  # Middle of adaptive range
    REASON="approaching_threshold_warning"
    
elif (( $(echo "$TS_COUNT > $THRESHOLD_CAUTION" | bc -l) )); then
    NEW_MODE="adaptive"
    NEW_LEVEL=35  # Low in the adaptive range
    REASON="approaching_threshold_caution"
    
else
    NEW_MODE="moderate"
    NEW_LEVEL=10  # Low in the moderate range
    REASON="below_threshold_normal"
fi

# Only create a new file if the mode or level has changed
if [[ "$NEW_MODE" != "$CURRENT_MODE" || "$NEW_LEVEL" != "$CURRENT_LEVEL" ]]; then
    echo "Updating control file: Mode=$NEW_MODE, Level=$NEW_LEVEL"
    
    # Create the new control file
    cat > "$CONTROL_FILE" << EOL
# filepath: $CONTROL_FILE
# Optimization mode control file with enhanced structure
# This file is read by otelcol-main and written by otelcol-observer
# Changes trigger pipeline configuration updates

# Mode configuration - determines active pipeline optimization level
mode: "$NEW_MODE"

# Metadata for tracking and coordination
last_updated: "$TIMESTAMP"
reason: "$REASON"
ts_count: $TS_COUNT
config_version: $NEW_VERSION
correlation_id: "$CORRELATION_ID"

# Fine-grained optimization level (0-100)
# 0-25: moderate, 26-75: adaptive, 76-100: ultra
optimization_level: $NEW_LEVEL

# Current thresholds (graduated levels for improved stability)
thresholds:
  moderate: $THRESHOLD_MODERATE
  caution: $THRESHOLD_CAUTION
  warning: $THRESHOLD_WARNING
  ultra: $THRESHOLD_ULTRA

# State transition information for debugging and audit
state:
  previous_mode: "$CURRENT_MODE"
  transition_timestamp: "$TIMESTAMP"
  transition_duration_seconds: 0
  stability_period_seconds: 300  # Don't change mode again for 5 minutes
EOL
    
    echo "Control file updated successfully"
else
    echo "No changes required: Mode=$CURRENT_MODE, Level=$CURRENT_LEVEL"
fi
