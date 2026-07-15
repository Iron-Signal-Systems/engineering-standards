#!/usr/bin/env bash
set -Eeuo pipefail
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[[ -n "$repo_root" ]] || { printf 'FAIL: not in a Git work tree\n' >&2; exit 1; }
profile="${1:-portable}"
python_cmd="${ISRAS_PYTHON:-python3}"
"$python_cmd" "$repo_root/tools/isras/doctor.py" --repo-root "$repo_root" --profile "$profile"
