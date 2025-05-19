#!/bin/bash
# generate-load.sh - Generate sample load for the SA-OMF getting started example

# Create a bunch of processes with different names to be picked up by the priority_tagger
echo "Starting database-like processes..."
for i in {1..3}; do
    (while true; do echo "database-worker-$i is running..."; sleep 10; done) &
    echo "Started database-worker-$i (PID: $!)"
done

echo "Starting API-like processes..."
for i in {1..5}; do
    (while true; do echo "api-service-$i is running..."; sleep 10; done) &
    echo "Started api-service-$i (PID: $!)"
done

echo "Starting worker processes..."
for i in {1..10}; do
    (while true; do echo "worker-$i is running..."; sleep 10; done) &
    echo "Started worker-$i (PID: $!)"
done

echo "Starting miscellaneous processes..."
for i in {1..20}; do
    (while true; do echo "misc-process-$i is running..."; sleep 10; done) &
    echo "Started misc-process-$i (PID: $!)"
done

echo "Load generation is running."
echo "Press Ctrl+C to stop all generated processes."

# Wait for user to press Ctrl+C
function cleanup {
    echo "Stopping all generated processes..."
    pkill -f "database-worker-"
    pkill -f "api-service-"
    pkill -f "worker-"
    pkill -f "misc-process-"
    echo "All generated processes have been stopped."
}

trap cleanup EXIT

# Wait indefinitely
while true; do
    sleep 1
done