# Iron Signal Engineering Standards Glossary

## ISRAS

**Expanded name:** Iron Signal Repository Assurance Standard.

**Definition:** The organization-wide Iron Signal Systems standard governing
repository reproducibility, validation, historical verification, change
control, evidence, acceptance, release, deployment verification, recovery, and
long-term maintainability.

**Not equivalent to:** Information System Risk Assessment.

**Relationship to risk management:** ISRAS may require a project to maintain
threat models, information-system risk assessments, risk registers, findings,
exceptions, and remediation evidence. Those remain separate assurance
artifacts. ISRAS governs how those artifacts and their related implementation
and evidence are versioned, validated, accepted, and maintained; it is not the
method used to calculate or classify information-system risk.

## Acceptance

A formal decision that an exact source commit and its applicable evidence meet
a defined boundary. Acceptance does not imply production readiness unless that
claim is explicitly within scope and supported by separate evidence.

## Canonical repository

The authoritative remote repository named by `REPOSITORY-ASSURANCE.json` from
which accepted source and project-owned assurance inputs must be reconstructable.

## Canonical validation

Validation performed in the exact accepted host, toolchain, service, database,
and operating-system profile required for a project boundary.

## Fresh-clone validation

Validation that obtains the exact pushed commit from the canonical repository
into a clean checkout and proves that no required project-owned input exists
only on a developer system.

## Historical checkpoint

An accepted or candidate source commit, gate, environment profile, and evidence
boundary recorded in `tools/validation/checkpoints.json`.

## Portable validation

Validation intended to run on each approved development platform or on clean
hosted runners without requiring privileged production, canonical, or
specialized infrastructure.

## Specialized validation

A campaign requiring a bounded lab or environment such as a Windows Active
Directory forest, multi-host failover topology, recovery system, or performance
and capacity environment.
