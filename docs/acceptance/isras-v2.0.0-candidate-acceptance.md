# ISRAS v2.0.0 Candidate Formal Acceptance Record

## Decision

- **Status:** Accepted for release finalization
- **Decision date:** 2026-07-16
- **Accepted candidate source commit:** `4aff00dfdc88154390252898210abc336fa8b2fc`
- **Candidate evidence commit:** `b0c982221acde7873307d010aca73ed2e386eb99`
- **Accepted predecessor:** `c379417720faa595fa5cb89a1dfdb2259d6cb95e`
- **Required source branch:** `dev`
- **Repository version at decision:** `1.0.1`
- **Proposed release tag:** `isras-v2.0.0` — not yet created
- **Validation gate:** `tools/isras/validate_isras_v2_candidate.py`
- **Environment profile:** `portable`
- **Runner identity:** `John’s Mac mini (Johns-Mac-mini.local)`
- **Correctness result:** `PASS`
- **Decision authority:** Iron Signal Systems repository owner

This record becomes canonical when committed and pushed to `dev`. The accepted
candidate source remains the exact source commit shown above. The later commit
that adds this decision record is governance evidence about the accepted
candidate and does not replace or alter the accepted source.

## Accepted candidate boundary

The decision accepts the ISRAS v2.0.0 candidate for release finalization,
including:

- mandatory exact-release inheritance and non-weakening governance;
- Engineering Standards Impact Assessment requirements;
- mandatory phase-entry and phase-exit compliance reviews;
- the `DOCUMENTED`, `IMPLEMENTED`, `VALIDATED`, and `ACCEPTED` control
  maturity model;
- the bounded-authority and privilege-non-propagation invariant;
- the normative prohibition on unrestricted execution contexts;
- hostile testing of authority, trust, lifecycle, and operational boundaries;
- minimum standards evidence for accepted phases;
- machine-readable schemas, templates, validators, regression tests, and
  migration guidance;
- preservation of the accepted ISRAS v1 normative history.

## Validation result

The exact pushed candidate passed:

- portable validation;
- fresh-clone and remote-completeness validation;
- isolated historical revalidation of accepted ISRAS v1.0.1;
- integration-enabled ISRAS regression testing;
- complete ISRAS v2 candidate validation with 40 PASS and 0 FAIL;
- source-manifest verification against the exact candidate source.

## Evidence

- **Evidence directory:** `docs/acceptance/evidence/isras-v2.0.0-candidate/`
- **Evidence JSON:** `docs/acceptance/evidence/isras-v2.0.0-candidate/acceptance-evidence.json`
- **Evidence JSON SHA-256:** `0e4516f76032008075a844ddc43cb44fdb90ae09ab31b9af113b32923f082cd7`
- **Candidate evidence commit:** `b0c982221acde7873307d010aca73ed2e386eb99`
- **Candidate source manifest SHA-256:** `b095d5802fd27f162b6cbe6ffdaf9279f15c966c3bdfa126601f38c787947c6e`
- **Campaign started:** `2026-07-16T07:57:34+00:00`
- **Campaign finished:** `2026-07-16T07:58:02.476901+00:00`

## Warnings and limitations

- Resource observations were not recorded for this documentation and validator
  campaign.
- Performance budgets, security findings, and operational readiness were not
  evaluated as product-runtime claims.
- Independent third-party review is not claimed.

## Non-claims

- This decision does not change `VERSION` from `1.0.1`.
- This decision does not create or authorize movement of an
  `isras-v2.0.0` tag.
- This decision does not yet register an accepted ISRAS v2.0.0 checkpoint.
- This decision does not silently alter any adopting repository's pinned ISRAS
  release.
- This decision does not prove that adopting products are production ready or
  free of vulnerabilities.

## Release-finalization authorization

This formal candidate acceptance authorizes a separate, reviewable
release-finalization change that shall:

1. change `VERSION` from `1.0.1` to `2.0.0`;
2. create or update the final ISRAS v2.0.0 release and acceptance record;
3. regenerate and verify `SOURCE-SHA256SUMS.txt`;
4. rerun complete validation on the exact pushed finalization commit;
5. create and verify the signed `isras-v2.0.0` tag at the exact validated
   release commit;
6. register the accepted immutable ISRAS v2.0.0 checkpoint in a subsequent
   governed change;
7. publish the exact release commit and source-manifest digest.

Repository-by-repository adoption remains deliberate. Each adopter must perform
an ESIA and pin the exact accepted release. Iron Atlas must first complete its
current pinned ISRAS v1.0.1 `RECORDED` adoption.
