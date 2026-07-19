# Hosted SSH Signer Trust

## Purpose

ISRAS hosted validation shall verify an SSH-signed consuming-project commit only
after establishing an explicit signer-trust source controlled by the exact pinned
Engineering Standards source. A clean hosted runner has no durable local
`gpg.ssh.allowedSignersFile`; signature presence alone is not verification.

## Authority boundary

The consuming commit shall not select, replace, extend, or weaken the signer
trust used to validate itself. Hosted signer trust is therefore committed under:

```text
trust/ssh/iron-signal-systems.allowed-signers
trust/ssh/iron-signal-systems.allowed-signers.sha256
trust/ssh/manifest.json
```

The reusable workflow is called by an immutable Engineering Standards commit.
The trust files and bootstrap tool must be tracked at that exact commit, match
their Git blobs, and match the committed SHA-256 and signer manifest before any
target-local Git trust configuration is created.

## Runtime configuration

The pinned bootstrap copies the accepted allowed-signers bytes into a new private
runtime directory and configures only the checked-out target repository:

```text
gpg.format=ssh
gpg.ssh.allowedSignersFile=<private runtime copy>
```

The runtime copy is not accepted as authority until its digest matches the pinned
source. Global runner configuration is not modified.

## Principal binding

A successful `git verify-commit HEAD` is necessary but not sufficient. The
reported SSH principal and key fingerprint must appear together in the pinned
manifest, and the verified principal must equal the exact commit's committer
email. This rejects a correct key placed under an unauthorized or incorrect
principal.

## Required rejection behavior

Hosted validation shall fail closed when any of the following is true:

- the trust source is missing, untracked, symbolic, modified, or digest-mismatched;
- the manifest and allowed-signers inventory differ;
- no trusted signer is declared;
- the commit is unsigned or cannot be verified;
- the signature uses a key outside the pinned signer inventory;
- the reported principal differs from the commit committer identity;
- the runtime trust directory already exists or cannot be secured.

No unsigned-commit exception or consuming-project trust override is permitted by
this baseline.

## Evidence retention

Successful trust establishment writes private evidence under
`.local/isras`. Failures write a private redacted log under
`.local/validation/logs`. Reusable hosted validation must retain both trees with
`if: always()` so a signature failure is not discarded merely because later
validation steps were skipped.

## Testing

The accepted implementation must exercise a clean temporary runner boundary and
prove:

- acceptance of the correct key and principal;
- rejection of missing trust;
- rejection of altered trust bytes;
- rejection of a wrong key;
- rejection of a wrong principal even when the cryptographic key is correct.

A real consuming-project hosted run remains required before formal adoption of a
release containing this boundary.

## Rotation

Signer additions, removals, principal changes, or key rotation are Engineering
Standards changes. They require review, regression testing, a new signed ISRAS
release, and explicit consuming-project upgrade. A floating GitHub account key
registry is discovery input during preparation; it is not runtime trust
authority.
