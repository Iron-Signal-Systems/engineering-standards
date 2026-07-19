# Changelog

## Unreleased

- Added release-bound hosted SSH signer trust sourced from the exact pinned
  Engineering Standards commit. The reusable workflow now verifies tracked trust
  bytes and their digest, configures a private target-local allowed-signers file,
  binds the verified principal and fingerprint to the commit committer identity,
  and rejects missing, altered, wrong-key, and wrong-principal trust.

- Retained both `.local/isras` and `.local/validation` from reusable hosted runs so
  repository-signature failure logs survive skipped later steps.

- Corrected GitHub Release asset upload transport to use `uploads.github.com` and
  added regression coverage requiring all six upload calls to use that host.

- Recorded that published `0.1.2` remains immutable but cannot establish formal
  consuming-project adoption when its required hosted validation fails.

## 0.1.2 — 2026-07-18

- Added fail-closed first project initialization from one explicitly selected,
  fully verified ISRAS GitHub Release. Initialization now requires the exact
  linker-bound validator artifact for that release before network or target
  authority is granted, uses one shared canonical origin parser, fixes runtime
  evidence to untracked `.local/isras`, generates stable timestamp-independent
  adoption evidence, and refuses partial, conflicting, dirty, symlinked, tracked,
  or mode-drifted targets.

- Added immutable reusable hosted validation. The called workflow checks out its
  own exact workflow repository and SHA, bootstrap-verifies the committed pin and
  release, downloads and digest-binds the published validator artifact, runs core
  repository and secret-protection checks, executes every committed project
  command, and retains validation evidence using read-only permissions and
  commit-pinned third-party actions.

- Added atomic no-overwrite publication, idempotent exact re-execution, rollback
  after mid-publication failure, a non-mutating project-owned Go format checker,
  hostile path and origin tests, and synchronized initialization/adoption
  documentation.
## 0.1.1 — 2026-07-18

- Corrected local release-tag discovery so the expected pre-tag state is
  accepted across supported Git versions without permitting command failures or
  ambiguous output. The workflow now uses read-only ref enumeration to
  distinguish an absent tag from an existing tag and from a failed Git
  inspection.

- Added the ISRAS Engineering Standards emblem as a repository documentation
  asset and rendered it in the README. The emblem is branding source only and
  is not part of the exact six-file downloadable release artifact set.

- Added controlled draft-first publication of the exact six-file deterministic
  ISRAS release artifact set. The separately named publication command requires
  a clean signed source, exact remote branch, GitHub-verified annotated tag,
  private artifact-build evidence, manifests, provenance, and embedded validator
  identity; rejects every preexisting release; uploads without clobbering;
  re-downloads and verifies remote bytes before and after publication; safely
  removes only its exact incomplete draft; retains private JSON and text
  evidence; and never creates or pushes a tag, moves a branch, or modifies a
  consuming project. The legacy `isras-release publish` entry point is disabled.

- Added fail-closed execution of one exact command declared by a consuming
  project's committed pin. A linker-bound release validator now requires exact
  validator, pin, target-origin, and target-commit identity; invokes argv without
  implicit shell interpretation; uses a credential-minimized isolated
  environment; bounds time, output, and Linux process descendants; rejects
  repository-state drift; and retains private redacted JSON and text evidence.

- Added explicit external target-repository selection through global `--repo`
  handling. Validator release identity is now independent from target Git identity;
  linker-bound `version` and `help` run outside Git; target discovery rejects
  missing, non-directory, non-Git, and symbolic-link paths; all execution and
  evidence remain rooted in one canonical target without process-wide `chdir`;
  and regression tests prove cross-repository isolation.

- Added deterministic release-artifact production from an exact signed source
  commit and annotated release tag. The producer embeds immutable release
  identity into the external validator, builds normalized framework and contract
  archives from committed sorted file lists, generates v1 provenance and both
  checksum manifests, records complete private build evidence, and commits the
  exact six-file artifact set atomically. Production performs no publication,
  archive extraction, consuming-project validation, or remote write.

- Added fail-closed release-artifact acquisition and verification. The project
  pin can now select an exact published GitHub release, require a GitHub-verified
  signed annotated tag at the pinned commit, acquire only declared assets,
  compare complete SHA-256 and SHA-512 values, cross-check both manifests,
  validate v1 provenance, retain full local evidence, and grant or deny a
  separate execution-authorization result. Local-directory verification remains
  available but cannot authorize execution because release and tag identity are
  not checked.

- Added the strict v1 `.isras/project.json` schema, standard-library Go parser,
  read-only project-pin declaration validation and inspection commands, release
  and workflow identity checks, artifact digest requirements, Go profile command
  declarations, bounded evidence paths, and hostile JSON regression tests.
  Terminal output truthfully labels artifact metadata as declared and unverified,
  abbreviates digest fingerprints, and performs no artifact acquisition, hashing,
  comparison, execution, or project mutation.

- Defined the pinned project framework: language-neutral ISRAS core
  requirements, language and platform profiles, Go as the first reference
  profile, immutable project release pins, external validator execution,
  versioned project-framework artifacts, and explicit project upgrades.
  Deprecated the copied-validator source-export model for new adoption while its
  replacement is implemented and validated.

- Censored release-workflow command arguments, streamed and captured subprocess
  output, retry diagnostics, retained logs, and final errors. Added bounded
  capture and line budgets plus fail-closed multiline private-key suppression so
  detected sensitive values cannot be reproduced through release automation.

- Updated GitHub-maintained workflow actions to `actions/checkout@v5` and
  `actions/setup-go@v6`, removing Node 20 deprecation annotations while
  preserving the Go implementation and existing Go validation behavior.

- Replaced the stale hard-coded validator profile with committed version and
  source identity metadata. The validator now reports reference versus
  project-owned export ownership, exact export source commit, target module, and
  current repository commit through a dedicated `version` command and dashboard
  header.
- Made project-validator export transactional: ordinary clones and linked
  worktrees are validated in an exact-commit scratch clone, deterministic module
  changes are applied and staged, existing requirements cannot disappear or
  change version, and failed applied validation restores the target boundary.
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
  `isras-v0.1.0`; active development remained explicitly non-release source until
  this release-preparation boundary.
- Corrected `isras-release check` so an existing local or remote release tag
  bound to a different commit fails before the expensive validation campaign;
  tag identity is checked again after validation to detect intervening changes.
- Added the repository-owned `isras-release` Go command with separated `check`
  and `tag` authority stages, retained local logs, bounded read retries, exact tag
  verification, and a separately controlled publication handoff.

## 0.1.0 — 2026-07-17

- Raised the required Go toolchain to Go 1.25.12 after CI identified reachable `net/url` standard-library vulnerabilities under Go 1.23.12.

- Restarted the active ISRAS implementation as a practical solo-developer
  baseline.
- Preserved the complete long-term ISRAS vision and terminology.
- Added a standard-library-first Go validation dashboard.
- Added Go formatting, vet, tests, builds, module-tidy, module-integrity, and
  `govulncheck` checks.
- Added repository-owned secret detection with censored output, deterministic
  finding identifiers, redaction plans, and bounded allowlist proposals.
- Required a local `*.log` for every validation failure.
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
