#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$repo_root"

mkdir -p .local/bin .local/validation/releases

echo "Building repository-owned clean-clone release validator..."
go build -trimpath \
  -o .local/bin/isras-release-validate \
  ./cmd/isras-release-validate
chmod 0755 .local/bin/isras-release-validate

echo "Running clean-clone release-validator tests..."
go test -count=1 ./internal/releasevalidation

echo
printf 'Built: %s\n' "$repo_root/.local/bin/isras-release-validate"
printf 'Run after pushing the exact commit:\n'
printf '  %s\n' "$repo_root/.local/bin/isras-release-validate"
