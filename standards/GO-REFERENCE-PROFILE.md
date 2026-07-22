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

## Minimum-version toolchain contract

The `go` directive in a consuming project's `go.mod` establishes the minimum Go
toolchain accepted by the ISRAS Go profile. It does not require byte-for-byte or
patch-exact equality with the declared version.

A selected `go1.25.12` toolchain satisfies `go 1.25.12`; so do later compatible
versions such as `go1.25.13`, `go1.26.0`, and valid custom-suffix builds such as
`go1.26.5-X:nodwarf5`. A toolchain below the declared minimum is rejected before
project command execution. Evidence and command output retain the exact selected
version so acceptance remains truthful without unnecessarily blocking newer
compatible toolchains.

## Child toolchain isolation

Go-profile project commands always receive `GOTOOLCHAIN=local` and `GOENV=off`.
These values are fixed by ISRAS after processing the bounded inherited environment,
so caller settings cannot authorize a toolchain download, a silent toolchain
switch, or an alternate Go environment file.

The selected Go executable directory is placed first in the bounded command
`PATH`. This binds a child command's `go` lookup to the same local toolchain
identity that was checked against the project minimum.

## Governed module-file parsing

ISRAS reads each declared module's `go.mod` through a bounded, fail-closed parser. The module file must be repository-contained, non-symbolic, regular, and within the governed size limit. Missing, malformed, and duplicate `go` directives are rejected; valid line comments, block comments, and quoted directive values are handled deliberately.

An optional `toolchain` directive is recorded but never authorizes download or switching. Execution remains fixed to `GOTOOLCHAIN=local` and `GOENV=off`.

## Go toolchain evidence

Every Go-profile project-command evidence record includes a version 2
`go_toolchain` object. The JSON and text forms identify the exact selected Go
executable, its directory and reported version, the project's declared minimum,
the optional `toolchain` directive, the fixed `GOTOOLCHAIN=local` and `GOENV=off`
values, and whether the selected version satisfied the minimum.

Negative evidence is retained when a selected toolchain is below the project
minimum. The evidence shows the selected identity and false comparison result, and
the project command does not execute.

## Complete module inventory

A project declaring the Go profile must contain a governed root `go.mod`. ISRAS
enumerates repository-owned source through Git using tracked paths plus untracked
paths that are not ignored. It excludes the reserved `.local/` runtime tree and
validates every resulting `go.mod` through the governed parser. Generated release,
validation, project-command, and local-tool evidence cannot enter the module set.

The inventory is sorted by repository-relative `go.mod` path and rejects duplicate
module identities, missing root modules, symbolic or non-regular module files, and
unreadable or malformed declarations. The selected Go version must satisfy every
module minimum. The project-level minimum is the highest minimum found across the
inventory; the project-level optional toolchain value reflects the root module,
while each module retains its own directive in evidence.

Adding or removing a module changes the generated inventory evidence
automatically. Stale root-only evidence is not permitted.

## Bounded Git inventory dependency

Repository-owned module enumeration invokes an absolute Git executable selected
from the same bounded system directories permitted by project-command execution.
It does not depend on, copy, or expand the caller's `PATH`. The Git subprocess
receives a minimal deterministic environment and cannot displace the separately
selected Go executable.

## Pinned govulncheck declaration

Every Go-profile pin declares `known_vulnerabilities` exactly as
`["govulncheck", "./..."]`. The tool's approved package and version remain
governed by `validation/tool-versions.json`. Explicit acquisition and exact build
identity verification occur before project-command execution; the project command
itself never downloads, installs, or changes the scanner.

## Govulncheck streaming-protocol parsing

The Go profile parses pinned govulncheck JSON as a sequence of concatenated
message objects rather than JSON Lines or one aggregate document. The first
message must be configuration, every message must contain exactly one supported
field, and malformed, ambiguous, unsupported, or unknown message boundaries fail
closed. Finding reachability is classified from the first trace frame as module,
package, symbol, or unknown.

## Exact govulncheck binary identity

