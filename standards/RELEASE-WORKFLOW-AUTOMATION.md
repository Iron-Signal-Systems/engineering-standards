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

Before the expensive validation campaign begins, `check` inspects the declared
version and the derived local and remote release tag. A development declaration
such as `0.1.1-development` fails before release validation because only a stable
`MAJOR.MINOR.PATCH` value can define a release candidate. An existing tag that
identifies any commit other than the current candidate is also a hard failure
because release tag names are immutable. The command repeats tag-identity
inspection after validation so an intervening tag change cannot be silently
accepted.

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

## Output censoring boundary

Release automation shall censor terminal output and retained workflow logs before
bytes reach either destination. The same censoring boundary applies to:

- displayed command arguments;
- streamed child-process standard output and standard error;
- captured command output written to logs;
- command failures and wrapped errors;
- read-retry diagnostics;
- remote URLs and GitHub CLI diagnostics; and
- final workflow summaries and failure reasons.

Structured command output may remain uncensored only in bounded process memory
for the minimum time required to parse authoritative Git or GitHub state. It
shall be censored before it is logged, displayed, or incorporated into an error.

Credential-shaped assignments and command flags, authorization headers, URL
userinfo, supported GitHub, AWS, and Slack token forms, and private-key material
shall be replaced with explicit `[REDACTED]` markers. Multiline private-key
blocks shall be suppressed across write boundaries rather than handled as
independent lines.

An incomplete output line is bounded to 64 KiB and captured subprocess output is
bounded to 1 MiB. Content exceeding those limits is discarded or truncated with
an explicit marker rather than emitted without complete censoring context.
Censoring does not change a command's exit status or convert a failed release
stage into a pass.

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

A published version cannot be reused for later source. After publication, active
development advances to a suffixed value such as `0.1.1-development`. That value
records the next development cycle but cannot pass release workflow preflight.
A later release-preparation change removes the suffix, finalizes the changelog
and release notes, and establishes the stable candidate version.

When `VERSION` derives a tag that already identifies a different commit, the
workflow fails and directs the developer to advance `VERSION`, the changelog,
and the release notes.
