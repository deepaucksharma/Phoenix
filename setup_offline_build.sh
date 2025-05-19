#!/bin/bash

# Setup script for Phoenix project in offline mode

# Check if Go is installed
GO_PATH=$(which go 2>/dev/null)
if [ -z "$GO_PATH" ]; then
  echo "Error: Go is not installed or not in PATH"
  echo ""
  echo "Please install Go using one of these methods:"
  echo "1. Use your system package manager:"
  echo "   sudo apt-get install golang    # Debian/Ubuntu"
  echo "   sudo yum install golang        # CentOS/RHEL"
  echo "   sudo pacman -S go              # Arch Linux"
  echo ""
  echo "2. Download from https://golang.org/dl/ and follow installation instructions"
  echo "   Note: This project requires Go 1.22 or higher"
  echo ""
  echo "After installation, make sure Go is in your PATH and try again."
  exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "Go version $GO_VERSION detected"

# Check if Go version is at least 1.22
GO_MAJOR=$(echo $GO_VERSION | cut -d. -f1)
GO_MINOR=$(echo $GO_VERSION | cut -d. -f2)
if [ "$GO_MAJOR" -lt 1 ] || ([ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -lt 22 ]); then
  echo "Error: Go version 1.22 or higher is required"
  echo "Your current Go version is $GO_VERSION"
  echo "Please upgrade Go and try again"
  exit 1
fi

# Ensure we use vendor directory regardless of GO111MODULE setting
export GO111MODULE=on
export GOPROXY=off  # Force offline mode
export GOSUMDB=off  # Disable checksum verification

echo "Build environment set to use vendored dependencies"
echo ""
echo "To build the project:"
echo "make build"
echo ""
echo "To run tests:"
echo "make test"
echo ""
echo "To run with default config:"
echo "make run"
echo ""
echo "Setup complete. You can now build and run the Phoenix project offline."