#!/usr/bin/env bash
# Phoenix v3 Ultimate Process-Metrics Stack - Adaptive Controller Script
# Revision 2025-05-22 · v3.0-final-uX
# Queries Prometheus for KPIs, implements PID-lite logic (profile selection),
# and updates the optimization_mode.yaml control file.

set -euo pipefail # Strict mode

# --- Configuration (Defaults, can be overridden by environment variables) ---
PROM_API_ENDPOINT="${PROMETHEUS_URL:-http://prometheus:9090}/api/v1"
CONTROL_FILE_PATH="${CONTROL_SIGNAL_FILE:-/app/control_signals/optimization_mode.yaml}"
TEMPLATE_FILE_PATH="${OPT_MODE_TEMPLATE_PATH:-/app/optimization_mode_template.yaml}"

# PID-lite "Set Point" for the 'optimised' pipeline's active time series count
# This is informational for now, as the script uses fixed thresholds for profile switching.
TARGET_OPTIMISED_TS_COUNT_SETPOINT="${TARGET_OPTIMIZED_PIPELINE_TS_COUNT:-20000}"

# Thresholds for discrete profile switching based on 'optimised' pipeline's TS count
# These are critical for the decision logic.
CONSERVATIVE_MAX_TS_THRESHOLD="${THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS:-15000}"
AGGRESSIVE_MIN_TS_THRESHOLD="${THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS:-25000}"
# "balanced" profile is used for TS counts between these two.

# Conceptual PID-lite Gains (not directly used for threshold adjustment in this script version)
# KP="${PID_KP:-0.20}" # Proportional gain
# KI="${PID_KI:-0.05}" # Integral gain
# INTEGRAL_STATE_FILE="/tmp/phoenix_pid_integral.state" # File to store integral term

STABILITY_PERIOD_SECONDS="${ADAPTIVE_CONTROLLER_STABILITY_SECONDS:-120}"
CORRELATION_ID_PREFIX="${CORRELATION_ID_PREFIX:-pv3ux}"

# Metric names as exposed by otelcol-observer's Prometheus endpoint (after relabeling)
# These query the 'phoenix_observer_kpi_store' namespace and 'phoenix_pipeline_output_cardinality_estimate' metric name
# with a 'phoenix_pipeline_label' to distinguish them.
METRIC_FULL_TS_QUERY='phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate{phoenix_pipeline_label="full_fidelity",job="otelcol-observer-metrics"}'
METRIC_OPTIMISED_TS_QUERY='phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate{phoenix_pipeline_label="optimised",job="otelcol-observer-metrics"}'
METRIC_EXPERIMENTAL_TS_QUERY='phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate{phoenix_pipeline_label="experimental",job="otelcol-observer-metrics"}'

# --- Logging ---
log_ts() { date -u +"%Y-%m-%dT%H:%M:%SZ"; }
log_info() { echo "[$(log_ts)] [CTL] INFO: $*"; }
log_warn() { echo "[$(log_ts)] [CTL] WARN: $*"; }
log_error() { echo "[$(log_ts)] [CTL] ERROR: $*"; }

# --- Helper to query Prometheus ---
query_prometheus_value() {
  local query_string="$1"
  local default_value="${2:-0}"
  local response_file="/tmp/prom_response_$$.json"
  local value
  local http_status

  # log_info "Querying Prometheus: $query_string"
  http_status=$(curl -s -m 10 -G "${PROM_API_ENDPOINT}/query" \
    --data-urlencode "query=${query_string}" \
    -o "$response_file" \
    -w "%{http_code}")

  if [[ "$http_status" -ne 200 ]]; then
    log_warn "Prometheus query failed for '$query_string'. HTTP Status: $http_status. Assuming default: $default_value. Response: $(cat $response_file || echo 'empty')"
    value="$default_value"
  else
    value=$(jq -r '.data.result[0].value[1] // "null"' "$response_file")
    if [[ "$value" == "null" || -z "$value" ]]; then
      # log_warn "Metric for '$query_string' not found or no data. Assuming default: $default_value."
      value="$default_value"
    fi
  fi
  rm -f "$response_file"
  # Ensure it's a number for bc, stripping potential scientific notation if jq doesn't handle it well
  sanitized_value=$(echo "$value" | awk '{printf "%.0f", $1}')
  [[ "$sanitized_value" =~ ^[0-9]+$ ]] && echo "$sanitized_value" || echo "$default_value"
}

