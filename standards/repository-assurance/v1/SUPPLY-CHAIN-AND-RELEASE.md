# Software Supply Chain and Release Assurance

## Development checks

- dependency lock and checksum verification;
- vulnerability analysis;
- secret scanning;
- workflow permission analysis;
- immutable references for external actions and reusable workflows;
- dependency and license inventory.

## Release outputs

A release should produce as applicable:

- source archive;
- binaries or packages;
- deployment files;
- migration manifest;
- SPDX or CycloneDX SBOM;
- SHA-256 manifest;
- build environment record;
- acceptance summary;
- provenance;
- signature or artifact attestation.

## Separation

Keep development, release, and installation evidence separate.

## Trusted build

A release builder must be disposable or returned to a known-good state and must
not share a developer home directory, personal SSH keys, caches, or unrelated
credentials.

## Compromise response

Document:

- vulnerable or compromised dependency response;
- signing-key compromise;
- workflow compromise;
- runner compromise;
- artifact revocation;
- trusted rebuild;
- replacement release and customer notification.
