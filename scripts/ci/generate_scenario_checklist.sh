#!/bin/bash
# generate_scenario_checklist.sh - produce checklist from master scenario table

set -euo pipefail

TABLE_FILE="docs/testing/master_scenario_table.md"
TARGET_DOC="docs/testing/validation-framework.md"

if [ ! -f "$TABLE_FILE" ]; then
  echo "Table file $TABLE_FILE not found" >&2
  exit 1
fi

# Build checklist from table
CHECKLIST=$(awk -F'|' 'NR>4 && /^\|/ {
  id=$2; desc=$3;
  gsub(/^ +| +$/, "", id);
  gsub(/^ +| +$/, "", desc);
  if(id!="") printf("- [ ] %s - %s\n", id, desc);
}' "$TABLE_FILE")

# Replace section in target doc
awk -v checklist="$CHECKLIST" '
/<!-- scenario-checklist:start -->/ {
  print;
  print checklist;
  in_section=1;
  next;
}
/<!-- scenario-checklist:end -->/ {
  print;
  in_section=0;
  next;
}
!in_section { print }
' "$TARGET_DOC" > "$TARGET_DOC.tmp"
mv "$TARGET_DOC.tmp" "$TARGET_DOC"

# Also print checklist to stdout
echo "$CHECKLIST"