# --- Main Controller Logic ---
log_info "Starting adaptive controller cycle (PID-Lite Profile Selector)."
log_info "Control File: $CONTROL_FILE, Template: $TEMPLATE_FILE"
log_info "Optimised TS Thresholds -> Conservative Max: $CONSERVATIVE_MAX_TS_THRESHOLD, Aggressive Min: $AGGRESSIVE_MIN_TS_THRESHOLD"
log_info "Stability Period: $STABILITY_PERIOD_SECONDS seconds"

# 1. Read current/previous state from control file
PREV_OPTIMIZATION_PROFILE="conservative"
PREV_CONFIG_VERSION=0
PREV_LAST_PROFILE_CHANGE_UNIX=$(($(date +%s) - STABILITY_PERIOD_SECONDS - 1))
PREV_FULL_TS_FROM_FILE=0
PREV_OPTIMISED_TS_FROM_FILE=0
PREV_EXPERIMENTAL_TS_FROM_FILE=0
PREV_COST_REDUCTION_FROM_FILE="0.0"
PREV_LAST_PROFILE_CHANGE_ISO_FROM_FILE="1970-01-01T00:00:00Z"
PREV_PIPELINE_EXPERIMENTAL_ENABLED_FROM_FILE="false"

if [ -f "$CONTROL_FILE" ]; then
  if yq eval '.optimization_profile' "$CONTROL_FILE" &> /dev/null; then
    PREV_OPTIMIZATION_PROFILE=$(yq eval '.optimization_profile' "$CONTROL_FILE" | tr -d '"')
    PREV_CONFIG_VERSION=$(yq eval '.config_version' "$CONTROL_FILE" | tr -d '"')
    PREV_LAST_PROFILE_CHANGE_ISO_FROM_FILE=$(yq eval '.last_profile_change_timestamp' "$CONTROL_FILE" | tr -d '"')
    PREV_FULL_TS_FROM_FILE=$(yq eval '.current_metrics.full_ts' "$CONTROL_FILE" 2>/dev/null || echo 0)
    PREV_OPTIMISED_TS_FROM_FILE=$(yq eval '.current_metrics.optimized_ts' "$CONTROL_FILE" 2>/dev/null || echo 0)
    PREV_EXPERIMENTAL_TS_FROM_FILE=$(yq eval '.current_metrics.experimental_ts' "$CONTROL_FILE" 2>/dev/null || echo 0)
    PREV_COST_REDUCTION_FROM_FILE=$(yq eval '.current_metrics.cost_reduction_ratio' "$CONTROL_FILE" 2>/dev/null || echo "0.0")
    PREV_PIPELINE_EXPERIMENTAL_ENABLED_FROM_FILE=$(yq eval '.pipelines.experimental_enabled' "$CONTROL_FILE" 2>/dev/null || echo "false")

    if [[ -n "$PREV_LAST_PROFILE_CHANGE_ISO_FROM_FILE" && "$PREV_LAST_PROFILE_CHANGE_ISO_FROM_FILE" != "null" && "$PREV_LAST_PROFILE_CHANGE_ISO_FROM_FILE" != "1970-01-01T00:00:00Z" ]]; then
      PREV_LAST_PROFILE_CHANGE_UNIX=$(date -d "$PREV_LAST_PROFILE_CHANGE_ISO_FROM_FILE" +%s)
    fi
    log_info "Read previous state - Profile: $PREV_OPTIMIZATION_PROFILE, Version: $PREV_CONFIG_VERSION, Last Profile Change UNIX: $PREV_LAST_PROFILE_CHANGE_UNIX"
  else
    log_warn "Control file $CONTROL_FILE found but malformed. Using default previous state."
  fi
else
  log_info "Control file $CONTROL_FILE not found. Initializing with default previous state."
fi

# 2. Fetch current KPIs from Prometheus (via otelcol-observer)
CURRENT_FULL_TS=$(query_prometheus_value "$METRIC_FULL_TS_QUERY" "$PREV_FULL_TS_FROM_FILE")
CURRENT_OPTIMISED_TS=$(query_prometheus_value "$METRIC_OPTIMISED_TS_QUERY" "$PREV_OPTIMISED_TS_FROM_FILE")
CURRENT_EXPERIMENTAL_TS=$(query_prometheus_value "$METRIC_EXPERIMENTAL_TS_QUERY" "$PREV_EXPERIMENTAL_TS_FROM_FILE")
log_info "Current KPIs - Full_TS: $CURRENT_FULL_TS, Optimised_TS: $CURRENT_OPTIMISED_TS, Experimental_TS: $CURRENT_EXPERIMENTAL_TS"

