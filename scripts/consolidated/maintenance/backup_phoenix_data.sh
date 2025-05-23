#!/bin/bash

# Phoenix Data Backup Script
# Backs up Prometheus data, configurations, and benchmark results

set -euo pipefail

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/var/backups/phoenix}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
PROMETHEUS_DATA_DIR="${PROMETHEUS_DATA_DIR:-./data/prometheus}"
GRAFANA_DATA_DIR="${GRAFANA_DATA_DIR:-./data/grafana}"
BENCHMARK_DB="${BENCHMARK_DB:-./data/benchmark/results.db}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

error_exit() {
    echo -e "${RED}[ERROR] $1${NC}" >&2
    exit 1
}

# Create backup directory
create_backup_dir() {
    local backup_path="$BACKUP_DIR/phoenix_backup_$TIMESTAMP"
    mkdir -p "$backup_path" || error_exit "Failed to create backup directory"
    echo "$backup_path"
}

# Backup Prometheus data
backup_prometheus() {
    local backup_path=$1
    log "Backing up Prometheus data..."
    
    if [ -d "$PROMETHEUS_DATA_DIR" ]; then
        # Create snapshot using Prometheus API
        if curl -s -X POST http://localhost:9090/api/v1/admin/tsdb/snapshot > /dev/null 2>&1; then
            # Find the latest snapshot
            latest_snapshot=$(ls -t "$PROMETHEUS_DATA_DIR/snapshots" 2>/dev/null | head -1)
            
            if [ -n "$latest_snapshot" ]; then
                tar -czf "$backup_path/prometheus_data.tar.gz" \
                    -C "$PROMETHEUS_DATA_DIR/snapshots" "$latest_snapshot" || {
                    log "Warning: Failed to backup Prometheus snapshot"
                }
                
                # Clean up snapshot
                rm -rf "$PROMETHEUS_DATA_DIR/snapshots/$latest_snapshot"
            fi
        else
            # Fallback to direct copy
            tar -czf "$backup_path/prometheus_data.tar.gz" \
                -C "$(dirname "$PROMETHEUS_DATA_DIR")" \
                "$(basename "$PROMETHEUS_DATA_DIR")" || {
                log "Warning: Failed to backup Prometheus data"
            }
        fi
    else
        log "Prometheus data directory not found"
    fi
}

# Backup Grafana data
backup_grafana() {
    local backup_path=$1
    log "Backing up Grafana data..."
    
    if [ -d "$GRAFANA_DATA_DIR" ]; then
        tar -czf "$backup_path/grafana_data.tar.gz" \
            -C "$(dirname "$GRAFANA_DATA_DIR")" \
            "$(basename "$GRAFANA_DATA_DIR")" || {
            log "Warning: Failed to backup Grafana data"
        }
    else
        log "Grafana data directory not found"
    fi
}

# Backup configurations
backup_configs() {
    local backup_path=$1
    log "Backing up configurations..."
    
    # Backup all config directories
    for config_dir in configs configs/monitoring/prometheus configs/monitoring/grafana; do
        if [ -d "$config_dir" ]; then
            tar -czf "$backup_path/$(basename $config_dir)_configs.tar.gz" \
                "$config_dir" || {
                log "Warning: Failed to backup $config_dir"
            }
        fi
    done
    
    # Backup environment file
    if [ -f .env ]; then
        cp .env "$backup_path/env_backup" || log "Warning: Failed to backup .env"
    fi
}

# Backup benchmark database
backup_benchmark_db() {
    local backup_path=$1
    log "Backing up benchmark database..."
    
    if [ -f "$BENCHMARK_DB" ]; then
        # Use SQLite backup command for consistency
        sqlite3 "$BENCHMARK_DB" ".backup '$backup_path/benchmark.db'" 2>/dev/null || {
            # Fallback to file copy
            cp "$BENCHMARK_DB" "$backup_path/benchmark.db" || {
                log "Warning: Failed to backup benchmark database"
            }
        }
    else
        log "Benchmark database not found"
    fi
}

