#!/usr/bin/env bash
set -Eeuo pipefail
repo_root="$(git rev-parse --show-toplevel)"
python3 "$repo_root/tools/isras/validate_fresh_clone.py" --repo-root "$repo_root"
