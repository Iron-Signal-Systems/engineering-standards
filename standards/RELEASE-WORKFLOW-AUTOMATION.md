# Release Workflow Automation

## Purpose

The repository-owned `isras-release` command performs repeatable release-source
checks and local signed-tag preparation through typed Go code rather than pasted
shell fragments. It does not replace release policy, signed-source requirements,
or clean-clone validation. It applies those controls consistently and retains a
local workflow log.

The command is built from committed source:

```bash
./tools/build-release-command.sh
```

## Staged authority

The workflow is deliberately split into two stages.

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
signed annotated local `isras-vMAJOR.MINOR.PATCH` tag. It does not push the tag,
change a remote branch, create a GitHub Release, or upload an asset.

The explicit `--confirm` flag is required because the stage changes a local Git
ref. An existing tag is accepted only when it is annotated, has a valid local
signature, and resolves to the exact tested commit.

## Publication handoff

The legacy command form below is disabled:

```bash
./.local/bin/isras-release publish --confirm
```

It returns a failure directing the operator to the separately named publication
command. This prevents the earlier workflow from creating an assetless release,
pushing a tag during publication, moving `main`, or bypassing deterministic
artifact verification.

After the signed annotated tag has been deliberately pushed through a separately
reviewed Git operation and the deterministic release artifacts have been
produced, publication uses:

```bash
./tools/build-release-publication.sh
./.local/bin/isras-release-publication check --version MAJOR.MINOR.PATCH
./.local/bin/isras-release-publication publish --version MAJOR.MINOR.PATCH --confirm
```

The complete remote-tag, artifact, draft, upload, remote-byte verification,
cleanup, and publication rules are defined in
[`RELEASE-PUBLICATION.md`](RELEASE-PUBLICATION.md).

## Git transport boundary

The `isras-release` source and tag stages may read the configured Git remote. The
local tag stage does not push it. Any push of the accepted annotated tag remains
a separately reviewed Git write before publication preflight can pass.

GitHub Release publication uses authenticated `gh` API operations only in the
separately named publication command. Neither command asks for, displays, or
stores an authentication token itself.

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
bounded. Content exceeding the relevant limit is discarded or truncated with an
explicit marker rather than emitted without complete censoring context.
Censoring does not change a command's exit status or convert a failed release
stage into a pass.

## Failure behavior

Every source or tag invocation writes a private local log under:

```text
.local/validation/releases/release-workflow-*.log
```

A failed stage exits only the `isras-release` child process. It cannot terminate
the developer's interactive shell. Read-only network operations are retried in a
bounded manner. The command performs no remote write.

Publication evidence and draft cleanup behavior are separate and are defined in
`RELEASE-PUBLICATION.md`.

## Idempotence

`check` and `tag` may be rerun after interruption. Existing local and remote tags
are accepted only when they exactly match the tested release identity. Conflicting
state stops the workflow for investigation.

A published version cannot be reused for later source. After publication, active
development advances to a suffixed value such as `0.1.1-development`. That value
records the next development cycle but cannot pass release workflow preflight.
A later release-preparation change removes the suffix, finalizes the changelog
and release notes, and establishes the stable candidate version.

When `VERSION` derives a tag that already identifies a different commit, the
workflow fails and directs the developer to advance `VERSION`, the changelog,
and the release notes.
