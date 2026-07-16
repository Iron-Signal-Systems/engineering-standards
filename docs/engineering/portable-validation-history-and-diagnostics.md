# Portable Validation History and Failure Diagnostics

## Status

This document defines the repository-owned behavior for portable validation in
local worktrees, shallow clones, GitHub pull-request merge refs, and native
operating-system matrix jobs.

## Required Git history

Portable regression tests may validate accepted historical source boundaries.
A checkout containing only the current event commit is therefore insufficient.
Before portable project tests run, the validator shall discover and verify:

- every accepted commit registered in `tools/validation/checkpoints.json`; and
- every `base_commit` declared by an active change-classification record under
  `docs/acceptance/`.

Missing accepted checkpoints should be fetched through their immutable release
tag. Missing classification bases should be fetched by exact commit identity.
The resulting object must resolve as a Git commit before validation continues.

GitHub workflows perform this acquisition while the workflow token is scoped to
`contents: read`. The repository-owned portable entrypoint repeats the preflight
as a verification boundary and may acquire missing public history when ordinary
remote access permits it.

## Isolated repository-tool bootstrap

Portable stages run under Python isolated mode. Because `python -I` removes the
invoked script directory from `sys.path`, tools that import the repository-owned
sibling `common.py` module must not be executed directly.
`tools/isras/invoke_repo_tool.py` is the bounded bootstrap: it adds only
`tools/isras` to the isolated interpreter path, verifies the selected tool stays
inside the repository, and executes it with the original stage arguments. Ambient
`PYTHONPATH` and user-site packages remain excluded.

## Pull-request merge refs

GitHub may run pull-request workflows against a synthetic merge commit. That is
useful compatibility evidence and is not interchangeable with exact candidate
source validation. The portable workflow may test the synthetic merge result,
while the dedicated candidate gate separately tests the exact pushed candidate
commit. Both identities must be printed in their respective evidence.

## Failure output contract

A generalized nonzero result is not sufficient. A failing stage shall print:

- a stable `failure_code`;
- the stage and exact validator path;
- the tested Git commit;
- workflow, job, and runner operating system when available;
- the exact command;
- the exit code; and
- the validator's detailed assertion output, including expected and observed
  values when the validator has them.

Regression tests for this contract shall set an explicit synthetic GitHub runner
context instead of inheriting ambient workflow variables, compare filesystem
paths with platform-native path objects rather than hard-coded separators, and
prove that streamed subprocess handles are closed before the stage returns.

History failures additionally identify the required commit, its purpose, the
fetch ref, whether acquisition was attempted, and sanitized fetch stdout and
stderr. Authentication material must never be printed.

## Stable failure-code families

- `ISRAS-CI-HISTORY-001` — required historical commit unavailable.
- `ISRAS-PORTABLE-HISTORY-001` — portable history stage failed.
- `ISRAS-PORTABLE-ENVIRONMENT-001` — environment profile failed.
- `ISRAS-PORTABLE-POLICY-001` — assurance policy validation failed.
- `ISRAS-PORTABLE-RELEASE-STATE-001` — release-state validation failed.
- `ISRAS-PORTABLE-PROJECT-001` — project checks or unit regressions failed.
- `ISRAS-PORTABLE-RUNNER-001` — the structured runner could not execute.

## Evidence interpretation

Failure because a required Git object was absent is a checkout-completeness
failure, not an operating-system incompatibility. Matrix summaries shall retain
the individual job result, but engineering review shall classify the root cause
from the structured validator output rather than treating repeated symptoms as
independent defects.
