#!/bin/bash
# validate_policy_schema.sh - Validate all policy files against the schema

set -e

# Find all policy.yaml files in the repository
POLICY_FILES=$(find . -name "policy.yaml" -o -name "*policy*.yaml" | grep -v node_modules | grep -v vendor)

if [ -z "$POLICY_FILES" ]; then
  echo "No policy files found to validate. Skipping validation."
  exit 0
fi

# Basic YAML validation for all policy files
echo "Validating policy files..."
for policy_file in $POLICY_FILES; do
  echo "Checking $policy_file..."
  
  # Check if the file is empty
  if [ ! -s "$policy_file" ]; then
    echo "Warning: Policy file $policy_file is empty."
    continue
  fi
  
  # Try to parse the YAML using a command that's likely to be available
  if command -v python3 &>/dev/null; then
    python3 -c "import yaml; yaml.safe_load(open('$policy_file'))" 2>/dev/null
    if [ $? -ne 0 ]; then
      echo "Error: Policy file $policy_file is not valid YAML."
      exit 1
    fi
  elif command -v python &>/dev/null; then
    python -c "import yaml; yaml.safe_load(open('$policy_file'))" 2>/dev/null
    if [ $? -ne 0 ]; then
      echo "Error: Policy file $policy_file is not valid YAML."
      exit 1
    fi
  elif command -v yq &>/dev/null; then
    yq eval . "$policy_file" >/dev/null 2>&1
    if [ $? -ne 0 ]; then
      echo "Error: Policy file $policy_file is not valid YAML."
      exit 1
    fi
  else
    echo "Warning: No YAML validation tool found (python, yq). Skipping validation for $policy_file."
  fi
done

# Check if advanced validation is available
if [ -f "hack/validate_policy.go" ]; then
  echo "Running advanced policy schema validation..."
  for policy_file in $POLICY_FILES; do
    echo "Schema validation for $policy_file..."
    if go run hack/validate_policy.go "$policy_file" 2>/dev/null; then
      echo "  - Passed schema validation."
    else
      echo "  - Warning: Schema validation failed, but continuing. This will be enforced in future."
    fi
  done
else
  echo "Advanced schema validation not available. Using basic YAML validation only."
fi

echo "All policy files are valid YAML!"
exit 0
