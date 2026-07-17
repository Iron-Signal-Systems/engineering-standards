# ISRAS Release Artifact Contract

## Purpose

An accepted ISRAS release is both an assurance-tool release and a versioned
project-framework release. It must contain enough immutable information for a
project to select, verify, execute, and later reconstruct the exact standard it
adopted.

## Release identity

Every release shall bind the following to the same tested source:

- stable semantic version;
- signed immutable tag;
- exact source commit;
- release title and notes;
- artifact manifests;
- validator binaries;
- framework and contract artifacts;
- provenance;
- migration metadata;
- reusable workflow identity when provided.

A mismatch among these identities is a release failure.

## Required artifact classes

The first production contract is expected to include:

```text
isras-validator-<os>-<arch>
isras-project-framework.tar.gz
isras-contracts.tar.gz
isras-migration-from-previous.json
SHA256SUMS
SHA512SUMS
provenance.json
```

The exact supported operating systems, architectures, names, and formats are
versioned by the release contract.

A release may also provide:

- detached signatures;
- software bills of materials;
- reproducible-build records;
- container artifacts;
- platform-specific wrappers;
- offline bundles;
- profile-specific framework overlays.

## Validator artifact

A validator binary shall report its embedded or bound release identity. It shall
refuse to represent itself as another version merely because a project pin
requests that version.

The validator shall validate a target repository. It shall not modify the target
unless a separately named, explicitly authorized command has modification
authority.

## Framework artifact

The project-framework artifact contains repository-level templates, schemas, and
guidance for initializing or assessing a project. It may contain profile
overlays, but it shall preserve the distinction between universal core
requirements and language-specific guidance.

Generated project files become project-owned after review and adoption. The
framework implementation itself remains an ISRAS release artifact.

## Contract artifact

The contracts artifact contains the machine-readable schemas and requirement
identifiers necessary to interpret the release. It shall include enough metadata
to reject incompatible pin schemas, profiles, commands, and evidence formats.

## Manifest and digest requirements

Every downloadable artifact shall be covered by SHA-256 and SHA-512 manifests.
Manifests shall be generated from the final release bytes and shall themselves be
bound to the release identity.

A consuming project shall verify the digest from its committed pin or an
equivalent release-bound trust record before execution or extraction.

A digest mismatch is fail-closed. Tooling shall not automatically retry against a
different release.

## Provenance

Release provenance shall identify at least:

- source repository;
- exact source commit;
- release tag;
- declared version;
- build environment and toolchain identity;
- artifact names and digests;
- validation campaign identity;
- publication time;
- known evidence limitations;
- signer or release authority.

Provenance is evidence of how the artifact was produced. It is not by itself a
claim of independent audit or universal production fitness.

## Reusable workflow boundary

A reusable GitHub workflow may provide hosted execution for projects. A project
shall reference an immutable workflow commit associated with its pinned release,
not a floating branch.

The workflow shall read and verify the project pin, acquire the exact artifacts,
and execute the same release identity used locally. It shall not silently select
a later release.

## Release immutability

Published tags, release artifacts, manifests, and provenance are immutable. A
defect in an accepted release is corrected by a later release and migration
record, not by replacing bytes under the old identity.

A release may be declared deprecated or revoked through a separately signed and
versioned advisory mechanism. That status does not permit mutation of the
original release.

## Availability and retention

Iron Signal Systems shall retain accepted release artifacts for the supported
lifetime of projects pinned to them or provide a verified archival replacement.
Projects may retain verified local copies for offline and recovery use.

Loss of the live download location shall not invalidate an artifact whose exact
bytes, digest, release identity, and provenance remain verifiable.

## Acceptance gate

The release artifact contract is not complete until tests prove:

- every published byte is covered by required manifests;
- validator identity matches the release;
- framework and contracts match the release;
- local and hosted consumption use the same pin;
- altered artifacts fail verification;
- floating references are rejected;
- offline verified acquisition is distinguishable from online validation;
- no consuming project's runtime dependency graph is modified by validator use.
