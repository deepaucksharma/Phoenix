#!/bin/bash
# codex-chunker.sh - Package git diff context for Codex

set -e

FILES=$(git diff --name-only main...HEAD)

if [ -z "$FILES" ]; then
  echo "No changed files between main and HEAD"
  exit 0
fi

echo "$FILES" | tar --zstd -cf .codex-context.tar.zst -T -

echo "Created .codex-context.tar.zst"

