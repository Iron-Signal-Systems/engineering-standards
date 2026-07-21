# Go module inventory boundary

**Status:** WORKSTREAM A CANDIDATE — NOT RELEASED OR ADOPTABLE

## Contract

Every project declaring the Go profile must have a governed root module. ISRAS
discovers every repository-owned source `go.mod` through Git's tracked and
nonignored-untracked path inventory, validates each file and declaration, rejects
duplicate module paths, and emits a stable sorted inventory. The reserved
`.local/` tree is excluded because it contains runtime evidence and local tools,
not project source.

The selected Go toolchain must satisfy all discovered module minimums. The highest
module minimum becomes the project-level minimum used for the overall decision.
Each module retains its own path, directory, module identity, minimum, optional
toolchain directive, and comparison result in project-command evidence version 2.

## Exclusions

Discovery does not enter `.git` or the fixed `.local/isras` runtime-evidence
boundary. It does not follow symbolic links and does not treat generated evidence
as source modules.

## Validation

Focused tests cover:

- deterministic three-module discovery;
- duplicate module identity rejection;
- missing root module;
- symbolic and non-regular `go.mod` paths;
- highest-minimum selection;
- nested below-minimum denial;
- exact per-module JSON/text evidence;
- evidence synchronization after module removal.

## Boundary

This record does not authorize vulnerability scanning acceptance, a commit, a
release, a consumer change, or Workstream B.
