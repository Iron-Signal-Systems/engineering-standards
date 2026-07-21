# Testing and Validation

## 1. Committed tests

Tests and validation programs are repository assets. The exact successful test
source shall remain available in the commit history associated with the tested
implementation.

Temporary commands may be used for investigation, but a result shall not be
accepted until the required check exists as committed project-owned source or a
declared standard tool invocation.

## 2. Go baseline

Go projects shall run, as applicable:

```text
gofmt validation
go vet ./...
go test -count=1 ./...
go build ./...
go mod tidy -diff
go mod verify
govulncheck ./...
```

Normal validation shall not rewrite source. Repair actions shall be explicit and
labeled as modifying the working tree.

`govulncheck` is the bounded external tool exception. The project pins the
required version in `validation/tool-versions.json`. If the tool or current
vulnerability data is unavailable, validation shall not claim that no known
vulnerability exists.

### Validator version and source identity

A repository-owned validator shall report a committed, machine-readable identity.
The identity shall distinguish the Engineering Standards reference repository from
a project-owned export and shall include, as applicable:

- the ISRAS profile and declared standard version;
- the canonical Engineering Standards source repository;
- the exact source commit used for an export;
- the adopting Go module;
- the current repository commit where the validator is running.

Reference identity metadata shall match the repository `VERSION` file. Exported
identity metadata shall pin an exact source commit and shall not claim reference
repository ownership. Unknown identity fields, unsupported schema versions,
invalid commit identities, missing target modules, and version drift shall fail
closed before validation results are rendered.

The validator shall expose this evidence through a dedicated `version` command and
shall include the ownership class in normal dashboard headers. Project-owned
exports do not silently inherit later Engineering Standards versions.

### Transactional project-validator export

A project-owned validator export shall be assembled and validated against a
scratch clone of the target's exact clean commit before the target working tree
is modified. Ordinary clones and linked worktrees shall be identified through
Git rather than through assumptions about the physical `.git` path.

The normal export operation may run `go mod tidy` and stage deterministic
`go.mod` or `go.sum` changes required by copied validator imports. It shall:

- show module-file changes for review;
- allow an existing requirement to move from indirect to direct;
- reject removal or version changes of existing requirements;
- require a second `go mod tidy -diff` to be empty;
- run tests, vet, build, and module verification before and after transfer;
- apply the exact proven patch and stage it without committing or pushing;
- restore the recorded target commit after any failed applied validation;
- bound Go command duration so network or tool execution cannot hang silently.

## 3. Secret protection

Tracked, staged, modified, and untracked source shall be scanned for likely
credentials, private key material, authorization headers, embedded URL
credentials, dangerous credential filenames, and suspicious sensitive
assignments.

The working tree and staged Git index are separate evidence sources. A clean
working-tree copy shall not conceal sensitive content already staged for commit,
and a clean staged copy shall not conceal sensitive content present only in the
working tree. Identical index and working-tree content may be scanned once.

Dangerous credential filenames shall be evaluated before content-size,
text-encoding, or binary-content skips. A credential container does not become
acceptable merely because its contents are binary or exceed the text-scanning
limit.

Staged-index findings shall be identified as staged evidence and shall not offer
automatic working-tree redaction. The affected path must be unstaged, corrected,
restaged, and revalidated.

Approved external-secret reference schemes are `secret://`, `vault://`,
`keyring://`, and `credential://`. Recognition of those references suppresses
only assignment-literal classification; embedded user/password material and
other credential-shaped content inside a URI remain findings. Unknown schemes
are not automatically trusted.

Valid Go source is classified using Go syntax positions. Identifier and selector
expressions such as `config.ClientSecret` are references rather than committed
literal values. Quoted string literals remain scannable, and malformed Go-like
source receives ordinary text classification rather than inferred semantics.

For shell source, variable references, command substitutions, arithmetic
expansions, and dynamically constructed assignments are not treated as committed
assignment literals. Their command bodies and surrounding source remain subject
to the ordinary scanner rules.

Scanner regression fixtures shall construct credential-shaped test values at
runtime so the scanner package remains safe under its own repository-wide scan.

The scanner shall:

- never display or log the complete detected value;
- identify findings with stable IDs and non-secret fingerprints;
- fail on unresolved findings;
- provide a redaction workflow for safely replaceable findings;
- permit a bounded allowlist only for verified false positives, placeholders,
  or deliberately inert test fixtures;
- refuse an allow workflow for private keys and strongly credential-shaped
  findings;
- keep the actual detected value out of allowlists and proposals.

The reference scanner offers allowlist actions only for bounded documentation,
example, `testdata`, and `_test.go` contexts. Ordinary application and deployment
source must be corrected rather than excepted.

## 4. Validation modes

- `development`: permits an intentionally modified working tree but reports it.
- `commit`: requires a clean working tree and verifies the exact current commit.
- `release`: applies commit requirements and is reserved for a declared release
  procedure.

## 5. Result interpretation

- `PASS`: the declared check completed and its expected condition was met.
- `FAIL`: a required check failed or could not produce a trustworthy result.
- `WARN`: the condition needs review but does not independently invalidate the
  current mode.
- `INFO`: supporting context or an available path/action.

### Exact govulncheck acquisition boundary

The approved command package and version are read from
`validation/tool-versions.json`. Hosted validation installs that exact identity
into `.local/tools/bin` in an explicit network-enabled acquisition step, verifies
the resulting binary's embedded command path, module root, and version, and only
then makes the directory available to project-command execution.

The runtime command declaration remains exactly `govulncheck ./...`. Runtime
validation shall fail when the binary is absent or its identity differs; it shall
never fall back to `go run`, `go install`, `latest`, or an implicit upgrade.

