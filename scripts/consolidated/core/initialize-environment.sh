#!/usr/bin/env bash
# Initializes the Phoenix-vNext Ultimate Stack environment
# Revision 2025-05-22 · v3.0-final-uX

echo "Initializing Phoenix-vNext Ultimate Stack environment..."
SCRIPT_DIR_INIT="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# Navigate up from scripts/consolidated/core/ to project root
PROJECT_ROOT="$(dirname "$(dirname "$(dirname "$SCRIPT_DIR_INIT")")")"
cd "$PROJECT_ROOT" || { echo "ERROR: Failed to change to project root directory '$PROJECT_ROOT'"; exit 1; }

# Create .env file from template if it doesn't exist
if [ ! -f ".env" ]; then
  echo "INFO: Creating .env file from .env.template."
  echo "IMPORTANT: Please edit .env with your New Relic API keys, OTLP endpoint, and review other defaults."
  cp .env.template .env
else
  echo "INFO: .env file already exists. Ensure it's up-to-date with .env.template structure."
fi

# Source .env to make variables available for opt_mode.yaml templating and other checks
set -a # Automatically export all variables
# shellcheck disable=SC1091
source .env.template # Source template first for ALL defaults
if [ -f ".env" ]; then
  # shellcheck disable=SC1091
  source .env # Override with user-set values from actual .env
fi
set +a

