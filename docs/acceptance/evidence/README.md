# Acceptance Evidence Records

This directory contains sanitized, durable acceptance records and artifact
hashes. Large raw logs may remain in protected CI or release storage when the
committed record identifies their cryptographic hashes and retention location.

A candidate record must not be represented as accepted. Acceptance records must
identify exact source and standard commits, the runner, environment fingerprint,
validator, timestamps, applicable predecessor, results, warnings, non-claims,
and evidence artifact hashes.

## Retained candidate campaigns

- [`isras-v2.0.1-candidate/`](isras-v2.0.1-candidate/) — exact pushed
  BSD-licensed patch candidate; formal acceptance remains pending.