### Govulncheck protocol parser validation

The vulnerability-scanner protocol parser is tested independently with synthetic
concatenated JSON streams. Tests cover valid configuration, progress, SBOM, OSV,
and finding messages; malformed JSON; scalar and null values; empty or multi-
field messages; unknown fields; missing or duplicate configuration; missing
advisory identities; and module, package, symbol, and unknown finding levels.

These parser tests do not contact the vulnerability database or execute a scanner.
Live tool identity, execution, evidence, and policy behavior are validated in
separate controlled steps.

### Govulncheck tool-identity validation

The exact-tool boundary is tested with synthetic selected-Go probes and scanner
files. Tests cover the approved declaration, unknown and trailing JSON,
unsupported package and version declarations, symlinked configuration, missing
tools, symlinked tools, wrong command package, wrong module, wrong version, and
successful SHA-256 and build-Go-version capture.

A missing scanner must fail before the selected Go probe runs, demonstrating that
the identity verifier has no installer or acquisition behavior.

### Govulncheck per-module runner validation

The per-module runner uses fake scanners to prove deterministic coverage, exact
`-json ./...` arguments, module-specific working directories, selected-Go PATH
precedence, forced local/off settings, bounded output and timeout behavior,
process termination, repository mutation detection, protocol parsing, and
hostile inventory rejection.

After synthetic tests pass, a guarded candidate test runs the exact approved
local scanner against every discovered Engineering Standards module and records
the resulting protocol summary in the validation log.

### Govulncheck evidence-projection validation

Synthetic tests prove deterministic module ordering, exact inventory coverage, identity matching, fixed package scope, scanner and protocol field projection, and defensive cloning of retained slice data. Coverage drift fails before project-command evidence integration.

### Govulncheck evidence schema and renderer validation

Tests marshal typed scanner evidence, inspect the governed v2 JSON shape, render the corresponding text evidence, and verify schema definitions and the dedicated pass example. Existing v1 and non-vulnerability v2 artifacts remain unchanged.

### Govulncheck runtime-orchestrator validation

Dependency-injected tests prove exact call order and path derivation, complete
module/evidence propagation, boundary-error short-circuiting, absolute
configuration requirements, module/package observation behavior, reachable
finding failure, unknown-level failure, deterministic advisory ordering, and the
absence of an acquisition path.

### Complete govulncheck Execute validation

Integration tests prove specialized dispatch, ordinary-command separation, configuration-path confinement, typed pass and reachable-finding failure evidence, and conditional schema requirements. A guarded live test invokes public `Execute` against a temporary governed project using the real pinned scanner and configuration.

### Govulncheck exception-document validation

Synthetic tests cover valid deterministic ordering and reject unsupported schema versions, duplicate exact scopes, wildcards, traversal, reserved evidence paths, missing symbols, weak governance text, absent or duplicate controls, self-approval, future or non-UTC approval, expiration, remediation after expiration, multiline data, unknown fields, multiple JSON values, relative or escaped files, symlinked paths, and nonregular files.

### Exact exception reconciliation validation

Tests prove receiver-qualified symbol normalization, exact finding retention, duplicate-trace aggregation, deterministic used/unused/unexcepted ordering, unknown-finding summaries, one-field scope mismatch behavior, and fail-closed rejection of duplicate modules, duplicate exception scopes, or symbol-count/detail drift.

### Exception-aware policy and evidence validation

Tests cover absent and present documents, digest retention, symlink rejection,
governance-data projection, defensive cloning, exact-used success, unknown
failure, unexcepted failure, unused-record failure, exception-aware runtime
success and mismatch failure, JSON/text rendering, and schema/example
synchronization. The guarded live public `Execute` test is rerun with no
exception document to prove the zero-exception production path.

## Documentation-impact validation

The documentation-impact evaluator is tested with documentation-only changes,
synchronized and unsynchronized implementation changes, schema/example
coordination, workflow-specific standards, overlapping self-governance rules,
deterministic ordering, unsafe changed paths, invalid pattern forms, duplicate
identifiers, unknown fields, multiple JSON values, repository escape, and
symlinked policy paths.

### Documentation-impact Git, CLI, and hosted enforcement

Validation covers exact commit IDs, repository-root identity, merge-base
selection, rename-disabled changed-path collection, NUL parsing, bounded Git
output, deterministic evidence, policy digests, passing and failing reports,
failure-evidence retention, atomic private writes, symlinked evidence paths, CLI
option parsing, workflow event-range resolution, always-retained artifacts, and a
temporary full-candidate commit evaluated by the newly built validator.

### Governed documentation-impact policy integration

The test suite loads `validation/documentation-impact-policy.json` from the
repository rather than validating only synthetic fixtures. It discovers current
`internal/release*` and `cmd/isras-release*` directories, requires an exact
trailing-slash trigger for each one, and evaluates a synchronized release change
set through the actual policy.

## Workstream A complete local acceptance

The complete A1-A6 candidate is assembled and committed only in a disposable
clone based on the exact PR base. The acceptance record is included before the
commit so that no documentation is added after testing.

The campaign runs complete tests, race tests, vet, build, validator commands,
JSON and shell syntax validation, documentation and workflow contracts,
documentation-impact enforcement, and live public govulncheck `Execute`
validation with the already-acquired exact pinned scanner.

A separate disposable commit changes implementation without the required
changelog, standard, or record. Acceptance requires that the
documentation-impact command reject that commit and retain structured JSON and
text failure evidence. The actual working branch remains unstaged and
uncommitted.
