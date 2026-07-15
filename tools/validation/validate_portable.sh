#!/usr/bin/env bash
set -Eeuo pipefail
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[[ -n "$repo_root" ]] || { printf 'FAIL: not in a Git work tree\n' >&2; exit 1; }
cd "$repo_root"
python_cmd="${ISRAS_PYTHON:-python3}"
"$python_cmd" tools/isras/doctor.py --repo-root "$repo_root" --profile portable
"$python_cmd" tools/isras/validate_policy.py --repo-root "$repo_root"
"$python_cmd" tools/isras/validate_release_state.py --repo-root "$repo_root"
"$python_cmd" tools/isras/portable_project_checks.py --repo-root "$repo_root"
printf '\nPortable validation PASSED.\n'
