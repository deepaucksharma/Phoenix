#!/bin/bash
# create-local-env.sh - Create a local development environment file

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Create .env.local file for development settings
cat > "${PROJECT_ROOT}/.env.local" << EOF
# Local Development Environment Variables
# This file is not checked into git (.gitignore)

# Build settings
export GOCACHE="${PROJECT_ROOT}/.gocache"
export GOEXPERIMENT=loopvar
export GOPRIVATE=github.com/deepaucksharma/Phoenix

# Optional: Uncomment to use specific GOPROXY
# export GOPROXY=direct

# Development helpers
# Uncomment to add ./bin to PATH for easy execution
# export PATH="\${PATH}:${PROJECT_ROOT}/bin"

# Docker settings
# export DOCKER_BUILDKIT=1
EOF

echo "Created .env.local with development settings"
echo "Add your custom environment variables to .env.local"
echo "This file is ignored by git and will not be committed"

# Create convenience aliases for development
ALIASES_FILE="${PROJECT_ROOT}/.aliases"
cat > "${ALIASES_FILE}" << EOF
# Development aliases for Phoenix project
# Source this file in your shell with:
# source .aliases

alias pnx-build='make fast-build'
alias pnx-run='make fast-run'
alias pnx-test='make test-unit'
alias pnx-verify='make verify'
alias pnx-docker='make docker && make docker-run'
EOF

echo "Created .aliases with useful development shortcuts"
echo "Source it with: source .aliases"

# Create a simple development launcher script
LAUNCHER="${PROJECT_ROOT}/scripts/dev/launch-dev.sh"
cat > "${LAUNCHER}" << EOF
#!/bin/bash
# launch-dev.sh - Quick development launcher

set -e

# Source environment variables
if [ -f ".env.local" ]; then
  source .env.local
fi

# Build and run in development mode
make fast-build && make fast-run

EOF
chmod +x "${LAUNCHER}"

echo "Created launch-dev.sh for quick development startup"
echo "Run with: scripts/dev/launch-dev.sh"

# Done
echo "Local development environment created successfully"