#!/bin/bash
# cleanup.sh - Clean up and standardize the Phoenix codebase

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "======================================================"
echo "  Phoenix Project Cleanup"
echo "======================================================"

# Make all script files executable
find "${PROJECT_ROOT}/scripts" -name "*.sh" -exec chmod +x {} \;

echo "1. Removing redundant files..."
# Remove redundant Makefiles
rm -f "${PROJECT_ROOT}/Makefile.new" 
rm -f "${PROJECT_ROOT}/Makefile.streamlined"
rm -f "${PROJECT_ROOT}/Makefile.docker"

# Remove redundant README files
rm -f "${PROJECT_ROOT}/README.md.new"
rm -f "${PROJECT_ROOT}/README.updated.md"

# Remove redundant docker-compose files
rm -f "${PROJECT_ROOT}/docker-compose.enhanced.yml"

# Remove redundant build scripts
rm -f "${PROJECT_ROOT}/build.sh"

# Clean up development guide duplicates
rm -f "${PROJECT_ROOT}/docs/development-guide.streamlined.md"

echo "2. Standardizing Go imports..."
# Run standardize-imports script if it exists
if [ -f "${PROJECT_ROOT}/scripts/cleanup/standardize-imports.sh" ]; then
    "${PROJECT_ROOT}/scripts/cleanup/standardize-imports.sh"
else
    echo "Standardize imports script not found. Skipping..."
fi

echo "3. Standardizing YAML formatting..."
# Run standardize-yaml script if it exists
if [ -f "${PROJECT_ROOT}/scripts/cleanup/standardize-yaml.sh" ]; then
    "${PROJECT_ROOT}/scripts/cleanup/standardize-yaml.sh"
else
    echo "Standardize YAML script not found. Skipping..."
fi

echo "4. Running go mod tidy and vendor..."
cd "${PROJECT_ROOT}"
go mod tidy
go mod vendor

echo "5. Ensuring proper file permissions..."
chmod +x "${PROJECT_ROOT}/scripts/setup/setup-offline-build.sh"

echo "6. Validating configuration files..."
# Run config validation scripts if they exist
if [ -f "${PROJECT_ROOT}/scripts/validation/validate_policy_schema.sh" ]; then
    "${PROJECT_ROOT}/scripts/validation/validate_policy_schema.sh"
else
    echo "Policy validation script not found. Skipping..."
fi

echo ""
echo "======================================================"
echo "  Cleanup Complete"
echo "======================================================"
echo ""
echo "The following actions were performed:"
echo "  - Removed redundant files"
echo "  - Standardized Go imports"
echo "  - Standardized YAML formatting"
echo "  - Updated Go module dependencies"
echo "  - Set proper file permissions"
echo "  - Validated configuration files"
echo ""
echo "For a complete summary of changes, see:"
echo "  docs/cleanup-summary.md"