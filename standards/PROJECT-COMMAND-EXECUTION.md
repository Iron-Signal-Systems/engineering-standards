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

## Go toolchain minimum and PATH preservation

For a project declaring the `go` profile, ISRAS resolves the active `go`
executable from the validator process before constructing the isolated project
command environment. The exact selected toolchain directory is placed first in
the bounded command `PATH`; unrelated caller `PATH` entries are not inherited.

The consuming project's `go.mod` `go` directive is a minimum version, not an
exact-version lock. ISRAS accepts the declared version and later valid Go
versions, including valid custom suffixes such as `go1.26.5-X:nodwarf5`, and
rejects versions below the declared minimum. `GOTOOLCHAIN=local` and `GOENV=off`
are used for the version query so validation never silently downloads or switches
toolchains while establishing this boundary.

## Fixed Go-profile child environment

For every project command in a project declaring the `go` profile, the child
environment fixes `GOTOOLCHAIN=local` and `GOENV=off` after bounded inherited
environment processing. Caller values such as `GOTOOLCHAIN=auto` or
`GOENV=/caller/path` cannot override these controls.

The exact selected Go executable directory remains first in the bounded child
`PATH`. A project-owned command that invokes `go` therefore reaches the same
selected local toolchain that ISRAS probed, without inheriting unrelated caller
`PATH` components. Projects that do not declare the Go profile retain the existing
bounded inherited-environment behavior.

## Go module declaration boundary

Before selecting Go, ISRAS parses the applicable repository-owned `go.mod`. It rejects missing, malformed, duplicate, oversized, unreadable, symbolic, non-regular, or path-escaped module files. The optional `toolchain` value is descriptive evidence only and cannot weaken the selected local toolchain boundary.

## Project-command evidence schema v2

Go-profile execution uses the governed
`schemas/isras-project-command-execution-v2.schema.json` contract. Version 2
preserves every version 1 command, timeout, output, repository-drift, validator,
target, environment-name, redaction, and stream field while adding a typed
`go_toolchain` object.

The object records the selected executable and directory, exact reported Go
version, project minimum, optional project `toolchain` directive, fixed effective
`GOTOOLCHAIN` and `GOENV` values, and the minimum-satisfaction result. Successful
Go-profile commands require complete selected-toolchain evidence and a true
minimum result. Below-minimum rejection retains the discovered executable,
reported version, declared minimum, fixed environment policy, and false result
before the project command can execute.

The version 1 schema remains immutable as a historical contract. Producers emit
version 2 after this behavioral change; consumers select schema handling by the
explicit `schema_version` field.

## Multi-module execution boundary

Before any Go-profile project command executes, ISRAS discovers and validates the
complete repository-owned source module set. Discovery uses Git's tracked and
nonignored-untracked inventory and excludes the reserved `.local/` runtime tree.
The selected executable is checked against every source module's `go` minimum,
not only the root module. One unsatisfied nested source module denies execution
before the project command starts.

Project-command evidence schema version 2 includes the sorted module inventory:
repository-relative `go.mod` path, module directory, declared module path, Go
minimum, optional toolchain directive, and per-module minimum result.

## Bounded source-inventory tool resolution

The source-module inventory resolves Git from governed absolute system
directories before invocation. Caller `PATH` contents are not inherited for this
operation. This keeps repository discovery available even when Go selection tests
or controlled execution expose only the selected Go directory.

## Govulncheck identity probe

The pinned scanner identity probe is separate from scanner execution. It invokes
the exact selected Go executable with `version -m` against the exact scanner
path, using `GOTOOLCHAIN=local`, `GOENV=off`, deterministic locale, bounded
system paths, bounded output, and a fixed timeout. The probe never consults the
complete caller PATH and never installs or upgrades a tool.

## Mandatory Go vulnerability-command specialization

`known_vulnerabilities` is a specialized Go-profile operation rather than an
ordinary single-directory project command. The implementation expands the one
governed declaration into one exact pinned scanner invocation per discovered
module. Every invocation retains the existing timeout, output, process-tree,
redaction, and repository-state controls while adding module identity and
streaming-protocol validation.

## Govulncheck evidence output

Project-command evidence v2 has an additive typed `govulncheck` section. The section is optional for non-vulnerability commands and records exact scanner identity plus one reconciled module result per governed `go.mod`. The runtime-dispatch step makes this section mandatory for `known_vulnerabilities`.

## Govulncheck pre-exception result policy

The specialized vulnerability operation evaluates parsed findings rather than
trusting the scanner exit code. Module- and package-level findings remain
recorded observations. Symbol-level findings are reachable and fail until an
exact governed exception exists. Unknown-level findings fail closed.

## Specialized `known_vulnerabilities` dispatch

`Execute` preserves the ordinary command path for all other operations. For `known_vulnerabilities`, it derives the exact runtime configuration from the governed evidence boundary, invokes the selected-Go and pinned-scanner orchestrator, and finalizes v2 evidence from per-module results. Failure evidence remains available when reachable findings or other post-scan policy checks fail.

## Govulncheck exception evidence

A passing `known_vulnerabilities` result requires typed exception-evaluation
evidence. The evidence records document presence, path, digest, schema version,
evaluation time, exact used exceptions, unused records, unexcepted reachable
findings, and unknown-finding summaries. Failure evidence retains the same
reconciliation whenever scanner execution completed successfully.

## Validator-owned hosted tool boundary

A hosted adapter shall not install validator-owned executables into the
consuming repository outside the fixed runtime-evidence boundary. The exact
governed `govulncheck` executable is installed in runner-owned temporary storage.

The reusable workflow passes its absolute path through
`ISRAS_GOVULNCHECK_EXECUTABLE`. The release validator accepts that path only
when it is clean, absolute, outside the target repository, a regular executable,
and not a symbolic link. Exact module and version verification remains
mandatory before execution.

This environment value is a validator-host integration boundary. It is not a
consumer-controlled project command, application dependency, mutable pin, or
permission to weaken clean-tree enforcement.
