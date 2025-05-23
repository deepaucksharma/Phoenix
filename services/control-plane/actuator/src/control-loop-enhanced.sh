#!/usr/bin/env bash
# Phoenix Enhanced Control Loop - With Advanced Stability Features
# Based on original but with additional safety mechanisms

set -euo pipefail

# --- Enhanced Configuration ---
PROM_API_ENDPOINT="${PROMETHEUS_URL:-http://prometheus:9090}/api/v1"
CONTROL_FILE_PATH="${CONTROL_SIGNAL_FILE:-/app/control_signals/optimization_mode.yaml}"
TEMPLATE_FILE_PATH="${OPT_MODE_TEMPLATE_PATH:-/app/optimization_mode_template.yaml}"
LOCK_FILE="/tmp/phoenix_control_lock"
LOCK_TIMEOUT=30
STATE_FILE="/tmp/phoenix_control_state.json"

# Metric queries (defaults can be overridden via environment)
METRIC_OPTIMISED_TS_QUERY="${METRIC_OPTIMISED_TS_QUERY:-phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate{phoenix_pipeline_label=\"optimised\",job=\"otelcol-observer-metrics\"}}"
METRIC_CARDINALITY_EXPLOSION_ALERT="${METRIC_CARDINALITY_EXPLOSION_ALERT:-phoenix_observer_kpi_store_phoenix_cardinality_explosion_alert_count{job=\"otelcol-observer-metrics\"}}"

# Profile thresholds
CONSERVATIVE_MAX_TS_THRESHOLD="${THRESHOLD_OPTIMIZATION_CONSERVATIVE_MAX_TS:-15000}"
AGGRESSIVE_MIN_TS_THRESHOLD="${THRESHOLD_OPTIMIZATION_AGGRESSIVE_MIN_TS:-25000}"

# Enhanced hysteresis and stability
HYSTERESIS_FACTOR="${HYSTERESIS_FACTOR:-0.1}"
STABILITY_PERIOD_SECONDS="${ADAPTIVE_CONTROLLER_STABILITY_SECONDS:-120}"
OSCILLATION_DETECTION_WINDOW="${OSCILLATION_DETECTION_WINDOW:-600}" # 10 minutes
MAX_CHANGES_PER_WINDOW="${MAX_CHANGES_PER_WINDOW:-3}"
EMERGENCY_LOCKOUT_DURATION="${EMERGENCY_LOCKOUT_DURATION:-300}" # 5 minutes

# Cardinality explosion thresholds
EXPLOSION_RATE_THRESHOLD="${EXPLOSION_RATE_THRESHOLD:-10000}" # series/sec
EXPLOSION_ABSOLUTE_THRESHOLD="${EXPLOSION_ABSOLUTE_THRESHOLD:-1000000}" # 1M series
RISK_SCORE_THRESHOLD="${RISK_SCORE_THRESHOLD:-0.7}"

# --- Logging ---
log_ts() { date -u +"%Y-%m-%dT%H:%M:%SZ"; }
log_info() { echo "[$(log_ts)] [CTL] INFO: $*"; }
log_warn() { echo "[$(log_ts)] [CTL] WARN: $*"; }
log_error() { echo "[$(log_ts)] [CTL] ERROR: $*"; }
log_debug() { echo "[$(log_ts)] [CTL] DEBUG: $*"; }

# --- State Management ---
load_state() {
    if [ -f "$STATE_FILE" ]; then
        cat "$STATE_FILE"
    else
        echo '{
            "change_history": [],
            "oscillation_count": 0,
            "last_emergency_timestamp": 0,
            "consecutive_errors": 0
        }'
    fi
}

save_state() {
    local state="$1"
    echo "$state" > "$STATE_FILE"
}

update_change_history() {
    local state="$1"
    local new_profile="$2"
    local timestamp=$(date +%s)
    
    # Add new change to history
    local history=$(echo "$state" | jq ".change_history += [{\"timestamp\": $timestamp, \"profile\": \"$new_profile\"}]")
    
    # Remove changes older than window
    local cutoff=$((timestamp - OSCILLATION_DETECTION_WINDOW))
    history=$(echo "$history" | jq ".change_history |= map(select(.timestamp > $cutoff))")
    
    # Count oscillations (profile changes)
    local change_count=$(echo "$history" | jq '.change_history | length')
    history=$(echo "$history" | jq ".oscillation_count = $change_count")
    
    echo "$history"
}

check_oscillation() {
    local state="$1"
    local oscillation_count=$(echo "$state" | jq -r '.oscillation_count')
    
    if [ "$oscillation_count" -gt "$MAX_CHANGES_PER_WINDOW" ]; then
        log_warn "Oscillation detected: $oscillation_count changes in $OSCILLATION_DETECTION_WINDOW seconds"
        return 0
    fi
    return 1
}

check_emergency_lockout() {
    local state="$1"
    local last_emergency=$(echo "$state" | jq -r '.last_emergency_timestamp')
    local now=$(date +%s)
    
    if [ $((now - last_emergency)) -lt "$EMERGENCY_LOCKOUT_DURATION" ]; then
        log_warn "Emergency lockout active. Time remaining: $((EMERGENCY_LOCKOUT_DURATION - (now - last_emergency)))s"
        return 0
    fi
    return 1
}