Before any mandatory Go vulnerability scan, the implementation loads the
approved command package and exact version from the governed tool-version file
and verifies the already acquired binary through the selected Go executable's
`version -m` inspection. The command package, module root, and exact version must
match. The scanner path must be absolute, regular, executable, and nonsymlink,
and its SHA-256 digest is recorded. Missing or mismatched tools fail without
installation or fallback.

## Per-module govulncheck execution

The exact verified scanner runs once for every repository-owned Go module in
deterministic `go.mod` path order. Each invocation uses `-json ./...` from the
module directory with the selected Go directory first in a bounded PATH,
`GOTOOLCHAIN=local`, `GOENV=off`, isolated caches, bounded execution, process-
tree termination, protocol validation, and repository-mutation detection.
Successful process exit is not treated as sufficient scan evidence.

## Typed govulncheck evidence projection

The verified scanner identity and every per-module result are projected into deterministic typed evidence only after exact coverage reconciliation with the governed module inventory. Missing, duplicate, ungoverned, or identity-drifting module results fail before JSON or text evidence is emitted.

## Govulncheck evidence schema v2

Go vulnerability evidence v2 records the exact verified scanner identity and one deterministic result for every governed Go module. JSON and text outputs include protocol configuration, advisory identities, SBOM data, finding levels, timing, boundary flags, and bounded per-module streams. Evidence v1 remains unchanged.

## Govulncheck runtime orchestration

The Go profile joins selected-toolchain enforcement, exact scanner identity,
per-module execution, protocol parsing, and typed evidence through one internal
orchestrator. The scanner executable is derived only from the target repository's
`.local/tools/bin/govulncheck` path and is never installed or substituted during
project-command execution.

## Mandatory vulnerability runtime dispatch

The Go profile routes `known_vulnerabilities` through the specialized per-module govulncheck runtime. Ordinary command execution is not used for this operation. A passing result requires selected-Go evidence and typed scanner evidence covering every governed module.

## Governed vulnerability exceptions

A reachable vulnerability may be considered for exception only through the versioned govulncheck exception document. Every record must name the exact advisory, `go.mod`, module, package, and symbol; document justification and compensating controls; identify an accountable owner and independent approver; expire at a canonical UTC time; and carry a remediation owner, target date, and plan. Wildcards, broad scopes, expired approvals, self-approval, and unmatched records are prohibited.

## Exact exception reconciliation

Govulncheck exceptions match only an exact symbol-level finding identity: advisory ID, governed `go.mod` path, vulnerable module, vulnerable package, and canonical symbol. Receiver-qualified methods use the protocol receiver plus function name, including pointer forms such as `(*Service).Handle`. Duplicate traces may increase an occurrence count but cannot broaden the scope of one exception.

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

## Additive profile boundary

The Go reference profile is additive to the language-neutral ISRAS core. It
translates core outcomes into Go-specific toolchain, module, formatting, testing,
build, and vulnerability-validation evidence. It does not redefine ISRAS as a Go
product and does not make Go a universal requirement for Iron Signal Systems
repositories.

A non-Go or mixed-language repository remains eligible for governance when an
accepted profile and project declaration provide equivalent, reviewable evidence
for the applicable core requirements.

## Bounded Git trust for module inventory

Repository-owned Go module discovery invokes the bounded system Git executable
with caller global and system configuration disabled. When Git ownership
protection applies to a mounted validation workspace, the command may declare
only the exact cleaned target repository root through command-scoped
`safe.directory`.

Wildcard trust, parent-directory trust, inherited caller trust configuration,
and disabling Git ownership protection are prohibited. The trust declaration
authorizes Git to inspect the exact repository; it does not broaden which module
paths enter the governed inventory.

## Hosted govulncheck executable ownership

The Go profile's hosted `govulncheck` binary is validator-owned runtime tooling.
It shall be installed outside the consuming repository, verified against the
exact governed module and version, and supplied to the release validator through
the bounded hosted-tool interface.

The consuming repository remains clean before the first project command.
Projects are not required to ignore validator tool directories, and the standard
must not use `.gitignore` changes to hide validator-created files.
