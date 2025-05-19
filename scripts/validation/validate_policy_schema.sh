#!/bin/bash
# validate_policy_schema.sh - Validate all policy files against the schema

set -e

# Find all policy.yaml files in the repository
POLICY_FILES=$(find . -name "policy.yaml" -o -name "*policy*.yaml")

echo "Validating policy files..."
for policy_file in $POLICY_FILES; do
  echo "Checking $policy_file..."
  go run hack/validate_policy.go "$policy_file"
  if [ $? -ne 0 ]; then
    echo "Error: Policy file $policy_file failed validation"
    exit 1
  fi
done

echo "All policy files valid!"
exit 0
