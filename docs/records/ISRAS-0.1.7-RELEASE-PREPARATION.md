# ISRAS 0.1.7 Corrective Release Preparation

**Status:** CORRECTIVE RELEASE-PREPARATION CANDIDATE

**Preparation date:** 2026-07-22

## Source boundary

- Immutable published 0.1.6 source: `d4045d90680b91e60edec62380d619b770edab12`
- Accepted hosted-tool correction commit: `67d3b86badba51195f8c2598a0313930800172bd`
- Governed correction merge: `37fc4071cf0fa5e0919084c5c071f273e7168699`
- Corrective release version: `0.1.7`
- Planned signed tag: `isras-v0.1.7`
- Release-preparation branch: `release/isras-v0.1.7-preparation`

## Corrected defect

The 0.1.6 reusable hosted adapter installed validator-owned `govulncheck`
tooling inside the consuming repository before commit-mode project commands.
The accepted correction moves that executable to runner-owned temporary storage,
preserves exact scanner verification, and rejects unsafe external executable
paths without weakening clean-tree enforcement.

## Reusable-source cleanup

Reusable Engineering Standards source records contain no consuming-project name,
consumer commit, workflow-run identifier, job identifier, or retained consumer
artifact identifier. The defect and correction are recorded at the reusable
standards boundary.

## 0.1.6 preservation

The `isras-v0.1.6` signed tag and published six-asset release remain immutable.
They are not moved, deleted, repointed, replaced, or republished.

## Remaining governed sequence

The 0.1.7 candidate requires signed commit verification, exact hosted CI,
governed merge, post-merge and clean-clone release validation, signed annotated
tagging, deterministic six-asset production and reproduction, separately
reviewed tag push, publication preflight, controlled draft-first publication,
remote-byte verification, and final release verification.

Consumer repositories remain outside this source release boundary until 0.1.7
is fully published and independently adopted or upgraded by each project.