# --- Cardinality Analysis ---
calculate_cardinality_risk() {
    local current_ts="$1"
    local growth_rate="$2"
    local explosion_count="$3"
    
    # Risk score based on multiple factors
    local risk_score=0
    
    # Factor 1: Absolute cardinality
    if [ "$current_ts" -gt "$EXPLOSION_ABSOLUTE_THRESHOLD" ]; then
        risk_score=$(echo "$risk_score + 0.4" | bc -l)
    elif [ "$current_ts" -gt $((EXPLOSION_ABSOLUTE_THRESHOLD / 2)) ]; then
        risk_score=$(echo "$risk_score + 0.2" | bc -l)
    fi
    
    # Factor 2: Growth rate
    if (( $(echo "$growth_rate > $EXPLOSION_RATE_THRESHOLD" | bc -l) )); then
        risk_score=$(echo "$risk_score + 0.4" | bc -l)
    elif (( $(echo "$growth_rate > $((EXPLOSION_RATE_THRESHOLD / 2))" | bc -l) )); then
        risk_score=$(echo "$risk_score + 0.2" | bc -l)
    fi
    
    # Factor 3: Active explosions
    if [ "$explosion_count" -gt 0 ]; then
        risk_score=$(echo "$risk_score + 0.2" | bc -l)
    fi
    
    echo "$risk_score"
}

# --- Enhanced Prometheus Query ---
query_prometheus_with_validation() {
    local query="$1"
    local default="$2"
    local description="$3"
    
    local value=$(query_prometheus_value "$query" "$default")
    
    # Validate value is reasonable
    if [ "$value" -lt 0 ]; then
        log_warn "Invalid negative value for $description: $value. Using default: $default"
        echo "$default"
        return
    fi
    
    # Check for unrealistic spikes (10x previous value)
    if [ -n "$4" ] && [ "$4" -gt 0 ]; then
        local previous="$4"
        if [ "$value" -gt $((previous * 10)) ]; then
            log_warn "Unrealistic spike in $description: $value (previous: $previous). Using previous value."
            echo "$previous"
            return
        fi
    fi
    
    echo "$value"
}

# Helper to query Prometheus with retries and validation
query_prometheus_value() {
    local query_string="$1"
    local default_value="${2:-0}"
    local response_file="/tmp/prom_response_$$.json"
    local value
    local http_status
    local retry_count=3
    local retry_delay=2

    for ((attempt=1; attempt <= retry_count; attempt++)); do
        http_status=$(curl -s -m 10 -G "${PROM_API_ENDPOINT}/query" \
            --data-urlencode "query=${query_string}" \
            -o "$response_file" \
            -w "%{http_code}")

        if [[ "$http_status" -eq 200 ]]; then
            value=$(jq -r '.data.result[0].value[1] // "null"' "$response_file")
            if [[ "$value" != "null" && -n "$value" ]]; then
                break
            fi
        fi

        if [[ "$attempt" -lt "$retry_count" ]]; then
            log_warn "Prometheus query attempt $attempt/$retry_count failed. Retrying in ${retry_delay}s..."
            sleep "$retry_delay"
        fi
    done

    if [[ "$http_status" -ne 200 ]]; then
        log_warn "Prometheus query failed for '$query_string'. HTTP Status: $http_status. Assuming default: $default_value. Response: $(cat $response_file || echo 'empty')"
        value="$default_value"
    elif [[ "$value" == "null" || -z "$value" ]]; then
        log_warn "Metric for '$query_string' not found or no data after $retry_count attempts. Assuming default: $default_value."
        value="$default_value"
    fi

    rm -f "$response_file"

    if [[ -n "$value" ]]; then
        sanitized_value=$(echo "$value" | awk '{printf "%.0f", $1}')
        if [[ "$sanitized_value" =~ ^[0-9]+$ ]]; then
            echo "$sanitized_value"
            return 0
        else
            log_warn "Value '$value' is not a valid number. Using default: $default_value"
        fi
    fi

    echo "$default_value"
}

