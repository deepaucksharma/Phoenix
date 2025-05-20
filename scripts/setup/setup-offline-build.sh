#!/bin/bash
# setup-offline-build.sh - Prepare the environment for offline builds

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "======================================================"
echo "  Setting up offline build environment for Phoenix"
echo "======================================================"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    echo "Please install Go 1.24 or later: https://golang.org/doc/install"
    exit 1
fi

# Create vendor directory if it doesn't exist
if [ ! -d "${PROJECT_ROOT}/vendor" ]; then
    echo "Creating vendor directory..."
    cd "${PROJECT_ROOT}"
    go mod vendor
    echo "Vendor directory created successfully"
else
    echo "Vendor directory already exists, checking for updates..."
    cd "${PROJECT_ROOT}"
    go mod tidy
    go mod vendor
    echo "Vendor directory updated successfully"
fi

# Install required tools
echo "Installing required development tools..."

# Check and install golangci-lint
if ! command -v golangci-lint &> /dev/null; then
    echo "Installing golangci-lint..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
else
    echo "golangci-lint is already installed"
fi

# Check and install mockgen
if ! command -v mockgen &> /dev/null; then
    echo "Installing mockgen..."
    go install github.com/golang/mock/mockgen@v1.6.0
else
    echo "mockgen is already installed"
fi

# Check and install govulncheck
if ! command -v govulncheck &> /dev/null; then
    echo "Installing govulncheck..."
    go install golang.org/x/vuln/cmd/govulncheck@latest
else
    echo "govulncheck is already installed"
fi

# Generate GitHub hook for commit message formatting
if [ -d "${PROJECT_ROOT}/.git" ]; then
    echo "Setting up Git hooks..."
    if [ ! -f "${PROJECT_ROOT}/.git/hooks/commit-msg" ]; then
        cat > "${PROJECT_ROOT}/.git/hooks/commit-msg" <<EOF
#!/bin/sh
# Enforce commit message format

commit_msg=\$(cat "\$1")
required_pattern="^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\(.+\))?: .{1,50}"

if ! echo "\$commit_msg" | grep -E "\$required_pattern" > /dev/null; then
    echo "ERROR: Invalid commit message format."
    echo "Please use the format: type(scope): description"
    echo "Where type is one of: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert"
    echo "And description is max 50 characters"
    exit 1
fi
EOF
        chmod +x "${PROJECT_ROOT}/.git/hooks/commit-msg"
        echo "Git commit-msg hook installed"
    else
        echo "Git commit-msg hook already exists"
    fi
fi

# Generate .golangci.yml if it doesn't exist
if [ ! -f "${PROJECT_ROOT}/.golangci.yml" ]; then
    echo "Creating default .golangci.yml configuration..."
    cat > "${PROJECT_ROOT}/.golangci.yml" <<EOF
run:
  timeout: 5m
  skip-dirs:
    - vendor
    - test/testdata

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports
    - misspell
    - typecheck

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
EOF
    echo ".golangci.yml created"
else 
    echo ".golangci.yml already exists"
fi

# Create a workspace cache directory for Go
mkdir -p "${PROJECT_ROOT}/.gocache"
echo "export GOCACHE=${PROJECT_ROOT}/.gocache" > "${PROJECT_ROOT}/.env.local"
echo "Local build environment set up with cached Go modules"

echo ""
echo "======================================================"
echo "  Offline build environment setup complete"
echo "======================================================"
echo ""
echo "To use offline build mode, run:"
echo "  make build GO_OFFLINE_ENV=true"
echo ""
echo "Verify the setup with:"
echo "  make verify"