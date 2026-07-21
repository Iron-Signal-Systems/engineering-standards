# ISRAS 0.1.6 corrective release preparation record

**Status:** CORRECTIVE RELEASE-PREPARATION CANDIDATE

**Preparation date:** 2026-07-21

## Source boundary

- Immutable signed but unpublished 0.1.5 commit:
  `23ff4052650e8a5e92a7e416cb23c74fdf92a098`
- Immutable signed tag: `isras-v0.1.5`
- Corrective release version: `0.1.6`
- Planned signed tag: `isras-v0.1.6`
- Corrective branch:
  `fix/release-artifact-go-minimum-baseline-0.1.6`

## Defect

The 0.1.5 project-command boundary correctly enforced the `go.mod` `go`
directive as a minimum and accepted later compatible custom toolchains. The
tagged release-artifact producer instead required exact equality with that
directive. Artifact production therefore stopped safely on
`go1.26.5-X:nodwarf5` before any canonical asset set or GitHub Release existed.

## Correction

The producer now uses governed Go version comparison semantics:

- exact or later compatible valid Go versions pass;
- valid custom suffixes are accepted;
- versions below the minimum fail; and
- the actual selected toolchain remains bound into provenance and
  reproducibility evidence.

Tests cover exact-minimum, later patch, later custom, below-minimum, and invalid
version identities.

## 0.1.5 preservation

The `isras-v0.1.5` tag is not moved, deleted, repointed, or published. It remains
immutable, signed, unpublished, and non-adoptable.

## Remaining governed sequence

The corrective commit still requires signed commit verification, exact hosted
CI, governed merge, clean-clone release validation, signed 0.1.6 tagging,
deterministic six-asset reproduction, publication preflight, controlled
draft-first publication, remote-byte verification, and final release
verification.

Consumer repositories remain outside this corrective release boundary until
0.1.6 is fully published and accepted.
