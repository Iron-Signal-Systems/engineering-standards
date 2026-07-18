# Project Command Execution

## Purpose

A consuming repository may declare project-owned validation commands in its
committed `.isras/project.json`. This contract defines the first execution
boundary for running one exact declared command through a linker-bound ISRAS
release validator.

The command surface is:

```bash
isras-validator-linux-amd64 \
  --repo /absolute/project/path \
  project-command run NAME
```

Only one exact command name is accepted per invocation. There is no implicit
`run-all`, wildcard, prefix, or fallback selection.

## Authorization boundary

Execution is denied unless all of the following are true:

- the validator is a linker-bound `release-artifact` identity;
- validator profile, stable version, release tag, source repository, and exact
  source commit match the committed project pin;
- the target repository origin resolves to the exact
  `project.repository` identity in the pin;
- the target `HEAD` remains the commit discovered before authorization;
- `.isras/project.json` is a regular file with no symbolic-link component;
- the committed `HEAD` copy, staged index copy, and working-tree copy of the pin
  are byte-identical;
- the requested command name exists in the committed pin;
- private evidence storage can be created before the process starts.

A modified or merely staged pin cannot authorize execution, including in
`development` mode. A reference-repository validator or deprecated
project-owned export cannot execute a project command.

## Exact process invocation

Commands remain JSON argument arrays. The first element is resolved as the
executable and the remaining elements are passed directly to `execve` semantics.
The validator does not concatenate arguments and does not invoke an implicit
shell.

An explicitly declared repository-local executable:

- must be a normalized relative path inside the target;
- must be a tracked regular executable file;
- must match the exact `HEAD` content and mode;
- must not contain a symbolic-link path component.

Executables selected by name are resolved once from the caller's bounded
absolute-component `PATH`; the resolved absolute path is retained as evidence.
The child receives a narrower path containing that executable directory and
existing system binary directories.

Generic privilege or process launchers such as `sudo`, `doas`, `su`, `env`,
`nohup`, `setsid`, and `xargs` are rejected. Known shell executables are rejected
when supplied an opaque command-string option such as `-c`, `/c`, `-Command`, or
`-EncodedCommand`. A directly executed committed script with a reviewed shebang
remains permitted.

## Working directory and environment

The child working directory is the canonical target root. The validator never
changes its own process-wide working directory.

The child does not inherit the complete caller environment. It receives:

- an isolated temporary `HOME`, `TMPDIR`, `XDG_CACHE_HOME`, `GOCACHE`, and
  `GOPATH` removed after execution;
- `LANG=C` and `LC_ALL=C`;
- the sanitized executable path;
- bounded non-secret toolchain controls such as Go toolchain, module-policy,
  compiler, certificate, timezone, and reproducibility variables when present;
- `ISRAS_PROJECT_ROOT`, `ISRAS_PROJECT_COMMAND`, and
  `ISRAS_VALIDATOR_SOURCE_COMMIT` identity values.

Credential variables, SSH agents, cloud tokens, GitHub tokens, generic secret
variables, and arbitrary caller variables are not inherited.

## Runtime bounds

Each invocation has:

- a 20-minute wall-clock timeout;
- a 1 MiB raw-output budget for stdout;
- a separate 1 MiB raw-output budget for stderr;
- a Linux process group terminated on timeout, output overflow, cancellation,
  and after the main process exits so background descendants do not survive.

Exceeding either output budget cancels the complete process group. Evidence
records hashes of the retained bounded prefix and explicitly marks that the
limit was exceeded.

## Repository-state boundary

Before execution, the validator records target `HEAD` and Git-visible state while
excluding only the declared evidence directory. `commit` and `release` modes
require that state to be clean.

After execution, `HEAD` and Git-visible state must exactly match the pre-execution
snapshot. A command that changes tracked, staged, or non-ignored untracked state
fails even when the process exits zero. The validator cannot prove the absence of
a transient write that was completely reverted before the final snapshot; that
limitation remains explicit.

## Evidence

A private run directory is created below:

```text
<evidence.directory>/project-commands/
```

The run directory is mode `0700`. `execution.json` and `execution.txt` are mode
`0600` and are written atomically. A durable pending marker is created before
process launch; inability to create it denies execution.

Evidence includes:

- validator and target identities;
- command name and sanitized argv evidence;
- resolved executable and target working directory;
- inherited environment names, never their values;
- start, finish, duration, timeout, output limit, and exit status;
- repository-state drift decision;
- byte counts and SHA-256/SHA-512 digests for each retained stream;
- redacted stdout and stderr.

Credential-shaped values and private-key material are redacted before retention.
Raw output and raw credential-shaped command arguments are not retained. The v1
JSON contract is:

```text
schemas/isras-project-command-execution-v1.schema.json
```

Failure to finalize both evidence files makes the execution result unacceptable,
even if the child process exited zero.

## Acceptance tests

Acceptance requires committed tests proving:

- exact arguments are not expanded or reinterpreted;
- unapproved environment values do not reach the child;
- validator, pin, target origin, and target commit mismatches deny execution;
- working-tree and staged pin drift deny execution before process start;
- repository-local executable drift and symbolic links fail closed;
- opaque shell strings and generic launchers fail closed;
- timeouts and output floods terminate execution;
- background descendants do not survive completion;
- nonzero exits retain private redacted evidence;
- Git-visible target mutation invalidates an otherwise successful command;
- evidence paths containing symbolic links fail closed;
- an embedded release validator can execute one committed command against an
  explicit external target without changing the caller's working directory.

## Assurance boundary

This step does not:

- execute downloaded validator or framework bytes;
- initialize or upgrade a project;
- modify a project pin;
- authorize commands not explicitly declared by name;
- publish artifacts, tags, releases, or commits;
- push, merge, deploy, or modify Iron Atlas;
- claim sandbox or container isolation beyond the documented process, environment,
  timeout, output, evidence, and repository-state controls.
