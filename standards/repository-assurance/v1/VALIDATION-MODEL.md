# Validation Model

## Developer validation

Fast local checks support iteration. They are necessary but not independent
evidence.

## Portable validation

Runs on clean, ordinary approved systems and fixed GitHub-hosted runners without
private infrastructure access or sensitive secrets. A portable or policy
workflow must not allow the caller to select a self-hosted runner.

The environment doctor validates operating system, architecture, command
presence, command-version patterns, pinned Python validation modules, and named
required environment variables. It can emit a machine-readable fingerprint.

Typical checks:

- committed-tree, staged, and working-tree whitespace and formatting;
- compilation;
- unit and race testing;
- dependency integrity;
- static security analysis;
- JSON Schema, workflow YAML, source-manifest, migration-manifest, and evidence validation;
- documentation synchronization;
- fixture regeneration;
- integration-test compilation;
- credential-pattern checks.

## Fresh-clone validation

Clones the canonical remote into a disposable directory, checks out the exact
pushed commit, and runs the portable entrypoint. It detects missing untracked or
ignored project inputs.

## Canonical validation

Runs exact environment-specific acceptance such as PostgreSQL, systemd,
deployment units, pinned toolchains, resource telemetry, or approved native
host controls.

## Specialized campaigns

Examples include:

- Windows Active Directory multi-forest trust;
- backup and point-in-time recovery;
- database and service failover;
- network degradation;
- multi-instance replay and idempotency;
- workstation reconnect;
- capacity and performance;
- compromise recovery and trusted rebuild.

## Historical validation

Checks out an accepted historical commit in a disposable clone and runs the gate
from that historical tree.

## Result vocabulary

Use separate outcomes:

```text
Correctness result: PASS | FAIL
Resource observation: RECORDED | NOT_RECORDED | NOT_APPLICABLE
Performance budget: PASS | FAIL | NOT_EVALUATED | NOT_APPLICABLE
Security findings: NONE | RECORDED | NOT_EVALUATED
Operational readiness: ACCEPTED | NOT_ACCEPTED | NOT_EVALUATED
```
