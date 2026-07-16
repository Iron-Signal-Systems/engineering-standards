# ISRAS v2.0.1 Candidate and Acceptance Plan

## Status

`CANDIDATE PREPARATION — NOT FORMALLY ACCEPTED`

## Target release

- Target version: `2.0.1`
- Proposed signed tag: `isras-v2.0.1`
- Current repository `VERSION` during candidate preparation: `2.0.0`
- Release type: patch release

The repository version remains `2.0.0` during candidate preparation. A later,
separate release-source change may set `VERSION` to `2.0.1` only after the exact
candidate source and its evidence are formally accepted for release
finalization.

## Patch-release rationale

ISRAS v2.0.1 is intended to publish the BSD 3-Clause licensing decision in the
signed release line and retain the completed v2.0.0 release/checkpoint records.

This candidate does not change:

- the ISRAS v1 normative tree;
- the ISRAS v2 normative tree;
- control semantics;
- schemas or governed templates;
- adopter validation contracts;
- mandatory engineering workflows.

The licensing and release-governance updates therefore do not require a major
or minor normative-standard release.

## Exact lineage

- Accepted ISRAS v2.0.0 release commit:
  `781246e69f8a9a382c25040f94b62dfe3b25ba89`
- Accepted ISRAS v2.0.0 signed tag: `isras-v2.0.0`
- ISRAS v2.0.0 annotated tag object:
  `a7a09a02798e2b2c905f2686820fd30890f62bc6`
- ISRAS v2.0.0 checkpoint-registration commit:
  `a1861291110efccaad9c587a99aaaf2de6f21812`
- First BSD-3-Clause source boundary:
  `5c07b428b206e4f4e5d7e33d6f5811d7d4e6e739`
- ISRAS v2.0.0 signing-key fingerprint:
  `SHA256:FiH+Jk7HHrNkvDEQTehI/aCfkmKpivtsqmkl5TmmMSE`

The exact v2.0.1 candidate source will be the later commit that contains this
plan, its candidate validator and gate, synchronized documentation, tests, and
the regenerated source manifest. That identity is recorded after the commit
exists; this plan intentionally contains no self-referential placeholder.

## Candidate scope

The candidate shall:

1. retain the complete BSD 3-Clause license text in root `LICENSE`;
2. retain `BSD-3-Clause` as the explicit repository licensing decision;
3. record `5c07b428b206e4f4e5d7e33d6f5811d7d4e6e739` as the first exact BSD-licensed source
   boundary;
4. retain the immutable v2.0.0 release and checkpoint identities;
5. add a v2.0.1-specific candidate validator and exact-pushed-source gate;
6. add regression coverage for the v2.0.1 candidate boundary;
7. preserve all accepted normative standard trees unchanged;
8. regenerate and verify `SOURCE-SHA256SUMS.txt`.

## Candidate acceptance criteria

The exact pushed candidate commit must satisfy all of the following:

- root `VERSION` remains `2.0.0`;
- BSD-3-Clause license text and scope are complete and internally consistent;
- the v2.0.0 release, tag object, signing fingerprint, and checkpoint remain
  exact;
- no files under `standards/repository-assurance/v1/` or
  `standards/repository-assurance/v2/` differ from the accepted v2.0.0 release;
- no schemas, governed templates, reusable workflows, or integration guides
  differ from the accepted v2.0.0 release;
- source-manifest verification passes;
- current release-state validation passes;
- portable and fresh-clone validation pass;
- the complete integration-enabled regression suite passes;
- accepted v1.0.1 and v2.0.0 historical checkpoints revalidate;
- retained evidence identifies the exact pushed candidate commit and all
  campaign artifacts by SHA-256;
- formal candidate acceptance is recorded in a later governed commit.

## Release-finalization sequence

After formal candidate acceptance:

1. create a separate release-source change declaring `VERSION` `2.0.1`;
2. add the v2.0.1 release-finalization record and frozen release gate;
3. regenerate the source manifest and rerun the exact pushed-source campaign;
4. create and verify the SSH-signed annotated `isras-v2.0.1` tag;
5. fast-forward `main` without force to the exact release commit;
6. verify remote `dev`, remote `main`, and the peeled tag target converge;
7. register the immutable v2.0.1 checkpoint in a later governed `dev` commit.

## Non-claims

This candidate does not claim:

- that ISRAS v2.0.1 is accepted or released;
- that `main` or any release tag has moved;
- that adopters automatically receive the BSD-licensed release;
- independent third-party review;
- runtime performance, security, or operational-readiness assurance beyond the
  repository-governance validation actually executed.
