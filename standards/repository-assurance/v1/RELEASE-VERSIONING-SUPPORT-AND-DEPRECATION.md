# Release, Versioning, Support, and Deprecation

## Purpose

This document defines the minimum release-governance requirements for ISRAS
and for repositories adopting it.

## Version identity

A release must identify:

- an exact 40-character source commit;
- an immutable release or acceptance tag;
- the applicable ISRAS commit;
- the release version;
- the acceptance evidence identity and SHA-256 digest;
- artifact and manifest SHA-256 values;
- compatibility and support statements.

Human-readable tags and version strings do not replace exact commit identity.

## Versioning

Repositories must publish a documented versioning model. Semantic Versioning
is the default for reusable standards, libraries, services, and externally
consumed interfaces unless an accepted project record defines a better model.

A breaking change includes an incompatible change to a public interface,
schema, migration contract, deployment contract, validation contract,
evidence schema, or supported operational workflow.

## Acceptance and release tags

Acceptance and release tags must be annotated. They must be
cryptographically signed when an approved signing identity and verification
path exist.

A repository that cannot sign tags must record a time-bounded signing
exception containing:

- reason;
- owner;
- affected tags;
- compensating controls;
- remediation milestone;
- expiration or review date.

Protected acceptance and release tags must not be moved or deleted through
ordinary maintenance.

ISRAS release tags use the protected namespace `isras-*`.

## Signed tag as acceptance decision

When approved signing is available, the signed annotated release tag is the
authoritative acceptance-decision object.

The tag target is the exact accepted source commit. The signed annotation
must contain:

- decision status and date;
- version and predecessor;
- validation gate and environment;
- runner identity;
- evidence digest and durable location;
- assurance outcomes;
- warnings, exceptions, and non-claims.

The acceptance plan must be committed before candidate freeze. Final evidence
generated after the freeze is retained outside the source tree and bound to
the decision through its digest in the signed tag annotation.

A separate post-acceptance source commit is not required merely to record the
decision.

## Release-branch convergence

At release finalization:

- the signed tag must peel to the validated candidate commit;
- `main` must identify that same exact commit;
- `dev` must identify that same exact commit;
- all three identities must be independently verified from the canonical
  remote.

New development may advance `dev` only after finalization is complete.

## Compatibility statement

Every release must identify:

- supported upgrade sources;
- supported downgrade or rollback behavior;
- database and schema compatibility;
- configuration compatibility;
- supported operating systems and architectures;
- supported external integrations;
- explicitly unsupported combinations.

## Support boundaries

Each repository must publish:

- supported versions or branches;
- security-fix policy;
- maintenance expectations;
- response and remediation process;
- end-of-support criteria;
- archival and migration expectations.

## Deprecation

A deprecation must state:

- the deprecated element;
- replacement or migration path;
- first deprecated version;
- planned removal boundary;
- compatibility impact;
- evidence that dependents were identified;
- exception process when migration cannot be completed in time.

## Release notes

Release notes are required for accepted releases and must distinguish:

- implementation changes;
- security changes;
- migrations;
- operational changes;
- compatibility changes;
- known issues;
- warnings and non-claims;
- required administrator or operator action.
