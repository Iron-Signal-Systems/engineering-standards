# ISRAS v1.0.0 Formal Acceptance Record

## Decision

- **Status:** Accepted
- **Decision date:** 2026-07-15
- **Accepted source commit:** `f9655ddbbf04430fc468aab405f2ed880df3e97d`
- **Standard commit:** `f9655ddbbf04430fc468aab405f2ed880df3e97d`
- **Required source branch:** `dev`
- **Acceptance tag:** `isras-v1.0.0`
- **Validation gate:** `tools/validation/phase-gates/validate_isras_v1_candidate.sh`
- **Environment profile:** `portable`
- **Runner identity:** `IT2544-arch-wsl2`
- **Correctness result:** `PASS`

This record becomes canonical when merged into `dev`. The acceptance tag and
`main` promotion must target the exact accepted source commit shown above, not
the later commit that adds this acceptance record.

## Accepted boundary

ISRAS v1.0.0 accepts the Iron Signal Repository Assurance Standard definition,
scope, native-first execution model, repository governance, clean-clone
validation, native operating-system matrix, historical checkpoint model,
environment declarations, source-manifest verification, secure-development
lifecycle, evidence schema, release governance, recovery requirements, and
language and project assurance profiles contained in the accepted source
commit.

## Validation result

The merged `dev` source commit passed:

- repository policy validation;
- source-manifest verification;
- portable validation;
- fresh-clone and remote-completeness validation;
- the ISRAS v1 formal candidate gate;
- hosted Ubuntu validation;
- hosted macOS validation;
- hosted Windows PowerShell validation;
- integration-tool validation.

## Evidence

- **Evidence record:** `docs/acceptance/evidence/isras-v1.0.0-candidate/acceptance-evidence.json`
- **Evidence SHA-256:** `d00d6cbfd629c629cc5f64a5eb51eb66765f53dc1502e90cb62d47ff2346ac52`
- **Source commit:** `f9655ddbbf04430fc468aab405f2ed880df3e97d`
- **Started:** `2026-07-15T20:09:45+00:00`
- **Finished:** `2026-07-15T20:12:06.389359+00:00`
- **Resource observation:** `NOT_APPLICABLE`
- **Performance budget:** `NOT_APPLICABLE`
- **Security findings:** `NOT_EVALUATED`
- **Operational readiness:** `NOT_EVALUATED`

## Warnings

- Independent human review was not available for the initial ISRAS v1 acceptance.

## Non-claims

- This acceptance does not prove that adopting products are production ready.
- This acceptance does not prove the absence of vulnerabilities.
- This acceptance does not complete product-specific canonical or specialized campaigns.

## Release finalization

After this acceptance record is merged:

1. create the annotated `isras-v1.0.0` tag at `f9655ddbbf04430fc468aab405f2ed880df3e97d`;
2. push and verify the immutable tag;
3. promote the same exact accepted source commit to `main`;
4. protect the acceptance tag from ordinary movement or deletion.

The acceptance-record commit is evidence about the accepted source. It does not
replace or alter the accepted source commit.
