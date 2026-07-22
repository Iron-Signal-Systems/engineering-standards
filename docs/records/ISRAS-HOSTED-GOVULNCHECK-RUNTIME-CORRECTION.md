# ISRAS Hosted Govulncheck Runtime Correction

**Status:** ACCEPTED SOURCE CORRECTION — RELEASE 0.1.7 REQUIRED

**Observed:** 2026-07-22

## Immutable affected boundary

- Published affected release: `isras-v0.1.6`
- Affected release source: `d4045d90680b91e60edec62380d619b770edab12`
- Accepted correction commit: `67d3b86badba51195f8c2598a0313930800172bd`
- Governed merge commit: `37fc4071cf0fa5e0919084c5c071f273e7168699`
- Corrective release: `isras-v0.1.7`

## Defect

Validation of a consuming repository showed that the 0.1.6 reusable hosted
workflow installed the exact governed `govulncheck` binary at:

```text
target/.local/tools/bin/govulncheck
```

When the consuming repository did not ignore that validator-owned path,
commit-mode project-command execution correctly observed a Git-visible untracked
file and failed before the first project command.

The failure did not invalidate signed-tag verification, six-asset verification,
repository validation, secret protection, selected-Go enforcement, or exact
scanner installation. It exposed an ownership-boundary defect in the hosted
adapter.

## Correction

The reusable hosted adapter now:

1. installs exact `govulncheck` under runner-owned temporary storage;
2. verifies the exact governed module and version;
3. exports the absolute executable path through
   `ISRAS_GOVULNCHECK_EXECUTABLE`; and
4. leaves the consuming repository free of validator-owned tool binaries.

The release validator accepts the external executable only when the path is:

- a clean absolute path;
- outside the target repository;
- a regular executable file;
- executable; and
- not a symbolic link.

The repository-local path remains only as a compatibility fallback for existing
local integrations. The reusable hosted workflow does not use it.

## Regression boundary

Focused tests cover:

- runner-owned executable acceptance;
- target-owned path rejection;
- relative path rejection;
- symbolic-link rejection;
- compatibility fallback behavior; and
- reusable-workflow rejection of target-owned tool installation.

The complete Engineering Standards validation and supported Linux platform
workflows passed on the accepted correction commit.

## Release and claim boundary

The published `isras-v0.1.6` tag and assets remain immutable. They are not
replaced, republished, or silently edited.

The correction becomes adoption authority only through the separately prepared,
validated, signed, published, and remotely verified `isras-v0.1.7` release.
