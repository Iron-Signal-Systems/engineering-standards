#!/usr/bin/env bash
set -Eeuo pipefail
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[[ -n "$repo_root" ]] || { printf 'FAIL: not in a Git work tree\n' >&2; exit 1; }
cd "$repo_root"
[[ "$(git branch --show-current)" == "dev" ]] || {
  printf 'FAIL: ISRAS v1 formal candidate gate requires branch dev\n' >&2
  exit 1
}
[[ -z "$(git status --porcelain)" ]] || {
  printf 'FAIL: ISRAS v1 formal candidate gate requires a clean working tree\n' >&2
  exit 1
}
expected_origin="git@github.com:Iron-Signal-Systems/engineering-standards.git"
actual_origin="$(git remote get-url origin)"
[[ "$actual_origin" == "$expected_origin" ]] || {
  printf 'FAIL: canonical origin mismatch: %s\n' "$actual_origin" >&2
  exit 1
}
python_cmd="${ISRAS_PYTHON:-python3}"
"$python_cmd" tools/isras/validate_policy.py --repo-root "$repo_root"
"$python_cmd" tools/isras/verify_source_manifest.py --repo-root "$repo_root"
./tools/validation/validate_portable.sh
./tools/validation/validate_fresh_clone.sh
printf '\nISRAS v1 candidate gate PASSED. This is validation evidence, not formal acceptance.\n'
