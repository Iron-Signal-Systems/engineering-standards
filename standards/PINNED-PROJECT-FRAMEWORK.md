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

Every adopting project shall commit one authoritative project pin at:

```text
.isras/project.json
```

The v1 machine-readable contract is committed at
[`schemas/isras-project-v1.schema.json`](../schemas/isras-project-v1.schema.json).
The strict standard-library Go parser is documented in
[`standards/PROJECT-PIN-SCHEMA.md`](PROJECT-PIN-SCHEMA.md).

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

The v1 parser validates structure and identity only. Artifact verification,
project initialization, and command execution remain separately authorized
boundaries. The initializer may generate a candidate pin only after it has
verified the exact accepted release under the project-initialization contract.

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

Once accepted and published, the `0.1.2` release validator supports an explicit
release selection:

```text
isras-validator-linux-amd64 --repo /path/to/project project-pin initialize --release isras-v0.1.2 --go-defaults
```

Initialization requires the exact linker-bound validator artifact for the
requested release, then resolves and verifies the immutable signed release, exact
commit, six assets, both digests, manifests, provenance, and reusable workflow
before publishing any target file. It fixes evidence to untracked `.local/isras`,
prepares the complete project-owned adoption set, applies without replacement,
validates canonical pin generation, and leaves the result uncommitted and
unpushed for review. It never silently chooses a newer release. See
[`PROJECT-INITIALIZATION-AND-ADOPTION.md`](PROJECT-INITIALIZATION-AND-ADOPTION.md).

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

## Hosted SSH signer trust

A reusable hosted workflow validating an SSH-signed project commit shall establish
trust from the exact pinned Engineering Standards source before invoking repository
validation. The consuming commit cannot authorize its own signer. The workflow
shall verify the tracked trust bytes and digest, create a private target-local
`gpg.ssh.allowedSignersFile`, bind the reported principal and fingerprint to the
commit committer identity, and retain both success evidence and failure logs. See
[`HOSTED-SSH-SIGNER-TRUST.md`](HOSTED-SSH-SIGNER-TRUST.md).

## No silent inheritance

Changes to `engineering-standards/dev`, `main`, a later tag, or a reusable
workflow do not modify a pinned project. The project changes only through an
explicit initialization correction, pin repair, or upgrade.

## Current transition

The `0.1.2-development` boundary replaces new source-export adoption with verified
release bootstrap, a canonical project pin, an immutable caller workflow, and a
small project-owned format checker. The old source-export model remains
deprecated and must not be used for new adoption. This replacement becomes
authoritative only after publication of the accepted `isras-v0.1.2` release.
