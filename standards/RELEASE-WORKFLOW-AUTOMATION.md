# Release Workflow Automation

## Purpose

The repository-owned `isras-release` command performs the repetitive release
checks and publication mechanics through typed Go code rather than pasted shell
fragments. It does not replace the release policy, signed-source requirements,
or clean-clone validation. It applies those controls consistently and retains a
local workflow log.

The command is built from committed source:

```bash
./tools/build-release-command.sh
```

## Staged authority

The workflow is deliberately split into three stages.

### Check

```bash
./.local/bin/isras-release check
```

`check` performs bounded authoritative network reads and creates ignored local
validation evidence. It does not create or move a Git ref, push a ref, move
`main`, or create a GitHub Release.

The stage requires:

- the configured release branch, normally `dev`;
- a completely clean repository;
- local `HEAD` equal to the authoritative remote release branch;
- a stable `MAJOR.MINOR.PATCH` value in `VERSION`;
- matching changelog and release-note artifacts;
- a verified exact commit signature;
- complete commit-mode validation; and
- clean-clone release-mode validation of the exact pushed commit.

### Tag

```bash
./.local/bin/isras-release tag --confirm
```

`tag` repeats the complete candidate checks and then creates or verifies the
signed annotated local `isras-vMAJOR.MINOR.PATCH` tag. It does not push the tag
or change a remote branch.

The explicit `--confirm` flag is required because the stage changes a local Git
ref. An existing tag is accepted only when it is annotated, has a valid local
signature, and resolves to the exact tested commit.

### Publish

```bash
./.local/bin/isras-release publish --confirm
```

`publish` repeats the complete checks and requires the signed local tag created
by the prior stage. It then:

1. pushes the exact annotated tag when it is not already present;
2. verifies that the remote tag object and peeled commit exactly match the local
   tag and tested commit;
3. fast-forwards remote `main` to the tested release commit without force;
4. creates or verifies the non-draft, non-prerelease GitHub Release using the
   committed release notes; and
5. proves the final remote state.

The explicit `--confirm` flag is required because the stage performs remote
writes. `publish` refuses to rewrite a divergent `main` branch.

## GitHub CLI boundary

Git repository transport remains governed by the configured Git remote, such as
SSH. GitHub Release publication uses the authenticated `gh` command because it
is a GitHub API operation rather than Git repository transport.

The command checks `gh auth status` before attempting publication. It never asks
for, displays, or stores an authentication token itself.

## Failure behavior

Every invocation writes a private local log under:

```text
.local/validation/releases/release-workflow-*.log
```

A failed stage exits only the `isras-release` child process. It cannot terminate
the developer's interactive shell. Read-only network operations are retried in a
bounded manner. Remote writes are not blindly retried; after an uncertain push,
the command reads the authoritative remote state and accepts success only when
the exact expected object is proven.

## Idempotence

A stage may be rerun after interruption. Existing local tags, remote tags,
`main`, and GitHub Releases are accepted only when they exactly match the tested
release identity and required publication state. Conflicting state stops the
workflow for investigation.
