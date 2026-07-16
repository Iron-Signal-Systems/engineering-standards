# Changelog

## Unreleased

- Makes portable diagnostic regressions deterministic across local, Linux, macOS, and Windows execution by isolating GitHub context, comparing native paths structurally, and closing streamed subprocess handles.
- Makes portable CI acquire and verify accepted checkpoint and change-classification base commits before regression tests, and adds structured stage-level diagnostics so failures identify the exact validator, tested commit, required object, command, observed result, and exit code.
- Establishes the development-only ISRAS v3 assurance-hardening candidate without modifying accepted ISRAS v1 or v2 normative trees.
- Requires clean-room release bootstrap from an absent environment, exact Python executable identity, isolated pip operation, retained upstream wheel provenance, exact wheelhouse contents, and exact final installed-distribution accounting.
- Makes SHA-512 tracked-source accounting operate on the Git index or an exact commit rather than an ambiguous working-tree view.
- Adds artifact-level evidence validation that independently verifies repository and commit identity, tracked paths, committed validator bytes, environment artifacts, artifact-internal source/campaign/environment markers, declared controls and tests, extracted outcomes, and incompatible reuse.
- Adds a machine-readable control-level external standards crosswalk, prohibits premature full-coverage claims, and makes unpinned baselines formal phase-entry blockers.
- Makes this repository's own governing ISRAS v2.0.1 release identity explicit and internally consistent with `RELEASE_ASSURED`.
- Adds GitHub ruleset evidence collection and offline validation of targets, includes, excludes, exact required checks, tag mutation restrictions, and bypass actors and modes.
- Adds C0 through C6 change classes with C3 security and C4 schema as parallel impact branches, changed-path escalation, and an actual C5 classification for this candidate.
- Records completion of the SSH-signed ISRAS v2.0.1 release and exact convergence of remote `dev`, remote `main`, and the peeled tag target at `d34fad82781a4e8485f8907fbfd34f236fa79ad2`.
- Records annotated tag object `f4eacec519c96be225ffd37276cc646d3712ab0f`, source-manifest SHA-256 `8f54ed1e9bfee251bf89b4c5f12edf11ac1e25ef0d145ba745301f2d05787ef1`, and signing-key fingerprint `SHA256:FiH+Jk7HHrNkvDEQTehI/aCfkmKpivtsqmkl5TmmMSE`.
- Registers `isras-v2.0.1` as an immutable historical checkpoint using the frozen v2.0.1 release-source gate and leaves the accepted release source unchanged.

## 2.0.1 — BSD-licensed patch release — 2026-07-16

- Prepares the formally authorized v2.0.1 release-source boundary with
  `VERSION` `2.0.1`, a frozen release validator, and an exact-source phase gate.
- Retains accepted candidate source `6543a5a93f078f47d87aa3b8ed8ebd2024cec373`, evidence commit
  `9dbe4d9696ff4a9838fd83cb0f6f652087710f98`, and formal acceptance commit `57d23742e60d29bf6f46d15b8f64f0497bb260cd`.
- Requires exact pushed-source validation, an SSH-signed annotated
  `isras-v2.0.1` tag, non-force `main` promotion, and exact branch/tag
  convergence before release completion.
- Formally accepts ISRAS v2.0.1 candidate source
  `6543a5a93f078f47d87aa3b8ed8ebd2024cec373` for release finalization.
- Binds the decision to evidence commit `9dbe4d9696ff4a9838fd83cb0f6f652087710f98`, evidence JSON
  SHA-256 `42d7dce7500929647af001f47bbbdf30ae7bef88c598d0aba8edd2424564d2b9`, and candidate source-manifest SHA-256
  `e2b6488a7f670b0c81d873478154d03438a9c5f21a8bf05010863fbe1e4fd7e8`.
- Authorizes a later, separate release-source change while leaving `VERSION`
  `2.0.0`, `main`, `isras-v2.0.0`, and the v2.0.0 checkpoint unchanged.
- Retains the successful exact pushed-candidate campaign for commit
  `6543a5a93f078f47d87aa3b8ed8ebd2024cec373`.
- Adds schema-conforming v2.0.1 candidate evidence, environment fingerprint,
  complete gate output, focused validation logs, and SHA-256 artifact accounting.
- Records candidate evidence without claiming formal acceptance, changing
  `VERSION`, moving `main`, creating `isras-v2.0.1`, or registering a checkpoint.
- Prepares an ISRAS v2.0.1 patch candidate to carry BSD-3-Clause into the
  signed release line without changing normative ISRAS controls.
- Records `5c07b428b206e4f4e5d7e33d6f5811d7d4e6e739` as the first exact
  BSD-licensed source boundary.
- Adds the v2.0.1 candidate plan, validator, exact-pushed-source gate, and
  regression coverage while retaining `VERSION` `2.0.0` until formal candidate
  acceptance and a separate release-source change.
- Adopts the BSD 3-Clause License (`BSD-3-Clause`) for repository-authored
  materials in source revisions containing the root `LICENSE` file.
- Records the prospective licensing boundary without modifying the immutable
  signed `isras-v2.0.0` release source.
