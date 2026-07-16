# ISRAS v2 Release, Versioning, Support, and Deprecation

## 1. Major-version decision

ISRAS v2.0.0 is a major release because it introduces mandatory phase-entry and
phase-exit reviews, new evidence contracts, hostile authority-boundary testing,
control-maturity semantics, mandatory standards inheritance, and a platform-wide
bounded-authority invariant. These changes alter validation contracts and
supported engineering workflows.

## 2. Accepted candidate lineage

The release source derives from the formally accepted candidate lineage:

- candidate source: `4aff00dfdc88154390252898210abc336fa8b2fc`;
- retained candidate evidence commit: `b0c982221acde7873307d010aca73ed2e386eb99`;
- formal candidate-acceptance authorization commit: `24e911b7c4a63735bcef9b4b84ab9b62ace10298`;
- accepted predecessor: `c379417720faa595fa5cb89a1dfdb2259d6cb95e`.

The accepted v1 source, normative tree, tags, and checkpoint history remain
intact.

## 3. Release-source boundary

The ISRAS v2.0.0 release source:

- declares repository root `VERSION` as `2.0.0`;
- contains the complete v2 normative tree, schemas, templates, validators,
  tests, migration guidance, acceptance lineage, and release-finalization
  record;
- preserves the accepted v1 normative tree unchanged;
- uses `tools/isras/validate_isras_v2_release.py` and
  `tools/validation/phase-gates/validate_isras_v2_release.sh` as the frozen v2
  release-source gates;
- does not silently alter any adopting repository's pinned release.

The exact release commit is the commit identified by the verified signed
annotated `isras-v2.0.0` tag after remote `dev`, remote `main`, and the peeled
tag target converge.

## 4. Release completion order

Release finalization is complete only after:

1. the exact release-source commit is pushed to `dev`;
2. the complete exact-commit validation campaign passes from the canonical
   remote;
3. the SSH-signed annotated `isras-v2.0.0` tag is created and verifies;
4. the tag peels to the exact validated release commit;
5. `main` is fast-forwarded without force to that same commit;
6. remote `dev`, remote `main`, and the peeled tag target are independently
   verified as identical;
7. the tag annotation or release metadata publishes the exact commit,
   source-manifest digest, evidence digest, decision authority, and validation
   result;
8. the immutable accepted v2.0.0 checkpoint is registered in a subsequent
   governed change.

The signed annotated tag is the authoritative acceptance-decision object. No
later source commit is required merely to describe release acceptance.

## 5. Pinned adopter support

Publication of v2.0.0 does not silently change a repository pinned to v1.0.1.
The repository remains governed by v1.0.1 until it completes an applicable ESIA
and adopts the exact accepted v2.0.0 release.

## 6. Compatibility and deprecation

A repository-specific extension remains compatible only if it preserves every
inherited control. Deprecation of a control, schema, validator option, or
supported workflow shall state replacement, migration, support period, and
acceptance impact.

A change that weakens a mandatory control, changes the meaning of accepted
evidence, or invalidates a supported workflow requires a major release.
