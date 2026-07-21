# Govulncheck exception-aware policy and evidence

**Status:** WORKSTREAM A CANDIDATE — A4.3 COMPLETE WORKING-TREE CANDIDATE — NOT RELEASED OR ADOPTABLE

## Scope

This step integrates the governed exception document and exact reconciliation
results into the mandatory govulncheck runtime, evidence schema v2, JSON output,
text output, and result policy.

It does not commit, push, merge, tag, release, modify Iron File Intelligence or
Iron Atlas, or begin Workstream B.

## Exception source

The only recognized exception path is:

```text
.isras/govulncheck-exceptions.json
```

The file is optional. Absence means that no exceptions are declared. A present
file must satisfy the accepted schema and parser boundary. The runtime records
its repository-relative path and SHA-256 digest and rejects symlinked parents,
nonregular files, oversized files, parse failures, expiration, governance
failures, or changes while it is being evaluated.

## Result policy

After every governed module is scanned:

- module-level findings remain recorded observations;
- package-level findings remain recorded observations;
- unknown-level findings fail closed;
- reachable symbol findings without an exact exception fail;
- unused or unmatched exception records fail;
- only exact used exceptions permit their matching reachable finding.

An exact exception must match advisory ID, governed `go.mod`, vulnerable module,
vulnerable package, and canonical symbol. A match does not silence or alter the
scanner stream; it changes only the governed result evaluation after the original
finding has been retained.

## Evidence

Evidence v2 records:

- whether the exception document exists;
- governed path, SHA-256, schema version, and evaluation time;
- every used exception and its full approval, controls, expiration, and
  remediation data;
- the exact matching finding, fixed versions, and occurrence count;
- every unused exception;
- every unexcepted reachable finding;
- every unknown-finding module summary.

A passing `known_vulnerabilities` result requires this exception-evaluation
evidence even when no exception file exists.

## Truthful limitations

Govulncheck reachability is static analysis. Dynamic behavior through reflection,
`unsafe`, assembly, plugins, generated behavior, or unanalyzed execution paths
may not be represented completely. Results also depend on the selected local Go
toolchain and the vulnerability database available during the scan.

Module- and package-level findings indicate dependency exposure but are not
treated as proof of symbol reachability. Exceptions apply only to exact
symbol-level findings emitted by the scanner and cannot suppress scanner output,
database records, or limitations.
