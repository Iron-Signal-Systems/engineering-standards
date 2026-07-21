# ISRAS 0.1.5 release preparation record

**Status:** RELEASE-PREPARATION CANDIDATE — NOT TAGGED OR PUBLISHED

**Preparation date:** 2026-07-21

## Source boundary

- Merged Workstream A commit:
  `428a29f776457fe523c3fc39bd84928be845eef5`
- Accepted Workstream A head:
  `570e229b5bd16a386efda88912f448dedb665d8b`
- Planned release version: `0.1.5`
- Planned signed tag: `isras-v0.1.5`
- Planned preparation branch: `release/isras-v0.1.5-preparation`

The merged commit preserved the exact accepted candidate tree and passed the
complete post-merge release-precondition campaign before this preparation began.

## Preparation scope

This change set closes the accumulated `Unreleased` changelog as `0.1.5` and
updates only the release identity and release-facing documentation:

- `VERSION`;
- validator identity metadata;
- `CHANGELOG.md`;
- repository release guidance;
- project-adoption guidance;
- 0.1.5 release notes; and
- this preparation record.

It does not change validator, release-publication, workflow, schema, or test
implementation.

## Accepted engineering boundary

The release candidate includes:

- bounded selected-Go project-command execution;
- minimum-version and multi-module Go enforcement;
- project-command evidence schema version 2;
- exact pinned govulncheck identity and per-module execution;
- streaming-protocol and typed vulnerability evidence;
- exact governed vulnerability exceptions;
- ISRAS repository identity and language-neutral core boundaries;
- versioned fail-closed documentation-impact policy and CLI enforcement;
- native and container Linux validation; and
- exact-root Git trust for bind-mounted workspaces.

## Remaining governed sequence

This preparation commit is not publication authority. Completion still requires:

1. push the exact signed preparation commit;
2. open and review a release-preparation pull request;
3. pass hosted validation on the exact preparation head;
4. merge to `dev` without rewriting the signed commit;
5. run exact clean-clone release validation on merged `dev`;
6. create and verify the signed annotated `isras-v0.1.5` tag;
7. push the exact tag;
8. produce and review the deterministic six-asset release;
9. pass read-only publication preflight;
10. publish through the controlled draft-first process; and
11. re-download and verify every remote asset and final release field.

Consumer pin or adoption changes occur only after publication and remain outside
this repository's release-preparation boundary.

## Claim boundary

This record does not claim independent review, certification, universal
production fitness, publication, or consumer adoption. No tag, GitHub Release,
release asset, consumer repository, Atlas repository, or Workstream B state is
modified by this preparation step.
