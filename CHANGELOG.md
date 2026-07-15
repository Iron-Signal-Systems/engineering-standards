# Changelog

## Unreleased — v1 pre-acceptance hardening

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
- Adds candidate acceptance structure and checkpoint-recording tooling.

## 1.0.0 — Initial candidate standard

- Establishes the initial ISRAS v1 candidate architecture and baseline tooling.
