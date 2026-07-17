# Changelog

## Unreleased

- Corrected secret-scanner semantics for approved external-secret references,
  Go identifier and selector expressions, Go comments, and shell dynamic
  assignments while preserving detection of string literals, malformed source,
  embedded URL credentials, unknown schemes, and hardcoded command-body values.
- Corrected the secret-scanning boundary so staged Git-index content and
  working-tree content are evaluated independently. A clean working-tree copy can
  no longer conceal a staged finding, and dangerous credential filenames now
  fail before binary, encoding, or size-based content skips.
- Added required native CI validation for Ubuntu Server 22.04 LTS alongside the
  existing Ubuntu Server 24.04 LTS validation, plus official OCI userland CI for
  Arch Linux and supported Fedora Server 43 and 44 release lines. Added weekly
  scheduled validation, exact merged-`dev` validation, and explicit documentation
  distinguishing native evidence from container-userland evidence.
- Started the `0.1.1-development` cycle after publishing and freezing
  `isras-v0.1.0`; active `dev` work is now explicitly non-release source.
- Corrected `isras-release check` so an existing local or remote release tag
  bound to a different commit fails before the expensive validation campaign;
  tag identity is checked again after validation to detect intervening changes.
- Added the repository-owned `isras-release` Go command with separated `check`, `tag`, and `publish` authority stages, retained local logs, bounded read retries, exact tag verification, safe `main` fast-forwarding, and GitHub Release publication through authenticated `gh`.

## 0.1.0 — 2026-07-17

- Raised the required Go toolchain to Go 1.25.12 after CI identified reachable `net/url` standard-library vulnerabilities under Go 1.23.12.

- Restarted the active ISRAS implementation as a practical solo-developer
  baseline.
- Preserved the complete long-term ISRAS vision and terminology.
- Added a standard-library-first Go validation dashboard.
- Added Go formatting, vet, test, build, module-tidy, module-integrity, and
  `govulncheck` checks.
- Added repository-owned secret detection with censored output, deterministic
  finding identifiers, redaction plans, and bounded allowlist proposals.
- Required a local `*.log` for each failed validation check.
- Required exact, context-specific remediation commands in terminal output.
- Declared Arch Linux as the primary development platform, with supported
  Ubuntu Server LTS and Fedora Server releases as default server targets.
- Added an archive-and-restart installer that preserves the previous repository
  through a branch, signed tag, local Git bundle, and digest manifest before
  replacing the active tree.
- Added context-aware commit-signature diagnostics for unsigned commits, missing
  OpenPGP keys, GitHub web-flow commits, and SSH allowed-signers failures.
- Removed automatic commit-amendment guidance from signature remediation and
  added regression tests proving that unsafe recommendation cannot return.

- Added repository-owned clean-clone release validation that proves the exact
  pushed branch tip, clones the canonical origin, checks out the exact commit,
  rebuilds committed validation tooling, runs release-mode validation, and
  retains local review evidence.
