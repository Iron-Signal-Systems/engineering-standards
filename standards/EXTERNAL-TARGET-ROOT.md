# External Target Repository Root

## Purpose

An ISRAS validator is an Engineering Standards tool operating on a separately
selected target repository. The validator's release identity and the target
repository's Git identity are different security boundaries and must not be
silently substituted for each other.

This contract defines explicit target selection for local execution. It does not
execute project-declared commands, initialize or upgrade a project, publish a
release, or modify Iron Atlas.

## Command boundary

A target repository may be selected with the global option:

```bash
isras-validator-linux-amd64 \
  --repo /src/example-project \
  project-pin validate
```

The option may appear before or after the command name, but it may be declared
only once. The validator resolves one canonical Git repository root from the
selected directory and uses that root for the complete invocation.

When `--repo` is omitted, the current working directory remains the target
selection input for backward-compatible repository-local use.

## Target path rules

The selected target must:

- exist;
- be a directory;
- be at or below one canonical Git repository root;
- contain no symbolic-link component in the selected or discovered path;
- resolve to a nonzero 40-character Git `HEAD` commit;
- remain within the bounded path-length and single-line input limits.

Missing paths, files, symbolic links, non-Git directories, invalid Git roots, and
repositories without a resolvable commit fail closed.

A subdirectory may be supplied. Git determines the canonical repository root,
and all later operations use that canonical root.

## No process-wide directory change

The validator does not call `chdir` to enter the target. Git, Go, secret scanning,
project-pin reads, evidence writes, and bounded repair operations receive the
canonical target root explicitly.

This prevents one target invocation from contaminating a later invocation and
allows the validator to be launched from `/tmp`, an administrator directory, or
a CI tool directory without changing caller state.

## Validator identity separation

A release validator obtains its identity only from linker-bound release values:

- stable ISRAS version;
- signed release tag;
- exact Engineering Standards source commit;
- release-artifact ownership.

Those values are available without discovering a target repository. A consuming
project cannot replace them with its own commit or with
`validation/isras-validator-identity.json`.

The target repository commit is retained separately in the validation runner and
repository evidence. It is not reported as the validator's repository commit.

The repository-owned development validator may locate its own identity from the
repository containing the executable or current development invocation. That
identity lookup is distinct from external target selection.

## Standalone commands

A linker-bound release validator must run these commands outside every Git
repository:

```bash
isras-validator-linux-amd64 version
isras-validator-linux-amd64 help
```

Neither command discovers, reads, hashes, validates, or modifies a target
repository.

Unknown commands also fail as command errors rather than being misreported as
repository-discovery failures.

## Rooted execution and remediation

Every actual command launched by the validator uses the canonical target as its
working directory without changing the parent process.

Validator rerun commands preserve the explicit `--repo` target. Direct Git, Go,
file-review, and remediation examples are target-root-qualified so copying a
suggested command cannot silently operate on the caller's current directory.

Failure logs, artifact-verification evidence, secret-review plans, and local
allowlist proposals are written only below the selected target's declared local
evidence boundary.

## Isolation tests

Acceptance tests must prove:

- an embedded release validator runs `version` and `help` from outside Git;
- `project-pin validate` succeeds against an explicit external target;
- global options work before or after the command name;
- two target repositories do not contaminate each other's identity or output;
- repository discovery does not change the process working directory;
- selected symbolic-link paths fail closed;
- missing, non-directory, and non-Git targets fail closed;
- validator identity remains the Engineering Standards release identity;
- target commit identity remains separate;
- no project-declared command is executed in this step.

## Assurance boundary

External target selection establishes only where read-only or separately named
validator operations act. It does not authorize artifact execution, project
command execution, initialization, upgrade application, commit, push, merge,
tag creation, release publication, or deployment.
