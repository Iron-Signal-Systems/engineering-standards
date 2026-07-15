#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 1 ]] || { printf 'Usage: %s <checkpoint>\n' "$0" >&2; exit 2; }
repo_root="$(git rev-parse --show-toplevel)"
python3 "$repo_root/tools/isras/validate_checkpoint.py" \
  --repo-root "$repo_root" \
  --checkpoint "$1"
