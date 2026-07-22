# Release Artifact Production

## Purpose

The release-artifact producer creates the exact local bytes later published for
an accepted ISRAS release. It produces artifacts only from a clean repository at
an exact signed commit and signed annotated release tag.

Artifact production is not publication. This step performs no GitHub Release
creation, upload, tag push, branch update, archive extraction, validator use
against a consuming project, or consuming-project modification.

## Command

Build the producer:

```bash
./tools/build-release-artifacts.sh
```

After a stable release commit has passed its acceptance campaign and a signed
annotated tag points directly to that commit, produce the local artifact set:

```bash
./.local/bin/isras-release-artifacts build \
  --version 0.1.1 \
  --published-at 2026-07-17T20:00:00Z \
  --validation-campaign isras-v0.1.1-release-acceptance \
  --release-authority 'Iron Signal Systems release authority'
```

`--published-at` is an explicit, reviewable RFC3339 timestamp recorded in the
release provenance. It is never silently replaced with the current time during a
rebuild.

## Required source boundary

Production requires all of the following:

- a clean Git repository, including no untracked files;
- an exact nonzero 40-character `HEAD` commit;
- a valid signature on that commit;
- a stable `MAJOR.MINOR.PATCH` value in `VERSION` at the exact commit;
- an annotated signed `isras-vMAJOR.MINOR.PATCH` tag;
- the tag resolving directly to `HEAD`;
- canonical Engineering Standards `origin` identity;
- an exact patch-level minimum Go directive in `go.mod`;
- a valid selected Go toolchain satisfying that declared minimum, including
  later compatible releases and valid custom toolchain suffixes.

The complete source boundary is checked before and after production. A changed
commit, tag, toolchain, or working tree invalidates the candidate.

## Produced artifact set

The initial production set is exactly:

```text
isras-validator-linux-amd64
isras-project-framework.tar.gz
isras-contracts.tar.gz
provenance.json
SHA256SUMS
SHA512SUMS
```

No extra file is placed in the artifact directory.

## Embedded validator identity

The validator binary contains linker-bound immutable values for:

- stable ISRAS version;
- release tag;
- exact Engineering Standards source commit;
- release-artifact ownership class.

A release validator does not read the deprecated copied
`validation/isras-validator-identity.json` from a consuming repository. A target
file cannot claim release-artifact ownership.

The producer executes only the binary's read-only `version` command and requires
its reported identity to match the signed source boundary before packaging
continues. The command requires no target repository. The produced binary later
selects a consuming repository through the explicit boundary defined in
[`EXTERNAL-TARGET-ROOT.md`](EXTERNAL-TARGET-ROOT.md). It does not validate a
consuming project during artifact production.

## Deterministic archives

The framework and contract archives use explicit committed file lists:

```text
release/framework-files.txt
release/contract-files.txt
```

The lists must be LF-terminated, sorted, unique, normalized relative paths. Each
path is read from the exact Git commit, not from the working tree.

Archive production normalizes:

- path order;
- owner and group identity;
- file mode to tracked regular or executable mode;
- modification time to the Unix epoch;
- cleared gzip timestamp, name, comment, and normalized operating-system marker;
- archive prefix and format.

A file-list change is a release-source change and therefore changes the archive
digests.

## Provenance and manifests

`provenance.json` matches `schemas/isras-provenance-v1.schema.json` and binds the
validator, framework, and contract artifact bytes to the release identity,
toolchain, validation campaign, explicit publication timestamp, release
authority, and known limitations.

The checksum manifests contain the exact non-manifest artifact set, sorted by
artifact name, with two spaces between each complete digest and filename. Both
manifests include the provenance file. They do not include themselves.

The manifests are then hashed and recorded in local artifact-build evidence so a
project pin can declare all six final assets.

## Atomic output and evidence

Production occurs in a newly created temporary directory under the final output
parent. The final output path must not already exist. The completed exact file set
is atomically renamed into place only after all production and boundary checks
pass.

Complete JSON and text evidence is written under:

```text
.local/validation/releases/isras-vMAJOR.MINOR.PATCH/
```

The evidence records every final artifact size and complete SHA-256 and SHA-512
digest. Evidence files use mode `0600`; containing directories use mode `0700`.

If evidence cannot be written, the produced output directory is removed and the
operation fails.

## Publication handoff

The producer's exact artifact directory and private `artifact-build.json` report
are the only accepted local inputs to the publication boundary. Publication
recomputes all sizes and digests, revalidates both manifests and provenance, and
re-executes the validator's read-only identity command before any GitHub Release
is created.

Artifact production does not imply publication authority. The separate
[`RELEASE-PUBLICATION.md`](RELEASE-PUBLICATION.md) contract requires the remote
signed tag, release absence, explicit confirmation, draft-first upload, remote
byte verification, and final publication evidence.

## Rebuild and reproducibility claim

The producer removes the Go build identifier, disables VCS auto-embedding, uses
`-trimpath`, disables CGO, and fixes the validator target to `linux/amd64`.

A byte-for-byte reproducibility claim is valid only when the exact source commit,
signed tag, file lists, selected Go toolchain identity, builder implementation,
and explicit provenance inputs are identical. The producer records those inputs;
it does not
claim reproducibility across a different toolchain or platform without evidence.

## Acceptance boundary

This implementation is accepted only when tests prove:

- development versions cannot produce release artifacts;
- unsigned commits, lightweight tags, invalid tags, and tag drift fail closed;
- dirty repositories and noncanonical origins fail closed;
- toolchains below the declared minimum fail closed while later compatible
  releases and valid custom suffixes are accepted;
- selected-toolchain identity drift during production fails closed;
- archive file lists reject unsafe, missing, duplicate, and unsorted paths;
- repeated archive production from identical inputs is byte-identical;
- validator identity is embedded and checked;
- manifests contain the exact required artifact set;
- provenance binds the exact core artifact bytes;
- partial output is removed after failure;
- an existing output directory is never overwritten;
- complete evidence is retained before success is reported;
- no artifact is uploaded, extracted, or used against a consuming project.