# Export current metrics
export_current_metrics() {
    local backup_path=$1
    log "Exporting current metrics..."
    
    # Export key metrics
    metrics_file="$backup_path/metrics_export.json"
    
    {
        echo "{"
        echo '  "timestamp": "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'",'
        echo '  "metrics": {'
        
        # Export cardinality metrics
        curl -s "http://localhost:9090/api/v1/query?query=phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate" | \
            jq -r '.data.result[] | "    \"\(.metric.pipeline)_cardinality\": \(.value[1]),"' 2>/dev/null || true
            
        # Export control state
        curl -s http://localhost:8081/metrics | \
            jq -r 'to_entries | map("    \"\(.key)\": \(.value),") | .[]' 2>/dev/null || true
            
        echo '    "backup_version": "1.0"'
        echo "  }"
        echo "}"
    } > "$metrics_file"
}

# Create backup manifest
create_manifest() {
    local backup_path=$1
    local manifest_file="$backup_path/manifest.json"
    
    {
        echo "{"
        echo '  "backup_timestamp": "'$TIMESTAMP'",'
        echo '  "backup_version": "1.0",'
        echo '  "contents": ['
        
        local first=true
        for file in "$backup_path"/*; do
            if [ -f "$file" ] && [ "$(basename "$file")" != "manifest.json" ]; then
                if [ "$first" = false ]; then echo ","; fi
                echo -n '    {'
                echo -n '"file": "'$(basename "$file")'",'
                echo -n '"size": '$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo 0)','
                echo -n '"checksum": "'$(sha256sum "$file" | cut -d' ' -f1)'"'
                echo -n '}'
                first=false
            fi
        done
        
        echo ""
        echo "  ]"
        echo "}"
    } > "$manifest_file"
}

# Cleanup old backups
cleanup_old_backups() {
    log "Cleaning up old backups..."
    
    find "$BACKUP_DIR" -name "phoenix_backup_*" -type d -mtime +$RETENTION_DAYS -exec rm -rf {} + 2>/dev/null || {
        log "Warning: Failed to clean some old backups"
    }
}

# Verify backup
verify_backup() {
    local backup_path=$1
    log "Verifying backup..."
    
    local errors=0
    
    # Check manifest
    if [ ! -f "$backup_path/manifest.json" ]; then
        log "Error: Manifest file missing"
        ((errors++))
    fi
    
    # Verify file checksums
    if [ -f "$backup_path/manifest.json" ]; then
        while IFS= read -r line; do
            if [[ $line =~ \"file\":\ \"([^\"]+)\" ]]; then
                file="${BASH_REMATCH[1]}"
                if [ ! -f "$backup_path/$file" ]; then
                    log "Error: File missing: $file"
                    ((errors++))
                fi
            fi
        done < "$backup_path/manifest.json"
    fi
    
    return $errors
}

# Main backup function
perform_backup() {
    log "Starting Phoenix backup..."
    
    # Create backup directory
    backup_path=$(create_backup_dir)
    log "Backup directory: $backup_path"
    
    # Perform backups
    backup_prometheus "$backup_path"
    backup_grafana "$backup_path"
    backup_configs "$backup_path"
    backup_benchmark_db "$backup_path"
    export_current_metrics "$backup_path"
    
    # Create manifest
    create_manifest "$backup_path"
    
    # Verify backup
    if verify_backup "$backup_path"; then
        log "Backup completed successfully"
        
        # Create latest symlink
        ln -sfn "$backup_path" "$BACKUP_DIR/latest"
        
        # Cleanup old backups
        cleanup_old_backups
        
        echo -e "${GREEN}✓ Backup completed: $backup_path${NC}"
        return 0
    else
        error_exit "Backup verification failed"
    fi
}

# Show usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Options:
    -d, --backup-dir DIR    Backup directory (default: $BACKUP_DIR)
    -r, --retention DAYS    Retention period in days (default: $RETENTION_DAYS)
    -l, --list              List existing backups
    -h, --help              Show this help message

Environment variables:
    BACKUP_DIR              Backup directory path
    RETENTION_DAYS          Number of days to retain backups
    PROMETHEUS_DATA_DIR     Prometheus data directory
    GRAFANA_DATA_DIR        Grafana data directory
    BENCHMARK_DB            Benchmark database path
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--backup-dir)
            BACKUP_DIR="$2"
            shift 2
            ;;
        -r|--retention)
            RETENTION_DAYS="$2"
            shift 2
            ;;
        -l|--list)
            ls -la "$BACKUP_DIR"/phoenix_backup_* 2>/dev/null || echo "No backups found"
            exit 0
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Execute backup
perform_backup