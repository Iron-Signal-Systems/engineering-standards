# Secure Development Lifecycle

ISRAS implements repository-level controls within a broader secure development
lifecycle.

## Lifecycle

1. Governance and ownership.
2. Requirements, including security, privacy, accessibility, resilience, and
   performance.
3. Threat modeling and abuse-case analysis.
4. Architecture and trust-boundary definition.
5. Bounded implementation planning with prohibited work.
6. Source, tests, fixtures, and synchronized documentation.
7. Portable pull-request verification.
8. Canonical and specialized environment validation.
9. Formal acceptance and frozen checkpoint.
10. Release build, SBOM, hashes, provenance, and signing or attestation.
11. Deployment verification, migration validation, and rollback.
12. Monitoring, capacity, backup, restore, continuity, and incident response.
13. Vulnerability intake, remediation, disclosure, revocation, and trusted
    rebuild.
14. End-of-life, archival, migration, and key retirement.

## Required traceability

Material changes should link:

```text
requirement
  ↕
architecture or decision
  ↕
implementation
  ↕
test and hostile validation
  ↕
acceptance evidence
  ↕
release and deployment identity
```

## NIST SSDF alignment

ISRAS is designed to integrate secure practices into each repository's existing
development lifecycle. It does not replace project-specific engineering,
security, privacy, accessibility, or operational standards.
