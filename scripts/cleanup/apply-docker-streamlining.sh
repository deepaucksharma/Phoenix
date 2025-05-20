#!/bin/bash
# apply-docker-streamlining.sh - Apply Docker streamlining changes

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "======================================================"
echo "  Applying Docker Streamlining Changes to Phoenix"
echo "======================================================"

# Make all script files executable
chmod +x "${PROJECT_ROOT}/build.sh"

# Move new files to replace old ones
echo "Updating Makefile with Docker-enhanced version..."
mv "${PROJECT_ROOT}/Makefile.docker" "${PROJECT_ROOT}/Makefile"

echo "Updating docker-compose.yml with enhanced version..."
mv "${PROJECT_ROOT}/docker-compose.enhanced.yml" "${PROJECT_ROOT}/docker-compose.yml"

# Create a backup of the existing development guide if it exists
if [ -f "${PROJECT_ROOT}/docs/development-guide.md" ]; then
  echo "Backing up existing development guide..."
  cp "${PROJECT_ROOT}/docs/development-guide.md" "${PROJECT_ROOT}/docs/development-guide.md.bak"
fi

echo "Adding universal build script..."
echo "The build.sh script provides a simplified interface for new users"
echo "It supports both local and Docker-based development"

# Check if .air.toml was created
if [ -f "${PROJECT_ROOT}/.air.toml" ]; then
  echo "Hot reload configuration (.air.toml) added for development"
fi

# Create a quick start section in the README
echo "Updating README with quick start section..."
if [ -f "${PROJECT_ROOT}/README.md" ]; then
  # Add quick start section if it doesn't exist
  if ! grep -q "## Quick Start" "${PROJECT_ROOT}/README.md"; then
    TEMP_FILE=$(mktemp)
    cat > "$TEMP_FILE" << 'EOF'

## Quick Start

The fastest way to get started with Phoenix:

```bash
# Build and run with a single command (works on any system with Bash)
./build.sh run

# Use hot reload for rapid development (requires Docker)
./build.sh --hot-reload

# For more options
./build.sh --help
```

For more detailed instructions, see the [Development Guide](docs/development-guide.md).

EOF
    
    # Find the architecture section and add quick start before it
    sed -i '/^## Architecture/i\'"$(cat $TEMP_FILE)" "${PROJECT_ROOT}/README.md"
    rm "$TEMP_FILE"
  fi
fi

echo ""
echo "======================================================"
echo "  Docker Streamlining Changes Applied Successfully"
echo "======================================================"
echo ""
echo "New features available:"
echo "  1. Universal build script: ./build.sh"
echo "  2. Hot reload development: docker-compose up hot-reload"
echo "  3. Enhanced Makefile with Docker targets"
echo "  4. Improved Docker Compose configuration"
echo ""
echo "Getting started:"
echo "  ./build.sh --help       - Show build script options"
echo "  ./build.sh run          - Build and run locally"
echo "  ./build.sh --hot-reload - Start hot reload server"
echo ""
echo "For more information, see:"
echo "  docs/development-guide.md - Comprehensive development guide"
echo "  docs/ci-cd.md             - Build and CI/CD documentation"