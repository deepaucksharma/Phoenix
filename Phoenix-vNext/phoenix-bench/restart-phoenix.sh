#!/bin/bash

echo "Restarting Phoenix-vNext with process metrics optimization..."

# Stop services
docker-compose down

# Start services with the new configuration
docker-compose up -d

# Wait for system to start
echo "Waiting 30 seconds for system startup..."
sleep 30

# Run tests
./test-process-metrics.sh

# Output success message
echo -e "\nPhoenix-vNext has been restarted with process metrics optimization enabled!"
echo "Access the Process Metrics Dashboard at: http://localhost:3000"
