#!/bin/bash

# Phoenix Data Restore Script
# Restores Phoenix data from backups

set -euo pipefail

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/var/backups/phoenix}"
RESTORE_PROMETHEUS="${RESTORE_PROMETHEUS:-true}"
RESTORE_GRAFANA="${RESTORE_GRAFANA:-true}"
RESTORE_CONFIGS="${RESTORE_CONFIGS:-true}"
RESTORE_BENCHMARK="${RESTORE_BENCHMARK:-true}"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

error_exit() {
    echo -e "${RED}[ERROR] $1${NC}" >&2
    exit 1
}

warning() {
    echo -e "${YELLOW}[WARNING] $1${NC}"
}

# Find backup to restore
find_backup() {
    local backup_path=""
    
    if [ -n "${1:-}" ]; then
        # Specific backup requested
        if [ -d "$1" ]; then
            backup_path="$1"
        elif [ -d "$BACKUP_DIR/$1" ]; then
            backup_path="$BACKUP_DIR/$1"
        else
            error_exit "Backup not found: $1"
        fi
    else
        # Use latest backup
        if [ -L "$BACKUP_DIR/latest" ]; then
            backup_path=$(readlink -f "$BACKUP_DIR/latest")
        else
            # Find most recent backup
            backup_path=$(ls -dt "$BACKUP_DIR"/phoenix_backup_* 2>/dev/null | head -1)
        fi
    fi
    
    if [ -z "$backup_path" ] || [ ! -d "$backup_path" ]; then
        error_exit "No backup found to restore"
    fi
    
    echo "$backup_path"
}

