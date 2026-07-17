#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$repo_root"

mkdir -p .local/bin .local/validation/releases

echo "Building repository-owned release workflow command..."
go build -trimpath \
  -o .local/bin/isras-release \
  ./cmd/isras-release
chmod 0755 .local/bin/isras-release

echo "Running release-workflow tests..."
go test -count=1 ./internal/releaseworkflow

echo
printf 'Built: %s\n' "$repo_root/.local/bin/isras-release"
printf 'Read-only candidate check:\n'
printf '  %s check\n' "$repo_root/.local/bin/isras-release"
printf 'Signed local tag stage:\n'
printf '  %s tag --confirm\n' "$repo_root/.local/bin/isras-release"
printf 'Remote publication stage:\n'
printf '  %s publish --confirm\n' "$repo_root/.local/bin/isras-release"
