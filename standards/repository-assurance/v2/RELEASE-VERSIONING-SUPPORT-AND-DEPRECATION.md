# ISRAS v2 Release, Versioning, Support, and Deprecation

## 1. Major-version decision

ISRAS v2.0.0 is a major release because it introduces mandatory phase-entry and
phase-exit reviews, new evidence contracts, hostile authority-boundary testing,
control-maturity semantics, and a platform-wide bounded-authority invariant.
These changes alter validation contracts and supported engineering workflows.

## 2. Candidate boundary

During candidate development:

- the accepted v1 source and checkpoint history remain intact;
- the v2 normative tree is developed separately;
- schemas, templates, validators, tests, migration guidance, and acceptance plan
  are completed before release finalization;
- the repository root `VERSION` remains `1.0.1`;
- no `isras-v2.0.0` release tag is created;
- no adopting repository claims accepted ISRAS v2 compliance.

## 3. Finalization order

After the complete candidate passes acceptance:

1. record the exact accepted candidate commit;
2. update `VERSION` to `2.0.0` in a dedicated release-finalization change;
3. regenerate and verify the source manifest;
4. rerun the complete v2 candidate and release-state validators;
5. push the exact release commit;
6. create and verify the signed `isras-v2.0.0` tag;
7. publish release notes and source-manifest digest;
8. register the immutable accepted checkpoint.

The accepted release commit is the commit identified by the signed release tag.

## 4. Pinned adopter support

Publication of v2.0.0 does not silently change a repository pinned to v1.0.1.
The repository remains governed by v1.0.1 until it completes an ESIA and adopts
the exact accepted v2.0.0 release.

## 5. Compatibility and deprecation

A repository-specific extension remains compatible only if it preserves every
inherited control. Deprecation of a control, schema, validator option, or
supported workflow shall state replacement, migration, support period, and
acceptance impact.

A change that weakens a mandatory control, changes the meaning of accepted
evidence, or invalidates a supported workflow requires a major release.
