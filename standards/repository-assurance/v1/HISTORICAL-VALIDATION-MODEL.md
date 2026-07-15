# Historical Validation Model

## Purpose

Accepted phases and steps must remain revalidatable after later development.

## Required method

1. Read the accepted commit and gate from `tools/validation/checkpoints.json`.
2. Clone the canonical remote into a disposable directory.
3. Check out the exact commit.
4. Create the historical branch name required by the original gate.
5. Run the environment doctor applicable to that checkpoint.
6. Run the gate from the historical tree.
7. retain the sanitized result and hashes.
8. Remove the disposable checkout and runtime state.

## Why current-tree execution is insufficient

Historical gates may correctly assert the absence of later files, migrations,
routes, identities, or authority. Running such a gate directly against a later
tree would produce a valid failure.

## Validator errata

Do not silently modify an accepted historical gate.

An erratum records:

- original checkpoint and result;
- validator defect;
- corrected validator;
- impact analysis;
- corrected result;
- acceptance disposition.
