#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$repo_root"

mkdir -p .local/bin .local/validation/logs

echo "Building repository-owned ISRAS validator..."
go build -trimpath -o .local/bin/isras-validate ./cmd/isras-validate
chmod 0755 .local/bin/isras-validate

echo "Running validator unit tests..."
go test -count=1 ./...

echo
printf 'Built: %s\n' "$repo_root/.local/bin/isras-validate"
printf 'Run:   %s\n' "$repo_root/.local/bin/isras-validate all"
