# Existing Repository Adoption Checklist

## Inventory first

- [ ] Identify canonical origin and branches.
- [ ] Identify every local-only test, fixture, validator, setup note, and tool.
- [ ] Identify ignored files currently required for tests.
- [ ] Identify personal paths and machine assumptions.
- [ ] Identify accepted historical commits and gates.
- [ ] Identify sensitive and specialized environments.
- [ ] Identify current secrets and credential sources.

## Record

- [ ] Add assurance manifest.
- [ ] Add governance and SDL documents.
- [ ] Add environment profiles.
- [ ] Add checkpoint registry.
- [ ] Add CODEOWNERS classification.

## Reproduce

- [ ] Add portable entrypoints.
- [ ] Add environment doctor.
- [ ] Add fresh-clone validation.
- [ ] Add historical validator.
- [ ] Run from every approved development system.

## Observe

- [ ] Add non-blocking hosted workflows.
- [ ] Correct missing dependencies and nondeterminism.
- [ ] Confirm no sensitive runner is reachable from public PRs.

## Enforce

- [ ] Require pull requests.
- [ ] Require stable checks.
- [ ] Protect development, release, and accepted tags.
- [ ] Document emergency bypass.

## Accept

- [ ] Record exact adopted standard commit.
- [ ] Record deviations.
- [ ] Record evidence and non-claims.
- [ ] Create the repository-assurance acceptance tag.

## Safe existing-repository application

Preview all conflicts with `--dry-run`, then use `--skip-existing` to write only
missing baseline files. Existing validation, documentation, workflow, and
governance files must be merged deliberately. Do not use `--force` as a generic
upgrade mechanism.
