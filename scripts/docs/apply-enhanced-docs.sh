#!/bin/bash
# Apply enhanced documentation changes

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "======================================================"
echo "  Applying Enhanced Documentation Updates to Phoenix"
echo "======================================================"

# Create directories if they don't exist
mkdir -p "${PROJECT_ROOT}/docs/guides"
mkdir -p "${PROJECT_ROOT}/docs/images"

# Verify the presence of updated documentation files
enhanced_docs=(
  "${PROJECT_ROOT}/docs/guides/quick-start-guide.md"
  "${PROJECT_ROOT}/docs/guides/build-process.md"
  "${PROJECT_ROOT}/docs/guides/configuration-guide.md"
  "${PROJECT_ROOT}/docs/guides/monitoring-guide.md"
  "${PROJECT_ROOT}/docs/guides/troubleshooting-guide.md"
  "${PROJECT_ROOT}/docs/guides/README.md"
  "${PROJECT_ROOT}/docs/README.md"
  "${PROJECT_ROOT}/docs/images/pid-controller-visualization.svg"
)

for file in "${enhanced_docs[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo "Error: Required file $file not found!"
    exit 1
  fi
done

echo "All required enhanced documentation files are present."
echo ""
echo "Enhanced Documentation Structure:"
echo "- docs/README.md: Updated main documentation index"
echo "- docs/guides/README.md: New guides index"
echo "- docs/guides/quick-start-guide.md: Fast onboarding guide"
echo "- docs/guides/build-process.md: Detailed build system documentation"
echo "- docs/guides/configuration-guide.md: Comprehensive configuration guide"
echo "- docs/guides/monitoring-guide.md: Guide for monitoring adaptive behavior"
echo "- docs/guides/troubleshooting-guide.md: Solutions for common issues"
echo "- docs/images/pid-controller-visualization.svg: New visualization"
echo ""
echo "These documentation updates expand on the previous changes by:"
echo "1. Providing structured guide content for different user needs"
echo "2. Offering detailed step-by-step instructions for common tasks"
echo "3. Including troubleshooting guidance for common issues"
echo "4. Adding visualization examples for monitoring"
echo ""
echo "Next steps:"
echo "1. Review the enhanced documentation"
echo "2. Consider adding more examples and use cases"
echo "3. Add additional visualizations for key concepts"
echo "======================================================"