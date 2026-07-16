#!/usr/bin/env bash
set -Eeuo pipefail
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[[ -n "$repo_root" ]] || { printf 'FAIL: not in a Git work tree\n' >&2; exit 1; }

venv="${ISRAS_TOOLS_VENV:-$repo_root/.isras-tools-venv}"
python_cmd="${ISRAS_BOOTSTRAP_PYTHON:-python3}"
requirements="$repo_root/tools/requirements.txt"

"$python_cmd" -m venv "$venv"
venv_python="$venv/bin/python"

# Developer bootstrap intentionally uses the interpreter-provided pip and does
# not implicitly upgrade pip. Release evidence must use bootstrap_tools_release.sh.
"$venv_python" -m pip install --disable-pip-version-check -r "$requirements"
"$venv_python" "$repo_root/tools/environment/record_tool_environment.py" \
  --bootstrap-mode developer \
  --requirements "$requirements" \
  --output "$venv/isras-tool-environment.json"

printf 'ISRAS developer tool environment created at %s\n' "$venv"
printf 'Set ISRAS_PYTHON=%s to use it.\n' "$venv_python"
printf 'NOTE: developer bootstrap is not release-assurance evidence.\n'
