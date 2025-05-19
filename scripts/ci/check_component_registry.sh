#!/bin/bash
# check_component_registry.sh - Ensure all components are properly registered

set -e

MAIN_FILE="cmd/sa-omf-otelcol/main.go"

# Find all component directories in a cross-platform way
# Exclude common utility packages that are not actual processors
find_dirs() {
  local basedir=$1
  local dirs=()
  
  # Use find without -printf for better cross-platform compatibility
  for d in $(find $basedir -mindepth 1 -maxdepth 1 -type d); do
    dirname=$(basename "$d")
    if [[ "$basedir" != "internal/processor" || "$dirname" != "base" ]]; then
      dirs+=("$dirname")
    fi
  done
  
  echo "${dirs[@]}"
}

# Get components

PROCESSORS=($(find_dirs "internal/processor"))
EXTENSIONS=($(find_dirs "internal/extension"))
CONNECTORS=($(find_dirs "internal/connector"))

# Get the module name from go.mod

MODULE_NAME=$(grep "^module " go.mod | awk '{print $2}')
if [ -z "$MODULE_NAME" ]; then
  MODULE_NAME="github.com/deepaucksharma/Phoenix" # Default fallback if not found
fi

echo "Using module name: $MODULE_NAME"

# Check if main file exists

if [ ! -f "$MAIN_FILE" ]; then
  echo "Warning: Main file $MAIN_FILE does not exist or is not accessible. Skipping component checks."
  exit 0
fi

# Check each processor

echo "Checking processor registration..."
for proc in "${PROCESSORS[@]}"; do
  if [ -z "$proc" ]; then
    continue
  fi
  
  if ! grep -q "$MODULE_NAME/internal/processor/$proc" "$MAIN_FILE" && ! grep -q "internal/processor/$proc" "$MAIN_FILE"; then
    echo "Warning: Processor '$proc' might not be imported in $MAIN_FILE"
  fi
  
  if ! grep -q "$proc\.NewFactory" "$MAIN_FILE"; then
    echo "Warning: Processor '$proc' factory might not be registered in $MAIN_FILE"
  fi
done

# Check each extension

echo "Checking extension registration..."
for ext in "${EXTENSIONS[@]}"; do
  if [ -z "$ext" ]; then
    continue
  fi
  
  if ! grep -q "$MODULE_NAME/internal/extension/$ext" "$MAIN_FILE" && ! grep -q "internal/extension/$ext" "$MAIN_FILE"; then
    echo "Warning: Extension '$ext' might not be imported in $MAIN_FILE"
  fi
  
  if ! grep -q "$ext\.NewFactory" "$MAIN_FILE"; then
    echo "Warning: Extension '$ext' factory might not be registered in $MAIN_FILE"
  fi
done

# Check each connector

echo "Checking connector registration..."
for conn in "${CONNECTORS[@]}"; do
  if [ -z "$conn" ]; then
    continue
  fi
  
  if ! grep -q "$MODULE_NAME/internal/connector/$conn" "$MAIN_FILE" && ! grep -q "internal/connector/$conn" "$MAIN_FILE"; then
    echo "Warning: Connector '$conn' might not be imported in $MAIN_FILE"
  fi
  
  if ! grep -q "$conn\.NewFactory" "$MAIN_FILE"; then
    echo "Warning: Connector '$conn' factory might not be registered in $MAIN_FILE"
  fi
done

echo "Component registry check completed!"
exit 0
