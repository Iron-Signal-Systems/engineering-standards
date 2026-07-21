# Govulncheck pinned-tool boundary

**Status:** WORKSTREAM A CANDIDATE — A4.1 ONLY — NOT RELEASED OR ADOPTABLE

## Decision

Go-profile projects declare the vulnerability command exactly as:

```json
["govulncheck", "./..."]
```

The declaration cannot embed `go run`, a version, a wrapper, or a different
package scope. The approved command package and version are governed by
`validation/tool-versions.json`.

## Acquisition

Hosted workflows perform one explicit network-enabled acquisition step before
project-command execution. They read the package and version from the governed
file, install into `.local/tools/bin` with `GOTOOLCHAIN=local` and `GOENV=off`,
and verify the binary's embedded command path, module root, and exact version.

Project-command execution does not install, upgrade, or silently replace the
tool.

## Validation in this step

This candidate adds project-pin parser tests for implicit acquisition and changed
scope, verifies the currently installed local tool identity, checks workflow
synchronization, and reruns project-pin and project-command regression suites.

## Remaining A4 work

This step does not yet scan every discovered module, classify reachable findings,
record per-module scan output, or implement governed vulnerability exceptions.
Those remain later A4 substeps.
