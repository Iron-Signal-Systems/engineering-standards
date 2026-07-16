# ISRAS v2.0.1 Candidate Formal Acceptance Record

## Decision

- **Status:** Accepted for release finalization
- **Decision date:** 2026-07-16
- **Accepted candidate source commit:** `6543a5a93f078f47d87aa3b8ed8ebd2024cec373`
- **Candidate evidence commit:** `9dbe4d9696ff4a9838fd83cb0f6f652087710f98`
- **Accepted predecessor:** `781246e69f8a9a382c25040f94b62dfe3b25ba89`
- **Required source branch:** `dev`
- **Repository version at decision:** `2.0.0`
- **Proposed release tag:** `isras-v2.0.1` — not yet created
- **Validation gate:**
  `tools/validation/phase-gates/validate_isras_v2_0_1_candidate.sh`
- **Environment profile:** `portable`
- **Runner identity:** `John's Mac (Johns-Mac-mini.local)`
- **Correctness result:** `PASS`
- **Decision authority:** Iron Signal Systems repository owner

This record becomes canonical when committed and pushed to `dev`. The accepted
candidate source remains the exact source commit shown above. The later commit
that adds this decision record is governance evidence about the accepted
candidate and does not replace or alter the accepted source.

## Accepted candidate boundary

The decision accepts the ISRAS v2.0.1 BSD-licensed patch candidate for release
finalization, including:

- the complete BSD 3-Clause License in root `LICENSE`;
- the explicit `BSD-3-Clause` repository licensing decision;
- the first exact BSD-licensed source boundary at
  `5c07b428b206e4f4e5d7e33d6f5811d7d4e6e739`;
- preservation of the immutable ISRAS v2.0.0 release, tag, and checkpoint;
- preservation of the ISRAS v1 and v2 normative trees without change;
- preservation of schemas, governed templates, integration guides, and reusable
  workflows without change;
- the v2.0.1 candidate validator, exact-pushed-source gate, and regression
  coverage;
- synchronized licensing, release-governance, acceptance, validation, and
  source-manifest records.

## Validation result

The exact pushed candidate passed:

- source-manifest verification;
- current release-state validation;
- portable validation;
- fresh-clone and remote-completeness validation;
- integration-enabled regression testing;
- isolated historical revalidation of accepted ISRAS v1.0.1;
- isolated historical revalidation of accepted ISRAS v2.0.0;
- complete ISRAS v2.0.1 candidate validation with **43 PASS and 0 FAIL**.

## Evidence

- **Evidence directory:**
  `docs/acceptance/evidence/isras-v2.0.1-candidate/`
- **Evidence JSON:**
  `docs/acceptance/evidence/isras-v2.0.1-candidate/acceptance-evidence.json`
- **Evidence JSON SHA-256:** `42d7dce7500929647af001f47bbbdf30ae7bef88c598d0aba8edd2424564d2b9`
- **Candidate evidence commit:** `9dbe4d9696ff4a9838fd83cb0f6f652087710f98`
- **Candidate source manifest SHA-256:** `e2b6488a7f670b0c81d873478154d03438a9c5f21a8bf05010863fbe1e4fd7e8`
- **Campaign started:** `2026-07-16T10:16:04.797950+00:00`
- **Campaign finished:** `2026-07-16T10:16:44.611439+00:00`
- **Candidate source commit:** `6543a5a93f078f47d87aa3b8ed8ebd2024cec373`

## Warnings and limitations

- GitHub SSH over port 443 was used because earlier non-authoritative attempts
  over port 22 timed out.
- Resource observations were not recorded for this documentation and validator
  campaign.
- Performance budgets, security findings, and operational readiness were not
  evaluated as product-runtime claims.
- Independent third-party review is not claimed.

## Non-claims

- This decision does not change `VERSION` from `2.0.0`.
- This decision does not create, sign, or authorize movement of
  `isras-v2.0.1`.
- This decision does not move `main`.
- This decision does not register an accepted ISRAS v2.0.1 checkpoint.
- This decision does not modify the immutable ISRAS v2.0.0 release or tag.
- This decision does not silently alter any adopting repository's pinned ISRAS
  release.
- This decision does not prove that adopting products are production ready or
  free of vulnerabilities.

## Release-finalization authorization

This formal candidate acceptance authorizes a separate, reviewable
release-finalization change that shall:

1. change `VERSION` from `2.0.0` to `2.0.1`;
2. add the ISRAS v2.0.1 release-finalization record and frozen release gate;
3. regenerate and verify `SOURCE-SHA256SUMS.txt`;
4. rerun complete validation on the exact pushed release-source commit;
5. create and verify the SSH-signed annotated `isras-v2.0.1` tag at the exact
   validated release commit;
6. fast-forward `main` without force to that same exact release commit;
7. verify remote `dev`, remote `main`, and the peeled tag target converge;
8. publish the exact release commit, source-manifest digest, evidence digest,
   decision authority, and validation result;
9. register the immutable ISRAS v2.0.1 checkpoint in a subsequent governed
   `dev` commit.

Repository-by-repository adoption remains deliberate. Each adopter must
complete any applicable Engineering Standards Impact Assessment and pin the
exact accepted release.