# 3. Calculate Cost Reduction Ratio (Optimised vs Full)
CURRENT_COST_REDUCTION_RATIO="0.0"
# Ensure CURRENT_FULL_TS is numeric and greater than 0 for division
if [[ "$CURRENT_FULL_TS" =~ ^[0-9]+(\.[0-9]+)?$ && $(echo "$CURRENT_FULL_TS > 0" | bc -l) -eq 1 ]]; then
  CURRENT_COST_REDUCTION_RATIO=$(echo "scale=3; (1 - ($CURRENT_OPTIMISED_TS / $CURRENT_FULL_TS))" | bc)
  # Ensure it's between 0 and 1, can be negative if opt > full
  if (( $(echo "$CURRENT_COST_REDUCTION_RATIO < 0" | bc -l) )); then CURRENT_COST_REDUCTION_RATIO="0.0"; fi
  if (( $(echo "$CURRENT_COST_REDUCTION_RATIO > 1" | bc -l) )); then CURRENT_COST_REDUCTION_RATIO="1.0"; fi
fi
log_info "Current Cost Reduction Ratio (Opt vs Full): $CURRENT_COST_REDUCTION_RATIO"

# 4. PID-lite logic: Determine desired optimisation_profile based on CURRENT_OPTIMISED_TS
PROPOSED_PROFILE=""
TRIGGER_REASON_TEXT=""

# Ensure thresholds are treated as numbers by bc
CONSERVATIVE_MAX_TS_NUM=$(echo "$CONSERVATIVE_MAX_TS_THRESHOLD" | bc)
AGGRESSIVE_MIN_TS_NUM=$(echo "$AGGRESSIVE_MIN_TS_THRESHOLD" | bc)
CURRENT_OPTIMISED_TS_NUM=$(echo "$CURRENT_OPTIMISED_TS" | bc)

if (( $(echo "$CURRENT_OPTIMISED_TS_NUM > $AGGRESSIVE_MIN_TS_NUM" | bc -l) )); then
  PROPOSED_PROFILE="aggressive"
  TRIGGER_REASON_TEXT="Optimised TS ($CURRENT_OPTIMISED_TS_NUM) > Aggressive Min TS ($AGGRESSIVE_MIN_TS_NUM)"
elif (( $(echo "$CURRENT_OPTIMISED_TS_NUM < $CONSERVATIVE_MAX_TS_NUM" | bc -l) )); then
  PROPOSED_PROFILE="conservative"
  TRIGGER_REASON_TEXT="Optimised TS ($CURRENT_OPTIMISED_TS_NUM) < Conservative Max TS ($CONSERVATIVE_MAX_TS_NUM)"
else
  PROPOSED_PROFILE="balanced"
  TRIGGER_REASON_TEXT="Optimised TS ($CURRENT_OPTIMISED_TS_NUM) in balanced range [$CONSERVATIVE_MAX_TS_NUM - $AGGRESSIVE_MIN_TS_NUM]"
fi
log_info "Proposed Profile based on Optimised TS: $PROPOSED_PROFILE. Reason: $TRIGGER_REASON_TEXT"

# 5. Apply Hysteresis (Stability Control for profile changes)
EFFECTIVE_PROFILE="$PROPOSED_PROFILE"
TIMESTAMP_OF_LAST_ACTUAL_PROFILE_CHANGE_FOR_FILE="$PREV_LAST_PROFILE_CHANGE_ISO_FROM_FILE"