# Validate critical environment variables
validate_env_vars() {
  local missing_vars=()
  local placeholder_vars=()
  
  # Helper to check if a variable is set and not a placeholder
  check_var() {
    local var_name="$1"
    local var_value="${!var_name}"
    local placeholder="$2"
    
    if [[ -z "$var_value" ]]; then
      missing_vars+=("$var_name")
    elif [[ "$var_value" == *"$placeholder"* ]]; then
      placeholder_vars+=("$var_name")
    fi
  }
  
  # Check required variables when exports are enabled
  if [[ "$ENABLE_NR_EXPORT_FULL" == "true" ]]; then
    check_var "NEW_RELIC_LICENSE_KEY_FULL" "YOUR_NR_INGEST_LICENSE_KEY"
  fi
  
  if [[ "$ENABLE_NR_EXPORT_OPTIMISED" == "true" ]]; then
    check_var "NEW_RELIC_LICENSE_KEY_OPTIMISED" "YOUR_NR_INGEST_LICENSE_KEY"
  fi
  
  if [[ "$ENABLE_NR_EXPORT_EXPERIMENTAL" == "true" ]]; then
    check_var "NEW_RELIC_LICENSE_KEY_EXPERIMENTAL" "YOUR_NR_INGEST_LICENSE_KEY"
  fi
  
  # Report issues
  if [[ ${#missing_vars[@]} -gt 0 ]]; then
    echo "ERROR: The following required environment variables are not set:"
    printf "  - %s\n" "${missing_vars[@]}"
    return 1
  fi
  
  if [[ ${#placeholder_vars[@]} -gt 0 ]]; then
    echo "WARNING: The following environment variables still contain placeholder values:"
    printf "  - %s\n" "${placeholder_vars[@]}"
    echo "If you're exporting to New Relic, please update these values in your .env file."
  fi
  
  return 0
}

# Run validation if any exports are enabled
if [[ "$ENABLE_NR_EXPORT_FULL" == "true" || "$ENABLE_NR_EXPORT_OPTIMISED" == "true" || "$ENABLE_NR_EXPORT_EXPERIMENTAL" == "true" ]]; then
  echo "INFO: Validating New Relic export configuration..."
  validate_env_vars || {
    echo "ERROR: Environment validation failed. Please update your .env file."
    echo "       Continuing setup, but New Relic exports may not work properly."
  }
fi

echo "INFO: Creating required data directories..."
mkdir -p ./data/otelcol_main
mkdir -p ./data/prometheus
mkdir -p ./data/grafana
mkdir -p ./data/otelcol_observer # If observer were to use file storage

echo "INFO: Setting up control directory and initial optimization_mode.yaml..."
mkdir -p ./configs/control
CONTROL_FILE_PATH_INIT="./configs/control/optimization_mode.yaml"
TEMPLATE_FILE_PATH_INIT="./configs/templates/control/optimization_mode_template.yaml"

if [ ! -f "$TEMPLATE_FILE_PATH_INIT" ]; then
  echo "ERROR: Control signal template file $TEMPLATE_FILE_PATH_INIT not found!"
  echo "Please create it with the schema defined in the specification."
  # Create a minimal one if missing, so script doesn't fail hard
  cat <<EOF > "$TEMPLATE_FILE_PATH_INIT"
# Minimal Fallback Template - Please use the full one from spec
optimization_profile: conservative
last_updated: "1970-01-01T00:00:00Z"
trigger_reason: "fallback_template_init"
current_metrics:
  full_ts: 0
  optimized_ts: 0
  experimental_ts: 0
  cost_reduction_ratio: 0.0
config_version: 0
correlation_id: "fallback-init"
last_profile_change_timestamp: "1970-01-01T00:00:00Z" # Renamed from state.transition_timestamp
thresholds: # Renamed from active_thresholds for clarity in template
  conservative_max_ts: ${THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS}
  aggressive_min_ts: ${THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS}
pipelines: # Added as per spec
  full_fidelity_enabled: true
  optimized_enabled: true
  experimental_enabled: false
EOF
  echo "WARNING: Created a minimal fallback template. Please use the full one from the specification."
fi

if [ ! -f "$CONTROL_FILE_PATH_INIT" ]; then
  echo "INFO: Creating initial control file: $CONTROL_FILE_PATH_INIT from $TEMPLATE_FILE_PATH_INIT"
  
  # Use envsubst for simple substitution if yq is not preferred for init script
  # More robust: update-control-file.sh can initialize it on first run if template is just a source.
  # For init, a simple copy of template is often enough, letting actuator fill dynamic values.
  cp "$TEMPLATE_FILE_PATH_INIT" "$CONTROL_FILE_PATH_INIT"

  # Initial values that need to be dynamic at init time
  INIT_TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  INIT_VERSION=1 # Start with version 1
  INIT_CORRELATION_ID="${CORRELATION_ID_PREFIX:-pv3ux}-init-$(date +%s)-v${INIT_VERSION}"
  
  # Use yq to set dynamic initial values if available
  if command -v yq &> /dev/null; then
    yq eval -i ".last_updated = \"$INIT_TIMESTAMP\" | \
                 .config_version = $INIT_VERSION | \
                 .correlation_id = \"$INIT_CORRELATION_ID\" | \
                 .last_profile_change_timestamp = \"$INIT_TIMESTAMP\" | \
                 .trigger_reason = \"initial_environment_setup\" | \
                 .current_metrics.optimized_ts = 0 | \
                 .thresholds.conservative_max_ts = ${THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS} | \
                 .thresholds.aggressive_min_ts = ${THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS} \
                " "$CONTROL_FILE_PATH_INIT"
    echo "INFO: Initial control file populated using yq."
  else
    echo "WARNING: yq command not found. Initial control file might not be fully dynamic. Actuator script will populate it."
  fi
  chmod 644 "$CONTROL_FILE_PATH_INIT"
else
  echo "INFO: Control file $CONTROL_FILE_PATH_INIT already exists."
fi

echo "INFO: Setting up dashboards directory for Grafana..."
mkdir -p ./configs/monitoring/grafana/dashboards
GRAFANA_MAIN_DASHBOARD_PATH_INIT="./configs/monitoring/grafana/dashboards/phoenix-v3-ultra-overview.json"
if [ ! -f "$GRAFANA_MAIN_DASHBOARD_PATH_INIT" ]; then
    echo "INFO: Creating placeholder Grafana main overview dashboard: $GRAFANA_MAIN_DASHBOARD_PATH_INIT"
    # This should be the full dashboard JSON from the spec. For brevity, a stub:
    cat <<EOF > "$GRAFANA_MAIN_DASHBOARD_PATH_INIT"
{
  "title": "Phoenix v3 Ultra - Placeholder",
  "uid": "phoenix-v3-ultra-overview",
  "panels": [{"type": "text", "title": "Placeholder", "gridPos": { "x": 0, "y": 0, "w": 24, "h": 2 }, "options": {"content": "# Dashboard content to be populated from spec", "mode": "markdown"}}],
  "schemaVersion": 37, "version": 1
}
EOF
fi

echo "INFO: Creating Prometheus rules directory..."
mkdir -p ./configs/monitoring/prometheus/rules

# Create placeholder phoenix_rules.yml if it doesn't exist
PROM_RULES_FILE="./configs/monitoring/prometheus/rules/phoenix_rules.yml"
if [ ! -f "$PROM_RULES_FILE" ]; then
  echo "INFO: Creating placeholder Prometheus rules file: $PROM_RULES_FILE"
  cat <<EOF > "$PROM_RULES_FILE"
# Placeholder for Phoenix Prometheus Rules
# Add recording rules and alerts here as per specification.
# groups:
#   - name: phoenix_optimisation_kpis
#     rules:
#       - record: phoenix:cost_reduction_ratio
#         expr: 1 - (sum(phoenix_opt_final_output_phoenix_optimised_output_ts_active) / ignoring(pipeline_output_type) sum(phoenix_full_final_output_phoenix_full_output_ts_active))
#   - name: phoenix_alerts
#     rules:
#       - alert: PhoenixOptimizationDrift
#         expr: phoenix:cost_reduction_ratio < 0.4 for 10m
#         labels: {severity: warning}
#         annotations:
#           summary: Phoenix optimized pipeline no longer hitting 40% reduction
#           description: "Current cost reduction ratio is {{ \$value | printf \"%.2f\" }}. Check pipeline configurations and control loop."
EOF
fi


echo ""
echo "INFO: Environment initialization complete."
echo "IMPORTANT: Please ensure your .env file is correctly populated, especially:"
echo "  - NEW_RELIC_LICENSE_KEY_FULL, _OPTIMISED, _EXPERIMENTAL"
echo "  - NEW_RELIC_OTLP_ENDPOINT"
echo "  - Review THRESHOLD_* values for the control loop."
echo ""
echo "To generate CHECKSUMS.txt (as per spec section 9):"
echo "  (cd '$PROJECT_ROOT' && sha256sum configs/otel/collectors/*.yaml configs/control/*template.yaml > CHECKSUMS.txt)"
echo ""
echo "To start the stack: docker compose up -d"
echo "To monitor logs: docker compose logs -f [otelcol-main|otelcol-observer|control-loop-actuator|synthetic-metrics-generator]"
echo "Grafana: http://localhost:3000 (Default: ${GRAFANA_ADMIN_USER:-admin}/${GRAFANA_ADMIN_PASSWORD:-admin} or as per .env)"
echo "Prometheus: http://localhost:9090"