# ISRAS v1.0.0 Release Finalization Record

## Decision

**Status: COMPLETE**

ISRAS v1.0.0 was formally accepted and its release identity was finalized on
2026-07-15.

## Accepted release boundary

- Version: `1.0.0`
- Acceptance tag: `isras-v1.0.0`
- Accepted source commit:
  `f9655ddbbf04430fc468aab405f2ed880df3e97d`
- Release branch: `main`
- Verified release-branch commit:
  `f9655ddbbf04430fc468aab405f2ed880df3e97d`

## Signed tag boundary

- Annotated tag object:
  `3f7d4e7f5b340c65cfe74f757ba0a24b2f94cc2b`
- Peeled target:
  `f9655ddbbf04430fc468aab405f2ed880df3e97d`
- Signing format: `SSH`
- Signing algorithm: `ED25519`
- Signing identity: `kb2vhn@gmail.com`
- Signing-key fingerprint:
  `SHA256:lCGPFFvNgV2sb/tLWTptypn9lihvc8Q6er5C/KgKkSU`
- Signature verification: `PASS`

## Completed actions

1. The incorrect pre-acceptance tag was replaced through the authorized
   controlled-correction procedure.
2. The corrected annotated tag was cryptographically verified.
3. The corrected tag was verified to target the exact accepted source.
4. `main` was fast-forwarded without force to that same exact source.
5. Remote `main`, the tag object, and the peeled tag target were independently
   verified.

## Evidence lineage

- Accepted source:
  `f9655ddbbf04430fc468aab405f2ed880df3e97d`
- Formal acceptance record commit:
  `7b3d7921e430fc203324655f6e92a88344a56746`
- Tag-correction authorization commit:
  `43195fe98404f383f9c3719ac1b8d9b343e231a8`

## Result

ISRAS v1.0.0 is formally accepted, signed, released, and available for exact
commit adoption.

This finalization record is retained in the later repository history. It does
not move or redefine the immutable v1.0.0 source boundary.
