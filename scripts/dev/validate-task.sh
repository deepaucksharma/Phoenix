#!/bin/bash
# Validates task file format and content

set -e

# Check if a file is provided
if [ -z "$1" ]; then
  echo "Error: No task file provided"
  echo "Usage: $0 <task_file.yaml>"
  exit 1
fi

TASK_FILE="$1"

# Check if file exists
if [ ! -f "$TASK_FILE" ]; then
  echo "Error: File $TASK_FILE does not exist"
  exit 1
fi

# Check if yq is installed
if ! command -v yq &> /dev/null; then
  echo "Warning: yq is not installed, falling back to basic validation"
  
  # Basic validation without yq
  # Check if file is a valid YAML
  if command -v python3 &> /dev/null; then
    python3 -c "import yaml; yaml.safe_load(open('$TASK_FILE'))" || { echo "Error: Invalid YAML format in $TASK_FILE"; exit 1; }
  elif command -v python &> /dev/null; then
    python -c "import yaml; yaml.safe_load(open('$TASK_FILE'))" || { echo "Error: Invalid YAML format in $TASK_FILE"; exit 1; }
  else
    echo "Warning: Cannot validate YAML structure (python not found). Visual inspection required."
  fi

  # Check for required fields using grep
  for field in id title state priority created_at description; do
    if ! grep -q "$field:" "$TASK_FILE"; then
      echo "Error: Required field '$field' is missing in $TASK_FILE"
      exit 1
    fi
  done
else
  # Full validation with yq
  # Check required fields
  for field in id title state priority created_at description; do
    if [ -z "$(yq -r ".$field" "$TASK_FILE" 2>/dev/null)" ]; then
      echo "Error: Required field '$field' is missing or empty in $TASK_FILE"
      exit 1
    fi
  done

  # Validate state field
  STATE=$(yq -r '.state' "$TASK_FILE")
  if [[ ! "$STATE" =~ ^(open|in_progress|review|blocked|done)$ ]]; then
    echo "Error: Invalid state '$STATE' in $TASK_FILE. Must be one of: open, in_progress, review, blocked, done"
    exit 1
  fi

  # Validate priority field
  PRIORITY=$(yq -r '.priority' "$TASK_FILE")
  if [[ ! "$PRIORITY" =~ ^(low|medium|high|critical)$ ]]; then
    echo "Error: Invalid priority '$PRIORITY' in $TASK_FILE. Must be one of: low, medium, high, critical"
    exit 1
  fi

  # Validate date format
  DATE=$(yq -r '.created_at' "$TASK_FILE")
  if ! [[ "$DATE" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
    echo "Error: Invalid date format in created_at field. Use YYYY-MM-DD format."
    exit 1
  fi
fi

echo "Task file $TASK_FILE validated successfully"
exit 0
