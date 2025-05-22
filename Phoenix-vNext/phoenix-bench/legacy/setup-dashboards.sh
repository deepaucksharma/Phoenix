#!/bin/bash
# Dashboard Setup Script for Phoenix-vNext 5-Pipeline Comparison
# Imports all dashboards into Grafana and fixes configuration issues

set -e

# ANSI colors for better output formatting
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

GRAFANA_URL="http://localhost:3000"
GRAFANA_USER="admin"
GRAFANA_PASS="admin"
DASHBOARDS_DIR="configs/dashboards"
PRIMARY_DASHBOARD="phoenix-5-pipeline-comparison.json"

echo -e "${BLUE}=== Phoenix-vNext Dashboard Setup ===${NC}"
echo "$(date)"
echo

# Check if Grafana is accessible
if ! curl -s "$GRAFANA_URL/api/health" > /dev/null; then
    echo -e "${RED}❌ Grafana is not accessible at $GRAFANA_URL${NC}"
    echo -e "Please ensure Grafana is running with: ${CYAN}docker compose up -d grafana${NC}"
    exit 1
fi

# Function to fix Prometheus datasource UIDs in dashboard files
fix_dashboard_uids() {
    local dashboard_file="$1"
    local temp_file="${dashboard_file}.tmp"
    
    echo -e "${YELLOW}Fixing datasource UIDs in ${dashboard_file}...${NC}"
    
    # Replace various forms of Prometheus UIDs with consistent UID
    jq '
      (.. | objects | select(has("datasource"?)) | .datasource) |= 
      if . != null and ((.type == "prometheus") or (.type == "Prometheus")) then 
        .uid = "prometheus" 
      else 
        . 
      end
    ' "$dashboard_file" > "$temp_file"
    
    # Verify the transformation worked before replacing
    if [ -s "$temp_file" ]; then
        mv "$temp_file" "$dashboard_file"
        echo -e "${GREEN}✓ Fixed datasource UIDs in ${dashboard_file}${NC}"
    else
        echo -e "${RED}✗ Failed to fix datasource UIDs in ${dashboard_file}${NC}"
        rm -f "$temp_file"
    fi
}

# Create datasource for Prometheus
echo -e "${BLUE}📊 Setting up Prometheus datasource...${NC}"
curl -s -X POST \
  -H "Content-Type: application/json" \
  -u "$GRAFANA_USER:$GRAFANA_PASS" \
  "$GRAFANA_URL/api/datasources" \
  -d '{
    "name": "prometheus",
    "type": "prometheus",
    "url": "http://prometheus:9090",
    "access": "proxy",
    "uid": "prometheus",
    "isDefault": true
  }' | jq -r '.message // "Datasource created or already exists"'

# Fix UIDs in all dashboard files
echo -e "${BLUE}🔧 Fixing dashboard configurations...${NC}"
for dashboard_file in $DASHBOARDS_DIR/*.json; do
    if [ -f "$dashboard_file" ]; then
        fix_dashboard_uids "$dashboard_file"
    fi
done

# Import all dashboards
echo -e "${BLUE}📈 Importing Phoenix dashboards...${NC}"
for dashboard_file in $DASHBOARDS_DIR/*.json; do
    if [ -f "$dashboard_file" ]; then
        dashboard_name=$(basename "$dashboard_file")
        echo -e "${CYAN}Importing dashboard: ${dashboard_name}${NC}"
        
        # Ensure the dashboard has a unique title
        title=$(jq -r '.title' "$dashboard_file")
        if [ -z "$title" ] || [ "$title" == "null" ]; then
            # Extract name from filename if no title
            title="Phoenix $(echo $dashboard_name | sed 's/.json//')"
            # Update title in file
            jq --arg title "$title" '.title = $title' "$dashboard_file" > "${dashboard_file}.tmp"
            mv "${dashboard_file}.tmp" "$dashboard_file"
        fi
        
        # Wrap the dashboard JSON in the import format
        DASHBOARD_JSON=$(cat "$dashboard_file")
        IMPORT_PAYLOAD=$(jq -n --argjson dashboard "$DASHBOARD_JSON" '{
          dashboard: $dashboard,
          overwrite: true,
          inputs: []
        }')

        RESPONSE=$(curl -s -X POST \
          -H "Content-Type: application/json" \
          -u "$GRAFANA_USER:$GRAFANA_PASS" \
          "$GRAFANA_URL/api/dashboards/db" \
          -d "$IMPORT_PAYLOAD")

        # Parse response
        DASHBOARD_URL=$(echo "$RESPONSE" | jq -r '.url // empty')
        if [ -n "$DASHBOARD_URL" ]; then
            echo -e "${GREEN}✓ Dashboard $title imported successfully!${NC}"
        else
            echo -e "${RED}⚠️ Dashboard import response:${NC}"
            echo "$RESPONSE" | jq '.'
        fi
    fi
done

# Get primary dashboard URL for user access
PRIMARY_DASHBOARD_PATH="${DASHBOARDS_DIR}/${PRIMARY_DASHBOARD}"
if [ -f "$PRIMARY_DASHBOARD_PATH" ]; then
    PRIMARY_TITLE=$(jq -r '.title' "$PRIMARY_DASHBOARD_PATH")
    echo -e "${GREEN}🎯 Access your Phoenix 5-Pipeline Comparison Dashboard at:${NC}"
    echo -e "${CYAN}   $GRAFANA_URL/dashboards${NC}"
    echo -e "${CYAN}   Look for dashboard: $PRIMARY_TITLE${NC}"
fi

echo ""
echo -e "${GREEN}🚀 Phoenix Dashboard Setup Complete!${NC}"
echo ""
echo -e "${BLUE}📊 Available Pipeline Endpoints:${NC}"
echo -e "   Full Pipeline (Port 8888):         ${CYAN}http://localhost:8888/metrics${NC}"
echo -e "   Optimized Pipeline (Port 8889):    ${CYAN}http://localhost:8889/metrics${NC}" 
echo -e "   Ultra Pipeline (Port 8890):        ${CYAN}http://localhost:8890/metrics${NC}"
echo -e "   Experimental Pipeline (Port 8895): ${CYAN}http://localhost:8895/metrics${NC}"
echo -e "   Hybrid Pipeline (Port 8896):       ${CYAN}http://localhost:8896/metrics${NC}"
echo ""
echo -e "${BLUE}🎪 Current Cardinality Status:${NC}"
for port in 8888 8889 8890 8895 8896; do
    metric_count=$(curl -s -m 2 "http://localhost:$port/metrics" | grep -c "^[a-z]" || echo "0")
    if [ "$port" == "8888" ]; then pipeline="Full Pipeline     "
    elif [ "$port" == "8889" ]; then pipeline="Optimized Pipeline"
    elif [ "$port" == "8890" ]; then pipeline="Ultra Pipeline    "
    elif [ "$port" == "8895" ]; then pipeline="Experimental     "
    elif [ "$port" == "8896" ]; then pipeline="Hybrid Pipeline  "
    fi
    
    if [ "$metric_count" -gt 0 ]; then
        echo -e "   ${pipeline}: ${GREEN}$metric_count metrics${NC}"
    else
        echo -e "   ${pipeline}: ${YELLOW}$metric_count metrics (not responding)${NC}"
    fi
done
