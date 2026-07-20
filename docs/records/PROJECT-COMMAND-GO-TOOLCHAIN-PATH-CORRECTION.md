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

## Claim boundary

This change is not an accepted ISRAS release by itself. It requires review,
complete repository validation, hosted external-consumer regression validation,
a later signed immutable release, and explicit consuming-project upgrade. It does
not alter or retroactively repair ISRAS 0.1.4 or any consuming-project evidence.
