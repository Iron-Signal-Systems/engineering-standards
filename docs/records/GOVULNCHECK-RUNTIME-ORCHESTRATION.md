# Govulncheck runtime orchestration and pre-exception policy

**Status:** WORKSTREAM A CANDIDATE — A4.2 ORCHESTRATION ONLY — NOT RELEASED OR ADOPTABLE

## Scope

This step establishes the internal runtime orchestration that joins the accepted
Go-toolchain selection, exact scanner identity, deterministic per-module runner,
typed evidence projection, and pre-exception finding policy.

It does not yet modify `Execute`, stage the authoritative tool-version
configuration for consuming projects, run a project command through the new
dispatch path, implement vulnerability exceptions, or change release state.

## Exact orchestration

The orchestrator:

1. requires a nonnil context and absolute tool-version configuration path;
2. resolves the target repository root;
3. selects the exact caller-selected Go toolchain and governed module inventory;
4. derives the scanner path only as
   `.local/tools/bin/govulncheck` beneath the target repository;
5. verifies that exact already-acquired binary against the governed
   configuration through the selected Go executable;
6. runs every governed module through the accepted per-module runner;
7. reconciles the complete run against the selected module inventory;
8. projects deterministic typed evidence;
9. evaluates findings independently from the scanner process exit code.

The orchestrator has no installation, download, upgrade, `go run`, wrapper, or
caller-PATH fallback behavior.

## Pre-exception finding policy

Before governed exceptions exist:

- module-level findings are recorded and do not by themselves fail;
- package-level findings are recorded and do not by themselves fail;
- symbol-level findings are treated as reachable and fail;
- unknown-level findings fail closed;
- reachable advisory identities are sorted and included in the failure;
- successful scanner exit cannot override finding-policy failure.

A4.3 will introduce the only governed mechanism capable of accepting an exact
reachable advisory under bounded scope, approval, expiration, compensating
controls, and remediation requirements.

## Remaining A4.2 integration

The next controlled step will:

- stage the authoritative validator tool-version configuration for self and
  consuming-project workflows;
- route `known_vulnerabilities` through this orchestrator inside `Execute`;
- attach selected-Go and govulncheck evidence to finalized v2 JSON and text;
- require the `govulncheck` section for the specialized command;
- run the complete path through synthetic and live project-command validation.
