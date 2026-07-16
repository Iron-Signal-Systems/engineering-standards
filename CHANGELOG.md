# Changelog

## Unreleased

- No changes are recorded after the ISRAS v2.0.0 release source boundary.

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
