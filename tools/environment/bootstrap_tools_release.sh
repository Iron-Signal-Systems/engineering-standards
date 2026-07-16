#!/usr/bin/env bash
set -Eeuo pipefail
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[[ -n "$repo_root" ]] || { printf 'FAIL: not in a Git work tree\n' >&2; exit 1; }
[[ -n "${ISRAS_WHEELHOUSE:-}" ]] || {
  printf 'FAIL: ISRAS_WHEELHOUSE must identify an accepted platform wheelhouse\n' >&2
  exit 1
}

wheelhouse="$(cd "$ISRAS_WHEELHOUSE" && pwd)"
venv="${ISRAS_TOOLS_VENV:-$repo_root/.isras-tools-venv}"
python_cmd="${ISRAS_BOOTSTRAP_PYTHON:-python3}"

[[ ! -e "$venv" ]] || {
  printf 'FAIL: release tool environment path already exists: %s\n' "$venv" >&2
  printf 'Use a new empty path or remove the prior governed tool environment explicitly.\n' >&2
  exit 1
}

"$python_cmd" -I "$repo_root/tools/environment/verify_wheelhouse.py" \
  --repo-root "$repo_root" \
  --wheelhouse "$wheelhouse"

"$python_cmd" -I -m venv "$venv"
venv_python="$venv/bin/python"

pip_install() {
  PIP_CONFIG_FILE=/dev/null PYTHONNOUSERSITE=1 \
    "$venv_python" -I -m pip --isolated install \
      --disable-pip-version-check \
      --no-index \
      --no-cache-dir \
      --only-binary=:all: \
      --find-links "$wheelhouse/wheels" \
      --require-hashes \
      "$@"
}

# Force installation from the accepted wheel even when ensurepip happened to
# provide the same version string.
pip_install --force-reinstall --no-deps -r "$wheelhouse/bootstrap-pip.lock"

PIP_CONFIG_FILE=/dev/null PYTHONNOUSERSITE=1 \
  "$venv_python" -I "$repo_root/tools/environment/clean_tool_venv.py" --keep pip

pip_install -r "$wheelhouse/requirements.lock"

"$venv_python" -I "$repo_root/tools/environment/record_tool_environment.py" \
  --bootstrap-mode release \
  --requirements "$wheelhouse/requirements.lock" \
  --bootstrap-lock "$wheelhouse/bootstrap-lock.json" \
  --wheelhouse-manifest "$wheelhouse/SHA512SUMS" \
  --output "$venv/isras-tool-environment.json"

printf 'ISRAS release tool environment created at %s\n' "$venv"
printf 'Set ISRAS_PYTHON=%s to use it.\n' "$venv_python"
