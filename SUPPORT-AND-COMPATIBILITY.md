# Support and Compatibility Policy

This repository publishes and maintains the Iron Signal Repository Assurance
Standard (ISRAS).

## Supported standard versions

Until the first formal v1 acceptance, the `dev` branch is a candidate standard
and no production-support claim is made.

After acceptance:

- `main` represents the latest accepted standard release;
- `dev` contains current candidate development;
- exact accepted tags and commits remain immutable;
- security and correctness fixes are applied to supported releases according to
  their published release notes and risk.

## Compatibility

Adopting repositories pin an exact ISRAS commit. An ISRAS update may require a
separate repository-assurance change and validation campaign in the adopter.

Patch releases must not intentionally introduce incompatible schema, workflow,
or required-entrypoint changes. Minor or major releases document migration and
compatibility impact.

## End of support

A standard version may reach end of support only after a successor and migration
path are published. Accepted historical source, tags, schemas, and evidence must
remain available for verification.
