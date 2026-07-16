# Validation Entrypoints

- `validate_current.sh` — current checkout policy and portable validation
- `validate_portable.sh` / `Validate-Portable.ps1` — portable project checks
- `validate_fresh_clone.sh` — canonical remote completeness
- `validate_checkpoint.sh` — isolated historical checkpoint
- `validate_canonical.sh` — project-specific canonical environment

## Accepted historical checkpoints

The checkpoint registry binds accepted releases to immutable source commits and
their frozen historical gates:

- `isras-v1.0.0` -> `f9655ddbbf04430fc468aab405f2ed880df3e97d`
- `isras-v1.0.1` -> `c379417720faa595fa5cb89a1dfdb2259d6cb95e`

Revalidate an accepted checkpoint from an isolated clone with:

```bash
./tools/validation/validate_checkpoint.sh isras-v1.0.1
```

Historical validation checks out the accepted source on a branch named `dev`
inside an isolated clone so frozen gates retain their original branch
assumptions.

The bootstrap portable validator detects common project types. Replace or extend
it with explicit project checks before formal repository-assurance acceptance.
