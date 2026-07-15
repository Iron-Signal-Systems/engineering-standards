# ISRAS v1.0.0 Tag Correction Record

## Status

**Controlled replacement completed and verified.**

- Authorization recorded: `2026-07-15T22:01:13.778942+00:00`
- Completion verified: `2026-07-15`
- Acceptance tag: `isras-v1.0.0`
- Formally accepted source commit:
  `f9655ddbbf04430fc468aab405f2ed880df3e97d`
- Acceptance-record commit:
  `7b3d7921e430fc203324655f6e92a88344a56746`

## Condition that required correction

The original remote `isras-v1.0.0` tag was created before formal acceptance
and did not identify the accepted source boundary.

- Original remote tag object:
  `9381cc6824a7e38936966ae09a27b89084f0805a`
- Original tag target:
  `a84ca6a0d2a5d3f9e1ee9b89f68495a15d9ba33b`
- Original signature status: `UNSIGNED`
- Required accepted target:
  `f9655ddbbf04430fc468aab405f2ed880df3e97d`

## Authorized and completed replacement

The replacement tag has:

- Corrected tag object:
  `3f7d4e7f5b340c65cfe74f757ba0a24b2f94cc2b`
- Target commit:
  `f9655ddbbf04430fc468aab405f2ed880df3e97d`
- Signing format: `SSH`
- Signing algorithm: `ED25519`
- Signing identity: `kb2vhn@gmail.com`
- Signing-key fingerprint:
  `SHA256:lCGPFFvNgV2sb/tLWTptypn9lihvc8Q6er5C/KgKkSU`
- Signature verification: `PASS`

The incorrect remote tag was deleted only after its exact object and peeled
target were reverified. The prepared signed replacement was then published
and independently reverified.

No other tag was changed by the correction.

## Main promotion completion

After corrected-tag verification, `main` was fast-forwarded without force to:

`f9655ddbbf04430fc468aab405f2ed880df3e97d`

Remote verification confirmed that `main` and the peeled `isras-v1.0.0` tag
target were identical.

## Result

The v1.0.0 acceptance identity is corrected, signed, and complete.

This record documents the controlled correction. It does not change the
original accepted source commit or expand the scope of the v1.0.0 acceptance.
