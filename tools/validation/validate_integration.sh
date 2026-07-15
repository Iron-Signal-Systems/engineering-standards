#!/usr/bin/env bash
set -Eeuo pipefail
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[[ -n "$repo_root" ]] || { printf 'FAIL: not in a Git work tree\n' >&2; exit 1; }
cd "$repo_root"
python_cmd="${ISRAS_PYTHON:-python3}"
export PYTHONPATH="$repo_root/tests${PYTHONPATH:+:$PYTHONPATH}"
export ISRAS_RUN_INTEGRATION_TESTS=1
"$python_cmd" -m unittest -v \
  test_isras_tools.ISRToolsTests.test_acceptance_evidence_resolves_self_and_records_runner \
  test_isras_tools.ISRToolsTests.test_fresh_clone_and_historical_checkpoint
printf '\nISRAS integration-tool validation PASSED.\n'
