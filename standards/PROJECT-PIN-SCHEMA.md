# Project Pin Schema

## Purpose

Every adopting Iron Signal Systems project commits one authoritative ISRAS pin at:

```text
.isras/project.json
```

The pin selects one accepted ISRAS release and records the exact identities needed
for local validation, hosted validation, project initialization, evidence, and a
later explicit upgrade.

The v1 JSON Schema is committed at:

```text
schemas/isras-project-v1.schema.json
```

The Go implementation in `internal/projectpin` is the authoritative parser for
cross-field and bounded-validation rules that JSON Schema cannot express alone.

## Read-only commands

The reference validator exposes:

```bash
./.local/bin/isras-validate project-pin validate
./.local/bin/isras-validate project-pin inspect
```

Both commands are read-only. They load only the canonical `.isras/project.json`
inside the current repository. They do not download artifacts, execute declared
project commands, modify the project, update the pin, or select a newer release.
Project initialization is a separate explicit command and authority boundary.

`inspect` reports the pin as a declaration. It labels artifact counts and digests
as declared values, explicitly states that artifact verification was not
performed, and reports that no artifact bytes were acquired or hashed. Digest
values are abbreviated in ordinary terminal output so a metadata fixture cannot
be mistaken for cryptographic comparison evidence. Complete digest values remain
in the parsed pin for later machine comparison and evidence.

`inspect` reports command names but deliberately does not print command
arguments.

## Artifact verification command

The next read-only boundary is:

```bash
./.local/bin/isras-validate project-pin verify-artifacts
```

This command reads the exact published GitHub release, verifies its signed
annotated tag and source commit, acquires only the declared assets, hashes their
actual bytes, compares both digests, checks both manifests, validates provenance,
and writes local evidence. It never executes or extracts an artifact.

A local source directory may be checked with `--source-directory PATH`. Local
mode verifies bytes, manifests, and provenance but reports execution
authorization as DENIED because the published release record and signed tag were
not checked.

See
[`ARTIFACT-ACQUISITION-AND-VERIFICATION.md`](ARTIFACT-ACQUISITION-AND-VERIFICATION.md).

## Initialization command

A linker-bound accepted release validator may generate a first pin only through:

```bash
isras-validator-linux-amd64 --repo /src/example-project project-pin initialize --release isras-v0.1.2 --go-defaults
```

The command verifies the exact release and reusable workflow before writing,
requires a clean canonical Iron Signal Systems target, publishes the complete
adoption set without replacement, and leaves the changes uncommitted. See
[`PROJECT-INITIALIZATION-AND-ADOPTION.md`](PROJECT-INITIALIZATION-AND-ADOPTION.md).

## Top-level fields

The v1 object contains exactly:

- `schema_version`;
- `project`;
- `standard`;
- `artifacts`;
- `workflow`;
- `profiles`;
- `commands`;
- `evidence`.

Unknown and duplicate fields fail closed at every nesting level. Multiple JSON
values and trailing non-whitespace data are rejected.

The file is bounded to 256 KiB.

## Project identity

`project.repository` is the canonical repository identity. The initial profile
requires:

```text
github.com/Iron-Signal-Systems/REPOSITORY
```

The field contains no scheme, credentials, query, fragment, or `.git` suffix.

## Standard identity

The `standard` object requires:

- `profile`: exactly `ISRAS-SD` for the current baseline;
- `version`: a stable `MAJOR.MINOR.PATCH` value;
- `release_tag`: exactly `isras-v` followed by that version;
- `source_repository`: the canonical Engineering Standards repository;
- `source_commit`: the exact nonzero 40-character lowercase release commit.

Development versions, floating refs, version ranges, shortened commits, uppercase
commits, and all-zero placeholders are invalid.

## Release artifacts

Each artifact records:

- a supported `kind`;
- a safe basename;
- SHA-256 and SHA-512 digests;
- operating system and architecture only for a validator binary.

The v1 schema supports:

- `validator`;
- `framework`;
- `contracts`;
- `provenance`;
- `sha256-manifest`;
- `sha512-manifest`;
- optional `migration`.

A valid pin requires at least one validator artifact and exactly one framework,
contracts, provenance, SHA-256 manifest, and SHA-512 manifest artifact. At most
one migration artifact is allowed.

Artifact names must be unique. Validator operating-system and architecture pairs
must be unique. Digests must be lowercase, exact length, and nonzero.

The pin declares the artifact bytes expected by the selected release. Step 2
checks only that those declarations are structurally and semantically valid. It
does not establish that any artifact exists, was downloaded, was hashed, matches
a manifest, or is safe to execute.

