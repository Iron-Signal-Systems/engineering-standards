# Release and Acceptance Model

Acceptance identifies:

- exact source and standard commits;
- exact predecessor;
- gate, runner identity, and environment fingerprint;
- actual start and finish times;
- correctness, resource, performance, security, and readiness outcomes;
- evidence hashes;
- warnings and non-claims;
- accepted tag or release identity.

A passing acceptance record requires a clean exact commit already present on the
canonical development branch. Candidate validation does not itself record an
acceptance decision.

Release adds:

- clean trusted build;
- SBOM and dependency-license inventory;
- artifact hashes and provenance;
- signed artifacts, attestations, and signed annotated tags where supported;
- documented signing exceptions where signing is not yet available;
- compatibility, upgrade, rollback, support, and deprecation statements;
- installation and deployment identity.

The adopting repository must record the applicable central ISRAS release, versioning, support, and deprecation requirements in its assurance adoption record.
