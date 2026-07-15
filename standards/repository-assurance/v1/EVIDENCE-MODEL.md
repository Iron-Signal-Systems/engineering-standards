# Acceptance Evidence Model

## Evidence classes

### Repository evidence

Small, sanitized, durable records committed to Git:

- exact source commit;
- standard commit;
- accepted predecessor;
- gate identity;
- result totals;
- environment fingerprint;
- resource and raw-log hashes;
- artifact hashes;
- known warnings;
- deviations and explicit non-claims.

### Workflow evidence

CI logs and machine-readable artifacts retained by the CI platform according to
the repository retention policy.

### Restricted evidence

Sensitive operational, identity, recovery, or security evidence stored outside
the public source repository with access control and a committed digest.

## Redaction

Evidence must not contain:

- passwords or full connection strings;
- HMAC, signing, or encryption keys;
- bearer tokens;
- private certificates;
- protected production records;
- unrestricted database logs;
- personal or privileged identity data beyond approved synthetic fixtures.

## Evidence identity

Every evidence package identifies:

- repository;
- exact 40-character source and ISRAS commits;
- source branch and optional acceptance tag;
- workflow or validator;
- explicit runner identity and machine-readable environment fingerprint;
- actual start and end times with timezone offsets;
- correctness, resource, performance, security, and operational-readiness results;
- evidence schema version;
- file SHA-256 values;
- warnings and explicit non-claims.

A `PASS` acceptance record requires a clean exact source commit already present
at the canonical development-branch head. `SELF` is resolved to the exact source
commit only for the central standards repository. `UNPINNED-BOOTSTRAP` is not
permitted in formal passing evidence.
