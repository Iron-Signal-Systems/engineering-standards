# Pinned Project Framework

## Purpose

An Iron Signal Systems project adopts one accepted ISRAS release and remains
governed by that exact release until a deliberate, reviewed upgrade changes the
pin.

A project created while `isras-v0.1.5` is the accepted baseline references that
release and builds its repository framework, documentation, tests, validation,
release process, and evidence around the obligations defined by `0.1.5`.
Publication of `0.1.6` does not silently alter the project.

## The project pin

Every adopting project shall commit one authoritative project pin. The final
path and schema are release artifacts; the intended boundary is:

```text
.isras/project.json
```

The pin identifies at least:

- schema version;
- ISRAS profile;
- accepted standard version;
- immutable signed release tag;
- exact Engineering Standards release commit;
- validator artifact name and digest;
- selected language or platform profiles;
- project identity;
- project-owned validation declaration;
- compatibility or migration metadata required by the release.

Human-readable version, immutable tag, exact commit, and artifact digest are all
required. A floating branch, `latest`, mutable download URL without a digest, or
unresolved version range is not an acceptable pin.

## Single version authority

The committed project pin is authoritative. Repository wrappers, local
validation, CI, release checks, and upgrade tooling shall read the same pin.

A project shall not maintain independent ISRAS versions in multiple scripts and
workflow files. Integration files may contain the immutable workflow commit
required to start execution, but they must verify that it corresponds to the
release declared by the project pin.

## External execution boundary

The validator remains owned and released by the Engineering Standards
repository. A consuming project shall not ordinarily copy:

- `cmd/isras-validate`;
- validator implementation packages;
- validator implementation unit tests;
- Engineering Standards release internals.

The project may commit:

- its ISRAS pin;
- project identity and policy declarations;
- project-specific command declarations;
- bounded exceptions;
- a small acquisition and execution wrapper;
- CI integration;
- project-local evidence paths;
- generated project-framework files that become project-owned.

ISRAS tooling shall not become a product runtime dependency merely because the
project uses the Go profile.

## Project framework

An accepted ISRAS release may provide a versioned project framework or template.
That framework identifies the repository-level artifacts normally expected of a
new ISS project, including appropriate forms of:

- project README and purpose;
- changelog;
- contribution and change-control guidance;
- security reporting and security-sensitive boundaries;
- architecture documentation;
- testing and validation documentation;
- release and recovery documentation;
- validation configuration;
- CI entry points;
- repository-owned project commands;
- evidence and exception locations.

The framework governs required engineering artifacts. It does not impose one
application architecture or source layout across every language.

## New-project initialization

A future repository-owned command shall support an explicit release selection,
conceptually:

```text
isras project init --release isras-v0.1.5 --target /path/to/project
```

Initialization shall:

1. resolve the immutable release;
2. verify its signed identity and exact commit;
3. acquire release manifests and required artifacts;
4. verify every artifact digest;
5. inspect the target and selected profile;
6. prepare a reviewable project-framework plan;
7. apply only after explicit authorization;
8. validate the resulting project boundary;
9. stage or report changes without committing or pushing.

It shall not silently choose a newer release when an exact release was requested.

## Existing-project adoption

Adopting ISRAS in an established project is a migration, not a blind template
copy. Tooling shall inventory existing artifacts, identify satisfied
requirements, propose missing artifacts, and preserve project-owned design.

Existing source layouts and technical choices shall not be replaced solely to
match a reference template.

## Local acquisition and cache

The project wrapper may use an owner-controlled local cache for verified ISRAS
artifacts. Cache entries shall be keyed by immutable release identity and digest.
A cached artifact is accepted only after the same verification required for a
fresh download.

An offline validation mode may use previously verified artifacts. It must report
that network freshness was not checked and must not claim that a later release or
revocation status was evaluated.

## Validation identity

Every project validation result shall identify:

- ISRAS profile and version;
- Engineering Standards release tag and exact source commit;
- validator artifact digest;
- target repository identity;
- exact target commit or working-tree state;
- selected language profiles;
- execution mode;
- evidence location;
- self-validation or independent-review status.

This separates the authority that produced the validator from the project being
validated.

## No silent inheritance

Changes to `engineering-standards/dev`, `main`, a later tag, or a reusable
workflow do not modify a pinned project. The project changes only through an
explicit initialization correction, pin repair, or upgrade.

## Current transition

The existing source-export model is deprecated for new adoption because it copies
validator implementation source and tests into the consuming repository. It
remains present temporarily so its replacement can be implemented and validated
without an unreviewed deletion.

No project should perform a new source export while the pinned project framework
is being implemented.
