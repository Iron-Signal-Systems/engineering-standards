# Validation Entrypoints

- `validate_current.sh` — current checkout policy and portable validation
- `validate_portable.sh` / `Validate-Portable.ps1` — portable project checks
- `validate_fresh_clone.sh` — canonical remote completeness
- `validate_checkpoint.sh` — isolated historical checkpoint
- `validate_canonical.sh` — project-specific canonical environment

## ISRAS v3 development-candidate validation

The development-only assurance-hardening candidate uses:

`tools/validation/phase-gates/validate_isras_v3_candidate.sh`

This gate verifies base ancestry and absence of unstaged drift, the immutable v1 and v2 normative trees, internally consistent repository self-assurance, clean-room deterministic bootstrap controls, Git-index SHA-512 source accounting, machine-readable templates, the actual C5 candidate classification, control-level external-crosswalk coverage, and v3 regression tests.

The v3 gate is not an acceptance or release gate. It must not move `VERSION`,
`main`, or an accepted `isras-*` tag.

## Accepted historical checkpoints

The checkpoint registry binds accepted releases to immutable source commits and
their frozen historical gates:

- `isras-v1.0.0` -> `f9655ddbbf04430fc468aab405f2ed880df3e97d`
- `isras-v1.0.1` -> `c379417720faa595fa5cb89a1dfdb2259d6cb95e`
- `isras-v2.0.0` -> `781246e69f8a9a382c25040f94b62dfe3b25ba89`
- `isras-v2.0.1` -> `d34fad82781a4e8485f8907fbfd34f236fa79ad2`

The current accepted release is `isras-v2.0.1` and uses the frozen gate:

`tools/validation/phase-gates/validate_isras_v2_0_1_release.sh`

Revalidate it from an isolated clone with:

```bash
./tools/validation/validate_checkpoint.sh isras-v2.0.1
```

The v2.0.0 predecessor remains available through:

`tools/validation/phase-gates/validate_isras_v2_release.sh`

Historical validation checks out the accepted source on a branch named `dev`
inside an isolated clone so frozen gates retain their original branch
assumptions. Before the frozen gate runs, the validator executes that accepted
tree's own `tools/environment/bootstrap_tools.sh` or `Bootstrap-Tools.ps1`,
creates an isolated `.isras-tools-venv`, and supplies its exact interpreter as
`ISRAS_PYTHON`. This compatibility bootstrap is predecessor-revalidation
evidence; it is not ISRAS v3 deterministic release-bootstrap evidence.
Checkpoint registration does not move `main` or an accepted release tag.

The bootstrap portable validator detects common project types. Replace or
extend it with explicit project checks before formal repository-assurance
acceptance.

## Portable history preflight and structured diagnostics

Portable validation now discovers every accepted checkpoint commit and active
change-classification base before project regressions run. Shallow CI checkouts
acquire those exact objects and verify that each resolves as a commit. The
portable shell and PowerShell entrypoints use
`tools/isras/run_portable_validation.py`, which invokes repository tools through
the bounded isolated bootstrap `tools/isras/invoke_repo_tool.py` and prints stage,
validator, tested commit, workflow, job, command, exit code, and a stable failure code. See
`docs/engineering/portable-validation-history-and-diagnostics.md`.
