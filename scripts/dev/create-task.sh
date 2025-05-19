#!/bin/bash
# create-task.sh - Quickly create a new task

set -e

function show_usage() {
  echo "Usage: $0 \"Task title\" [options]"
  echo "Example: $0 \"Add anti-windup to PID controller\" --role=implementer --priority=high"
  echo ""
  echo "Options:"
  echo "  --role=ROLE          Assign to a specific role (architect, implementer, integrator, tester, etc.)"
  echo "  --priority=PRIORITY  Set priority (high, medium, low)"
  echo "  --area=AREA          Set the component area"
  echo "  --prefix=PREFIX      Set the task ID prefix (default: PID)"
  echo "  --sequential         Use sequential task ID instead of date-based"
  exit 1
}

# Default values

ROLE=""
PRIORITY="medium"
AREA=""
PREFIX="PID"
SEQUENTIAL=false

# Parse command line arguments

if [ $# -lt 1 ]; then
  show_usage
fi

TITLE="$1"
shift

# Process options

for arg in "$@"; do
  case $arg in
    --role=*)
      ROLE="${arg#*=}"
      VALID_ROLES=("architect" "implementer" "integrator" "tester" "devops" "doc-writer" "reviewer" "security-auditor" "planner")
      VALID=false
      for valid_role in "${VALID_ROLES[@]}"; do
        if [ "$ROLE" = "$valid_role" ]; then
          VALID=true
          break
        fi
      done
      if [ "$VALID" = false ]; then
        echo "Error: Invalid role '$ROLE'"
        echo "Valid roles: ${VALID_ROLES[*]}"
        exit 1
      fi
      ;;
    --priority=*)
      PRIORITY="${arg#*=}"
      if [[ ! "$PRIORITY" =~ ^(high|medium|low)$ ]]; then
        echo "Error: Priority must be 'high', 'medium', or 'low'"
        exit 1
      fi
      ;;
    --area=*)
      AREA="${arg#*=}"
      ;;
    --prefix=*)
      PREFIX="${arg#*=}"
      ;;
    --sequential)
      SEQUENTIAL=true
      ;;
    --help)
      show_usage
      ;;
    *)
      echo "Unknown option: $arg"
      show_usage
      ;;
  esac
done

# Generate task ID

if [ "$SEQUENTIAL" = true ]; then
  # Find the highest task number for the given prefix
  HIGHEST_NUM=0
  for task_file in tasks/*.yaml; do
    if [ -f "$task_file" ]; then
      FILENAME=$(basename "$task_file" .yaml)
      if [[ "$FILENAME" =~ ^${PREFIX}-([0-9]{3})$ ]]; then
        NUM=${BASH_REMATCH[1]}
        if [ "$NUM" -gt "$HIGHEST_NUM" ]; then
          HIGHEST_NUM=$NUM
        fi
      fi
    fi
  done
  NEXT_NUM=$((HIGHEST_NUM + 1))
  ID=$(printf "${PREFIX}-%03d" $NEXT_NUM)
else
  # Use date-based ID
  ID="${PREFIX}-$(date +%j%H%M)"
fi

# Ensure tasks directory exists

mkdir -p tasks

# Create the task file

cat > "tasks/$ID.yaml" << EOL
id: $ID
title: "$TITLE"
state: open
priority: $PRIORITY
created_at: "$(date +%Y-%m-%d)"
assigned_to: "$ROLE"
area: "$AREA"
depends_on: []
acceptance:
  - ""
description: |
  $TITLE
  
  <!-- Add a detailed description of what needs to be done -->
EOL

echo "Created tasks/$ID.yaml"

# If any important fields are missing, remind the user

MISSING=""
if [ -z "$ROLE" ]; then MISSING="$MISSING\n- Set the assigned_to field"; fi
if [ -z "$AREA" ]; then MISSING="$MISSING\n- Set the area field"; fi

if [ -n "$MISSING" ]; then
  echo -e "Don't forget to add additional details:$MISSING"
  echo "- Add acceptance criteria"
  echo "- Expand the description"
fi

# Open the file in an editor if available

if [ -n "$EDITOR" ]; then
  $EDITOR "tasks/$ID.yaml"
elif command -v nano >/dev/null 2>&1; then
  nano "tasks/$ID.yaml"
elif command -v vim >/dev/null 2>&1; then
  vim "tasks/$ID.yaml"
fi
