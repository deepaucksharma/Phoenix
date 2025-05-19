#!/bin/bash
# create-task.sh - Quickly create a new task

set -e

if [ $# -lt 1 ]; then
  echo "Usage: $0 \"Task title\""
  echo "Example: $0 \"Add anti-windup to PID controller\""
  exit 1
fi

TITLE="$*"
ID="PID-$(date +%j%H%M)"

cat > "tasks/$ID.yaml" << EOL
id: $ID
title: "$TITLE"
state: open
priority: medium
created_at: "$(date +%Y-%m-%d)"
assigned_to: ""
area: ""
depends_on: []
acceptance:
  - ""
description: |
  $TITLE
EOL

echo "Created tasks/$ID.yaml"
echo "Don't forget to add additional details:"
echo "- Set the area"
echo "- Add acceptance criteria"
echo "- Expand the description"
