# Govulncheck exact tool identity

**Status:** WORKSTREAM A CANDIDATE — A4.2 TOOL IDENTITY ONLY — NOT RELEASED OR ADOPTABLE

## Scope

This step establishes strict loading of the governed govulncheck declaration and
exact verification of an already acquired scanner binary. It does not install,
download, upgrade, execute a vulnerability scan, extend project-command evidence,
or implement vulnerability exceptions.

## Governed configuration

The loader reads an absolute, regular, nonsymlink
`validation/tool-versions.json`-compatible file with a bounded size.

It requires:

- configuration version `1`;
- a `govulncheck` declaration;
- command package `golang.org/x/vuln/cmd/govulncheck`;
- an exact semantic version such as `v1.6.0`;
- no unknown fields inside the govulncheck declaration;
- one JSON value only.

There is no hardcoded version fallback.

## Binary verification

The verifier requires absolute regular executable paths for both the selected Go
binary and govulncheck. Symbolic links, directories, missing files, and
nonexecutable files fail closed.

The selected Go executable runs exactly:

```text
go version -m <exact-govulncheck-path>
```

with:

- `GOTOOLCHAIN=local`;
- `GOENV=off`;
- a bounded PATH;
- deterministic locale;
- a bounded execution time;
- bounded stdout and stderr.

The embedded command package, module root, and exact version must match the
governed declaration. The verifier records the exact executable, directory,
build Go version, and SHA-256 digest.

## No acquisition path

Missing or mismatched tools fail. The verifier contains no `go install`, `go
run`, network, upgrade, or fallback path.

## Remaining A4.2 work

The exact verified identity is not yet connected to the per-module scanner
runner or project-command evidence. Those remain Step 12C and Step 12D.
