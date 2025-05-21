#!/bin/bash
# Initializes the Phoenix-vNext environment

echo "Initializing Phoenix-vNext environment..."

# Create .env file from template if it doesn't exist
if [ ! -f ".env" ]; then
  echo "Creating .env file from .env.template. Please edit .env with your API keys."
  cp .env.template .env
else
  echo ".env file already exists."
fi

# Create required directories
mkdir -p ./data/otelcol_main
mkdir -p ./data/prometheus
mkdir -p ./data/grafana
mkdir -p ./configs/control_signals
mkdir -p ./configs/dashboards # For Grafana JSON dashboards

# Create initial control_signals/opt_mode.yaml file
CONTROL_FILE_PATH="./configs/control_signals/opt_mode.yaml"
if [ ! -f "$CONTROL_FILE_PATH" ]; then
  echo "Creating initial control file: $CONTROL_FILE_PATH"
  CURRENT_ISO_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  INITIAL_CORRELATION_ID="init-$(date +%s)"
  # Get thresholds from environment or use defaults from .env.template if .env not sourced yet
  source .env # Load .env if it exists to get THRESHOLD_* vars
  cat <<EOF > "$CONTROL_FILE_PATH"
# Optimization mode control file with enhanced structure
# This file is read by otelcol-main and written by otelcol-observer
# Changes trigger pipeline configuration updates

# Mode configuration - determines active pipeline optimization level
mode: "moderate"

# Metadata for tracking and coordination
last_updated: "${CURRENT_ISO_TIME}"
reason: "initial_deployment"
ts_count: 0
config_version: 1
correlation_id: "${INITIAL_CORRELATION_ID}"

# Fine-grained optimization level (0-100)
# 0-25: moderate, 26-75: adaptive, 76-100: ultra
optimization_level: 0

# Current thresholds (graduated levels for improved stability)
thresholds:
  moderate: ${THRESHOLD_MODERATE:-300.0}
  caution: ${THRESHOLD_CAUTION:-350.0}
  warning: ${THRESHOLD_WARNING:-400.0}
  ultra: ${THRESHOLD_ULTRA:-450.0}

# State transition information for debugging and audit
state:
  previous_mode: "initial"
  transition_timestamp: "${CURRENT_ISO_TIME}"
  transition_duration_seconds: 0
  stability_period_seconds: 300  # Don't change mode again for 5 minutes
EOF
else
  echo "Control file $CONTROL_FILE_PATH already exists."
fi

# Create an empty Grafana dashboard JSON if one isn't provided (dashboard provisioning needs a file)
GRAFANA_DASHBOARD_PATH="./configs/dashboards/phoenix-5pipeline-dashboard.json"
if [ ! -f "$GRAFANA_DASHBOARD_PATH" ]; then
    echo "Creating placeholder Grafana dashboard JSON: $GRAFANA_DASHBOARD_PATH"
    cat <<EOF > "$GRAFANA_DASHBOARD_PATH"
{
  "title": "Phoenix-vNext Placeholder Dashboard",
  "uid": "phoenix-vnext-placeholder",
  "panels": [
    {
      "type": "text",
      "title": "Placeholder Panel",
      "gridPos": { "x": 0, "y": 0, "w": 24, "h": 3 },
      "options": {
        "content": "This is a placeholder dashboard. The main dashboard should be provisioned by 'grafana-dashboards.yaml' and the JSON file it points to.",
        "mode": "markdown"
      }
    }
  ]
}
EOF
fi


echo "Environment initialization complete."
echo "Ensure your .env file is populated with New Relic API keys."
echo "You can now run 'docker compose up -d'."
