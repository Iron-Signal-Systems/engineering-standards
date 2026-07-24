# ISRAS 0.1.8 Signer Rotation

**Status:** RELEASE-PREPARATION CANDIDATE

**Preparation date:** 2026-07-22

## Purpose

Add the Arch development host signing key for `kb2vhn@gmail.com` to the governed
hosted signer inventory. Atlas hosted validation rejected the correctly signed
candidate because ISRAS 0.1.7 did not contain this key.

## Trust boundary

- Principal: `kb2vhn@gmail.com`
- Fingerprint: `SHA256:CiONRPnsf/rG0Ix5LJmYeoCVdI4d1kRfYtQfQTp/vDQ`
- Registry source: `https://api.github.com/users/kb2vhn/ssh_signing_keys`

The GitHub registry was discovery input only. The committed allowed-signers
bytes, checksum, and manifest are the release authority. Existing signer entries
remain unchanged.

## Release boundary

ISRAS 0.1.7 remains immutable. The signer addition requires the signed
`isras-v0.1.8` source, complete validation, deterministic six-asset production,
publication, remote-byte verification, and explicit consuming-project upgrade.

This record proves the intended source change. It does not establish release
publication, independent review, consuming-project adoption, or formal
acceptance.
