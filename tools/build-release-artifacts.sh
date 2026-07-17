#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$repo_root"

mkdir -p .local/bin .local/validation/releases .local/releases

echo "Building repository-owned release-artifact producer..."
go build -trimpath \
  -o .local/bin/isras-release-artifacts \
  ./cmd/isras-release-artifacts
chmod 0755 .local/bin/isras-release-artifacts

echo "Running release-artifact producer tests..."
go test -count=1 ./internal/releaseartifactbuild ./cmd/isras-release-artifacts

echo
printf 'Built: %s\n' "$repo_root/.local/bin/isras-release-artifacts"
printf 'This tool produces local release artifacts only. It does not publish them.\n'
