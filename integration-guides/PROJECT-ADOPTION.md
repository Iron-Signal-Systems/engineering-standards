# Project Adoption

The active baseline is designed so each adopting project retains the exact Go
validator and its unit tests in that project's own Git history.

## Export the reference validator

From a clean `engineering-standards` working tree:

```bash
./tools/export-project-validator.sh /path/to/target-project
```

A read-only preflight is available:

```bash
./tools/export-project-validator.sh --dry-run /path/to/target-project
```

The target must be a clean, non-bare Git working tree with an existing `go.mod`
file. Ordinary clones and linked Git worktrees are both supported. The exporter
asks Git for the target worktree and repository state; it does not assume that
`.git` is a directory.

The exporter copies:

- `cmd/isras-validate/`;
- validator packages under `internal/isras/`;
- scanner, dashboard, and validator-identity unit tests;
- `validation/isras-validator-identity.json`;
- `validation/secret-allowlist.json`;
- `validation/tool-versions.json`;
- `tools/isras/build-validator.sh`.

## Validator identity boundary

The reference validator reads committed identity metadata from
`validation/isras-validator-identity.json`. Its declared standard version must
match the repository `VERSION` file. A mismatch is a validation startup failure,
not a cosmetic warning.

During export, the exporter replaces reference ownership with
`project-owned-export` and records:

- the declared ISRAS profile and standard version;
- the canonical Engineering Standards source repository;
- the exact Engineering Standards source commit used for the export;
- the adopting project's Go module path.

The target repository's current commit is discovered at runtime and is reported
separately from the immutable export source commit. This prevents an exported
validator from presenting itself as the live Engineering Standards repository or
from silently inheriting a later development version.

After building the target-owned validator, inspect the identity directly:

```bash
./.local/bin/isras-validate version
```

The normal validation dashboard also includes the version and ownership class in
its header. Updating an exported validator is a new reviewed export with a new
source commit, not an implicit upstream change.

## Transactional export boundary

The target working tree is not modified while the candidate export is being
assembled. The exporter:

1. records the exact clean target commit;
2. creates a detached scratch clone at that commit;
3. copies and rewrites the validator in the scratch clone;
4. runs `gofmt` and `go mod tidy` there;
5. rejects removal or version changes of existing module requirements;
6. permits an existing requirement to move from indirect to direct;
7. displays resulting `go.mod` and `go.sum` changes;
8. requires a second `go mod tidy -diff` to be empty;
9. runs all Go tests, vet, build, and module verification;
10. creates one exact Git patch from the proven scratch tree;
11. applies and stages that patch in the real target;
12. repeats module, test, vet, build, and verification checks in the target.

If any post-application check fails or the process receives a handled interrupt,
the exporter resets the target to the recorded commit and removes only the new
export paths. A worktree-specific transaction journal is retained only while the
real target is being modified.

Go commands are bounded by a default 900-second timeout so module resolution or
tests cannot remain hidden indefinitely. A project may set
`ISRAS_EXPORT_GO_TIMEOUT_SECONDS` to another positive integer when a larger,
reviewed validation budget is required.

The exporter stages the validator, `.gitignore`, and any deterministic `go.mod`
or `go.sum` changes. It never commits, pushes, tags, or changes a remote ref.

## Project ownership

After export, the project owns the copied source. Changes to the organization
reference validator do not silently alter an adopting repository. A later update
is a normal reviewed source change with a visible diff and rerun tests.

## Required project additions

Each project shall also document:

- its supported operating systems and deployment profiles;
- its project-specific validation commands;
- its security-sensitive change boundaries;
- any additional scanners or specialized tests;
- its release and recovery process;
- its current assurance status.
