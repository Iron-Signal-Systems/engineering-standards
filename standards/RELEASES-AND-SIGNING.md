# Releases and Signing

## Signing

Commits and annotated release tags shall be signed using the developer's
configured Git signing identity. Signatures establish attribution and integrity.
They do not establish independent review.

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
