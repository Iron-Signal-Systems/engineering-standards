# ISRAS Solo Developer Baseline

## 1. Purpose

This profile provides a serious but sustainable repository-assurance baseline
for projects implemented primarily by one developer.

## 2. Truthful status

The following statuses are permitted:

- `DEVELOPMENT`
- `SELF-VALIDATED`
- `EXTERNAL-REVIEW-PENDING`
- `INDEPENDENTLY-REVIEWED`
- `RELEASED`

Self-authored and self-validated work shall be labeled `SELF-VALIDATED`.
Cryptographic signing establishes attribution and integrity; it does not create
independent review.

## 3. Required repository practices

A project adopting this profile shall:

1. use Git as the authoritative source history;
2. sign commits and annotated release tags;
3. retain the exact test and validation source used for a successful commit;
4. prevent validation from depending on undocumented terminal-only scripts;
5. identify the exact commit being validated;
6. keep implementation, documentation, tests, status, and release records
   synchronized in the same change set;
7. create a local failure log for every failed check;
8. censor possible sensitive values from terminal output and logs;
9. provide exact context-specific remediation commands;
10. record known limitations and unsupported environments.

## 4. Change levels

### Routine

Documentation, comments, spelling, and mechanically harmless cleanup.

### Functional

Ordinary behavior, interfaces, tests, or implementation changes.

### Sensitive

Authentication, authorization, cryptography, audit, secrets, database
boundaries, migrations, deployment security, recovery, or privileged behavior.

### Release

A versioned release or formally accepted project milestone.

Each higher level includes applicable requirements from lower levels.

## 5. Non-claims

A passing self-validation run does not establish independent assurance,
certification, regulatory compliance, production readiness, complete security,
or absence of vulnerabilities.
