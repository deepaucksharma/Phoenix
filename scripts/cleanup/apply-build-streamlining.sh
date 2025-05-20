#!/bin/bash
# apply-build-streamlining.sh - Apply all build streamlining changes

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "======================================================"
echo "  Applying Build Streamlining Changes to Phoenix"
echo "======================================================"

# Make all script files executable
chmod +x "${PROJECT_ROOT}/scripts/setup/setup-offline-build.sh"
chmod +x "${PROJECT_ROOT}/scripts/setup/bash-completion.sh"
chmod +x "${PROJECT_ROOT}/scripts/dev/create-local-env.sh"

# Move new files to replace old ones
echo "Updating Makefile..."
mv "${PROJECT_ROOT}/Makefile.new" "${PROJECT_ROOT}/Makefile"

echo "Updating README.md..."
mv "${PROJECT_ROOT}/README.md.new" "${PROJECT_ROOT}/README.md"

# Run setup scripts
echo "Running setup-offline-build.sh..."
"${PROJECT_ROOT}/scripts/setup/setup-offline-build.sh"

echo "Running create-local-env.sh..."
"${PROJECT_ROOT}/scripts/dev/create-local-env.sh"

# Create .env.local if it doesn't exist
if [ ! -f "${PROJECT_ROOT}/.env.local" ]; then
  echo "Creating .env.local..."
  touch "${PROJECT_ROOT}/.env.local"
fi

# Source environment
if [ -f "${PROJECT_ROOT}/.env.local" ]; then
  echo "Sourcing .env.local..."
  source "${PROJECT_ROOT}/.env.local"
fi

# Run fast build to verify
echo "Running fast-build to verify changes..."
cd "${PROJECT_ROOT}"
make fast-build

echo ""
echo "======================================================"
echo "  Build Streamlining Changes Applied Successfully"
echo "======================================================"
echo ""
echo "New commands available:"
echo "  make fast-build              - Quick build for development"
echo "  make fast-run                - Quick run for development"
echo "  make verify                  - Run all verification checks"
echo ""
echo "For more information, see:"
echo "  docs/ci-cd.md                - Build and CI/CD documentation"
echo "  docs/project-planning/build-streamline-summary.md - Summary of changes"
echo ""
echo "To complete setup:"
echo "  source scripts/setup/bash-completion.sh  - Add Bash completion"
echo "  source .aliases                          - Add development shortcuts"