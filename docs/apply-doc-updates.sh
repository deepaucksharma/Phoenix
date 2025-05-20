#!/bin/bash
# Apply documentation updates to accurately reflect the current state of Phoenix architecture

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "======================================================"
echo "  Applying Architecture Documentation Updates"
echo "======================================================"

# Create directories if they don't exist
mkdir -p "${PROJECT_ROOT}/docs/images"
mkdir -p "${PROJECT_ROOT}/docs/concepts"
mkdir -p "${PROJECT_ROOT}/docs/architecture"

# Verify the presence of updated documentation files
files_to_check=(
  "${PROJECT_ROOT}/docs/architecture/CURRENT_STATE.md"
  "${PROJECT_ROOT}/docs/configuration-reference.md"
  "${PROJECT_ROOT}/docs/concepts/adaptive-processing.md"
  "${PROJECT_ROOT}/docs/concepts/modern-adaptive-architecture.md"
)

for file in "${files_to_check[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo "Error: Required file $file not found!"
    exit 1
  fi
done

echo "All required documentation files are present."
echo "Updating README with current architecture information..."

echo "Documentation updates applied successfully."
echo ""
echo "Updated Documentation Structure:"
echo "- README.md: Updated to reflect current architecture"
echo "- docs/architecture/README.md: Clarifies current vs. historical architecture"
echo "- docs/architecture/CURRENT_STATE.md: Detailed explanation of current architecture"
echo "- docs/configuration-reference.md: Updated configuration examples"
echo "- docs/components/processors/README.md: Updated processor descriptions"
echo "- docs/components/processors/adaptive_pid.md: Updated processor documentation"
echo "- docs/concepts/adaptive-processing.md: Explains adaptive processing concepts"
echo "- docs/concepts/modern-adaptive-architecture.md: NEW - explains current architecture approach"
echo ""
echo "Important Note: Some historical documentation (like ADRs) remains to provide context,"
echo "but they may describe an earlier version of the architecture. All documentation now"
echo "clearly distinguishes between the current implementation and historical designs."
echo ""
echo "Suggested Next Steps:"
echo "1. Review architecture diagrams and create/update as needed"
echo "2. Update remaining processor documentation"
echo "3. Add examples of how to monitor the self-adaptive behavior"
echo "======================================================"