# Support and Compatibility Policy

This repository publishes and maintains the Iron Signal Repository Assurance
Standard (ISRAS).

## Supported standard versions

ISRAS v2.0.x is the current major release line. ISRAS v1.0.x remains a supported
historical line for repositories that are deliberately pinned to an exact
accepted v1 release and have not completed an ISRAS v2 Engineering Standards
Impact Assessment and migration.

A release is accepted only when its exact source commit is jointly identified
by a verified signed annotated `isras-v*` tag and the remote `main` branch.

- ISRAS v2.0.0 is the first release requiring mandatory standards inheritance,
  phase-entry and phase-exit reviews, maturity-accurate evidence, hostile
  authority validation, and the bounded-authority invariant.
- ISRAS v1.0.1 remains the latest accepted v1 release.
- `main` represents the latest accepted standard source boundary.
- `dev` contains current development and may advance after a release.
- At release finalization, `dev`, `main`, and the signed tag must identify the
  same exact source commit.
- Exact accepted tags and commits remain immutable.
- A version number or branch name alone does not prove acceptance.
- Security and correctness fixes are applied according to release risk,
  compatibility impact, and published evidence.

## Compatibility

Adopting repositories pin an exact ISRAS commit. An ISRAS update requires a
separately reviewed repository-assurance change and applicable validation in the
adopting repository.

Moving from ISRAS v1 to ISRAS v2 is a major-version migration. It requires review
of the v2 migration guide, an Engineering Standards Impact Assessment when
applicable, exact release pinning, and completion of required phase-compliance
records. Publication of v2 does not silently alter an existing v1 pin.

Patch releases must not intentionally introduce incompatible schema, workflow,
evidence, or required-entrypoint changes. Minor or major releases must document
migration and compatibility impact.

## End of support

A standard version may reach end of support only after a successor and a usable
migration path are published. Accepted historical source, tags, schemas,
manifests, and evidence identities remain available for verification.
