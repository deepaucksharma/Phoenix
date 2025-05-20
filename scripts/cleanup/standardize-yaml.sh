#!/bin/bash
# standardize-yaml.sh - Ensure consistent YAML formatting

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "======================================================"
echo "  Standardizing YAML files formatting"
echo "======================================================"

# Check if yamllint is installed
if ! command -v yamllint &> /dev/null; then
    echo "yamllint not found. Please install with:"
    echo "  pip install yamllint"
    echo "or"
    echo "  apt-get install yamllint"
    echo "Skipping YAML standardization..."
    exit 0
fi

# Create yamllint config if it doesn't exist
if [ ! -f "${PROJECT_ROOT}/.yamllint" ]; then
    cat > "${PROJECT_ROOT}/.yamllint" << EOF
---
extends: default

rules:
  line-length: disable
  comments: disable
  comments-indentation: disable
  document-start: disable
  truthy: disable
  indentation:
    spaces: 2
    indent-sequences: true
    check-multi-line-strings: false
EOF
    echo "Created .yamllint configuration file"
fi

# Find and check YAML files
echo "Checking YAML files..."
yaml_files=$(find "${PROJECT_ROOT}" -name "*.yaml" -o -name "*.yml" | grep -v "vendor" | grep -v ".github")

# Check files for formatting issues
issues_found=0
for file in $yaml_files; do
    echo "- Checking ${file}"
    if ! yamllint -c "${PROJECT_ROOT}/.yamllint" "$file"; then
        issues_found=1
    fi
done

if [ $issues_found -eq 1 ]; then
    echo ""
    echo "YAML formatting issues were found. Please fix them manually."
    echo "You can use a YAML formatter or IDE plugin to fix the issues."
else
    echo ""
    echo "All YAML files follow the standard formatting rules."
fi

echo ""
echo "======================================================"
echo "  YAML Standardization Complete"
echo "======================================================"