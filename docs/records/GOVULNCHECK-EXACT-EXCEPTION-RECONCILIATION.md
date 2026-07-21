# Govulncheck exact exception reconciliation

**Status:** WORKSTREAM A CANDIDATE — A4.3 MATCHING ONLY — NOT RELEASED OR ADOPTABLE

## Purpose

This step preserves exact symbol-level finding identities from the govulncheck stream and reconciles them against the governed exception document. It does not yet permit an exception to change project-command status.

## Exact finding identity

For every symbol-level finding, the parser retains advisory ID, vulnerable module path, vulnerable package path, canonical symbol, and fixed version when reported.

The first trace frame is authoritative. Receiver and function are combined exactly: `*Service` plus `Handle` becomes `(*Service).Handle`. Duplicate traces for the same advisory and exact scope are collapsed into one reconciliation entry with an occurrence count and sorted fixed-version set.

## Reconciliation

An exception is used only when advisory ID, governed `go.mod` path, vulnerable module, vulnerable package, and canonical symbol all match exactly.

The result contains deterministic used exceptions, unused exceptions, reachable findings without an exception, and unknown-level finding summaries. A mismatch in any one scope field leaves the exception unused and the finding unexcepted.

## Fail-closed boundaries

Reconciliation rejects duplicate governed module results, duplicate exact exception scopes, missing module identities, reachable findings without exact identity, disagreement between symbol-level counts and retained detailed findings, and unsupported exception schema versions.

## Claim boundary

This step cannot accept a vulnerability. Runtime policy, used/unused evidence integration, exception-aware pass/fail behavior, and hosted validation remain later A4.3 work.

## Exception-aware vulnerability result policy

The mandatory Go vulnerability operation evaluates exact reconciliation after
all module scans. Unknown findings, reachable findings without an exact governed
exception, and unused exception records fail. A reachable finding may pass only
when one valid exception matches its advisory, governed `go.mod`, vulnerable
module, vulnerable package, and canonical symbol exactly.

The exception document is optional at
`.isras/govulncheck-exceptions.json`; absence means zero exceptions. A present
document is hashed and retained in evidence. Exceptions never suppress or alter
the original govulncheck stream.
