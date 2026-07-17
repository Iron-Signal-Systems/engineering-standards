# Changelog

## 0.1.0-development — 2026-07-16

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
