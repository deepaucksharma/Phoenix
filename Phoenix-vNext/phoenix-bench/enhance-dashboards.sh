#!/bin/bash
# Script to enhance the Phoenix-vNext dashboards with additional metric panels
# Adds new useful panels to monitor the system more effectively

set -e

# ANSI colors for better output formatting
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

DASHBOARDS_DIR="configs/dashboards"
UNIFIED_DASHBOARD="$DASHBOARDS_DIR/phoenix-unified-dashboard.json"
TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")
BACKUP_DIR="$DASHBOARDS_DIR/backups"

echo -e "${BLUE}=== Phoenix-vNext Dashboard Enhancement ===${NC}"
echo "$(date)"
echo

# Check if unified dashboard exists
if [ ! -f "$UNIFIED_DASHBOARD" ]; then
    echo -e "${YELLOW}⚠️ Unified dashboard not found. Please run unify-dashboards.sh first.${NC}"
    echo -e "   Creating unified dashboard now..."
    ./unify-dashboards.sh
    if [ ! -f "$UNIFIED_DASHBOARD" ]; then
        echo -e "${RED}❌ Failed to create unified dashboard${NC}"
        exit 1
    fi
fi

# Make backup of unified dashboard
mkdir -p "$BACKUP_DIR"
backup_path="$BACKUP_DIR/phoenix-unified-dashboard_$TIMESTAMP.json"
cp "$UNIFIED_DASHBOARD" "$backup_path"
echo -e "${GREEN}✓ Backed up unified dashboard to $(basename "$backup_path")${NC}"

# Enhance the dashboard with additional useful panels
echo -e "${BLUE}🔧 Enhancing dashboard with additional panels...${NC}"

# Add the schema coherence panel to track the alignment status
jq '
# Create a working copy of the dashboard we can modify
. as $original |

# Generate a new panel for Schema Coherence
{
  "datasource": {
    "type": "prometheus",
    "uid": "prometheus"
  },
  "description": "Monitors the alignment between defined thresholds and active modes",
  "fieldConfig": {
    "defaults": {
      "color": { "mode": "thresholds" },
      "mappings": [],
      "thresholds": {
        "mode": "absolute",
        "steps": [
          { "color": "green", "value": null },
          { "color": "red", "value": 0.5 }
        ]
      },
      "unit": "none"
    },
    "overrides": []
  },
  "gridPos": { "h": 8, "w": 12, "x": 0, "y": 0 },
  "id": 9999,
  "options": {
    "colorMode": "background",
    "graphMode": "none",
    "justifyMode": "auto",
    "orientation": "vertical",
    "reduceOptions": {
      "calcs": ["lastNotNull"],
      "fields": "",
      "values": false
    },
    "textMode": "auto"
  },
  "pluginVersion": "10.4.3",
  "title": "Schema Coherence Status",
  "type": "stat",
  "targets": [
    {
      "datasource": {
        "type": "prometheus",
        "uid": "prometheus"
      },
      "editorMode": "code",
      "expr": "phoenix_observer_mode",
      "legendFormat": "Current Operation Mode",
      "range": true,
      "refId": "A"
    }
  ]
} as $schema_panel |

# Generate a new panel for Control Signal Health
{
  "datasource": {
    "type": "prometheus",
    "uid": "prometheus"
  },
  "description": "Health status of the control signal mechanism for dynamic optimization",
  "fieldConfig": {
    "defaults": {
      "color": { "mode": "thresholds" },
      "mappings": [
        {
          "options": {
            "0": { "text": "Unhealthy" },
            "1": { "text": "Healthy" }
          },
          "type": "value"
        }
      ],
      "thresholds": {
        "mode": "absolute",
        "steps": [
          { "color": "red", "value": null },
          { "color": "yellow", "value": 0.5 },
          { "color": "green", "value": 1 }
        ]
      }
    },
    "overrides": []
  },
  "gridPos": { "h": 8, "w": 12, "x": 12, "y": 0 },
  "id": 9998,
  "options": {
    "colorMode": "background",
    "graphMode": "area",
    "justifyMode": "auto",
    "orientation": "horizontal",
    "reduceOptions": {
      "calcs": ["lastNotNull"],
      "fields": "",
      "values": false
    },
    "textMode": "auto"
  },
  "pluginVersion": "10.4.3",
  "title": "Control Signal Status",
  "type": "stat",
  "targets": [
    {
      "datasource": {
        "type": "prometheus",
        "uid": "prometheus"
      },
      "editorMode": "code",
      "expr": "up{job=\"otelcol-main-cardinality-scrape\"}",
      "legendFormat": "Observer Signal Health",
      "range": true,
      "refId": "A"
    }
  ]
} as $health_panel |

