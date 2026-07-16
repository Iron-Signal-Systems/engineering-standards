#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
PYTHON="${ISRAS_PYTHON:-python3}"

cd "$ROOT"

fail() {
    printf 'FAIL: %s\n' "$*" >&2
    exit 1
}

[[ "$(git branch --show-current)" == "dev" ]] ||
    fail "candidate gate requires branch dev"

[[ -z "$(git status --porcelain)" ]] ||
    fail "candidate gate requires a clean working tree"

git fetch origin --prune --tags

LOCAL_HEAD="$(git rev-parse HEAD)"
REMOTE_DEV="$(git rev-parse refs/remotes/origin/dev)"

[[ "$LOCAL_HEAD" == "$REMOTE_DEV" ]] ||
    fail "candidate gate requires the exact pushed remote dev commit"

"$PYTHON" tools/isras/verify_source_manifest.py --repo-root .

"$PYTHON" tools/isras/validate_release_state.py --repo-root .

"$PYTHON" tools/isras/validate_isras_v2_0_1_candidate.py \
    --repo-root .

ISRAS_PYTHON="$PYTHON" \
    ./tools/validation/validate_portable.sh

ISRAS_PYTHON="$PYTHON" \
    ./tools/validation/validate_fresh_clone.sh

ISRAS_RUN_INTEGRATION_TESTS=1 \
ISRAS_PYTHON="$PYTHON" \
    "$PYTHON" -m unittest -v \
        tests.test_engineering_standards_compliance \
        tests.test_isras_tools

ISRAS_PYTHON="$PYTHON" \
    "$PYTHON" tools/isras/validate_checkpoint.py \
        --repo-root . \
        --checkpoint isras-v1.0.1

ISRAS_PYTHON="$PYTHON" \
    "$PYTHON" tools/isras/validate_checkpoint.py \
        --repo-root . \
        --checkpoint isras-v2.0.0

printf '\nISRAS v2.0.1 exact pushed candidate gate PASSED.\n'
