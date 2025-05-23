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

# Include original query_prometheus_value function here...
query_prometheus_value() {
    # Original implementation from the file
    local query_string="$1"
    local default_value="${2:-0}"
    # ... (rest of original implementation)
}

# --- Main Enhanced Control Logic ---
main() {
    log_info "Starting enhanced adaptive controller with advanced stability features"
    
    # Load previous state
    local state=$(load_state)
    
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
        # Use normal threshold-based logic with hysteresis
        # (Original implementation logic here)
        proposed_profile="balanced" # Placeholder
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