# Generate a new row for threshold visualization
{
  "collapsed": false,
  "gridPos": {
    "h": 1,
    "w": 24,
    "x": 0,
    "y": 16
  },
  "id": 9997,
  "panels": [],
  "title": "Optimization Thresholds",
  "type": "row"
} as $threshold_row |

# Generate a new panel for Threshold Visualization
{
  "datasource": {
    "type": "prometheus",
    "uid": "prometheus"
  },
  "description": "Visualization of the current threshold settings compared to active metrics",
  "fieldConfig": {
    "defaults": {
      "color": { "mode": "palette-classic" },
      "custom": {
        "axisCenteredZero": false,
        "axisColorMode": "text",
        "axisLabel": "Active Series Count",
        "axisPlacement": "auto",
        "barAlignment": 0,
        "drawStyle": "line",
        "fillOpacity": 20,
        "gradientMode": "none",
        "hideFrom": {
          "legend": false,
          "tooltip": false,
          "viz": false
        },
        "lineInterpolation": "linear",
        "lineWidth": 2,
        "pointSize": 5,
        "scaleDistribution": {
          "type": "linear"
        },
        "showPoints": "never",
        "spanNulls": false,
        "stacking": {
          "group": "A",
          "mode": "none"
        },
        "thresholdsStyle": {
          "mode": "line+area"
        }
      },
      "mappings": [],
      "thresholds": {
        "mode": "absolute",
        "steps": [
          {
            "color": "green",
            "value": null
          },
          {
            "color": "yellow",
            "value": 300
          },
          {
            "color": "orange",
            "value": 375
          },
          {
            "color": "red",
            "value": 450
          }
        ]
      }
    },
    "overrides": []
  },
  "gridPos": {
    "h": 9,
    "w": 24,
    "x": 0,
    "y": 17
  },
  "id": 9996,
  "options": {
    "legend": {
      "calcs": ["mean", "max"],
      "displayMode": "table",
      "placement": "right",
      "showLegend": true
    },
    "tooltip": {
      "mode": "multi",
      "sort": "none"
    }
  },
  "pluginVersion": "10.4.3",
  "title": "Metrics vs Thresholds",
  "type": "timeseries",
  "targets": [
    {
      "datasource": {
        "type": "prometheus",
        "uid": "prometheus"
      },
      "editorMode": "code",
      "expr": "phoenix_opt_ts_active",
      "legendFormat": "Optimized Pipeline Series",
      "range": true,
      "refId": "A"
    },
    {
      "datasource": {
        "type": "prometheus",
        "uid": "prometheus"
      },
      "editorMode": "code",
      "expr": "phoenix_threshold_moderate",
      "legendFormat": "Moderate Threshold",
      "range": true,
      "refId": "B"
    },
    {
      "datasource": {
        "type": "prometheus",
        "uid": "prometheus"
      },
      "editorMode": "code",
      "expr": "phoenix_threshold_adaptive",
      "legendFormat": "Adaptive Threshold",
      "range": true,
      "refId": "C"
    },
    {
      "datasource": {
        "type": "prometheus",
        "uid": "prometheus"
      },
      "editorMode": "code",
      "expr": "phoenix_threshold_ultra",
      "legendFormat": "Ultra Threshold",
      "range": true,
      "refId": "D"
    }
  ]
} as $threshold_panel |

# Add new panels to the dashboard
.panels += [
  $schema_panel, 
  $health_panel,
  $threshold_row,
  $threshold_panel
] |

# Ensure panels are in a good order by adjusting y positions
.panels |= map(
  if .gridPos and .gridPos.y >= 0 then
    .gridPos.y += 27
  else
    .
  end
)
' "$UNIFIED_DASHBOARD" > "${UNIFIED_DASHBOARD}.new"

if [ -s "${UNIFIED_DASHBOARD}.new" ]; then
    mv "${UNIFIED_DASHBOARD}.new" "$UNIFIED_DASHBOARD"
    echo -e "${GREEN}✅ Enhanced unified dashboard with additional panels${NC}"
else
    echo -e "${RED}❌ Failed to enhance dashboard${NC}"
    rm -f "${UNIFIED_DASHBOARD}.new"
    exit 1
fi

echo
echo -e "${BLUE}🚀 Next steps:${NC}"
echo -e "1. Run ${CYAN}chmod +x ./setup-dashboards.sh${NC} to make the setup script executable"
echo -e "2. Run ${CYAN}./setup-dashboards.sh${NC} to import the enhanced dashboard into Grafana"
echo -e "3. Access Grafana at ${CYAN}http://localhost:3000${NC} to view the enhanced dashboard"
