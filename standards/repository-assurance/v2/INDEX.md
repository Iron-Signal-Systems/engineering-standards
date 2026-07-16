# ISRAS v2 Document Index

**ISRAS** means **Iron Signal Repository Assurance Standard**. ISRAS v2 is the
mandatory governing engineering contract for Iron Signal Systems repositories
that formally adopt an accepted v2 release.

ISRAS v2 is a candidate until its complete candidate tree is validated,
accepted, signed, tagged, and released. The repository root `VERSION` remains at
the currently accepted release during candidate development.

## Normative core

- [Standard](STANDARD.md)
- [Control Catalog](CONTROL-CATALOG.md)
- [Mandatory Governance and Inheritance](MANDATORY-GOVERNANCE-AND-INHERITANCE.md)
- [Bounded Authority and Privilege Non-Propagation](BOUNDED-AUTHORITY-AND-PRIVILEGE-NON-PROPAGATION.md)

## Phase governance and adoption

- [Engineering Standards Impact Assessment](ENGINEERING-STANDARDS-IMPACT-ASSESSMENT.md)
- [Phase Entry and Exit Compliance](PHASE-ENTRY-AND-EXIT-COMPLIANCE.md)
- [Migration Guide](MIGRATION-GUIDE.md)

## Validation and evidence

- [Hostile Authority Validation](HOSTILE-AUTHORITY-VALIDATION.md)
- [Evidence Model](EVIDENCE-MODEL.md)
- [Validation Model](VALIDATION-MODEL.md)

## Release governance

- [Release, Versioning, Support, and Deprecation](RELEASE-VERSIONING-SUPPORT-AND-DEPRECATION.md)

## Machine-readable contracts

- `schemas/engineering-standards-impact-assessment-v1.schema.json`
- `schemas/phase-standards-compliance-v1.schema.json`
- `schemas/authority-boundary-record-v1.schema.json`
- `templates/engineering-standards/`
- `tools/isras/validate_engineering_standards_compliance.py`
- `tools/isras/validate_isras_v2_candidate.py`
- `tests/test_engineering_standards_compliance.py`