if [[ "$PROPOSED_PROFILE" != "$PREV_OPTIMIZATION_PROFILE" ]]; then
  NOW_UNIX=$(date +%s)
  TIME_SINCE_LAST_CHANGE=$((NOW_UNIX - PREV_LAST_PROFILE_CHANGE_UNIX))
  if (( TIME_SINCE_LAST_CHANGE < STABILITY_PERIOD_SECONDS )); then
    EFFECTIVE_PROFILE="$PREV_OPTIMIZATION_PROFILE"
    TRIGGER_REASON_TEXT="Stability hold ($TIME_SINCE_LAST_CHANGE s < $STABILITY_PERIOD_SECONDS s). Maintained '$PREV_OPTIMIZATION_PROFILE'. Original intent: '$PROPOSED_PROFILE' ($TRIGGER_REASON_TEXT)"
    log_info "$TRIGGER_REASON_TEXT"
  else
    TIMESTAMP_OF_LAST_ACTUAL_PROFILE_CHANGE_FOR_FILE=$(log_ts) # Profile is changing, update this timestamp
    log_info "Profile changing from '$PREV_OPTIMIZATION_PROFILE' to '$EFFECTIVE_PROFILE'."
  fi
else
  log_info "Profile ('$PREV_OPTIMIZATION_PROFILE') remains unchanged based on TS counts."
fi

# 6. Determine pipeline enablement based on effective profile (as per spec)
EXPERIMENTAL_PIPELINE_ENABLED="false"
if [[ "$EFFECTIVE_PROFILE" == "aggressive" ]]; then
  EXPERIMENTAL_PIPELINE_ENABLED="true"
fi
# Full and Optimised are generally always "enabled" in terms of their config existing,
# but their data export to New Relic is controlled by ENABLE_NR_EXPORT_* vars
# and the profile tag is used by main collector to filter/route internally.

# 7. Write to control file
if ! [[ "$PREV_CONFIG_VERSION" =~ ^[0-9]+$ ]]; then PREV_CONFIG_VERSION=0; fi
NEW_VERSION=$((PREV_CONFIG_VERSION + 1))
NEW_CORRELATION_ID="${CORRELATION_ID_PREFIX_VAL}-$(date +%s)-v${NEW_VERSION}"
WRITE_TIMESTAMP_ISO=$(log_ts)

log_info "Writing to control file: $CONTROL_FILE"
log_info "  Effective Profile: $EFFECTIVE_PROFILE, Version: $NEW_VERSION, Correlation: $NEW_CORRELATION_ID"
log_info "  Experimental Pipeline to be Enabled by Collector: $EXPERIMENTAL_PIPELINE_ENABLED"

CONTROL_FILE_TEMP="${CONTROL_FILE}.tmp_actuator_$$"
if [ ! -f "$TEMPLATE_FILE" ]; then
  log_error "Template file $TEMPLATE_FILE not found. Cannot write control file."
  exit 1
fi

yq eval ".optimization_profile = \"$EFFECTIVE_PROFILE\" | \
         .config_version = $NEW_VERSION | \
         .correlation_id = \"$NEW_CORRELATION_ID\" | \
         .last_updated = \"$WRITE_TIMESTAMP_ISO\" | \
         .trigger_reason = \"$TRIGGER_REASON_TEXT\" | \
         .current_metrics.full_ts = $(echo "$CURRENT_FULL_TS" | bc) | \
         .current_metrics.optimized_ts = $(echo "$CURRENT_OPTIMISED_TS" | bc) | \
         .current_metrics.experimental_ts = $(echo "$CURRENT_EXPERIMENTAL_TS" | bc) | \
         .current_metrics.cost_reduction_ratio = $(echo "$CURRENT_COST_REDUCTION_RATIO" | bc) | \
         .thresholds.conservative_max_ts = $(echo "$CONSERVATIVE_MAX_TS_THRESHOLD" | bc) | \
         .thresholds.aggressive_min_ts = $(echo "$AGGRESSIVE_MIN_TS_THRESHOLD" | bc) | \
         .pipelines.experimental_enabled = $EXPERIMENTAL_PIPELINE_ENABLED | \
         .last_profile_change_timestamp = \"$TIMESTAMP_OF_LAST_ACTUAL_PROFILE_CHANGE_FOR_FILE\" \
        " "$TEMPLATE_FILE" > "$CONTROL_FILE_TEMP"

if [ $? -eq 0 ]; then
  mv "$CONTROL_FILE_TEMP" "$CONTROL_FILE"
  chmod 644 "$CONTROL_FILE"
  log_info "Control file successfully updated."
else
  log_error "Failed to update control file using yq. Temporary file not moved. Error: $?"
  rm -f "$CONTROL_FILE_TEMP"
fi

log_info "Adaptive controller cycle finished."