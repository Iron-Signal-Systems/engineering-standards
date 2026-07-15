# Support and Compatibility Policy

This repository publishes and maintains the Iron Signal Repository Assurance
Standard (ISRAS).

## Supported standard versions

ISRAS v1.0.x releases are supported when their exact source commit is
identified by a verified signed annotated `isras-v*` tag and the remote
`main` branch.

ISRAS v1.0.0 is the first formally accepted release.

- `main` represents the latest accepted standard source boundary;
- `dev` contains current development and may advance after a release;
- at release finalization, `dev`, `main`, and the signed tag must identify
  the same exact source commit;
- exact accepted tags and commits remain immutable;
- a version number or branch name alone does not prove acceptance;
- security and correctness fixes are applied according to release risk,
  compatibility impact, and published evidence.

## Compatibility

Adopting repositories pin an exact ISRAS commit. An ISRAS update requires a
separately reviewed repository-assurance change and applicable validation in
the adopting repository.

Patch releases must not intentionally introduce incompatible schema,
workflow, evidence, or required-entrypoint changes. Minor or major releases
must document migration and compatibility impact.

## End of support

A standard version may reach end of support only after a successor and a
usable migration path are published. Accepted historical source, tags,
schemas, manifests, and evidence identities remain available for
verification.
