# Release and Acceptance Model

## Acceptance boundary

Acceptance identifies:

- the exact source and applicable standard commits;
- the exact accepted predecessor;
- the validation gate, runner identity, and environment fingerprint;
- actual campaign start and finish times;
- correctness, resource, performance, security, and readiness outcomes;
- evidence locations and SHA-256 digests;
- warnings and non-claims;
- the accepted signed tag and release identity.

## Candidate freeze

A candidate acceptance requires a clean exact commit already present on the
canonical `dev` branch and retrievable from the canonical remote.

Candidate validation does not itself record an acceptance decision.

The in-tree acceptance plan, release notes, compatibility statements,
support boundary, and required validation entrypoints must be complete before
the candidate commit is frozen.

## Signed tag as the authoritative acceptance decision

When an approved signing identity and verification path exist, the
cryptographically signed annotated release tag is the authoritative
acceptance-decision object.

The signed tag inherently binds the decision to its exact target commit. Its
annotation must identify:

- decision status and date;
- release version;
- predecessor release and exact predecessor commit;
- validation gate and environment profile;
- runner identity;
- evidence location and SHA-256 digest;
- correctness and applicable assurance outcomes;
- warnings, exceptions, and non-claims.

This model prevents a later source commit from being required merely to
describe acceptance. It therefore prevents acceptance evidence from
unnecessarily leaving `main` behind `dev`.

## Acceptance plans and evidence storage

An in-tree acceptance plan is committed before candidate freeze. The plan
defines the required decision criteria but does not claim that acceptance
has already occurred.

Evidence generated after candidate freeze must be retained in an approved CI
artifact store or evidence repository. Its digest and durable location are
included in the signed tag annotation.

A later narrative or historical record may be added through a future release,
but it cannot alter the identity of the already accepted release.

## Release completion

Release finalization is complete only when:

1. the signed annotated tag verifies successfully;
2. the tag peels to the exact validated candidate commit;
3. remote `main` identifies that same exact commit;
4. remote `dev` identifies that same exact commit at finalization;
5. the evidence digest and decision metadata are retained;
6. the `isras-*` tag namespace is protected from ordinary movement or
   deletion.

After finalization, new development may advance `dev`. That normal
development does not change the accepted release identity.

## Additional release evidence

Release may additionally require:

- a clean trusted build;
- SBOM and dependency-license inventory;
- artifact hashes and provenance;
- signed artifacts and attestations;
- compatibility, upgrade, rollback, support, and deprecation statements;
- installation and deployment identity.

The adopting repository must record the applicable central ISRAS release and exact standard commit in its assurance adoption record.
