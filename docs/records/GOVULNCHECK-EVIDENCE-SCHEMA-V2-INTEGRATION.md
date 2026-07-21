# Govulncheck evidence schema v2 integration

**Status:** WORKSTREAM A CANDIDATE — A4.2 EVIDENCE OUTPUT ONLY — NOT RELEASED OR ADOPTABLE

## Scope

This step attaches the accepted typed govulncheck evidence structures to project-command evidence v2 and synchronizes JSON, text, schema, examples, tests, and documentation. It does not yet route the runtime `known_vulnerabilities` command through the per-module scanner runner.

## JSON evidence

Project-command `Result` now permits an additive `govulncheck` section containing exact scanner identity, per-module results, bounded streams, protocol configuration, SBOM data, advisory identities, and module/package/symbol/unknown finding counts.

The field remains optional during this intermediate step. The runtime-dispatch step will require it for the mandatory `known_vulnerabilities` operation once the runner is attached.

## Text evidence

The text renderer emits the same scanner identity, module coverage, protocol summary, advisory identities, finding counts, stream digests, and sanitized per-module stream output.

## Schema compatibility

Evidence schema v1 remains unchanged. Evidence schema v2 gains additive definitions for govulncheck evidence and a dedicated governed pass example. Existing non-vulnerability v2 evidence remains valid.

## Remaining A4.2 work

The next controlled step specializes runtime dispatch so `known_vulnerabilities` invokes the exact verified scanner for every governed module, projects the results, finalizes v2 JSON/text evidence, and fails closed on scanner, protocol, coverage, or reachable-finding errors.
