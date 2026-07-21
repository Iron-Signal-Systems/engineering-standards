# Govulncheck streaming-protocol parser

**Status:** WORKSTREAM A CANDIDATE — A4.2 PARSER ONLY — NOT RELEASED OR ADOPTABLE

## Scope

This step establishes the fail-closed parser for the pinned govulncheck streaming
JSON protocol. It does not yet resolve or execute the scanner, extend project-
command evidence, classify policy outcomes, or implement vulnerability
exceptions.

## Protocol boundary

The parser consumes a sequence of concatenated JSON values. Every value must be a
JSON object containing exactly one supported message field:

- `config`
- `progress`
- `SBOM`
- `osv`
- `finding`

The first message must be `config`, duplicate configuration messages are
rejected, and `protocol_version` is required. Unsupported top-level fields,
malformed JSON, scalar or null messages, empty messages, and messages containing
multiple fields fail closed.

## Finding classification

Finding level is determined from the first trace frame:

- function present: symbol level;
- package present without function: package level;
- module present without package or function: module level;
- otherwise: unknown level.

Unknown-level findings are recorded explicitly and are not silently interpreted
as safe.

## Deterministic summary

The parser records message counts, scanner configuration, sorted unique SBOM
roots and modules, sorted unique OSV advisory IDs, sorted unique finding advisory
IDs, and module/package/symbol/unknown finding counts.

## Validation

Synthetic tests cover valid concatenated streams and hostile boundaries without
invoking a live scanner. The live pinned-tool runner and typed evidence
integration remain later A4.2 steps.