# --- Main Enhanced Control Logic ---
main() {
    log_info "Starting enhanced adaptive controller with advanced stability features"
    
    # Load previous state
    local state=$(load_state)
    local PREV_OPTIMIZATION_PROFILE=$(echo "$state" | jq -r '.change_history[-1].profile // "balanced"')
    local PREV_LAST_CHANGE=$(echo "$state" | jq -r '.change_history[-1].timestamp // 0')
    PREV_OPTIMISED_TS_FROM_FILE=0
    
    # Check for emergency lockout
    if check_emergency_lockout "$state"; then
        log_warn "Controller in emergency lockout. Skipping this cycle."
        exit 0
    fi
    
    # Check for oscillation
    if check_oscillation "$state"; then
        log_warn "Oscillation detected. Forcing stable profile for this cycle."
        FORCED_PROFILE="balanced"
    fi
    
    # Query metrics with validation
    local current_optimised_ts=$(query_prometheus_with_validation \
        "$METRIC_OPTIMISED_TS_QUERY" \
        "$PREV_OPTIMISED_TS_FROM_FILE" \
        "optimised time series" \
        "$PREV_OPTIMISED_TS_FROM_FILE")
    
    # Calculate cardinality growth rate
    local growth_rate=$(query_prometheus_value \
        "rate($METRIC_OPTIMISED_TS_QUERY[5m])" \
        "0")
    
    # Get explosion alerts
    local explosion_count=$(query_prometheus_value \
        "$METRIC_CARDINALITY_EXPLOSION_ALERT" \
        "0")
    
    # Calculate risk score
    local risk_score=$(calculate_cardinality_risk \
        "$current_optimised_ts" \
        "$growth_rate" \
        "$explosion_count")
    
    log_info "Cardinality risk score: $risk_score (threshold: $RISK_SCORE_THRESHOLD)"
    
    # Determine profile with enhanced logic
    local proposed_profile=""
    
    if (( $(echo "$risk_score > $RISK_SCORE_THRESHOLD" | bc -l) )); then
        proposed_profile="aggressive"
        log_warn "High cardinality risk detected. Forcing aggressive optimization."

        # Update emergency timestamp
        state=$(echo "$state" | jq ".last_emergency_timestamp = $(date +%s)")
    elif [ -n "$FORCED_PROFILE" ]; then
        proposed_profile="$FORCED_PROFILE"
    else
        # Threshold based logic with hysteresis
        CONSERVATIVE_MAX_TS_NUM=$(echo "$CONSERVATIVE_MAX_TS_THRESHOLD" | bc)
        AGGRESSIVE_MIN_TS_NUM=$(echo "$AGGRESSIVE_MIN_TS_THRESHOLD" | bc)
        CURRENT_OPTIMISED_TS_NUM=$(echo "$current_optimised_ts" | bc)
        HYSTERESIS_FACTOR_NUM=$(echo "$HYSTERESIS_FACTOR" | bc -l)
        CONSERVATIVE_MAX_WITH_HYSTERESIS=$(echo "$CONSERVATIVE_MAX_TS_NUM * (1 + $HYSTERESIS_FACTOR_NUM)" | bc -l)
        AGGRESSIVE_MIN_WITH_HYSTERESIS=$(echo "$AGGRESSIVE_MIN_TS_NUM * (1 - $HYSTERESIS_FACTOR_NUM)" | bc -l)

        if [[ "$PREV_OPTIMIZATION_PROFILE" == "conservative" ]]; then
            if (( $(echo "$CURRENT_OPTIMISED_TS_NUM > $CONSERVATIVE_MAX_WITH_HYSTERESIS" | bc -l) )); then
                if (( $(echo "$CURRENT_OPTIMISED_TS_NUM > $AGGRESSIVE_MIN_TS_NUM" | bc -l) )); then
                    proposed_profile="aggressive"
                else
                    proposed_profile="balanced"
                fi
            else
                proposed_profile="conservative"
            fi
        elif [[ "$PREV_OPTIMIZATION_PROFILE" == "aggressive" ]]; then
            if (( $(echo "$CURRENT_OPTIMISED_TS_NUM < $AGGRESSIVE_MIN_WITH_HYSTERESIS" | bc -l) )); then
                if (( $(echo "$CURRENT_OPTIMISED_TS_NUM < $CONSERVATIVE_MAX_TS_NUM" | bc -l) )); then
                    proposed_profile="conservative"
                else
                    proposed_profile="balanced"
                fi
            else
                proposed_profile="aggressive"
            fi
        else
            if (( $(echo "$CURRENT_OPTIMISED_TS_NUM > $AGGRESSIVE_MIN_TS_NUM" | bc -l) )); then
                proposed_profile="aggressive"
            elif (( $(echo "$CURRENT_OPTIMISED_TS_NUM < $CONSERVATIVE_MAX_TS_NUM" | bc -l) )); then
                proposed_profile="conservative"
            else
                proposed_profile="balanced"
            fi
        fi

        if [[ "$proposed_profile" != "$PREV_OPTIMIZATION_PROFILE" ]]; then
            NOW=$(date +%s)
            TIME_SINCE_LAST_CHANGE=$((NOW - PREV_LAST_CHANGE))
            if (( TIME_SINCE_LAST_CHANGE < STABILITY_PERIOD_SECONDS )); then
                log_info "Stability hold ($TIME_SINCE_LAST_CHANGE s < $STABILITY_PERIOD_SECONDS s). Maintaining '$PREV_OPTIMIZATION_PROFILE'."
                proposed_profile="$PREV_OPTIMIZATION_PROFILE"
            else
                PREV_LAST_CHANGE=$NOW
            fi
        fi
    fi
    
    # Update state with new profile change
    if [ "$proposed_profile" != "$PREV_OPTIMIZATION_PROFILE" ]; then
        state=$(update_change_history "$state" "$proposed_profile")
    fi
    
    # Save state
    save_state "$state"
    
    # Continue with original control file update logic...
    log_info "Enhanced control cycle complete. Profile: $proposed_profile"
}

# Run main function
main "$@"