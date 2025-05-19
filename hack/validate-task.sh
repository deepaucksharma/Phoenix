#!/bin/bash
# validate-task.sh - Validate a task specification

set -e

if [ $# -ne 1 ]; then
  echo "Usage: $0 <task-file>"
  echo "Example: $0 tasks/PID-001.yaml"
  exit 1
fi

TASK_FILE=$1

# Check if file exists
if [ ! -f "$TASK_FILE" ]; then
  echo "Error: Task file not found at $TASK_FILE"
  exit 1
fi

# Check required fields
for field in id title state priority created_at assigned_to area description; do
  if ! grep -q "^$field:" "$TASK_FILE"; then
    echo "Error: Missing required field '$field' in $TASK_FILE"
    exit 1
  fi
done

# Validate state
STATE=$(grep "^state:" "$TASK_FILE" | cut -d ' ' -f 2-)
if [[ "$STATE" != "open" && "$STATE" != "in-progress" && "$STATE" != "review" && "$STATE" != "done" ]]; then
  echo "Error: Invalid state '$STATE' - must be one of: open, in-progress, review, done"
  exit 1
fi

# Validate priority
PRIORITY=$(grep "^priority:" "$TASK_FILE" | cut -d ' ' -f 2-)
if [[ "$PRIORITY" != "high" && "$PRIORITY" != "medium" && "$PRIORITY" != "low" ]]; then
  echo "Error: Invalid priority '$PRIORITY' - must be one of: high, medium, low"
  exit 1
fi

# Validate date format
DATE=$(grep "^created_at:" "$TASK_FILE" | cut -d '"' -f 2)
if ! [[ $DATE =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
  echo "Error: Invalid date format '$DATE' - must be YYYY-MM-DD"
  exit 1
fi

# Check acceptance criteria
if ! grep -q "^acceptance:" "$TASK_FILE"; then
  echo "Warning: No acceptance criteria defined in $TASK_FILE"
fi

echo "Task $TASK_FILE is valid."
exit 0
