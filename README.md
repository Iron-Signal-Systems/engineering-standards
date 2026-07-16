# Iron Signal Repository Assurance Standard

> **Built on purpose. Backed by discipline. Engineered to endure.**

## ISRAS vision

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

## Current implementation profile

This repository currently implements the **ISRAS Solo Developer Baseline** as a
development candidate. It is intentionally sized for a single developer while
retaining truthful engineering discipline and a path toward the complete ISRAS
vision.

The profile requires:

- signed commits and signed release tags;
- committed, reviewable test and validation source;
- exact commit identification;
- clear self-validation status without false independent-review claims;
- Go formatting, static analysis, tests, builds, module checks, and known
  vulnerability scanning;
- local secret detection before source is committed or pushed;
- automatic censoring of possible sensitive values in terminal output and logs;
- bounded redaction and allowlist workflows;
- a local `*.log` for every failed check;
- concise terminal dashboards with exact safe commands for the detected issue;
- declared support for Arch Linux, supported Ubuntu Server LTS releases, and
  supported Fedora Server releases unless a project declares a different scope.

The earlier ISRAS v1, v2, and v3 development work is preserved through the
archive branch, signed archive tag, and local Git bundle created by the restart
installer. That work remains available as a future source for team, production,
regulated, and independently reviewed profiles.

## Quick start

Build the repository-owned validator:

```bash
./tools/build-validator.sh
```

Run complete development validation:

```bash
./.local/bin/isras-validate all
```

Run commit validation after committing the exact candidate:

```bash
./.local/bin/isras-validate all --mode commit
```

Run only the secret scanner:

```bash
./.local/bin/isras-validate secrets
```

The validator prints only the commands relevant to a detected problem. Commands
are labeled as read-only, networked, proposal-creating, or working-tree
modifying actions.

## Assurance status

This tree is a development candidate. A successful local run establishes
**self-validation**, not independent review, certification, regulatory
compliance, production readiness, or absence of vulnerabilities.

See:

- [`standards/ISRAS-VISION.md`](standards/ISRAS-VISION.md)
- [`standards/SOLO-DEVELOPER-BASELINE.md`](standards/SOLO-DEVELOPER-BASELINE.md)
- [`standards/TESTING-AND-VALIDATION.md`](standards/TESTING-AND-VALIDATION.md)
- [`standards/FAILURE-LOGGING-AND-REMEDIATION.md`](standards/FAILURE-LOGGING-AND-REMEDIATION.md)
- [`standards/PLATFORM-SUPPORT.md`](standards/PLATFORM-SUPPORT.md)
- [`standards/RELEASES-AND-SIGNING.md`](standards/RELEASES-AND-SIGNING.md)
- [`docs/archive/README.md`](docs/archive/README.md)
- [`integration-guides/PROJECT-ADOPTION.md`](integration-guides/PROJECT-ADOPTION.md)
