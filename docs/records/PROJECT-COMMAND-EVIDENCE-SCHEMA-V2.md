# Project-command evidence schema v2

**Status:** WORKSTREAM A CANDIDATE — NOT RELEASED OR ADOPTABLE

## Revision purpose

Project-command evidence schema version 2 adds explicit selected-Go identity and
minimum-version decision evidence. It preserves the complete version 1 command,
timeout, output, repository-drift, validator, target, environment, redaction, and
stream boundary.

The historical `schemas/isras-project-command-execution-v1.schema.json` file
remains unchanged. The new governed contract is
`schemas/isras-project-command-execution-v2.schema.json`.

## Added Go evidence

The optional `go_toolchain` object is emitted for every Go-profile command and
contains the selected executable, selected directory, selected version, project
minimum, optional toolchain directive, fixed local/off environment, and minimum
comparison result.

A successful Go-profile command records complete identity and a true result. A
below-minimum selection records the discovered identity and false result, then
fails before project command execution.

## Boundary

This record does not accept PR #35, create a release, modify a consumer, or
authorize Workstream B.

## Govulncheck evidence output

Project-command evidence v2 has an additive typed `govulncheck` section. The section is optional for non-vulnerability commands and records exact scanner identity plus one reconciled module result per governed `go.mod`. The runtime-dispatch step makes this section mandatory for `known_vulnerabilities`.

## Govulncheck exception evidence

A passing `known_vulnerabilities` result requires typed exception-evaluation
evidence. The evidence records document presence, path, digest, schema version,
evaluation time, exact used exceptions, unused records, unexcepted reachable
findings, and unknown-finding summaries. Failure evidence retains the same
reconciliation whenever scanner execution completed successfully.
