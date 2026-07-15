# ISRAS v1.0.0 Tag Correction Authorization

## Status

**Authorized for controlled replacement; remote replacement pending.**

- Authorization recorded: `2026-07-15T22:01:13.778942+00:00`
- Acceptance tag: `isras-v1.0.0`
- Formally accepted source commit: `f9655ddbbf04430fc468aab405f2ed880df3e97d`
- Acceptance-record commit: `7b3d7921e430fc203324655f6e92a88344a56746`

## Condition requiring correction

The existing remote `isras-v1.0.0` tag was created before formal acceptance and does
not identify the accepted ISRAS v1.0.0 source boundary.

- Existing remote tag object: `9381cc6824a7e38936966ae09a27b89084f0805a`
- Existing tag target: `a84ca6a0d2a5d3f9e1ee9b89f68495a15d9ba33b`
- Existing signature status: `UNSIGNED`
- Required target: `f9655ddbbf04430fc468aab405f2ed880df3e97d`

The existing tag therefore cannot serve as the immutable ISRAS v1.0.0
acceptance identity.

## Authorized replacement

A new annotated tag has been prepared locally with:

- New tag object: `3f7d4e7f5b340c65cfe74f757ba0a24b2f94cc2b`
- Target commit: `f9655ddbbf04430fc468aab405f2ed880df3e97d`
- Signing format: `SSH`
- Signing algorithm: `ED25519`
- Signing identity: `kb2vhn@gmail.com`
- Signing-key fingerprint: `SHA256:lCGPFFvNgV2sb/tLWTptypn9lihvc8Q6er5C/KgKkSU`
- Local signature verification: `PASS`

## Controlled replacement procedure

The remote replacement is authorized only when all of the following remain
true:

1. the remote tag object still equals `9381cc6824a7e38936966ae09a27b89084f0805a`;
2. the remote tag still peels to `a84ca6a0d2a5d3f9e1ee9b89f68495a15d9ba33b`;
3. the locally prepared replacement object equals `3f7d4e7f5b340c65cfe74f757ba0a24b2f94cc2b`;
4. the replacement peels to `f9655ddbbf04430fc468aab405f2ed880df3e97d`;
5. local signature verification reports a good signature;
6. only `refs/tags/isras-v1.0.0` is deleted;
7. the signed replacement is pushed immediately after deletion;
8. the remote object, peeled target, and signature are verified afterward.

No other tag or branch may be altered as part of this correction.

## Main promotion boundary

`main` remains blocked from promotion until the corrected remote tag has been
verified.

When promoted, `main` must move by fast-forward to exactly:

`f9655ddbbf04430fc468aab405f2ed880df3e97d`

It must not move to the later acceptance-record commit
`7b3d7921e430fc203324655f6e92a88344a56746`.

## Non-claims

This authorization does not itself prove that the remote replacement or
`main` promotion has occurred. Completion must be verified and recorded
separately.
