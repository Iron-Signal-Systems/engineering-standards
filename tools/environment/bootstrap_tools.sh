#!/usr/bin/env bash
set -Eeuo pipefail
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[[ -n "$repo_root" ]] || { printf 'FAIL: not in a Git work tree\n' >&2; exit 1; }
venv="${ISRAS_TOOLS_VENV:-$repo_root/.isras-tools-venv}"
python3 -m venv "$venv"
"$venv/bin/python" -m pip install --upgrade pip
"$venv/bin/python" -m pip install -r "$repo_root/tools/requirements.txt"
printf 'ISRAS tool environment created at %s\n' "$venv"
printf 'Set ISRAS_PYTHON=%s to use it.\n' "$venv/bin/python"
