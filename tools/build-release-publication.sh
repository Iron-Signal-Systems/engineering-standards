#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$repo_root"

mkdir -p .local/bin .local/validation/releases

echo "Building controlled release-publication command..."
go build -trimpath \
  -o .local/bin/isras-release-publication \
  ./cmd/isras-release-publication
chmod 0755 .local/bin/isras-release-publication

echo "Running release-publication tests..."
go test -count=1 \
  ./internal/releasepublication \
  ./cmd/isras-release-publication

echo
printf 'Built: %s\n' "$repo_root/.local/bin/isras-release-publication"
printf 'Read-only publication preflight:\n'
printf '  %s check --version MAJOR.MINOR.PATCH\n' \
  "$repo_root/.local/bin/isras-release-publication"
printf 'Explicit publication after review:\n'
printf '  %s publish --version MAJOR.MINOR.PATCH --confirm\n' \
  "$repo_root/.local/bin/isras-release-publication"
printf 'This command never creates or pushes a Git tag and never moves main.\n'
