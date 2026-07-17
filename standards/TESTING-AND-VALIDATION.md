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
