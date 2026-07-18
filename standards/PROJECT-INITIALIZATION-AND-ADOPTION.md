# Project Initialization and Adoption

## Purpose

This contract defines the first supported boundary for adopting one accepted
ISRAS release into an existing Iron Signal Systems Go repository. Initialization
is a controlled release-consumption operation, not a template copy and not
permission to reorganize application source.

The implementation is exposed by the linker-bound release validator:

```bash
isras-validator-linux-amd64 \
  --repo /src/example-project \
  project-pin initialize \
  --release isras-vMAJOR.MINOR.PATCH \
  --go-defaults
```

Initialization changes no Git refs, creates no commit, performs no push, and
never chooses a release implicitly.

## Authority prerequisites

Initialization requires all of the following before the target may be modified:

1. an explicit stable `isras-vMAJOR.MINOR.PATCH` release tag;
2. the exact published six-asset GitHub Release;
3. a GitHub-verified signed annotated tag targeting the release source commit;
4. exact GitHub-recorded SHA-256 values and downloaded byte sizes;
5. complete locally observed SHA-256 and SHA-512 values;
6. valid SHA-256 and SHA-512 manifests;
7. provenance bound to the same release, source commit, and core artifacts;
8. a framework archive containing the reusable workflow declared by the pin;
9. an explicit `--go-defaults` profile authorization;
10. a canonical Iron Signal Systems GitHub origin for the target repository.

A release that lacks the reusable hosted workflow is not adoption-capable even
when its other artifacts are valid. This rule intentionally prevents
`isras-v0.1.1` from being used for complete adoption.

## Target preconditions

First installation requires:

- a resolvable Git repository HEAD;
- a canonical `github.com/Iron-Signal-Systems/REPOSITORY` origin;
- a clean index and working tree, including no untracked files;
- no existing, partial, conflicting, symbolic-link, or mode-drifted adoption
  paths.

An exact already-installed set is accepted as idempotent and produces no
additional change. Any partial or non-identical state fails closed and is not
repaired or overwritten automatically.

## Generated project-owned artifacts

The initial Go adoption set is exactly:

```text
.isras/project.json
.isras/adoption-verification.json
.isras/check-go-format
.github/workflows/isras-validation.yml
```

The project pin contains:

- the exact accepted release identity;
- the exact six artifact names and both complete digests;
- the exact reusable workflow source commit;
- the canonical target repository identity;
- the Go profile;
- explicit project-owned command arrays;
- the project evidence directory.

The caller workflow references the reusable workflow by the same 40-character
source commit recorded in the pin. It never references `dev`, `main`, `latest`,
or a mutable tag.

The format checker is a small project-owned executable. It enumerates tracked Go
source and runs `gofmt -l` without rewriting source. Other default commands use
bounded argv declarations and may be replaced only through a reviewed pin
change.

## Atomic publication and rollback

Initialization prepares every file privately before publication. It:

- validates every target path as repository-relative;
- rejects symbolic-link and non-directory path components;
- creates directories one component at a time;
- synchronizes created files and containing directories;
- publishes without replacing an existing path;
- removes every file and directory created by the operation when any later
  publication step fails.

No commit or push is performed. The resulting working-tree change is deliberately
left for human review.

## Reusable hosted validation

The accepted release contains:

```text
.github/workflows/validate-project.yml
```

The reusable workflow:

1. checks out the consuming repository at the caller event commit;
2. checks out the called workflow's own repository and exact workflow SHA using
   `job.workflow_repository` and `job.workflow_sha`;
3. builds a validator with that exact stable release identity embedded;
4. validates the committed project pin in commit mode;
5. verifies the exact published release and six artifact bytes;
6. executes each committed project command through the bounded project-command
   boundary;
7. uses read-only repository permissions and commit-pinned third-party actions.

The workflow must not derive Engineering Standards identity from the caller's
ordinary `github.sha` context.

## Failure behavior

Before target publication, failures leave the target unchanged. Release download
and verification use temporary storage outside the target repository. A failed
initialization does not create a pin, workflow, checker, evidence record, commit,
tag, branch, release, or remote write.

After publication begins, any failure invokes rollback of the exact paths created
by the current operation. Pre-existing paths are never deleted.

## Acceptance evidence

The implementation acceptance boundary includes tests for:

- exact release bootstrap and six-asset verification;
- rejection of a release without the reusable workflow;
- canonical project-origin handling;
- clean first installation;
- exact idempotent re-execution;
- conflict and partial-state rejection;
- executable-mode drift rejection;
- symbolic-link path rejection;
- rollback after a mid-publication failure;
- immutable caller-workflow generation;
- reusable-workflow source identity and pinned action SHAs;
- inclusion of the reusable workflow in the release framework archive.

## Scope limits

This boundary supports first adoption of the current Go profile. It does not yet
implement:

- automatic migration of an existing hand-authored pin;
- ISRAS release upgrades;
- non-Go profile initialization;
- source-layout reorganization;
- automatic commit, push, merge, tag, or release publication;
- independent audit or certification.

A development branch containing this implementation is not itself adoption
authority. Consuming repositories may use it only after the implementation is
included in an accepted signed release with the exact verified six-asset set.
