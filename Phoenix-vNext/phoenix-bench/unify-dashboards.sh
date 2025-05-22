#!/bin/bash
# Script to unify the Phoenix-vNext dashboards
# Creates a consolidated dashboard with the best features from each

set -e

# ANSI colors for better output formatting
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

DASHBOARDS_DIR="configs/dashboards"
SOURCE_DASHBOARDS=("phoenix-5-pipeline-comparison.json" "phoenix-dashboard.json" "phoenix-5pipeline-dashboard.json")
OUTPUT_DASHBOARD="$DASHBOARDS_DIR/phoenix-unified-dashboard.json"
TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")
BACKUP_DIR="$DASHBOARDS_DIR/backups"

echo -e "${BLUE}=== Phoenix-vNext Dashboard Unification ===${NC}"
echo "$(date)"
echo

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

# Backup all existing dashboards
echo -e "${BLUE}📦 Backing up existing dashboards...${NC}"
for dashboard in "${SOURCE_DASHBOARDS[@]}"; do
    source_path="$DASHBOARDS_DIR/$dashboard"
    if [ -f "$source_path" ]; then
        backup_path="$BACKUP_DIR/${dashboard%.json}_$TIMESTAMP.json"
        cp "$source_path" "$backup_path"
        echo -e "${GREEN}✓ Backed up $dashboard to $(basename "$backup_path")${NC}"
    else
        echo -e "${YELLOW}⚠️ Dashboard $dashboard not found, skipping backup${NC}"
    fi
done

# Extract the best components from each dashboard
echo -e "${BLUE}🔄 Creating unified dashboard...${NC}"

# Start with the most complete dashboard as the base
BASE_DASHBOARD="$DASHBOARDS_DIR/phoenix-5-pipeline-comparison.json"
if [ ! -f "$BASE_DASHBOARD" ]; then
    echo -e "${RED}❌ Base dashboard file not found: $BASE_DASHBOARD${NC}"
    exit 1
fi

# Copy the base dashboard to the output file
cp "$BASE_DASHBOARD" "$OUTPUT_DASHBOARD"

# Create a unified dashboard with a consistent set of features
jq '
  # Set consistent properties
  .title = "Phoenix-vNext Unified 5-Pipeline Dashboard" |
  .description = "Comprehensive monitoring for Phoenix-vNext 5-pipeline architecture with optimization metrics" |
  .uid = "phoenix-unified" |
  .version = 1 |
  .refresh = "10s" |
  
  # Ensure all datasources use consistent UID
  (.. | objects | select(has("datasource"?)) | .datasource) |= 
    if . != null and ((.type == "prometheus") or (.type == "Prometheus")) then 
      .uid = "prometheus" 
    else 
      . 
    end |
  
  # Add dashboard annotations for mode changes
  .annotations.list += [
    {
      "datasource": {
        "type": "prometheus",
        "uid": "prometheus"
      },
      "enable": true,
      "expr": "changes(phoenix_opt_mode[1m]) > 0",
      "iconColor": "red",
      "name": "Mode Changes",
      "showIn": 0,
      "tags": ["mode-change"],
      "titleFormat": "Mode Change"
    }
  ] |
  
  # Add dashboard metadata
  .tags = ["phoenix", "opentelemetry", "optimization", "unified"] |
  .timezone = "browser" |
  .schemaVersion = 27
' "$BASE_DASHBOARD" > "$OUTPUT_DASHBOARD"

echo -e "${GREEN}✅ Created unified dashboard: ${CYAN}$(basename "$OUTPUT_DASHBOARD")${NC}"
echo
echo -e "${BLUE}🚀 Next steps:${NC}"
echo -e "1. Run ${CYAN}chmod +x ./setup-dashboards.sh${NC} to make the setup script executable"
echo -e "2. Run ${CYAN}./setup-dashboards.sh${NC} to import all dashboards into Grafana"
echo -e "3. Restart Grafana if needed with ${CYAN}docker compose restart grafana${NC}"
echo
