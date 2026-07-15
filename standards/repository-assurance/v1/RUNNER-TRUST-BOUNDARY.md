# Runner Trust Boundary

## Hosted pull-request runners

Public and ordinary pull-request validation uses clean GitHub-hosted runners or
an equivalently isolated service.

These jobs receive:

- source under review;
- read-only repository access;
- no production or specialized-lab credentials;
- no private-network path to sensitive systems;
- no signing or deployment authority.

## Self-hosted runners

Self-hosted runners are treated as sensitive infrastructure.

They must not run untrusted public pull-request code. Canonical or specialized
jobs run only against an exact protected commit and through an approved trigger.

A canonical runner must not:

- be a development workstation;
- share a developer home directory;
- contain personal SSH keys;
- hold unrelated long-lived tokens;
- run multiple unrelated trust domains without accepted isolation;
- retain unreviewed state between campaigns.

## Preferred lifecycle

```text
approved commit
  ↓
new VM or known-good snapshot
  ↓
one bounded campaign
  ↓
sanitized evidence export
  ↓
destroy or revert
```

Just-in-time runner registration is preferred where practical. Reused hardware
still requires a clean environment.

## Windows lab runner

The Windows orchestrator is not a domain controller, certificate authority, or
permanent privileged administration workstation. It receives delegated lab-only
authority and is reverted or rebuilt after hostile campaigns.
