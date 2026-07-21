# Governed govulncheck exception document

**Status:** WORKSTREAM A CANDIDATE — A4.3 DECLARATION ONLY — NOT RELEASED OR ADOPTABLE

## Purpose

This step defines the only document shape that may later authorize an exact exception for a reachable Go vulnerability. Parsing the document does not yet suppress, accept, or change any scanner result.

## Exact scope

Every exception identifies one exact advisory ID, governed `go.mod` path, module path, package path, and symbol. Pointer-receiver symbols such as `(*Service).Handle` are exact valid names, while wildcard forms remain prohibited.

Wildcards, traversal, reserved `.local/` module paths, empty scope values, whitespace, backslashes, and control characters are rejected. An exception cannot apply to an advisory, module, package, or symbol that is not named exactly.

## Governance

Every exception requires substantive justification, unique compensating controls, an accountable owner, an independent approver, canonical UTC approval and expiration timestamps, an approval record, and a remediation owner, target date, and plan.

Expired records, future approvals, self-approval, duplicate exact scopes, duplicate controls, unknown fields, trailing JSON, oversized files, symbolic links, nonregular files, and repository escape fail closed.

## Claim boundary

The parser and schema alone do not permit an exception. Runtime matching, exact finding reconciliation, used/unused exception evidence, and exception-aware result policy remain later A4.3 steps.

## Govulncheck exception evidence

A passing `known_vulnerabilities` result requires typed exception-evaluation
evidence. The evidence records document presence, path, digest, schema version,
evaluation time, exact used exceptions, unused records, unexcepted reachable
findings, and unknown-finding summaries. Failure evidence retains the same
reconciliation whenever scanner execution completed successfully.
