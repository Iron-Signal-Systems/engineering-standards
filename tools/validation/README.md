# Validation Entrypoints

- `validate_current.sh` — current checkout policy and portable validation
- `validate_portable.sh` / `Validate-Portable.ps1` — portable project checks
- `validate_fresh_clone.sh` — canonical remote completeness
- `validate_checkpoint.sh` — isolated historical checkpoint
- `validate_canonical.sh` — project-specific canonical environment

## Current release-source validation

The ISRAS v2.0.1 release source uses:

`tools/validation/phase-gates/validate_isras_v2_0_1_release.sh`

Run this frozen gate only after the exact release-source commit is committed and
pushed to remote `dev`. The gate verifies policy, source manifest, synchronized
release state, the v2.0.1 release-source boundary, portable and fresh-clone
validation, the integration-enabled regression suite, and accepted v1.0.1 and
v2.0.0 historical checkpoints.

Signed-tag verification, non-force `main` promotion, and exact branch/tag
convergence remain separate completion checks.

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
