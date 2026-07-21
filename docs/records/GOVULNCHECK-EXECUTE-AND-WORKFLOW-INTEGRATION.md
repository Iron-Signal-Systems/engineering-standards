# Govulncheck Execute and hosted-workflow integration

**Status:** WORKSTREAM A CANDIDATE — A4.2 COMPLETE CANDIDATE — NOT RELEASED OR ADOPTABLE

## Scope

This step routes the mandatory Go-profile `known_vulnerabilities` operation through the accepted govulncheck runtime boundary and stages the exact governed runtime inputs in the reusable hosted workflow.

It does not implement vulnerability exceptions, commit, push, merge, tag, release, modify IFI or Atlas, or begin Workstream B.

## Execute specialization

`Execute` now delegates through an injectable internal dispatcher. Ordinary commands retain the established execution path. `known_vulnerabilities` instead:

- requires the Go profile and exact `["govulncheck", "./..."]` declaration;
- reads the runtime tool configuration only from the target evidence boundary;
- invokes the selected-Go and exact-tool runtime orchestrator;
- records selected-Go and typed per-module govulncheck evidence;
- treats per-module streams as authoritative rather than inventing a single aggregate process stream;
- finalizes v2 JSON and text evidence for both pass and failure;
- fails on scanner, protocol, coverage, mutation, timeout, output, reachable-finding, or unknown-finding errors.

A passing `known_vulnerabilities` document must contain the typed `govulncheck` section. Early failures may omit it when exact scanner evidence could not yet be established.

## Runtime configuration boundary

The staged configuration path is:

```text
.local/isras/runtime/tool-versions.json
```

The path is derived from the governed evidence directory, must remain beneath the target repository, and may not contain symbolic-link or non-directory parents. The exact scanner remains:

```text
.local/tools/bin/govulncheck
```

## Hosted workflow

The reusable consumer workflow already installs and verifies the exact scanner from `standard/validation/tool-versions.json`. It now also copies that exact configuration into the target runtime evidence boundary with private permissions before project-command execution.

## Validation

Synthetic integration tests execute the complete dispatcher with injected runtime results, prove ordinary commands cannot enter the specialized path, and retain typed failure evidence for reachable findings.

A guarded live test creates a temporary governed Go project, copies the real pinned scanner, stages the real tool-version configuration, invokes public `Execute`, and verifies complete v2 JSON/text evidence.
