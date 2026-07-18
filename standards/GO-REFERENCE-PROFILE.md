# Go Reference Profile

## Purpose

The Go reference profile is the first supported language profile for ISRAS. It
defines the normal Go-specific evidence used to satisfy the language-neutral core
standard.

The ISRAS validator, project bootstrap tooling, release tooling, and initial
reference workflow are implemented in Go. A consuming Go project does not import
those tools into its product runtime.

## Scope

A project selects the Go profile when Go is part of the project boundary being
validated. The profile applies to the declared Go modules, commands, generated
artifacts, tests, and dependency records identified by the project.

The project remains responsible for any non-Go components and may declare
additional profiles or project-specific validation commands when supported.

## Normal Go evidence

A conforming Go project normally provides evidence for:

- canonical formatting through `gofmt`;
- static analysis through `go vet`;
- complete package tests through `go test`;
- successful compilation through `go build`;
- deterministic module metadata through `go mod tidy -diff`;
- module integrity through `go mod verify`;
- declared Go and toolchain versions;
- reviewable `go.mod` and `go.sum` changes;
- known-vulnerability analysis through the accepted profile's selected tool;
- race, fuzz, integration, hostile-condition, or platform tests when required by
  project risk.

The exact command boundary is project-owned and committed. A project may use
repository-owned wrappers when they remain transparent, bounded, and complete.

## Project command declaration

The project shall declare the commands that satisfy each applicable profile
control. The declaration shall use argument arrays rather than an opaque shell
string unless shell behavior is itself necessary and reviewed.

Conceptual example:

```json
{
  "profile": "go",
  "commands": {
    "format_check": ["gofmt", "-l", "."],
    "static_analysis": ["go", "vet", "./..."],
    "test": ["go", "test", "./..."],
    "build": ["go", "build", "./..."],
    "module_consistency": ["go", "mod", "tidy", "-diff"],
    "module_integrity": ["go", "mod", "verify"]
  }
}
```

The schema and accepted commands are versioned release artifacts. The v1 project
pin is authoritative for command declaration, and one exact committed command may
be run through the separately governed
[`PROJECT-COMMAND-EXECUTION.md`](PROJECT-COMMAND-EXECUTION.md) boundary.

## Toolchain identity

The project shall declare and verify the Go language and toolchain versions it
uses. A toolchain change is a reviewed source change and must be validated across
the project's declared support boundary.

The profile may establish a minimum supported Go version or reject a version with
known applicable vulnerabilities. A project may require a newer version.

## Dependency boundary

ISRAS tooling shall not ordinarily add itself to a consuming project's `go.mod`
or `go.sum`. The validator is repository assurance tooling, not an application
library.

Project dependencies remain project-owned. ISRAS may require that they are:

- declared;
- integrity-verifiable;
- reviewed when changed;
- tested in the project;
- assessed for known applicable vulnerabilities;
- retained or removed through an explicit change.

## Repository layout

The profile may recommend conventional Go layouts such as `cmd/`, `internal/`,
or focused package directories. Those layouts are guidance, not universal core
requirements.

ISRAS shall validate required project artifacts and declared boundaries without
forcing one application structure onto every Go repository.

## Extension boundary

Future Go profile releases may add guidance for:

- multiple modules;
- generated source;
- database migrations;
- command-line applications;
- services and workers;
- cgo;
- cross-compilation;
- race and fuzz campaigns;
- reproducible release binaries;
- software bills of materials;
- deployment and recovery evidence.

Such additions require an accepted ISRAS release and an explicit upgrade by each
project.
