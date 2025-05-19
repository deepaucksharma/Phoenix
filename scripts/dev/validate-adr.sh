#!/bin/bash
# validate-adr.sh - Validate Architecture Decision Records

set -e

# Find all ADR files
ADR_FILES=$(find docs/adr -name "*.md")

# Check each ADR
for adr in $ADR_FILES; do
  echo "Checking $adr..."
  
  # Check file name format (YYYYMMDD-title.md)
  base=$(basename "$adr")
  if ! [[ $base =~ ^[0-9]{8}-.*\.md$ ]]; then
    echo "Error: ADR filename $base does not match format YYYYMMDD-title.md"
    exit 1
  fi
  
  # Check required sections
  for section in "# " "Date: " "## Status" "## Context" "## Decision" "## Consequences"; do
    if ! grep -q "$section" "$adr"; then
      echo "Error: ADR $base is missing required section: $section"
      exit 1
    fi
  done
  
  # Check status
  status=$(grep -A 1 "## Status" "$adr" | tail -n 1 | xargs)
  if [[ "$status" != "Proposed" && "$status" != "Accepted" && "$status" != "Deprecated" && "$status" != "Superseded" ]]; then
    echo "Error: ADR $base has invalid status: $status (must be Proposed, Accepted, Deprecated, or Superseded)"
    exit 1
  fi
done

echo "All ADRs are valid!"
exit 0
