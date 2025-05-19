#!/bin/bash
# create-branch.sh - Create a role-specific branch for a task

set -e

if [ $# -lt 2 ]; then
  echo "Usage: $0 <role> <task-id> [description]"
  echo "Example: $0 implementer PID-001 \"Add anti-windup\""
  exit 1
fi

ROLE=$1
TASK_ID=$2
DESC=${3:-$(grep -A 1 "title:" "tasks/$TASK_ID.yaml" | tail -n 1 | sed 's/^title: "\(.*\)"/\1/' | tr '[:upper:]' '[:lower:]' | tr ' ' '-')}

# Validate role
if [ ! -f "agents/$ROLE.yaml" ]; then
  echo "Error: Role '$ROLE' not found in agents/ directory"
  echo "Available roles: $(ls agents/ | sed 's/\.yaml//')"
  exit 1
fi

# Validate task
if [ ! -f "tasks/$TASK_ID.yaml" ]; then
  echo "Error: Task '$TASK_ID' not found in tasks/ directory"
  exit 1
fi

# Construct branch name
BRANCH="lane/$ROLE/$TASK_ID-$DESC"

# Create branch
git checkout -b "$BRANCH" main

# Update task state
sed -i 's/^state: .*/state: in-progress/' "tasks/$TASK_ID.yaml"
sed -i "s/^assigned_to: .*/assigned_to: \"$ROLE\"/" "tasks/$TASK_ID.yaml"

# Commit task state update
git add "tasks/$TASK_ID.yaml"
git commit -m "chore: Start work on $TASK_ID"

echo "Branch '$BRANCH' created and task updated to in-progress"
echo "When ready to create a PR, remember to include:"
echo "ROLE: $ROLE"
echo "TASKS: $TASK_ID"
