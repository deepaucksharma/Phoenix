#!/bin/bash
# check_component_registry.sh - Ensure all components are properly registered

set -e

MAIN_FILE="cmd/sa-omf-otelcol/main.go"

# Find all component directories
PROCESSORS=$(find internal/processor -mindepth 1 -maxdepth 1 -type d -printf "%f\n")
EXTENSIONS=$(find internal/extension -mindepth 1 -maxdepth 1 -type d -printf "%f\n")
CONNECTORS=$(find internal/connector -mindepth 1 -maxdepth 1 -type d -printf "%f\n")

# Check each processor
echo "Checking processor registration..."
for proc in $PROCESSORS; do
  if ! grep -q "\"github.com/yourorg/sa-omf/internal/processor/$proc\"" "$MAIN_FILE"; then
    echo "Error: Processor '$proc' is not imported in $MAIN_FILE"
    exit 1
  fi
  
  if ! grep -q "$proc.NewFactory()" "$MAIN_FILE"; then
    echo "Error: Processor '$proc' factory is not registered in $MAIN_FILE"
    exit 1
  fi
done

# Check each extension
echo "Checking extension registration..."
for ext in $EXTENSIONS; do
  if ! grep -q "\"github.com/yourorg/sa-omf/internal/extension/$ext\"" "$MAIN_FILE"; then
    echo "Error: Extension '$ext' is not imported in $MAIN_FILE"
    exit 1
  fi
  
  if ! grep -q "$ext.NewFactory()" "$MAIN_FILE"; then
    echo "Error: Extension '$ext' factory is not registered in $MAIN_FILE"
    exit 1
  fi
done

# Check each connector
echo "Checking connector registration..."
for conn in $CONNECTORS; do
  if ! grep -q "\"github.com/yourorg/sa-omf/internal/connector/$conn\"" "$MAIN_FILE"; then
    echo "Error: Connector '$conn' is not imported in $MAIN_FILE"
    exit 1
  fi
  
  if ! grep -q "$conn.NewFactory()" "$MAIN_FILE"; then
    echo "Error: Connector '$conn' factory is not registered in $MAIN_FILE"
    exit 1
  fi
done

echo "All components properly registered!"
exit 0
