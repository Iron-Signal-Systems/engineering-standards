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
- source commit;
- workflow or validator;
- runner or environment profile;
- start and end times;
- correctness result;
- evidence schema version;
- file SHA-256 values.