# Verify backup integrity
verify_backup() {
    local backup_path=$1
    log "Verifying backup integrity..."
    
    if [ ! -f "$backup_path/manifest.json" ]; then
        error_exit "Invalid backup: manifest.json not found"
    fi
    
    # Verify checksums
    local errors=0
    while IFS= read -r line; do
        if [[ $line =~ \"file\":\ \"([^\"]+)\".*\"checksum\":\ \"([^\"]+)\" ]]; then
            file="${BASH_REMATCH[1]}"
            expected_checksum="${BASH_REMATCH[2]}"
            
            if [ -f "$backup_path/$file" ]; then
                actual_checksum=$(sha256sum "$backup_path/$file" | cut -d' ' -f1)
                if [ "$actual_checksum" != "$expected_checksum" ]; then
                    warning "Checksum mismatch for $file"
                    ((errors++))
                fi
            else
                warning "File missing: $file"
                ((errors++))
            fi
        fi
    done < "$backup_path/manifest.json"
    
    if [ $errors -gt 0 ]; then
        error_exit "Backup verification failed with $errors errors"
    fi
    
    log "Backup verification passed"
}

# Stop Phoenix services
stop_services() {
    log "Stopping Phoenix services..."
    
    if command -v docker-compose &> /dev/null; then
        docker-compose stop || warning "Failed to stop some services"
    else
        warning "docker-compose not found - please stop services manually"
        read -p "Press Enter when services are stopped..."
    fi
}

# Start Phoenix services
start_services() {
    log "Starting Phoenix services..."
    
    if command -v docker-compose &> /dev/null; then
        docker-compose up -d || warning "Failed to start some services"
    else
        warning "docker-compose not found - please start services manually"
    fi
}

# Restore Prometheus data
restore_prometheus() {
    local backup_path=$1
    
    if [ "$RESTORE_PROMETHEUS" != "true" ]; then
        log "Skipping Prometheus restore"
        return
    fi
    
    log "Restoring Prometheus data..."
    
    if [ -f "$backup_path/prometheus_data.tar.gz" ]; then
        # Backup current data
        if [ -d "$PROMETHEUS_DATA_DIR" ]; then
            mv "$PROMETHEUS_DATA_DIR" "${PROMETHEUS_DATA_DIR}.bak.$(date +%Y%m%d_%H%M%S)"
        fi
        
        # Extract backup
        mkdir -p "$(dirname "$PROMETHEUS_DATA_DIR")"
        tar -xzf "$backup_path/prometheus_data.tar.gz" \
            -C "$(dirname "$PROMETHEUS_DATA_DIR")" || {
            error_exit "Failed to restore Prometheus data"
        }
        
        log "Prometheus data restored"
    else
        warning "Prometheus backup not found"
    fi
}

# Restore Grafana data
restore_grafana() {
    local backup_path=$1
    
    if [ "$RESTORE_GRAFANA" != "true" ]; then
        log "Skipping Grafana restore"
        return
    fi
    
    log "Restoring Grafana data..."
    
    if [ -f "$backup_path/grafana_data.tar.gz" ]; then
        # Backup current data
        if [ -d "$GRAFANA_DATA_DIR" ]; then
            mv "$GRAFANA_DATA_DIR" "${GRAFANA_DATA_DIR}.bak.$(date +%Y%m%d_%H%M%S)"
        fi
        
        # Extract backup
        mkdir -p "$(dirname "$GRAFANA_DATA_DIR")"
        tar -xzf "$backup_path/grafana_data.tar.gz" \
            -C "$(dirname "$GRAFANA_DATA_DIR")" || {
            error_exit "Failed to restore Grafana data"
        }
        
        log "Grafana data restored"
    else
        warning "Grafana backup not found"
    fi
}

# Restore configurations
restore_configs() {
    local backup_path=$1
    
    if [ "$RESTORE_CONFIGS" != "true" ]; then
        log "Skipping configuration restore"
        return
    fi
    
    log "Restoring configurations..."
    
    # Restore config archives
    for archive in "$backup_path"/*_configs.tar.gz; do
        if [ -f "$archive" ]; then
            log "Extracting $(basename "$archive")..."
            tar -xzf "$archive" || warning "Failed to extract $(basename "$archive")"
        fi
    done
    
    # Restore environment file
    if [ -f "$backup_path/env_backup" ]; then
        cp "$backup_path/env_backup" .env.restored
        log "Environment file restored to .env.restored"
        log "Review and rename to .env when ready"
    fi
}

# Restore benchmark database
restore_benchmark_db() {
    local backup_path=$1
    
    if [ "$RESTORE_BENCHMARK" != "true" ]; then
        log "Skipping benchmark database restore"
        return
    fi
    
    log "Restoring benchmark database..."
    
    if [ -f "$backup_path/benchmark.db" ]; then
        # Backup current database
        if [ -f "$BENCHMARK_DB" ]; then
            mv "$BENCHMARK_DB" "${BENCHMARK_DB}.bak.$(date +%Y%m%d_%H%M%S)"
        fi
        
        # Restore database
        mkdir -p "$(dirname "$BENCHMARK_DB")"
        cp "$backup_path/benchmark.db" "$BENCHMARK_DB" || {
            error_exit "Failed to restore benchmark database"
        }
        
        log "Benchmark database restored"
    else
        warning "Benchmark database backup not found"
    fi
}

# Show restore summary
show_summary() {
    local backup_path=$1
    
    echo ""
    echo "Restore Summary"
    echo "==============="
    echo "Backup: $(basename "$backup_path")"
    echo "Date: $(jq -r .backup_timestamp "$backup_path/manifest.json" 2>/dev/null || echo "Unknown")"
    echo ""
    echo "Components restored:"
    [ "$RESTORE_PROMETHEUS" = "true" ] && echo "  ✓ Prometheus data"
    [ "$RESTORE_GRAFANA" = "true" ] && echo "  ✓ Grafana data"
    [ "$RESTORE_CONFIGS" = "true" ] && echo "  ✓ Configurations"
    [ "$RESTORE_BENCHMARK" = "true" ] && echo "  ✓ Benchmark database"
    echo ""
    
    if [ -f "$backup_path/metrics_export.json" ]; then
        echo "Metrics at backup time:"
        jq -r '.metrics | to_entries | .[] | "  \(.key): \(.value)"' "$backup_path/metrics_export.json" 2>/dev/null || true
    fi
}

# Main restore function
perform_restore() {
    local backup_path=$1
    
    log "Starting Phoenix restore from: $backup_path"
    
    # Verify backup
    verify_backup "$backup_path"
    
    # Confirm restore
    echo ""
    echo -e "${YELLOW}WARNING: This will restore Phoenix data from backup.${NC}"
    echo -e "${YELLOW}Current data will be backed up with .bak suffix.${NC}"
    echo ""
    read -p "Continue with restore? (yes/no): " confirm
    
    if [ "$confirm" != "yes" ]; then
        log "Restore cancelled"
        exit 0
    fi
    
    # Stop services
    stop_services
    
    # Perform restore
    restore_prometheus "$backup_path"
    restore_grafana "$backup_path"
    restore_configs "$backup_path"
    restore_benchmark_db "$backup_path"
    
    # Show summary
    show_summary "$backup_path"
    
    # Start services
    echo ""
    read -p "Start Phoenix services now? (yes/no): " start_now
    if [ "$start_now" = "yes" ]; then
        start_services
    fi
    
    echo -e "${GREEN}✓ Restore completed successfully${NC}"
}

# Show usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS] [BACKUP_PATH]

Restore Phoenix data from backup. If no backup path is specified,
the latest backup will be used.

Options:
    --prometheus        Restore only Prometheus data
    --grafana          Restore only Grafana data
    --configs          Restore only configurations
    --benchmark        Restore only benchmark database
    --skip-prometheus  Skip Prometheus restore
    --skip-grafana     Skip Grafana restore
    --skip-configs     Skip configuration restore
    --skip-benchmark   Skip benchmark restore
    -l, --list         List available backups
    -h, --help         Show this help message

Examples:
    $0                                    # Restore latest backup
    $0 phoenix_backup_20240523_120000     # Restore specific backup
    $0 --prometheus --grafana             # Restore only Prometheus and Grafana
EOF
}

# Parse command line arguments
BACKUP_PATH=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --prometheus)
            RESTORE_PROMETHEUS=true
            RESTORE_GRAFANA=false
            RESTORE_CONFIGS=false
            RESTORE_BENCHMARK=false
            shift
            ;;
        --grafana)
            RESTORE_PROMETHEUS=false
            RESTORE_GRAFANA=true
            RESTORE_CONFIGS=false
            RESTORE_BENCHMARK=false
            shift
            ;;
        --configs)
            RESTORE_PROMETHEUS=false
            RESTORE_GRAFANA=false
            RESTORE_CONFIGS=true
            RESTORE_BENCHMARK=false
            shift
            ;;
        --benchmark)
            RESTORE_PROMETHEUS=false
            RESTORE_GRAFANA=false
            RESTORE_CONFIGS=false
            RESTORE_BENCHMARK=true
            shift
            ;;
        --skip-prometheus)
            RESTORE_PROMETHEUS=false
            shift
            ;;
        --skip-grafana)
            RESTORE_GRAFANA=false
            shift
            ;;
        --skip-configs)
            RESTORE_CONFIGS=false
            shift
            ;;
        --skip-benchmark)
            RESTORE_BENCHMARK=false
            shift
            ;;
        -l|--list)
            echo "Available backups:"
            ls -la "$BACKUP_DIR"/phoenix_backup_* 2>/dev/null || echo "No backups found"
            exit 0
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        -*)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
        *)
            BACKUP_PATH="$1"
            shift
            ;;
    esac
done

# Find and restore backup
backup_path=$(find_backup "$BACKUP_PATH")
perform_restore "$backup_path"