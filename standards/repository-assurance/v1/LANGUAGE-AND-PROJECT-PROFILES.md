# Language and Project Assurance Profiles

## Purpose

ISRAS defines common assurance behavior while allowing a repository to select
only the checks applicable to its languages, runtime, and product boundary.
Profiles do not weaken common repository, evidence, or historical-validation
requirements.

## Go

Applicable repositories must define and enforce, as appropriate:

- exact Go language and toolchain versions;
- `gofmt`, `go vet`, `go test`, and supported race testing;
- `go mod verify`;
- `govulncheck` or an accepted equivalent;
- direct and indirect dependency inventory;
- dependency-license inventory;
- reproducible build comparison or an explicit non-claim;
- build and release provenance.

## .NET

Applicable repositories must define and enforce, as appropriate:

- exact supported SDK versions;
- lock-file-backed restore with `--locked-mode` for accepted builds;
- analyzers and formatting policy;
- build and test execution;
- package vulnerability and license review;
- publish validation;
- reproducibility classification and provenance.

## Python

Applicable repositories must define and enforce, as appropriate:

- exact supported Python versions;
- locked direct and transitive dependencies for accepted builds;
- syntax, import, unit, and integration tests;
- accepted type checking and linting policy where used;
- deterministic fixture generation;
- package-build and installation validation;
- dependency vulnerability and license review.

## PowerShell

Applicable repositories must define and enforce, as appropriate:

- exact supported PowerShell and Windows versions;
- parser validation;
- PSScriptAnalyzer policy;
- Pester tests;
- `-WhatIf` and destructive-operation protections;
- marker, OU, path, and scope restrictions;
- least-authority execution identities;
- cleanup and teardown validation.

## SQL and database change

Applicable repositories must define and enforce, as appropriate:

- deterministic migration order;
- SHA-256 migration integrity;
- clean installation and supported upgrade paths;
- exact database major versions;
- transactional and recovery behavior;
- role, ownership, and privilege validation;
- concurrency and hostile tests;
- backup, restore, and point-in-time recovery evidence.

## Documentation and generated artifacts

Applicable repositories must define and enforce, as appropriate:

- link and reference validation;
- identifier uniqueness;
- status synchronization;
- terminology and required-section policy;
- deterministic generation;
- generated-file SHA-256 manifests;
- explicit distinction between normative, informative, candidate, and accepted
  material.

## Applicability

A repository records its selected profiles in `REPOSITORY-ASSURANCE.json` and
its exact commands and versions in environment and validation profiles.
Unavailable specialized capabilities must be reported as unavailable or not
applicable rather than misrepresented as a pass.