Acquisition, signed-release verification, local hashing, complete digest
comparison, manifest membership, provenance binding, and execution authorization
are separate later implementation steps.

## Reusable workflow identity

The `workflow` object requires:

- the canonical Engineering Standards repository;
- `.github/workflows/validate-project.yml`;
- an exact nonzero 40-character lowercase commit.

The workflow commit must equal `standard.source_commit`. This prevents the local
validator release and hosted workflow from silently following different ISRAS
versions.

## Profiles

The current v1 implementation supports the `go` project profile. A future
accepted ISRAS release may add another profile and update the schema and parser.
An older pinned release does not silently learn that profile.

Profiles are unique and ordered project declarations.

## Project-owned commands

Commands are JSON arrays, not opaque shell strings. The first element is the
executable and the remaining elements are its arguments.

For the Go profile, the pin requires command declarations named:

- `format_check`;
- `static_analysis`;
- `test`;
- `build`;
- `module_consistency`;
- `module_integrity`;
- `known_vulnerabilities`.

Additional project-specific commands are allowed when their names match the
schema.

The parser bounds command count, argument count, individual argument length, and
total argument bytes. Empty arguments and NUL, carriage-return, or newline
characters are rejected. The executable must be a single argument without
whitespace.

Declaration validation alone does not execute commands or claim that they satisfy
the project profile. A linker-bound release validator may execute one exact
committed declaration through `project-command run NAME` only after the separate
authorization, runtime, repository-state, and evidence controls in
[`PROJECT-COMMAND-EXECUTION.md`](PROJECT-COMMAND-EXECUTION.md) pass.

## Evidence location

`evidence.directory` is a normalized relative slash-separated path. Absolute
paths, traversal, backslashes, control characters, `.` and `..` segments, and
paths inside `.git` are rejected.

The directory is project-owned. The schema does not decide which evidence is
tracked or private; the pinned release and project policy define that boundary.

## Relational rules

The Go parser additionally enforces rules not fully represented by the JSON
Schema, including:

- release tag equals `isras-v` plus the version;
- workflow commit equals the source commit;
- required artifact-kind counts;
- unique artifact names and validator platforms;
- current supported profile selection;
- Go command-name requirements;
- file and aggregate size budgets.

Passing project-pin declaration validation proves only that the pin is
structurally acceptable to this implementation. Terminal output shall therefore
use `Declaration status: PASS`, `Artifact verification: NOT PERFORMED`,
`Artifacts declared`, and `Declared SHA-*` labels. It shall not use a bare
`Artifacts` heading or an unlabeled checksum list that can be mistaken for
verified evidence.

This step does not prove that the release tag exists, is signed, points to the
declared commit, or that the declared artifact bytes are available. Those checks
belong to artifact acquisition and release verification.

## Example shape

The following is illustrative. Angle-bracket values are not valid production
values and must be replaced from an accepted release manifest:

```json
{
  "schema_version": 1,
  "project": {
    "repository": "github.com/Iron-Signal-Systems/example-project"
  },
  "standard": {
    "profile": "ISRAS-SD",
    "version": "0.1.5",
    "release_tag": "isras-v0.1.5",
    "source_repository": "github.com/Iron-Signal-Systems/engineering-standards",
    "source_commit": "<exact-release-commit>"
  },
  "artifacts": [
    {
      "kind": "validator",
      "os": "linux",
      "arch": "amd64",
      "name": "isras-validator-linux-amd64",
      "sha256": "<exact-sha256>",
      "sha512": "<exact-sha512>"
    }
  ],
  "workflow": {
    "repository": "github.com/Iron-Signal-Systems/engineering-standards",
    "path": ".github/workflows/validate-project.yml",
    "commit": "<same-exact-release-commit>"
  },
  "profiles": ["go"],
  "commands": {
    "format_check": ["./.isras/check-go-format"],
    "static_analysis": ["go", "vet", "./..."],
    "test": ["go", "test", "./..."],
    "build": ["go", "build", "./..."],
    "module_consistency": ["go", "mod", "tidy", "-diff"],
    "module_integrity": ["go", "mod", "verify"],
    "known_vulnerabilities": ["go", "run", "golang.org/x/vuln/cmd/govulncheck@v1.6.0", "./..."]
  },
  "evidence": {
    "directory": ".local/validation"
  }
}
```

A production pin also includes the required framework, contracts, provenance,
and manifest artifacts described above.
