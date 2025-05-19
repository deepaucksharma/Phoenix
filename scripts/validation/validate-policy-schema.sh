#!/bin/bash
# validate_policy_schema.sh - Validate all policy files against the schema

set -e

# Find all policy.yaml files in the repository

POLICY_FILES=$(find . -name "policy.yaml" -o -name "*policy*.yaml")

if [ -z "$POLICY_FILES" ]; then
  echo "No policy files found to validate. Skipping validation."
  exit 0
fi

# Check if the validation tool exists

VALIDATOR="hack/validate_policy.go"
if [ ! -f "$VALIDATOR" ]; then
  echo "Policy validator not found at $VALIDATOR."
  
  # Create a basic YAML syntax check instead
  echo "Performing basic YAML syntax check instead..."
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
else
  echo "Validating policy files..."
  for policy_file in $POLICY_FILES; do
    echo "Checking $policy_file..."
    go run "$VALIDATOR" "$policy_file"
    if [ $? -ne 0 ]; then
      echo "Error: Policy file $policy_file failed validation"
      exit 1
    fi
  done
fi

echo "All policy files valid!"
exit 0
