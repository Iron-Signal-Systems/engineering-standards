# ISRAS v2.0.0 Release Completion and Checkpoint Record

## Decision

**Status: COMPLETE**

ISRAS v2.0.0 was formally accepted, signed, released, and converged on
2026-07-16.

## Immutable release boundary

- Version: `2.0.0`
- Signed annotated tag: `isras-v2.0.0`
- Accepted release commit: `781246e69f8a9a382c25040f94b62dfe3b25ba89`
- Annotated tag object: `a7a09a02798e2b2c905f2686820fd30890f62bc6`
- Peeled tag target: `781246e69f8a9a382c25040f94b62dfe3b25ba89`
- Source-manifest SHA-256: `262e275e63f1c7d104bb77c8799633121bad43d2fc58edf54594e5eda61555b7`
- Candidate-evidence JSON SHA-256: `0e4516f76032008075a844ddc43cb44fdb90ae09ab31b9af113b32923f082cd7`

At release finalization, remote `dev`, remote `main`, and the peeled signed tag
target all identified the exact accepted release commit.

## Signing identity

- Signing format: `SSH`
- Signing algorithm: `ED25519`
- Signing identity: `kb2vhn@gmail.com`
- Signing-key fingerprint: `SHA256:FiH+Jk7HHrNkvDEQTehI/aCfkmKpivtsqmkl5TmmMSE`
- Local signature verification: `PASS`
- Remote tag-object verification: `PASS`

The signed annotated tag is the authoritative ISRAS v2.0.0
acceptance-decision object.

## Validation result

- ISRAS v2.0.0 release-source validation: `45 PASS`, `0 FAIL`
- Integration-enabled regression suite: `29 tests`, `OK`
- Accepted ISRAS v1.0.1 historical checkpoint: `PASS`
- Exact pushed-source validation: `PASS`
- Source-manifest verification: `PASS`
- Remote release convergence: `PASS`
- Working-tree cleanliness after finalization: `PASS`

## Acceptance lineage

- Accepted predecessor: `c379417720faa595fa5cb89a1dfdb2259d6cb95e`
- Accepted candidate source: `4aff00dfdc88154390252898210abc336fa8b2fc`
- Candidate evidence commit: `b0c982221acde7873307d010aca73ed2e386eb99`
- Formal candidate-acceptance authorization commit: `24e911b7c4a63735bcef9b4b84ab9b62ace10298`
- Final release source: `781246e69f8a9a382c25040f94b62dfe3b25ba89`

## Immutable checkpoint registration

The checkpoint registry binds `isras-v2.0.0` to:

- commit `781246e69f8a9a382c25040f94b62dfe3b25ba89`;
- frozen gate
  `tools/validation/phase-gates/validate_isras_v2_release.sh`;
- required historical branch name `dev`;
- portable environment profile;
- expected result of zero failures.

Historical validation checks out the exact accepted source in an isolated clone
on a branch named `dev` and executes the gate retained in that source tree.

## Post-release development boundary

This completion and checkpoint record is a later governance change on `dev`.
It does not move or redefine remote `main`, the signed `isras-v2.0.0` tag, tag
object `a7a09a02798e2b2c905f2686820fd30890f62bc6`, or accepted source commit `781246e69f8a9a382c25040f94b62dfe3b25ba89`.

After this record is committed and pushed, `dev` may identify the later
checkpoint-registration commit while `main` and `isras-v2.0.0` remain fixed at
the accepted v2.0.0 release source.

## Adoption non-claim

No adopting repository silently inherits ISRAS v2.0.0. Each adopter must perform
the applicable Engineering Standards Impact Assessment, pin the exact accepted
release commit and manifest digest, and complete its own reviewed migration and
acceptance evidence.
