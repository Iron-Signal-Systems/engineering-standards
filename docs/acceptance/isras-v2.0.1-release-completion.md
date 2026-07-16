# ISRAS v2.0.1 Release Completion and Checkpoint Record

## Decision

**Status: COMPLETE**

ISRAS v2.0.1 was formally accepted, signed, released, and converged on
2026-07-16.

## Immutable release boundary

- Version: `2.0.1`
- Signed annotated tag: `isras-v2.0.1`
- Accepted release commit: `d34fad82781a4e8485f8907fbfd34f236fa79ad2`
- Annotated tag object: `f4eacec519c96be225ffd37276cc646d3712ab0f`
- Peeled tag target: `d34fad82781a4e8485f8907fbfd34f236fa79ad2`
- Source-manifest SHA-256: `8f54ed1e9bfee251bf89b4c5f12edf11ac1e25ef0d145ba745301f2d05787ef1`
- Candidate-evidence JSON SHA-256: `42d7dce7500929647af001f47bbbdf30ae7bef88c598d0aba8edd2424564d2b9`

At release finalization, remote `dev`, remote `main`, and the peeled signed tag
target all identified the exact accepted release commit.

## Signing identity

- Signing format: `SSH`
- Signing algorithm: `ED25519`
- Signing identity: `kb2vhn@gmail.com`
- Signing-key fingerprint: `SHA256:FiH+Jk7HHrNkvDEQTehI/aCfkmKpivtsqmkl5TmmMSE`
- Local signature verification: `PASS`
- Remote tag-object verification: `PASS`

The signed annotated tag is the authoritative ISRAS v2.0.1
acceptance-decision object.

## Validation result

- Exact pushed ISRAS v2.0.1 release-source campaign: `PASS`
- Integration-enabled regression suite: `35 tests`, `OK`, `2 skipped`
- Accepted ISRAS v1.0.1 historical checkpoint: `PASS`
- Accepted ISRAS v2.0.0 historical checkpoint: `PASS`
- Source-manifest verification: `PASS`
- Remote release convergence: `PASS`
- Working-tree cleanliness after finalization: `PASS`

## Acceptance lineage

- Accepted predecessor release: `781246e69f8a9a382c25040f94b62dfe3b25ba89`
- First BSD-3-Clause source boundary: `5c07b428b206e4f4e5d7e33d6f5811d7d4e6e739`
- Accepted candidate source: `6543a5a93f078f47d87aa3b8ed8ebd2024cec373`
- Candidate evidence commit: `9dbe4d9696ff4a9838fd83cb0f6f652087710f98`
- Formal candidate-acceptance authorization commit: `57d23742e60d29bf6f46d15b8f64f0497bb260cd`
- Final release source: `d34fad82781a4e8485f8907fbfd34f236fa79ad2`

## Immutable checkpoint registration

The checkpoint registry binds `isras-v2.0.1` to:

- commit `d34fad82781a4e8485f8907fbfd34f236fa79ad2`;
- frozen gate
  `tools/validation/phase-gates/validate_isras_v2_0_1_release.sh`;
- required historical branch name `dev`;
- portable environment profile;
- expected result of zero failures.

Historical validation checks out the exact accepted source in an isolated clone
on a branch named `dev` and executes the gate retained in that source tree.

## Post-release development boundary

This completion and checkpoint record is a later governance change on `dev`.
It does not move or redefine remote `main`, the signed `isras-v2.0.1` tag, tag
object `f4eacec519c96be225ffd37276cc646d3712ab0f`, or accepted source commit `d34fad82781a4e8485f8907fbfd34f236fa79ad2`.

After this record is committed and pushed, `dev` may identify the later
checkpoint-registration commit while `main` and `isras-v2.0.1` remain fixed at
the accepted v2.0.1 release source.

## Licensing boundary

ISRAS v2.0.1 is the first signed ISRAS release whose source contains the root
BSD 3-Clause `LICENSE`. The immutable v2.0.0 predecessor remains historically
unchanged and predates that licensing boundary.

## Adoption non-claim

No adopting repository silently inherits ISRAS v2.0.1. Each adopter must
perform any applicable Engineering Standards Impact Assessment, pin the exact
accepted release commit and manifest digest, and complete its own reviewed
migration or patch-adoption evidence.
