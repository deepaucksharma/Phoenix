#!/bin/bash
# install-hooks.sh - Install Git hooks

set -e

echo "Installing Git hooks..."

# Create hooks directory if it doesn't exist
mkdir -p .git/hooks

# Copy each hook file
for hook in $(ls .git-hooks/); do
  cp ".git-hooks/$hook" ".git/hooks/"
  chmod +x ".git/hooks/$hook"
  echo "Installed $hook"
done

echo "Git hooks installed successfully!"
