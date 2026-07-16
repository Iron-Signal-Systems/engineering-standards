#!/usr/bin/env bash
set -Eeuo pipefail

fail() {
    printf 'FAIL: %s\n' "$*" >&2
    return 1
}

canonical_slug() {
    local value="$1"
    value="${value#git@github.com:}"
    value="${value#ssh://git@github.com/}"
    value="${value#https://github.com/}"
    value="${value#http://github.com/}"
    value="${value%.git}"
    printf '%s\n' "$value"
}

main() {
    local repo_root
    local python_cmd
    local actual_origin

    repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
    [[ -n "$repo_root" ]] || fail "not in a Git work tree"
    cd "$repo_root"

    [[ "$(git branch --show-current)" == "dev" ]] ||
        fail "ISRAS v2.0.1 release gate requires branch dev"

    [[ -z "$(git status --porcelain)" ]] ||
        fail "ISRAS v2.0.1 release gate requires a clean working tree"

    actual_origin="$(git remote get-url origin)"
    [[ "$(canonical_slug "$actual_origin")" == \
        "Iron-Signal-Systems/engineering-standards" ]] ||
        fail "canonical origin mismatch: ${actual_origin}"

    [[ "$(tr -d '[:space:]' < VERSION)" == "2.0.1" ]] ||
        fail "ISRAS v2.0.1 release gate requires VERSION 2.0.1"

    python_cmd="${ISRAS_PYTHON:-python3}"

    "$python_cmd" tools/isras/validate_policy.py \
        --repo-root "$repo_root"
    "$python_cmd" tools/isras/verify_source_manifest.py \
        --repo-root "$repo_root"
    "$python_cmd" tools/isras/validate_release_state.py \
        --repo-root "$repo_root"
    "$python_cmd" tools/isras/validate_isras_v2_0_1_release.py \
        --repo-root "$repo_root"

    ISRAS_PYTHON="$python_cmd" \
        ./tools/validation/validate_portable.sh

    ISRAS_PYTHON="$python_cmd" \
        ./tools/validation/validate_fresh_clone.sh

    ISRAS_RUN_INTEGRATION_TESTS=1 \
    ISRAS_PYTHON="$python_cmd" \
        "$python_cmd" -m unittest -v \
            tests.test_engineering_standards_compliance \
            tests.test_isras_tools

    ISRAS_PYTHON="$python_cmd" \
        "$python_cmd" tools/isras/validate_checkpoint.py \
            --repo-root "$repo_root" \
            --checkpoint isras-v1.0.1

    ISRAS_PYTHON="$python_cmd" \
        "$python_cmd" tools/isras/validate_checkpoint.py \
            --repo-root "$repo_root" \
            --checkpoint isras-v2.0.0

    printf '\nISRAS v2.0.1 release-source gate PASSED.\n'
    printf 'Signed tag and branch convergence remain separate completion checks.\n'
}

main "$@"
