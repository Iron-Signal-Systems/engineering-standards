# ISRAS 0.1.6 Hosted Govulncheck Runtime Correction

**Status:** SOURCE CORRECTION CANDIDATE — NEW RELEASE REQUIRED

**Observed:** 2026-07-22

## Immutable affected boundary

- Published release: `isras-v0.1.6`
- Release source:
  `d4045d90680b91e60edec62380d619b770edab12`
- Consuming repository:
  `github.com/Iron-Signal-Systems/iron-file-intelligence`
- Consuming candidate:
  `1c160784dea399b5dbc12d6be3bdb4cc0f0ed37a`
- Hosted run: `29883861708`
- Hosted job: `88810263073`
- Retained artifact: `8515796354`
- Retained artifact SHA-256:
  `7f18dd8099e6080a1d4c35adcadf34e32938ca4cd76d9f2398dbfd90de83068d`

## Defect

The 0.1.6 reusable hosted workflow installed the exact governed
`govulncheck` binary at:

```text
target/.local/tools/bin/govulncheck
```

The consuming repository did not ignore `/.local/tools/`. Commit-mode project
command execution therefore observed a Git-visible untracked runtime tool before
the first sorted command (`build`) and failed closed with:

```text
commit and release modes require a clean target repository before project command execution
```

Release identity, signed-tag verification, six-asset verification, repository
validation, secret protection, the selected Go toolchain, and exact
`govulncheck` installation all completed before this failure.

## Correction

The hosted adapter now:

1. installs exact `govulncheck` into runner-owned
   `$RUNNER_TEMP/isras-tools/bin`;
2. verifies the installed module and version exactly as before;
3. exports the absolute executable path through the
   validator-owned `ISRAS_GOVULNCHECK_EXECUTABLE` boundary; and
4. leaves the consuming repository free of validator-owned tool binaries.

The release validator accepts that boundary only when the path is:

- a clean absolute path;
- outside the target repository;
- a regular executable file; and
- not a symbolic link.

The prior repository-local path remains only as a compatibility fallback for
existing local integrations. The hosted reusable workflow no longer uses it.

## Claim boundary

The published `isras-v0.1.6` release and assets remain immutable. They are not
replaced, republished, or silently edited. This source correction is not
consumer adoption authority.

A new immutable release is required after the exact correction candidate passes
repository validation, focused and complete tests, hosted validation, governed
merge, clean-clone release validation, deterministic asset reproduction,
controlled publication, remote-byte verification, and final release
verification.

IFI PR #1 remains a draft and Gate 1 remains unaccepted.
