# ISRAS v1.0.1 Acceptance Plan

## Status

**Candidate plan — this document is not an acceptance decision.**

## Intended release

- Version: `1.0.1`
- Intended tag: `isras-v1.0.1`
- Required source branch: `dev`
- Release branch: `main`
- Predecessor tag: `isras-v1.0.0`
- Predecessor source commit:
  `f9655ddbbf04430fc468aab405f2ed880df3e97d`

## Scope

The candidate includes:

- the durable v1.0.0 formal-acceptance evidence;
- the completed v1.0.0 tag-correction and finalization records;
- corrected public release, support, and security status;
- a signed-tag authoritative acceptance-decision model;
- release-branch convergence requirements;
- the protected `isras-*` tag namespace;
- exact-commit adopter quick-start instructions;
- release-state drift validation.

## Required validation

The exact merged candidate commit must pass:

- repository policy validation;
- source-manifest verification;
- portable validation;
- integration-tool validation;
- fresh-clone and remote-completeness validation;
- historical checkpoint validation;
- the ISRAS v1 candidate gate;
- hosted Ubuntu validation;
- hosted macOS validation;
- hosted Windows PowerShell validation.

## Acceptance evidence

Final evidence is generated only after the exact candidate commit is present
on remote `dev`.

The evidence must be retained in an approved artifact or evidence store. Its
durable location and SHA-256 digest must be included in the signed annotated
`isras-v1.0.1` tag.

## Authoritative acceptance decision

The verified SSH-signed annotated `isras-v1.0.1` tag is the authoritative
acceptance-decision object.

Its annotation must identify:

- decision status and date;
- exact predecessor;
- validation gate and environment;
- runner identity;
- evidence digest and location;
- correctness and applicable assurance outcomes;
- warnings and non-claims.

## Completion conditions

Acceptance is complete only when the canonical remote proves that:

- `refs/tags/isras-v1.0.1` is an annotated signed tag;
- the tag signature verifies through the approved allowed-signers boundary;
- the peeled tag target is the exact validated candidate commit;
- `refs/heads/main` identifies that same commit;
- `refs/heads/dev` identifies that same commit;
- the tag namespace is protected from ordinary update or deletion.

No later source commit is required merely to record the acceptance decision.
