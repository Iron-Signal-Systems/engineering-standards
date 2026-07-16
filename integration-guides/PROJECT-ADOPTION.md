# Project Adoption

The active baseline is designed so each adopting project retains the exact Go
validator and its unit tests in that project's own Git history.

## Export the reference validator

From a clean `engineering-standards` working tree:

```bash
./tools/export-project-validator.sh /path/to/target-project
```

The target must be a clean Git repository with an existing `go.mod` file. The
exporter copies:

- `cmd/isras-validate/`;
- validator packages under `internal/isras/`;
- scanner and dashboard unit tests;
- `validation/secret-allowlist.json`;
- `validation/tool-versions.json`;
- `tools/isras/build-validator.sh`.

It rewrites Go import paths to the target module, runs formatting, tests, vet,
build, module-tidy comparison, and module verification, then stages the files.
It does not commit or push.

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
