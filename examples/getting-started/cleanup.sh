#!/bin/bash
# cleanup.sh - Clean up the SA-OMF getting started example

# Stop running containers
echo "Stopping and removing containers..."
docker-compose down

# Remove created files
echo "Removing temporary files..."
rm -rf prometheus.yml grafana

echo "Cleanup complete!"