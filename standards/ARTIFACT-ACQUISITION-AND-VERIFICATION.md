# Artifact Acquisition and Verification

## Purpose

A project pin is a declaration until the selected release and its artifact bytes
are independently acquired and compared. This contract defines the read-only
verification boundary that converts a structurally valid pin into a verified
artifact set.

Artifact verification does not execute the validator, extract an archive, modify
project source, update the pin, select another release, commit, push, merge, tag,
or publish.

## Commands

The reference validator exposes:

```bash
./.local/bin/isras-validate project-pin verify-artifacts
```

With no source option, this command reads the exact GitHub release selected by the
project pin. It requires an authenticated or otherwise authorized GitHub CLI when
the Engineering Standards repository is not publicly readable.

A local comparison mode is available:

```bash
./.local/bin/isras-validate project-pin verify-artifacts \
  --source-directory /path/to/release-assets
```

Local-directory mode verifies file inventory, pin digests, manifests, and
provenance. It cannot prove the current GitHub release record or signed tag and
therefore never grants execution authorization.

## Online release identity

GitHub release verification requires all of the following:

- the exact pinned release tag exists as a published release;
- the release is neither a draft nor a prerelease;
- the release asset names exactly equal the pin's artifact names;
- each release asset is fully uploaded and within the size boundary;
- GitHub's release-asset SHA-256 metadata equals the pin;
- the release ref resolves to an annotated tag object, not a lightweight tag;
- the annotated tag has a GitHub-verified valid signature;
- the signed tag points directly to the exact pinned source commit.

The published release record and signed tag are checked again after download and
local hashing. Any intervening release, tag, asset-digest, or asset-size change
denies authorization.

A failure does not fall back to another tag, branch, release, or latest version.

## Acquisition

The implementation invokes GitHub CLI with the exact release tag and one exact
asset-name pattern for every declared artifact. Artifact names are already
restricted by the project-pin schema and cannot contain glob metacharacters,
path separators, credentials, or shell syntax.

Downloads are placed in a newly created temporary directory. The directory is
removed after verification. No release artifact is extracted or executed by this
step.

## Local byte verification

The verifier requires the source directory to contain exactly the declared file
set. Extra files, missing files, directories, symbolic links, non-regular files,
empty files, oversized files, duplicate names, and changes during hashing fail
closed.

Every artifact is read once through SHA-256 and SHA-512 hashers. The complete
observed values are compared against the complete values in the project pin.
Terminal output reports only PASS or FAIL. Complete expected and observed digests
are retained in local evidence.

A digest mismatch denies execution authorization. The verifier does not retry a
different download or substitute another release.

## Manifest layering

The project pin independently identifies the final bytes of `SHA256SUMS` and
`SHA512SUMS`. This avoids impossible self-referential manifest hashes.

Each manifest must then contain exactly one sorted entry for every non-manifest
release artifact, including `provenance.json`. Entries use the conventional form:

```text
<lowercase-digest><two spaces><artifact-name>
```

The verifier requires every manifest digest to match both the project pin and the
locally observed artifact bytes. Unknown, missing, duplicate, unsorted, malformed,
or unsafe entries fail closed.

## Provenance layering

`provenance.json` is itself covered by both manifests and by the project pin. Its
v1 schema is committed at:

```text
schemas/isras-provenance-v1.schema.json
```

Provenance binds the release profile, semantic version, tag, source repository,
exact source commit, build identity, validation campaign, publication time,
release authority, evidence limitations, and the core produced artifacts.

To avoid self-reference, provenance lists validator, framework, contracts, and
optional migration artifacts. It does not list itself or the checksum manifests.
The manifests cover provenance; the pin covers the manifests.

## Execution authorization

Online verification grants execution authorization only when all of these are
PASS:

- published release record;
- signed annotated tag;
- asset acquisition;
- exact asset inventory;
- complete SHA-256 and SHA-512 pin comparisons;
- SHA-256 manifest membership;
- SHA-512 manifest membership;
- provenance identity and artifact binding;
- downloaded file sizes matching the release record.

This authorization states only that the selected validator artifact set is the
one bound to the pinned release. It does not execute the validator or prove that
the consuming project passes ISRAS.

Local-directory verification always reports execution authorization as DENIED
because online release identity and tag verification were not performed.

## Evidence

Every verification attempt writes JSON and text evidence under the project's
committed `evidence.directory` location. Evidence files use mode `0600` and
include:

- source mode and location;
- start and finish times;
- release and tag status;
- acquisition and inventory status;
- complete expected and observed digests;
- manifest and provenance status;
- final execution authorization;
- a bounded failure reason when applicable.

Terminal output abbreviates no security decision: it reports explicit status
labels and evidence paths while keeping the complete digest comparison in the
machine-readable record.

## Failure boundary

No failed or incomplete verification authorizes execution. A network error,
authentication failure, malformed GitHub response, unavailable asset, altered
byte, manifest inconsistency, provenance inconsistency, evidence-write failure,
or timeout is a verification failure.
