# Windows Active Directory Lab Integration

## Role

A reproducible Windows AD lab is a specialized assurance environment, not an
ordinary public pull-request runner.

## Required separation

- the orchestrator is not a domain controller or certificate authority;
- the runner is not permanently privileged;
- import authority is delegated to exact lab OUs;
- destructive campaigns use snapshots or clean rebuilds;
- public pull requests cannot trigger the lab;
- lab credentials are not available to portable workflows.

## Deterministic inputs and observed identities

Scenario source, generators, expected counts, schemas, and SHA-256 manifests
should be deterministic.

Fresh AD rebuilds should not assume byte-identical SIDs, GUIDs, trust secrets,
timestamps, or replication metadata. Use stable fixture IDs and produce a
runtime observation manifest that maps fixture identity to observed AD identity.

## Campaigns

Test as applicable:

- forest-wide and selective authentication;
- one-way and two-way trusts;
- foreign security principals;
- cross-forest domain-local membership;
- nested groups;
- broken trusts;
- unresolved principals;
- disabled, stale, renamed, moved, deleted, and recreated identities;
- membership removal and replication delay;
- DNS, global catalog, Kerberos time, SPN, LDAPS, and certificate failure;
- least-privileged import and marker-scoped teardown.

## Platform boundary

AD establishes identity and account state. It does not become the entire
application authorization engine. Application policy, purpose, resource, scope,
approval, and decision state remain governed by the application platform.
