# Govulncheck per-module runner

**Status:** WORKSTREAM A CANDIDATE — A4.2 RUNNER ONLY — NOT RELEASED OR ADOPTABLE

## Scope

This step establishes deterministic execution of the exact already verified
govulncheck binary for every repository-owned Go module. It does not yet attach
the results to project-command evidence schema v2 or implement vulnerability
exceptions.

## Module coverage

The runner consumes the accepted sorted module inventory and:

- rejects an empty inventory;
- rejects duplicate `go.mod` paths before any scanner executes;
- revalidates every module directory and `go.mod` path;
- rejects traversal, symbolic links, nonregular module files, and mismatched
  directories;
- executes every accepted module exactly once;
- confirms the execution count equals the governed inventory count.

## Exact execution

Each module runs with:

- the exact verified scanner executable;
- arguments `-json ./...`;
- working directory equal to the module directory;
- selected Go directory first in bounded PATH;
- exact scanner directory in bounded PATH;
- `GOTOOLCHAIN=local`;
- `GOENV=off`;
- isolated HOME, temporary, cache, GOCACHE, and GOPATH directories;
- deterministic locale;
- bounded timeout and stdout/stderr;
- Linux process-group termination;
- Git-visible repository mutation detection.

The full caller PATH is never inherited.

## Protocol and result checks

Successful process exit alone is insufficient. The runner parses the complete
stream through the accepted fail-closed protocol parser and verifies any reported
scanner name, scanner version, source mode, symbol level, and Go version against
the selected and approved identities.

The runner retains per-module timings, exit status, environment names, bounded
stream evidence, protocol summary, advisory identities, and finding-level counts
in typed internal results. Evidence schema v2 integration remains Step 12D.

## Validation

Synthetic fake-scanner tests prove:

- deterministic multi-module execution;
- exact arguments and working directory;
- selected Go PATH precedence;
- forced local/off environment;
- caller PATH exclusion;
- nonzero-exit failure;
- malformed-protocol failure;
- repository-mutation failure;
- output-limit failure;
- timeout and descendant termination;
- duplicate and escaped inventory rejection.

A guarded live candidate test verifies the exact pinned local scanner against the
current Engineering Standards module after all synthetic gates pass.
