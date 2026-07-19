# Project Initialization and Adoption

## Purpose

This contract defines the first supported boundary for adopting one accepted
ISRAS release into an existing Iron Signal Systems Go repository. Initialization
is controlled release consumption, not a template copy and not permission to
reorganize application source.

The command is exposed only by the exact linker-bound validator artifact from the
release being adopted:

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

1. the running executable identifies itself as a linker-bound release artifact;
2. its profile, stable version, tag, source repository, and source commit match
   the explicitly requested release;
3. the exact published six-asset GitHub Release exists;
4. a GitHub-verified signed annotated tag targets the same source commit;
5. GitHub-recorded SHA-256 values and downloaded byte sizes match;
6. locally observed SHA-256 and SHA-512 values match;
7. both checksum manifests and provenance bind the same release and artifacts;
8. the framework archive contains the reusable workflow declared by the pin;
9. `--go-defaults` is explicitly authorized;
10. the target has one canonical Iron Signal Systems GitHub origin.

A reference-repository build, development build, project-owned export, validator
for another release, or release lacking the reusable workflow has no
initialization authority. This intentionally prevents `isras-v0.1.1` from being
used for complete adoption.

## Target and evidence preconditions

First installation requires:

- a resolvable Git repository HEAD;
- a canonical SSH or HTTPS
  `github.com/Iron-Signal-Systems/REPOSITORY` origin with no credentials, port,
  query, fragment, or path ambiguity;
- a clean index and working tree, including no untracked files;
- no existing partial, conflicting, symbolic-link, or mode-drifted adoption
  paths;
- the fixed runtime evidence directory `.local/isras` to be untracked and to
  contain no symbolic-link or non-directory component.

The evidence directory is not user-selectable in this boundary. Excluding an
arbitrary path from repository-state comparison could conceal changes to the pin,
workflow, or project-owned checker.

An exact already-installed set is accepted as idempotent. Bootstrap verification
is performed again, but durable adoption evidence excludes volatile run
timestamps and is canonical for the same verified release. Any partial or
non-identical state fails closed and is not repaired or overwritten.

## Generated project-owned artifacts

The initial Go adoption set is exactly:

```text
.isras/project.json
.isras/adoption-verification.json
.isras/check-go-format
.github/workflows/isras-validation.yml
```

The project pin records the exact release identity, six assets and both digests,
reusable workflow commit, target identity, Go profile, explicit command arrays,
and fixed `.local/isras` evidence directory.

The caller workflow references the reusable workflow by the same 40-character
source commit recorded in the pin. It never references `dev`, `main`, `latest`,
or a mutable release tag.

The format checker enumerates tracked Go source and runs `gofmt -l` without
rewriting source. Other default commands use bounded argv declarations and may be
changed only through a reviewed pin update.

## Atomic publication and rollback

Initialization prepares every file privately before publication. It validates
repository-relative paths, rejects symbolic-link and non-directory components,
creates directories one component at a time, synchronizes files and directories,
publishes without replacement, and removes paths created by the current operation
when a later publication step fails.

No commit or push is performed. The resulting working-tree change is left for
human review.

## Reusable hosted validation

The accepted release contains `.github/workflows/validate-project.yml`. The
reusable workflow:

1. checks out the consuming repository at the exact caller head commit;
2. checks out the called workflow's repository and exact `job.workflow_sha`;
3. builds an exact-source bootstrap verifier;
4. validates the committed pin and verifies the six published assets;
5. downloads `isras-validator-linux-amd64` from the pinned release;
6. verifies its release-recorded byte size and pinned SHA-256 and SHA-512 digests before execution;
7. runs repository and secret-protection boundaries with that published artifact;
8. revalidates the pin and release with the published artifact;
9. executes every committed project command through the bounded command boundary;
10. retains `.local/isras` as a GitHub Actions evidence artifact even when a later
    validation step fails.

The workflow uses read-only repository permissions and full-commit-pinned
third-party actions. It does not derive Engineering Standards identity from the
caller's ordinary `github.sha` value.

## Failure behavior

Before target publication, failures leave the target unchanged. Release download
and verification use temporary storage outside the target repository. A failed
initialization does not create a pin, workflow, checker, evidence record, commit,
tag, branch, release, or remote write.

After publication begins, failure invokes rollback of paths created by the
current operation. Pre-existing paths are never deleted.

## Acceptance evidence

The implementation acceptance boundary includes tests for:

- exact release bootstrap and six-asset verification;
- rejection of releases without the reusable workflow;
- rejection of reference, development, exported, mismatched, or inconsistent
  validator identities before target modification;
- one shared canonical origin parser for initialization and command execution;
- fixed untracked evidence-path enforcement;
- complete `Initialize()` idempotence across fresh timestamped verification runs;
- conflict, partial-state, executable-mode, and symbolic-link rejection;
- rollback after mid-publication failure;
- immutable caller-workflow generation;
- exact called-workflow identity and caller commit selection;
- published-validator byte, SHA-256, and SHA-512 binding;
- repository, secret, project-command, and retained-evidence workflow boundaries;
- inclusion of the reusable workflow in the release framework archive.

## Scope limits

This boundary supports first adoption of the Go profile. It does not implement
migration of hand-authored pins, release upgrades, non-Go initialization,
source-layout reorganization, automatic Git or release publication, or
independent certification.

A development branch containing this implementation is not adoption authority.
Consuming repositories may use it only after acceptance and publication as the
exact signed release with the verified six-asset set.
