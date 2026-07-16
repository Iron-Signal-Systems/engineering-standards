#!/usr/bin/env bash
set -Eeuo pipefail
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[[ -n "$repo_root" ]] || { printf 'FAIL: not in a Git work tree\n' >&2; exit 1; }
cd "$repo_root"
python_cmd="${ISRAS_PYTHON:-python3}"

printf '== ISRAS v3 assurance-hardening candidate validation ==\n'
"$python_cmd" tools/isras/validate_isras_v3_candidate.py --repo-root "$repo_root"
"$python_cmd" -m unittest -v tests.test_isras_v3_hardening
printf 'ISRAS v3 assurance-hardening candidate validation PASSED.\n'
