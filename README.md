# Iron Signal Engineering Standards

> **Built on purpose. Backed by discipline. Engineered to endure.**

This repository defines the common engineering, secure-development, repository
assurance, validation, acceptance, release, deployment, and operational evidence
standards for Iron Signal Systems projects.

The current normative standard is the **Iron Signal Repository Assurance
Standard (ISRAS) v2**. The accepted ISRAS v1 normative tree remains retained and
immutable for historical verification and repositories that are still
deliberately pinned to an accepted v1 release.

## What is ISRAS?

**ISRAS** stands for the **Iron Signal Repository Assurance Standard**. It is
the organization-wide Iron Signal Systems standard governing repository
reproducibility, validation, historical verification, change control, evidence,
acceptance, release, deployment verification, recovery, long-term
maintainability, engineering-standard inheritance, phase compliance, and
bounded authority.

ISRAS does **not** stand for “Information System Risk Assessment” and is not
itself a risk-assessment methodology. Projects may be required to maintain
information-system risk assessments, threat models, risk registers, findings,
and remediation evidence, but those remain separate assurance artifacts. ISRAS
governs how those artifacts and their related implementation and evidence are
versioned, validated, accepted, and maintained.

## Current status

The repository's declared source version is recorded in `VERSION`. A version
number alone is not acceptance evidence.

The latest accepted release is the exact source commit jointly identified by:

- a verified SSH-signed annotated `isras-v*` tag;
- the remote `main` branch;
- the applicable evidence digest and acceptance-decision metadata.

`dev` may advance with candidate work after a release. At release finalization,
however, `dev`, `main`, and the signed release tag must all identify the same
exact source commit.

The signed annotated tag is the authoritative acceptance-decision object. No
post-acceptance source commit is required merely to record the decision.

## Validation tool bootstrap

The policy validators use exactly pinned Python tools declared in
`tools/requirements.txt`. Install them into a recreatable local virtual
environment without requiring Docker or Podman:

```bash
chmod +x tools/environment/bootstrap_tools.sh
./tools/environment/bootstrap_tools.sh
export ISRAS_PYTHON="$PWD/.isras-tools-venv/bin/python"
```

PowerShell users can run `tools/environment/Bootstrap-Tools.ps1` and set
`ISRAS_PYTHON` to the generated virtual-environment Python executable.

## Governing rule

A change is not complete merely because it works on the developer's current
system. It is complete only when its exact pushed commit can be reconstructed,
validated, and evidenced from the canonical repository using declared
environments and committed project-owned assets.

ISRAS v2 additionally requires exact standards inheritance, non-weakening
governance, Engineering Standards Impact Assessments where applicable, phase
entry and exit reviews, maturity-accurate evidence, hostile authority testing,
and prevention of unrestricted execution contexts.

## Native-first portability

ISRAS does not require Docker or Podman as the universal answer to
reproducibility.

Projects must declare and validate their native host, virtual-machine, or
specialized-lab requirements. Containers may be used when useful, but they must
not hide undeclared dependencies or become the only validation path unless the
product's accepted deployment model is itself container-native.

## Repository contents

- `standards/repository-assurance/v2/` — current normative ISRAS v2 documents;
  start with [the v2 document index](standards/repository-assurance/v2/INDEX.md)
- `standards/repository-assurance/v1/` — retained accepted ISRAS v1 normative
  history for pinned adopters and historical verification
- `schemas/` — machine-readable assurance, environment, checkpoint, migration,
  phase-compliance, impact-assessment, authority-boundary, and evidence schemas
- `templates/repository-baseline/` — files copied into adopting repositories
- `templates/engineering-standards/` — ISRAS v2 phase, impact, and authority
  record templates
- `templates/workflows/` — GitHub Actions caller examples
- `.github/workflows/` — reusable organization workflows
- `tools/isras/` — adoption, policy, compliance, fresh-clone, checkpoint,
  manifest, release-state, and evidence tooling
- `integration-guides/` — project-specific adoption sequences
- `GLOSSARY.md` — authoritative terminology, including the ISRAS definition
- `SUPPORT-AND-COMPATIBILITY.md` — support, compatibility, and lifecycle policy
- `docs/engineering/adopter-quick-start.md` — exact-release adoption and
  signature-verification sequence

## ISRAS v2 adoption sequence

1. Read `standards/repository-assurance/v2/MIGRATION-GUIDE.md`.
2. Confirm the target repository's currently pinned ISRAS release.
3. Complete an Engineering Standards Impact Assessment when required.
4. Create a reviewed work branch in the target repository.
5. Pin the exact verified `isras-v2.0.0` release commit and source-manifest
   digest.
6. Apply and customize the governed baseline without weakening inherited
   controls.
7. Complete the required phase-entry review before implementation.
8. Run portable, fresh-clone, policy, compliance, and applicable specialized
   validation.
9. Complete the phase-exit review with exact pushed-source evidence.
10. Record adoption only after the applicable acceptance boundary passes.

Publication of ISRAS v2.0.0 does not silently change a repository pinned to
ISRAS v1.0.1.

## Standard versioning

Adopting repositories pin the standard and reusable workflows to an exact commit
SHA. A signed standard release tag is useful for humans and provides the
authoritative acceptance decision, but exact commit identity remains the
normative machine reference.