- Adds synchronized licensing documentation, contribution terms, validation,
  tests, and source-manifest coverage.
- Records completion of the SSH-signed ISRAS v2.0.0 release and exact convergence
  of remote `dev`, remote `main`, and the peeled tag target at
  `781246e69f8a9a382c25040f94b62dfe3b25ba89`.
- Records annotated tag object `a7a09a02798e2b2c905f2686820fd30890f62bc6` and signing-key fingerprint
  `SHA256:FiH+Jk7HHrNkvDEQTehI/aCfkmKpivtsqmkl5TmmMSE`.
- Registers `isras-v2.0.0` as an immutable historical checkpoint using the frozen
  v2 release-source gate.
- Adds exact checkpoint regression validation and synchronized acceptance and
  validation indexes.
- Preserves `main` and `isras-v2.0.0` at the immutable release source while
  allowing `dev` to advance with the post-release governance record.

## 2.0.0 — Governance and bounded authority — 2026-07-16

- Makes exact ISRAS release inheritance mandatory for deliberate adopters and
  prohibits silent weakening of inherited controls.
- Introduces Engineering Standards Impact Assessments for changes affected by
  newer accepted engineering standards.
- Requires phase-entry and phase-exit standards-compliance reviews.
- Defines `DOCUMENTED`, `IMPLEMENTED`, `VALIDATED`, and `ACCEPTED` control
  maturity and prohibits evidence overclaim.
- Establishes the bounded-authority and privilege-non-propagation invariant.
- Normatively prohibits unrestricted execution contexts and requires explicit
  separation of administrative, service, runtime, and ordinary-user authority.
- Requires hostile validation of authority, trust, lifecycle, and operational
  boundaries when applicable.
- Defines minimum standards evidence for accepted phases.
- Adds machine-readable phase-compliance, impact-assessment, and
  authority-boundary schemas and templates.
- Adds ISRAS v2 compliance, candidate, release-source, and regression
  validation.
- Preserves the accepted ISRAS v1 normative tree and immutable v1.0.1
  checkpoint.
- Documents deliberate migration from v1.0.1, including the Iron Atlas
  sequencing requirement.
- Requires the exact v2.0.0 release source, remote `dev`, remote `main`, and the
  SSH-signed annotated `isras-v2.0.0` tag to converge at finalization.

## 1.0.1 — Release hardening

- Records completion of the corrected signed v1.0.0 tag and exact `main`
  promotion.
- Corrects stale candidate, support, and security wording.
- Defines the signed annotated tag as the authoritative acceptance-decision
  object.
- Requires `dev`, `main`, and the signed tag to converge on the exact accepted
  commit at release finalization.
- Adds protected `isras-*` tag-namespace requirements.
- Adds exact-commit adopter quick-start instructions.
- Adds automated release-state drift validation.
- Clarifies the current all-rights-reserved licensing decision.
- Registers the formally accepted `isras-v1.0.1` release as an immutable
  historical checkpoint bound to
  `c379417720faa595fa5cb89a1dfdb2259d6cb95e`.
- Adds a focused regression assertion for the exact v1.0.1 checkpoint record.
- Documents isolated historical checkpoint revalidation.

## 1.0.0 — Accepted 2026-07-15

- Authorizes controlled replacement of the pre-acceptance unsigned
  `isras-v1.0.0` tag with an SSH-signed annotated tag targeting the exact
  formally accepted source commit.
- Makes the repository adopter safely encode generated JSON and Python string
  values so Windows paths and other backslash-containing values cannot create
  invalid assurance manifests or validation source files.
- Makes hosted workflow checkouts fetch the exact immutable GitHub event commit
  instead of fetching a mutable pull-request or branch ref and then comparing it
  with the event SHA.
- Defines ISRAS as the Iron Signal Repository Assurance Standard and explicitly
  distinguishes it from Information System Risk Assessment.
- Adds scope, glossary, language/project profiles, release versioning, support,
  compatibility, signed-tag, and deprecation requirements.
- Replaces recursive source hashing with tracked-file manifest generation and
  verification.
- Adds exact environment version and Python-tool dependency validation with
  machine-readable fingerprints.
- Hardens hosted and canonical workflows, prevents caller-selected sensitive
  runners, and creates clean exact-commit canonical checkouts.
- Makes Windows and Unix portable validation execute the same Python test and
  policy logic.
- Requires exact standard commits, runner identity, real timestamps, environment
  fingerprints, clean pushed source, and schema validation in acceptance
  evidence.
- Adds repository-wide schema, YAML, workflow, Markdown-link, source-manifest,
  evidence, and committed-whitespace validation.
- Makes the ISRAS tool tests reuse the exact active Python interpreter so
  temporary adopted repositories receive the same pinned validation
  dependencies as the parent validation campaign.
- Makes portable origin validation compare canonical repository identity across
  SSH and HTTPS transports while preserving exact-origin checks for canonical
  validation; hosted workflows now use isolated pinned Python environments on
  Linux, macOS, and Windows.
- Adds candidate acceptance structure and checkpoint-recording tooling.

## 1.0.0-rc0 — Initial candidate standard

- Establishes the initial ISRAS v1 candidate architecture and baseline tooling.
