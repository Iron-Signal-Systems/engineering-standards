#!/usr/bin/env bash
set -Eeuo pipefail
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[[ -n "$repo_root" ]] || {
  printf 'FAIL: not in a Git work tree\n' >&2
  printf 'failure_code=ISRAS-PORTABLE-ENTRYPOINT-001\n' >&2
  exit 1
}
python_cmd="${ISRAS_PYTHON:-python3}"
exec "$python_cmd" -I \
  "$repo_root/tools/isras/run_portable_validation.py" \
  --repo-root "$repo_root"
