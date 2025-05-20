#!/bin/bash
# standardize-imports.sh - Ensure consistent imports in Go files

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "======================================================"
echo "  Standardizing imports in Go files"
echo "======================================================"

# Install goimports if not already available
if ! command -v goimports &> /dev/null; then
    echo "Installing goimports..."
    go install golang.org/x/tools/cmd/goimports@v0.1.12
fi

# Run goimports on all Go files
echo "Running goimports on Go files..."
export PATH=$PATH:$HOME/go/bin
find "${PROJECT_ROOT}" -name "*.go" -not -path "*/vendor/*" -not -path "*/\.*" | xargs goimports -w

# Additional project-specific standardization
echo "Applying project-specific import standardization..."

# Ensure proper imports for all PID controller files
echo "Standardizing PID controller imports..."
for file in $(find "${PROJECT_ROOT}/internal/control/pid" -name "*.go" 2>/dev/null || echo ""); do
    if [ -f "$file" ]; then
        echo "- ${file}"
        # Ensure imports are logically grouped
        TEMP_FILE=$(mktemp)
        cat "$file" | awk '
        BEGIN { importing=0; std_imports=""; project_imports=""; third_party_imports=""; }
        /^import \(/ { importing=1; print; next; }
        /^\)/ && importing==1 { 
            importing=0; 
            if (std_imports != "") print std_imports;
            if (third_party_imports != "") print third_party_imports; 
            if (project_imports != "") print project_imports;
            print; 
            next; 
        }
        importing==1 && /^\t"github.com\/deepaucksharma\/Phoenix/ { project_imports = project_imports $0 "\n"; next; }
        importing==1 && /^\t"([^\.]+\.)+[^\/]+"/ { std_imports = std_imports $0 "\n"; next; }
        importing==1 { third_party_imports = third_party_imports $0 "\n"; next; }
        { print; }
        ' > "$TEMP_FILE"
        mv "$TEMP_FILE" "$file"
    fi
done

echo ""
echo "======================================================"
echo "  Import Standardization Complete"
echo "======================================================"
echo ""
echo "Go imports have been standardized with the following order:"
echo "  1. Standard library imports"
echo "  2. Third-party dependencies"  
echo "  3. Project imports (github.com/deepaucksharma/Phoenix/...)"