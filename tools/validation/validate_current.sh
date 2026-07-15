#!/usr/bin/env bash
set -Eeuo pipefail
repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"
python3 tools/isras/validate_policy.py --repo-root "$repo_root"
bash tools/validation/validate_portable.sh
