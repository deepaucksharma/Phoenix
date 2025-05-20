#!/bin/bash
# apply-streamlined-build.sh - Apply streamlined build process changes

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "======================================================"
echo "  Applying Streamlined Build Changes to Phoenix"
echo "======================================================"

# Update Makefile
echo "Updating Makefile with streamlined version..."
mv "${PROJECT_ROOT}/Makefile.streamlined" "${PROJECT_ROOT}/Makefile"

# Update development guide
echo "Updating development guide..."
mv "${PROJECT_ROOT}/docs/development-guide.streamlined.md" "${PROJECT_ROOT}/docs/development-guide.md"

# Remove build.sh
echo "Removing build.sh in favor of make interface..."
if [ -f "${PROJECT_ROOT}/build.sh" ]; then
  rm "${PROJECT_ROOT}/build.sh"
fi

# Check for offline-build.sh
echo "Checking offline-build.sh redirection script..."
if [ -f "${PROJECT_ROOT}/setup_offline_build.sh" ]; then
  # Make sure the target script exists
  mkdir -p "${PROJECT_ROOT}/scripts/setup"
  if [ ! -f "${PROJECT_ROOT}/scripts/setup/setup-offline-build.sh" ]; then
    echo "Warning: Target script scripts/setup/setup-offline-build.sh doesn't exist."
    echo "Creating a minimal version..."
    
    cat > "${PROJECT_ROOT}/scripts/setup/setup-offline-build.sh" << 'EOF'
#!/bin/bash
# setup-offline-build.sh - Set up offline build environment

set -e

echo "Setting up offline build environment..."

# Create vendor directory if it doesn't exist
if [ ! -d "vendor" ]; then
    echo "Creating vendor directory..."
    go mod vendor
fi

# Install required development tools
if ! command -v golangci-lint &> /dev/null; then
    echo "Installing golangci-lint..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
fi

if ! command -v mockgen &> /dev/null; then
    echo "Installing mockgen..."
    go install github.com/golang/mock/mockgen@latest
fi

echo "Offline build environment set up successfully"
EOF
    chmod +x "${PROJECT_ROOT}/scripts/setup/setup-offline-build.sh"
  fi
  
  echo "Using redirection script for backward compatibility..."
fi

# Create README section on make-based workflow
echo "Updating README with make-based workflow info..."
if [ -f "${PROJECT_ROOT}/README.md" ]; then
  # Add make-based workflow section if it doesn't exist
  if ! grep -q "## Development Workflow" "${PROJECT_ROOT}/README.md"; then
    TEMP_FILE=$(mktemp)
    cat > "$TEMP_FILE" << 'EOF'

## Development Workflow

The Phoenix project uses `make` as its primary development interface:

```bash
# Build the project
make fast-build

# Run with development config
make fast-run

# Run unit tests
make test-unit

# Start hot reload development server (requires Docker)
make hot-reload

# For help with all available commands
make help
```

For more detailed instructions, see the [Development Guide](docs/development-guide.md).

EOF
    
    # Find the architecture section and add workflow before it
    sed -i '/^## Architecture/i\'"$(cat $TEMP_FILE)" "${PROJECT_ROOT}/README.md"
    rm "$TEMP_FILE"
  fi
fi

echo ""
echo "======================================================"
echo "  Streamlined Build Changes Applied Successfully"
echo "======================================================"
echo ""
echo "The build process has been streamlined to focus on make:"
echo "  1. Removed build.sh in favor of make commands"
echo "  2. Removed multi-arch docker in favor of simpler targets"
echo "  3. Updated documentation to focus on make-based workflow"
echo ""
echo "Getting started:"
echo "  make help                - Show all available commands"
echo "  make fast-build          - Build quickly for development"
echo "  make fast-run            - Run with development config"
echo "  make hot-reload          - Start hot reload server"
echo ""
echo "For more information, see:"
echo "  docs/development-guide.md - Comprehensive development guide"
echo "  docs/ci-cd.md             - Build and CI/CD documentation"