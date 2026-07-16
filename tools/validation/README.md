# Validation Entrypoints

- `validate_current.sh` — current checkout policy and portable validation
- `validate_portable.sh` / `Validate-Portable.ps1` — portable project checks
- `validate_fresh_clone.sh` — canonical remote completeness
- `validate_checkpoint.sh` — isolated historical checkpoint
- `validate_canonical.sh` — project-specific canonical environment

## Current candidate validation

The ISRAS v2.0.1 patch candidate uses:

`tools/validation/phase-gates/validate_isras_v2_0_1_candidate.sh`

Run this gate only after the exact candidate commit is pushed to remote `dev`.
It verifies the local and remote commit identity, source manifest, BSD licensing
boundary, current release state, portable and fresh-clone validation, complete
regression tests, and accepted historical checkpoints.

## Accepted historical checkpoints

The checkpoint registry binds accepted releases to immutable source commits and
their frozen historical gates:

- `isras-v1.0.0` -> `f9655ddbbf04430fc468aab405f2ed880df3e97d`
- `isras-v1.0.1` -> `c379417720faa595fa5cb89a1dfdb2259d6cb95e`
- `isras-v2.0.0` -> `781246e69f8a9a382c25040f94b62dfe3b25ba89`

The v2.0.0 checkpoint uses the frozen gate:

`tools/validation/phase-gates/validate_isras_v2_release.sh`

Revalidate the current accepted release from an isolated clone with:

```bash
./tools/validation/validate_checkpoint.sh isras-v2.0.0
```

Historical validation checks out the accepted source on a branch named `dev`
inside an isolated clone so frozen gates retain their original branch
assumptions. Checkpoint registration does not move `main` or an accepted release
tag.

The bootstrap portable validator detects common project types. Replace or extend
it with explicit project checks before formal repository-assurance acceptance.
