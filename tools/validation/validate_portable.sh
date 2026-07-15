#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[[ -n "$repo_root" ]] || { printf 'FAIL: not in a Git work tree\n' >&2; exit 1; }
cd "$repo_root"

python3 tools/isras/validate_policy.py --repo-root "$repo_root"
python3 tools/isras/portable_project_checks.py --repo-root "$repo_root"
python3 -m unittest discover -s tests -p 'test_*.py'

printf '\nPortable validation PASSED.\n'
