# Engineering Standards Repository Self-Assurance

## Decision

The `engineering-standards` repository governs itself through an exact accepted
ISRAS release. It does not use a floating branch, current working tree, or
`SELF` as its governing baseline.

## Current governing release

- Release: `isras-v2.0.1`
- Exact commit: `d34fad82781a4e8485f8907fbfd34f236fa79ad2`
- Source-manifest SHA-256:
  `8f54ed1e9bfee251bf89b4c5f12edf11ac1e25ef0d145ba745301f2d05787ef1`
- State: signed, released, converged, and checkpointed
- Repository adoption maturity: `RELEASE_ASSURED`

The later release-completion and checkpoint record at `08a0a514ec308f76dbf80ffdcb8caa70ce6e345f` documents the
accepted boundary without redefining the immutable release source.

## Development state

The `dev` branch may contain later governed work. Candidate work does not
silently replace the governing release. The v3 assurance-hardening tree remains
development-only until its own formal lifecycle completes.

## Why self-pinning is required

A standards repository cannot prove its governing identity by pointing to
whatever source happens to be current. Self-assurance separates the accepted
governing release from later candidate source, evidence, decisions, release
source, and future self-adoption.

## Reassessment triggers

A new self-assurance review is required before formal v3 phase entry, before
promotion of a later governing release, when self-assurance semantics change,
when branch, tag, signing, evidence authority changes, or during compromise
recovery.

## Non-claim

Self-pinning does not constitute independent review or replace separation of
duties.
