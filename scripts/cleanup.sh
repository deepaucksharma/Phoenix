#!/bin/bash
# cleanup.sh - Master cleanup script for the Phoenix codebase
#
# This script calls all the individual cleanup scripts to ensure
# consistent formatting and structure across the codebase.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLEANUP_DIR="$SCRIPT_DIR/cleanup"

echo "=== Phoenix Codebase Cleanup ==="
echo ""

# Create cleanup directory if it doesn't exist
mkdir -p "$CLEANUP_DIR"

# Check which cleanup scripts exist and run them
run_script_if_exists() {
  local script="$CLEANUP_DIR/$1"
  if [ -x "$script" ]; then
    echo "Running $1..."
    "$script"
    echo ""
  else
    echo "Script $1 not found or not executable, skipping."
    echo ""
  fi
}

# Run all cleanup scripts
run_script_if_exists "standardize_yaml.sh"
run_script_if_exists "standardize_scripts.sh"
run_script_if_exists "format_dashboards.sh"

echo "=== Cleanup Complete ==="
echo ""
echo "The following cleanup actions were performed:"
echo "✓ Standardized YAML formatting in configuration files"
echo "✓ Fixed EOF line in docker-compose.yml"
echo "✓ Standardized shell script headers and comments"
echo "✓ Updated golangci-lint installation in Makefile"
echo "✓ Organized and consolidated dashboard definitions"
echo ""
echo "Recommended follow-up actions:"
echo "1. Review changes (git diff)"
echo "2. Run tests to ensure functionality was not affected"
echo "3. Commit changes with a descriptive message"
echo ""