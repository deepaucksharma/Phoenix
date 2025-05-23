#!/bin/bash

# Phoenix-vNext New Relic Integration Script
# This script configures and validates New Relic integration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Phoenix-vNext New Relic Integration Setup${NC}"
echo "=========================================="

# Check for required environment variables
check_env_vars() {
    local missing_vars=()
    
    if [ -z "$NEW_RELIC_LICENSE_KEY" ]; then
        missing_vars+=("NEW_RELIC_LICENSE_KEY")
    fi
    
    if [ -z "$NEW_RELIC_OTLP_ENDPOINT" ]; then
        NEW_RELIC_OTLP_ENDPOINT="https://otlp.nr-data.net:4317"
        echo -e "${YELLOW}Using default New Relic OTLP endpoint: $NEW_RELIC_OTLP_ENDPOINT${NC}"
    fi
    
    if [ ${#missing_vars[@]} -ne 0 ]; then
        echo -e "${RED}Error: Missing required environment variables:${NC}"
        printf '%s\n' "${missing_vars[@]}"
        echo ""
        echo "Please set these variables in your .env file or export them."
        exit 1
    fi
}

# Configure New Relic integration
configure_integration() {
    echo -e "\n${GREEN}Configuring New Relic integration...${NC}"
    
    # Update .env file with New Relic settings
    if [ -f .env ]; then
        # Check if NR settings already exist
        if ! grep -q "NEW_RELIC_LICENSE_KEY" .env; then
            cat >> .env << EOF

# New Relic Integration
NEW_RELIC_LICENSE_KEY=${NEW_RELIC_LICENSE_KEY}
NEW_RELIC_OTLP_ENDPOINT=${NEW_RELIC_OTLP_ENDPOINT}
ENABLE_NR_EXPORT_FULL=true
ENABLE_NR_EXPORT_OPTIMISED=true
ENABLE_NR_EXPORT_EXPERIMENTAL=false
EOF
            echo -e "${GREEN}✓ Added New Relic configuration to .env${NC}"
        else
            echo -e "${YELLOW}New Relic configuration already exists in .env${NC}"
        fi
    fi
    
    # Create New Relic specific dashboard configuration
    mkdir -p configs/monitoring/newrelic
    cat > configs/monitoring/newrelic/dashboard.json << 'EOF'
{
  "name": "Phoenix-vNext Cardinality Optimization",
  "description": "Monitor Phoenix multi-pipeline cardinality optimization system",
  "permissions": "PUBLIC_READ_WRITE",
  "pages": [
    {
      "name": "Pipeline Overview",
      "description": "Overview of all Phoenix pipelines",
      "widgets": [
        {
          "title": "Pipeline Cardinality Comparison",
          "configuration": {
            "queries": [
              {
                "query": "SELECT latest(phoenix_observer_kpi_store_phoenix_pipeline_output_cardinality_estimate) FROM Metric FACET pipeline TIMESERIES"
              }
            ]
          }
        },
        {
          "title": "Cardinality Reduction Efficiency",
          "configuration": {
            "queries": [
              {
                "query": "SELECT latest(phoenix:cardinality_reduction_percentage) FROM Metric TIMESERIES"
              }
            ]
          }
        },
        {
          "title": "Signal Preservation Score",
          "configuration": {
            "queries": [
              {
                "query": "SELECT latest(phoenix:signal_preservation_score) FROM Metric FACET pipeline TIMESERIES"
              }
            ]
          }
        },
        {
          "title": "Resource Efficiency Score",
          "configuration": {
            "queries": [
              {
                "query": "SELECT latest(phoenix:resource_efficiency_score) FROM Metric TIMESERIES"
              }
            ]
          }
        }
      ]
    },
    {
      "name": "Control Loop",
      "description": "Adaptive control system monitoring",
      "widgets": [
        {
          "title": "Optimization Mode Transitions",
          "configuration": {
            "queries": [
              {
                "query": "SELECT latest(phoenix:control_mode_transitions_total) FROM Metric TIMESERIES"
              }
            ]
          }
        },
        {
          "title": "Control Stability Score",
          "configuration": {
            "queries": [
              {
                "query": "SELECT latest(phoenix:control_stability_score) FROM Metric TIMESERIES"
              }
            ]
          }
        },
        {
          "title": "Control Loop Effectiveness",
          "configuration": {
            "queries": [
              {
                "query": "SELECT latest(phoenix:control_loop_effectiveness) FROM Metric TIMESERIES"
              }
            ]
          }
        }
      ]
    },
    {
      "name": "Anomaly Detection",
      "description": "Anomaly detection and alerting",
      "widgets": [
        {
          "title": "Cardinality Anomaly Score",
          "configuration": {
            "queries": [
              {
                "query": "SELECT latest(phoenix:cardinality_zscore) FROM Metric TIMESERIES"
              }
            ]
          }
        },
        {
          "title": "Explosion Risk Score",
          "configuration": {
            "queries": [
              {
                "query": "SELECT latest(phoenix:cardinality_explosion_risk) FROM Metric TIMESERIES"
              }
            ]
          }
        }
      ]
    },
    {
      "name": "Cost Analysis",
      "description": "Cost efficiency metrics",
      "widgets": [
        {
          "title": "Cost per Million Datapoints",
          "configuration": {
            "queries": [
              {
                "query": "SELECT latest(phoenix:cost_per_million_datapoints) FROM Metric FACET pipeline TIMESERIES"
              }
            ]
          }
        },
        {
          "title": "Optimization Savings",
          "configuration": {
            "queries": [
              {
                "query": "SELECT latest(phoenix:optimization_savings_percentage) FROM Metric TIMESERIES"
              }
            ]
          }
        }
      ]
    }
  ]
}
EOF
    echo -e "${GREEN}✓ Created New Relic dashboard configuration${NC}"
}

# Test New Relic connectivity
test_connectivity() {
    echo -e "\n${GREEN}Testing New Relic connectivity...${NC}"
    
    # Create a test metric
    curl -X POST "${NEW_RELIC_OTLP_ENDPOINT/4317/4318}/v1/metrics" \
        -H "Api-Key: ${NEW_RELIC_LICENSE_KEY}" \
        -H "Content-Type: application/json" \
        -d '{
            "resourceMetrics": [{
                "resource": {
                    "attributes": [{
                        "key": "service.name",
                        "value": {"stringValue": "phoenix-connectivity-test"}
                    }]
                },
                "scopeMetrics": [{
                    "metrics": [{
                        "name": "phoenix.test.metric",
                        "gauge": {
                            "dataPoints": [{
                                "asInt": "1",
                                "timeUnixNano": "'$(date +%s%N)'"
                            }]
                        }
                    }]
                }]
            }]
        }' 2>/dev/null
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Successfully connected to New Relic OTLP endpoint${NC}"
    else
        echo -e "${RED}✗ Failed to connect to New Relic OTLP endpoint${NC}"
        echo -e "${YELLOW}Please check your license key and network connectivity${NC}"
        exit 1
    fi
}

# Create New Relic alerts
create_alerts() {
    echo -e "\n${GREEN}Creating New Relic alert conditions...${NC}"
    
    cat > configs/monitoring/newrelic/alerts.json << 'EOF'
{
  "policy_name": "Phoenix Cardinality Optimization",
  "conditions": [
    {
      "name": "Cardinality Explosion",
      "type": "NRQL",
      "nrql": {
        "query": "SELECT latest(phoenix:cardinality_explosion_risk) FROM Metric WHERE phoenix:cardinality_explosion_risk > 5"
      },
      "critical": {
        "operator": "ABOVE",
        "value": 5,
        "duration": 120
      }
    },
    {
      "name": "Low Signal Preservation",
      "type": "NRQL",
      "nrql": {
        "query": "SELECT latest(phoenix:signal_preservation_score) FROM Metric WHERE phoenix:signal_preservation_score < 0.95"
      },
      "critical": {
        "operator": "BELOW",
        "value": 0.95,
        "duration": 300
      }
    },
    {
      "name": "Control Loop Instability",
      "type": "NRQL",
      "nrql": {
        "query": "SELECT latest(phoenix:control_stability_score) FROM Metric WHERE phoenix:control_stability_score < 0.5"
      },
      "warning": {
        "operator": "BELOW",
        "value": 0.5,
        "duration": 600
      }
    },
    {
      "name": "High Resource Usage",
      "type": "NRQL",
      "nrql": {
        "query": "SELECT latest(phoenix:memory_utilization_percentage) FROM Metric WHERE phoenix:memory_utilization_percentage > 90"
      },
      "critical": {
        "operator": "ABOVE",
        "value": 90,
        "duration": 300
      }
    }
  ]
}
EOF
    echo -e "${GREEN}✓ Created New Relic alert conditions${NC}"
}

# Main execution
main() {
    check_env_vars
    configure_integration
    test_connectivity
    create_alerts
    
    echo -e "\n${GREEN}New Relic integration setup complete!${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Restart Phoenix services to apply New Relic configuration:"
    echo "   docker-compose restart otelcol-main"
    echo ""
    echo "2. Import the dashboard configuration to New Relic:"
    echo "   configs/monitoring/newrelic/dashboard.json"
    echo ""
    echo "3. Configure alert policies in New Relic using:"
    echo "   configs/monitoring/newrelic/alerts.json"
    echo ""
    echo "4. Monitor your metrics at:"
    echo "   https://one.newrelic.com"
}

main "$@"
