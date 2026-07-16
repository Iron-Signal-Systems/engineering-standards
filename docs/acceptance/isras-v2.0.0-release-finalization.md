# ISRAS v2.0.0 Release Finalization Record

## Decision state

**Status: AUTHORIZED — COMPLETION REQUIRES SIGNED TAG AND BRANCH CONVERGENCE**

The ISRAS v2.0.0 candidate has been formally accepted and this source tree is
authorized for exact-commit release finalization.

## Release identity

- Version: `2.0.0`
- Required signed annotated tag: `isras-v2.0.0`
- Development branch at finalization: `dev`
- Release branch: `main`
- Accepted candidate source commit: `4aff00dfdc88154390252898210abc336fa8b2fc`
- Candidate evidence commit: `b0c982221acde7873307d010aca73ed2e386eb99`
- Formal candidate-acceptance authorization commit: `24e911b7c4a63735bcef9b4b84ab9b62ace10298`
- Accepted predecessor: `c379417720faa595fa5cb89a1dfdb2259d6cb95e`

The exact release commit is intentionally not written as a literal value inside
this source file because a commit cannot contain its own final object identity.
The verified signed tag object, its peeled commit target, and remote branch
convergence provide the immutable exact release identity.

## In-tree finalization boundary

This release source:

1. changes root `VERSION` from `1.0.1` to `2.0.0`;
2. publishes synchronized v2 release, support, compatibility, README, changelog,
   migration, and acceptance documentation;
3. preserves the accepted ISRAS v1 normative tree unchanged;
4. provides a frozen v2 release-source validator and phase gate;
5. regenerates and verifies the tracked-file source manifest;
6. retains the candidate evidence and formal candidate-acceptance lineage.

## Required exact-commit campaign

Before the release tag is created, the exact pushed release-source commit shall
pass:

- portable validation;
- fresh-clone and remote-completeness validation;
- the ISRAS v2 release-source validator;
- the integration-enabled regression suite;
- isolated historical revalidation of the accepted ISRAS v1.0.1 checkpoint;
- source-manifest verification;
- committed-whitespace validation.

## Completion criteria

Release finalization becomes complete only when:

1. the SSH-signed annotated `isras-v2.0.0` tag verifies successfully;
2. the tag peels to the exact validated release-source commit;
3. remote `dev` identifies that exact commit;
4. remote `main` identifies that exact commit;
5. `main` was advanced without force;
6. the tag annotation or GitHub release publishes the exact commit,
   source-manifest SHA-256, evidence digest, validation result, decision
   authority, and signing identity;
7. the protected `isras-*` namespace prevents ordinary tag movement or deletion.

After completion, `dev` may advance for future work. `main` and
`isras-v2.0.0` remain fixed at the accepted v2.0.0 source boundary.

## Checkpoint sequencing

The accepted immutable `isras-v2.0.0` checkpoint is registered only after tag
verification and branch convergence. The checkpoint-registration commit does
not redefine the release source.

## Adoption non-claim

Publication does not silently update any repository pinned to ISRAS v1.0.1.
Each adopter must perform an applicable Engineering Standards Impact Assessment,
pin the exact verified v2.0.0 commit and manifest digest, and complete its own
reviewed migration and acceptance evidence.
