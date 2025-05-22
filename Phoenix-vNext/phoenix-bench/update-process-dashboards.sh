#!/bin/bash

# Script to update Grafana dashboards for process metrics integration
echo "Updating Grafana dashboards for process metrics..."

# Update Process Dashboard
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer admin:admin" \
  --data-binary @/Users/deepaksharma/Desktop/src_main/Phoenix-vNext/phoenix-bench/configs/dashboards/phoenix-process-dashboard.json \
  http://localhost:3000/api/dashboards/db

echo -e "\nDashboards updated successfully!"

# Wait for user confirmation
echo -e "\nPress Enter to continue..."
read
