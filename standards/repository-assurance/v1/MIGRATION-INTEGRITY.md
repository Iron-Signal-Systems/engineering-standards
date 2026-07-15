# Migration Integrity

Repositories with database or state migrations must maintain a deterministic,
cryptographically bound migration manifest.

Record:

- migration identifier;
- relative path;
- SHA-256;
- expected order;
- phase or release introduced;
- accepted tag or source commit;
- transaction behavior.

The target system records:

- migration identifier and hash;
- release-manifest hash;
- application time;
- applying identity;
- tool version;
- source commit;
- execution result.

An already-recorded migration whose content hash changed must fail closed.
Corrections use a new migration rather than rewriting accepted history.

Hostile tests cover altered bytes, whitespace changes, reordering, missing or
extra files, duplicate identifiers, manifest corruption, interrupted execution,
wrong release identity, and database-record mismatch.
