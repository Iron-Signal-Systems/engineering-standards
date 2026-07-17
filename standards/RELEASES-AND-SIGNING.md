# Releases and Signing

## Signing

Commits and annotated release tags shall be signed using the developer's
configured Git signing identity. Signatures establish attribution and integrity.
They do not establish independent review.

Validation shall distinguish between:

- an unsigned commit;
- a signed commit whose signer trust material is unavailable locally;
- an SSH signature that cannot be matched through the configured allowed-signers
  file;
- a present but invalid signature; and
- a cryptographically verified signature.

No failed signature check may automatically recommend amending a commit. Rewriting
history requires a separate, explicit decision after determining whether the
commit has already been pushed or incorporated into another branch.

## GitHub-created commits

GitHub web-interface merge and editing operations may create OpenPGP-signed
commits using GitHub's web-flow signing identity. Local verification requires the
published GitHub web-flow public key to be obtained through the official GitHub
source, inspected, and deliberately imported into the local GPG keyring.

A missing public key means local verification is unavailable. It does not permit
the validator to claim that the signature passed, and it does not by itself prove
that the signature is invalid.

## Development behavior

During development, an unsigned current `HEAD` may be reported as a warning while
new changes remain uncommitted. The next commit shall be created with the
configured signing identity. The validator shall recommend a new signed commit,
not amendment of the existing commit.

In commit and release modes, an unsigned exact commit is a failure. The validator
shall first direct the developer to determine whether the commit is already
published and to review this policy before any history correction.

## Development acceptance

A self-validated candidate shall record:

- exact repository;
- exact pushed commit;
- validation mode and result;
- relevant environment identity;
- retained committed test source;
- warnings and known limitations;
- status `SELF-VALIDATED`.

## Release baseline

A release shall additionally require:

- clean-clone validation of the exact pushed commit;
- complete applicable project validation;
- release notes;
- declared supported platforms;
- rollback or recovery guidance where applicable;
- a signed annotated tag;
- confirmation that the tag resolves to the exact tested source.

Independent review shall be recorded only when performed by a qualified person
other than the author.

## Tag naming

ISRAS releases use the tag form `isras-vMAJOR.MINOR.PATCH`. A release tag shall
be signed, annotated, and point directly to the exact commit that passed the
required release validation. The first practical Solo Developer Baseline release
uses `isras-v0.1.0`.

## Clean-clone validation

Before a release tag is created, the exact pushed candidate commit shall pass
the repository-owned clean-clone campaign defined in
[`CLEAN-CLONE-RELEASE-VALIDATION.md`](CLEAN-CLONE-RELEASE-VALIDATION.md).

The campaign retains local review evidence but does not itself create a tag or
make a release claim.
