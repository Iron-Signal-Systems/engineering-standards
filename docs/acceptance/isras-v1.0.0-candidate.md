# ISRAS v1.0.0 Acceptance Candidate

## Status

Candidate only. This document does not record formal acceptance.

## Candidate boundary

The candidate must contain the authoritative ISRAS definition, scope,
native-first execution model, secure-development lifecycle, repository and
workflow governance, environment and toolchain declarations, source-manifest
verification, historical validation, evidence schema, release governance,
recovery, specialized-lab, and project-profile requirements.

## Required validation

Before acceptance:

1. merge the complete hardening change to `dev`;
2. verify the exact commit is pushed;
3. run portable validation on the approved Arch WSL, another Linux or macOS
   development host, and Windows PowerShell;
4. run the hosted native OS matrix;
5. run `tools/validation/phase-gates/validate_isras_v1_candidate.sh` on `dev`;
6. generate an acceptance-evidence JSON record with an explicit runner identity;
7. verify evidence against the schema;
8. record the checkpoint using its exact 40-character commit;
9. create a signed annotated `isras-v1.0.0` tag, or record a time-bounded signing
   exception;
10. promote the exact accepted commit to `main`.

## Required non-claims

Initial ISRAS v1 acceptance does not prove independent human review, production
readiness of adopting products, absence of vulnerabilities, or completion of
product-specific canonical and specialized campaigns.
