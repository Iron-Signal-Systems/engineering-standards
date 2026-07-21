# Project-command Go toolchain PATH correction

**Status:** CORRECTION CANDIDATE — NOT RELEASED OR ADOPTABLE

## Defect

The ISRAS 0.1.4 project-command environment replaced the caller `PATH` with a
bounded system-only path. In GitHub-hosted validation this discarded the exact Go
toolchain configured by `actions/setup-go` and allowed an older runner-provided Go
binary to be selected. Iron File Intelligence run `29737532949` exposed the
defect while executing commit `35cfe652a53b120998cf92bfedc39dc695f32cb0`.

## Correction

For Go-profile projects, the validator now resolves the active caller-selected
`go` executable before isolation, verifies that its reported version satisfies the
consuming project's `go.mod` minimum, and places only that exact toolchain
directory ahead of the existing bounded command path.

Version comparison uses Go toolchain ordering. Exact minimum versions and later
valid versions are accepted, including valid custom suffixes. Versions below the
minimum are rejected. The correction does not inherit the caller's entire `PATH`,
does not enable toolchain auto-download, and does not weaken opaque-launcher,
repository-drift, timeout, output, or evidence controls.

The complete test fixture boundary also declares a minimal `go.mod` for every
fixture that claims the Go profile. This keeps generic project-command behavior
tests valid while preserving the rule that a Go-profile project must declare its
minimum Go version.

## Child-environment correction

Go-profile child commands force `GOTOOLCHAIN=local` and `GOENV=off` after bounded
environment inheritance. Caller values cannot replace either control. The selected
Go directory remains first in the bounded child `PATH`, so a project-owned command
that invokes `go` reaches the same local executable identity that ISRAS probed.

Focused hostile regression coverage sets caller values equivalent to
`GOTOOLCHAIN=auto` and an alternate `GOENV` path, then verifies the child observes
only `local` and `off`. The same regression verifies both `command -v go` and
`go env GOVERSION` against the selected fixture toolchain. A non-Go request retains
the pre-existing inherited values, proving the correction is profile-scoped.

## Governed module parser

The correction replaces first-match line scanning with bounded parsing that supports valid comments, rejects missing, malformed, or duplicate directives, records an optional `toolchain` declaration, and enforces repository-contained regular files. The parser accepts a relative module path for later multi-module inventory use. The directive never authorizes acquisition or switching.

## Evidence schema revision

The correction advances generated project-command evidence to schema version 2
without changing the historical version 1 schema. The typed JSON and text outputs
record the selected Go executable, selected directory, exact reported version,
project minimum, optional project `toolchain` directive, fixed effective
`GOTOOLCHAIN` and `GOENV` values, and the minimum-satisfaction result.

Focused tests cover both a successful newer custom-suffix toolchain and a
below-minimum rejection. The negative case retains exact toolchain evidence with
`go_minimum_satisfied=false` and proves the project command did not execute. A
governed version 2 Go-pass example is retained beside the schemas.

## Multi-module inventory

The correction discovers every repository-owned `go.mod`, validates each through
the governed parser, rejects duplicate module identities and hostile file
boundaries, and sorts the inventory by repository-relative module-file path. The
selected Go implementation must satisfy every module minimum. A nested module
above the selected version fails before project command execution and retains
per-module negative evidence.

Evidence schema version 2 now carries the exact module set, allowing the later
mandatory vulnerability gate to prove that every discovered module was scanned.

## Repository-owned source correction

The A4 pre-implementation inventory found 23 filesystem `go.mod` files in the
Engineering Standards working tree: one source module and 22 generated copies
under `.local/validation/releases`. A raw filesystem walk would incorrectly treat
those historical evidence snapshots as active modules.

Module discovery now uses Git's tracked plus nonignored-untracked source inventory
and excludes the reserved `.local/` runtime tree. Regression tests prove that
tracked and current nonignored source modules are included while generated
validation and project-command evidence modules are excluded. This correction is
required before vulnerability scanning can claim coverage of every source module.

## Bounded Git resolution correction

The first repository-owned inventory implementation invoked `git` through the
caller's current `PATH`. Existing selected-Go regression tests intentionally
reduce `PATH` to the fake selected Go directory, so module discovery failed before
Go selection could be tested.

The corrected inventory resolves an absolute Git executable from the bounded
system directories already allowed by project-command execution and invokes it
with a minimal environment. Tests prove inventory succeeds when caller `PATH`
contains no Git executable. The established `non-regular` hostile module-path
error wording is also preserved.

## Claim boundary

This change is not an accepted ISRAS release by itself. It requires review,
complete repository validation, hosted external-consumer regression validation,
a later signed immutable release, and explicit consuming-project upgrade. It does
not alter or retroactively repair ISRAS 0.1.4 or any consuming-project evidence.

## Specialized `known_vulnerabilities` dispatch

`Execute` preserves the ordinary command path for all other operations. For `known_vulnerabilities`, it derives the exact runtime configuration from the governed evidence boundary, invokes the selected-Go and pinned-scanner orchestrator, and finalizes v2 evidence from per-module results. Failure evidence remains available when reachable findings or other post-scan policy checks fail.
