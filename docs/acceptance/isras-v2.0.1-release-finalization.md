# ISRAS v2.0.1 Release Finalization Record

## Decision state

**Status: AUTHORIZED — COMPLETION REQUIRES SIGNED TAG AND BRANCH CONVERGENCE**

The ISRAS v2.0.1 candidate has been formally accepted and this source tree is
authorized for exact-commit release finalization.

## Release identity

- Version: `2.0.1`
- Required signed annotated tag: `isras-v2.0.1`
- Development branch at finalization: `dev`
- Release branch: `main`
- Accepted candidate source commit: `6543a5a93f078f47d87aa3b8ed8ebd2024cec373`
- Candidate evidence commit: `9dbe4d9696ff4a9838fd83cb0f6f652087710f98`
- Formal candidate-acceptance authorization commit: `57d23742e60d29bf6f46d15b8f64f0497bb260cd`
- Accepted predecessor release: `781246e69f8a9a382c25040f94b62dfe3b25ba89`
- First BSD-3-Clause source boundary: `5c07b428b206e4f4e5d7e33d6f5811d7d4e6e739`
- Candidate evidence JSON SHA-256: `42d7dce7500929647af001f47bbbdf30ae7bef88c598d0aba8edd2424564d2b9`
- Candidate source-manifest SHA-256: `e2b6488a7f670b0c81d873478154d03438a9c5f21a8bf05010863fbe1e4fd7e8`

The exact release commit is intentionally not written as a literal value inside
this source file because a commit cannot contain its own final object identity.
The verified signed tag object, its peeled commit target, and remote branch
convergence provide the immutable exact release identity.

## In-tree finalization boundary

This release source:

1. changes root `VERSION` from `2.0.0` to `2.0.1`;
2. publishes the BSD-3-Clause licensing decision in the release line;
3. preserves the ISRAS v1 and v2 normative trees unchanged;
4. preserves schemas, governed templates, integration guides, and reusable
   workflows unchanged;
5. adds a frozen v2.0.1 release-source validator and phase gate;
6. regenerates and verifies the tracked-file source manifest;
7. retains the exact candidate source, evidence, and formal-acceptance lineage.

## Required exact-commit campaign

Before the release tag is created, the exact pushed release-source commit shall
pass:

- source-manifest verification;
- current release-state validation;
- the frozen ISRAS v2.0.1 release-source validator;
- portable validation;
- fresh-clone and remote-completeness validation;
- the integration-enabled regression suite;
- isolated historical revalidation of accepted ISRAS v1.0.1;
- isolated historical revalidation of accepted ISRAS v2.0.0;
- committed-whitespace validation.

## Completion criteria

Release finalization becomes complete only when:

1. the SSH-signed annotated `isras-v2.0.1` tag verifies successfully;
2. the tag peels to the exact validated release-source commit;
3. remote `dev` identifies that exact commit;
4. remote `main` identifies that exact commit;
5. `main` was advanced without force;
6. the tag annotation or GitHub release publishes the exact commit,
   source-manifest SHA-256, evidence digest, validation result, decision
   authority, and signing identity;
7. the protected `isras-*` namespace prevents ordinary tag movement or deletion.

Until every criterion passes, the latest accepted release remains
`isras-v2.0.0`. After completion, `dev` may advance for future work while
`main` and `isras-v2.0.1` remain fixed at the accepted v2.0.1 source boundary.

## Checkpoint sequencing

The immutable `isras-v2.0.1` checkpoint is registered only after signed-tag
verification and exact branch convergence. The later checkpoint-registration
commit does not redefine the release source.

## Adoption non-claim

Publication does not silently update any adopting repository. Each adopter must
perform any applicable Engineering Standards Impact Assessment, pin the exact
verified v2.0.1 release commit and manifest digest, and complete its own reviewed
migration or patch-adoption evidence.
