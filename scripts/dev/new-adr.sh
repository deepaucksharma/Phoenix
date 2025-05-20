#!/bin/bash
# new-adr.sh - Create a new Architecture Decision Record

set -e

if [ $# -lt 1 ]; then
  echo "Usage: $0 \"Title of ADR\""
  echo "Example: $0 \"Use HyperLogLog for Cardinality Estimation\""
  exit 1
fi

TITLE=$1
DATE=$(date +%Y%m%d)
FILENAME="${DATE}-$(echo $TITLE | tr '[:upper:]' '[:lower:]' | tr ' ' '-').md"
FULLPATH="docs/architecture/adr/$FILENAME"

# Check if file already exists

if [ -f "$FULLPATH" ]; then
  echo "Error: ADR already exists at $FULLPATH"
  exit 1
fi

# Create ADR file

cat > "$FULLPATH" << EOL
# $(echo $TITLE)

Date: $(date +%Y-%m-%d)

## Status

Proposed

## Context

[Describe the context and problem statement]

## Decision

[Describe the decision that was made]

## Consequences

[Describe the consequences of the decision]

## Alternatives Considered

[Describe alternatives that were considered]

## Implementation Notes

[Provide any implementation guidance]

## References

[List any references or related documents]
EOL

echo "ADR created at $FULLPATH"
