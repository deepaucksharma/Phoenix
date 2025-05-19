#!/usr/bin/env bash
set -euo pipefail

echo "== Running gosec =="
gosec ./...

echo "== Running govulncheck =="
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
