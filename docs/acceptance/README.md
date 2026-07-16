# Acceptance Records

This directory retains sanitized, durable acceptance plans, historical records,
evidence identities, release-completion records, and correction records.

Large or restricted logs remain in approved evidence storage or CI artifacts.
Durable v1 and v2 records identify artifacts by SHA-256 and exact source
boundary. New v3 candidate evidence uses SHA-512 as its primary artifact-binding
digest while retaining SHA-256 where accepted history or an external tool
contract requires it.

For v1.0.1 and later, the verified signed annotated release tag is the
authoritative acceptance-decision object. In-tree plans and candidate records
predeclare and preserve the governing criteria; later completion records retain
verified remote and cryptographic identities without redefining the immutable
release source.

## Accepted historical records

- [ISRAS v1.0.0 formal acceptance](isras-v1.0.0.md)
- [ISRAS v1.0.0 tag correction](isras-v1.0.0-tag-correction.md)
- [ISRAS v1.0.0 release finalization](isras-v1.0.0-release-finalization.md)
- [ISRAS v2.0.0 candidate formal acceptance](isras-v2.0.0-candidate-acceptance.md)
- [ISRAS v2.0.0 release-source finalization record](isras-v2.0.0-release-finalization.md)
- [ISRAS v2.0.0 release completion and checkpoint](isras-v2.0.0-release-completion.md)
- [ISRAS v2.0.1 candidate formal acceptance](isras-v2.0.1-candidate-acceptance.md)
- [ISRAS v2.0.1 release-source finalization record](isras-v2.0.1-release-finalization.md)
- [ISRAS v2.0.1 release completion and checkpoint](isras-v2.0.1-release-completion.md)

## Development-only future candidate

- [ISRAS v3.0.0 assurance-hardening plan](isras-v3.0.0-plan.md)
- [ISRAS v3.0.0 candidate change classification](isras-v3.0.0-change-classification.json)

The v3 plan is not accepted, released, or inherited by adopters.

## Retained acceptance plans

- [ISRAS v1.0.1 acceptance plan](isras-v1.0.1-plan.md)
- [ISRAS v2.0.0 candidate and acceptance plan](isras-v2.0.0-plan.md)
- [ISRAS v2.0.1 candidate and acceptance plan](isras-v2.0.1-plan.md)

## Retained candidate evidence

- [`isras-v2.0.1-candidate/`](evidence/isras-v2.0.1-candidate/) — accepted and released as `isras-v2.0.1`
- [`isras-v2.0.0-candidate/`](evidence/isras-v2.0.0-candidate/)
