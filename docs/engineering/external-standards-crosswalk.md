# ISRAS External Standards Crosswalk

## Status

The controlling development-candidate mapping is the machine-readable
[`external-standards-crosswalk.json`](external-standards-crosswalk.json),
validated against
[`external-standards-crosswalk-v1.schema.json`](../../schemas/external-standards-crosswalk-v1.schema.json).
It maps every inherited and candidate ISRAS control individually.

ISRAS remains its own standard and may impose stronger repository,
historical-integrity, evidence, and acceptance requirements. The crosswalk is
an informative translation aid; it does not claim certification, equivalence,
authorization, regulatory compliance, or complete implementation.

## Mapping states

- `COVERED` — reserved for requirement-level human review establishing full
  correspondence. The current candidate validator rejects this state to prevent
  premature overclaim.
- `PARTIALLY_COVERED` — ISRAS contributes to part of the external outcome.
- `PROJECT_RESPONSIBILITY` — product, deployment, data, jurisdiction, or
  operations must supply the requirement.
- `NOT_APPLICABLE` — outside the mapped control's scope.

## Baseline governance

Each baseline records a version and source identifier. A baseline marked
`REVIEW_REQUIRED` is an explicit phase-entry blocker, not a soft warning for
formal acceptance. The current candidate still requires immutable baseline pins
for OpenSSF Scorecard and OWASP SAMM before formal ISRAS v3 phase entry.

The candidate record includes NIST SSDF, NIST CSF 2.0, NIST SP 800-53, SLSA,
OpenSSF Scorecard, OWASP SAMM, OWASP ASVS, CIS Software Supply Chain Security
guidance, and FBI CJIS Security Policy relationships. CJIS mappings identify
engineering contribution only; deployment-specific CJIS obligations remain the
responsibility of the applicable agency, system, agreements, and operating
environment.

## Validation

Development review:

```bash
python3 tools/isras/validate_external_standards_crosswalk.py \
  --repo-root . \
  --record docs/engineering/external-standards-crosswalk.json
```

Formal phase-entry or acceptance review adds `--require-all-pinned`. That mode
fails while any baseline remains unpinned or marked for review.
