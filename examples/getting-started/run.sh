#!/bin/bash
# run.sh - Start the SA-OMF getting started example

set -e

# Create directories for Grafana and Prometheus
mkdir -p grafana/provisioning/datasources grafana/provisioning/dashboards grafana/dashboards

# Create Prometheus config
cat > prometheus.yml << EOF
global:
  scrape_interval: 10s
  evaluation_interval: 10s

scrape_configs:
  - job_name: 'saomf'
    static_configs:
      - targets: ['host.docker.internal:8889']
EOF

# Create Grafana datasource
cat > grafana/provisioning/datasources/prometheus.yml << EOF
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
EOF

# Create Grafana dashboard provisioning
cat > grafana/provisioning/dashboards/default.yml << EOF
apiVersion: 1

providers:
  - name: 'Default'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    options:
      path: /var/lib/grafana/dashboards
EOF

# Create basic dashboard
cat > grafana/dashboards/sa-omf-overview.json << EOF
{
  "annotations": {
    "list": []
  },
  "editable": true,
  "fiscalYearStartMonth": 0,
  "graphTooltip": 0,
  "links": [],
  "liveNow": false,
  "panels": [
    {
      "datasource": {
        "type": "prometheus",
        "uid": "prometheus"
      },
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "palette-classic"
          },
          "custom": {
            "axisCenteredZero": false,
            "axisColorMode": "text",
            "axisLabel": "",
            "axisPlacement": "auto",
            "barAlignment": 0,
            "drawStyle": "line",
            "fillOpacity": 10,
            "gradientMode": "none",
            "hideFrom": {
              "legend": false,
              "tooltip": false,
              "viz": false
            },
            "lineInterpolation": "linear",
            "lineWidth": 1,
            "pointSize": 5,
            "scaleDistribution": {
              "type": "linear"
            },
            "showPoints": "auto",
            "spanNulls": false,
            "stacking": {
              "group": "A",
              "mode": "none"
            },
            "thresholdsStyle": {
              "mode": "off"
            }
          },
          "mappings": [],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              }
            ]
          },
          "unit": "none"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 0,
        "y": 0
      },
      "id": 1,
      "options": {
        "legend": {
          "calcs": [],
          "displayMode": "list",
          "placement": "bottom",
          "showLegend": true
        },
        "tooltip": {
          "mode": "single",
          "sort": "none"
        }
      },
      "title": "Adaptive TopK Value",
      "type": "timeseries"
    }
  ],
  "refresh": "10s",
  "schemaVersion": 38,
  "style": "dark",
  "tags": [],
  "templating": {
    "list": []
  },
  "time": {
    "from": "now-15m",
    "to": "now"
  },
  "timepicker": {},
  "timezone": "",
  "title": "SA-OMF Overview",
  "version": 0,
  "weekStart": ""
}
EOF

# Start the monitoring stack
echo "Starting monitoring stack..."
docker-compose up -d

# Check for the collector binary
if [ ! -f "../../bin/sa-omf-otelcol" ]; then
    echo "Building collector..."
    (cd ../../ && make build)
fi

# Start the collector
echo "Starting SA-OMF collector..."
../../bin/sa-omf-otelcol --config=config.yaml &
COLLECTOR_PID=$!

echo "Example is running!"
echo "- Collector:   http://localhost:8889 (metrics endpoint)"
echo "- Prometheus:  http://localhost:9090"
echo "- Grafana:     http://localhost:3000 (admin/admin)"
echo ""
echo "Press Ctrl+C to stop"

# Handle graceful shutdown
function cleanup {
    echo "Stopping collector..."
    kill $COLLECTOR_PID
    
    echo "Monitoring stack will continue running."
    echo "Run ./cleanup.sh to stop and clean up the monitoring stack."
}

trap cleanup EXIT

# Wait for process to complete or Ctrl+C
wait $COLLECTOR_PID