# ISRAS v2 Evidence Model

## 1. Purpose

Evidence binds engineering claims to exact source, standards, environments,
identities, validators, tests, decisions, and acceptance boundaries.
Documentation describing an intended control is not evidence that the control
is implemented, validated, or accepted.

## 2. Minimum accepted-phase evidence

Every accepted phase shall retain:

- repository and phase identifier;
- exact repository commit and branch;
- adopted ISRAS version, signed tag, commit, and source-manifest digest;
- phase-entry and phase-exit review records;
- ESIA when required;
- applicable controls;
- non-applicable controls and justifications;
- impact classifications;
- required and actual maturity classifications;
- implementation and validation obligations;
- authority boundary records;
- hostile tests performed and results;
- deviations, exceptions, and deferments;
- resource-observation status;
- historical predecessor status;
- reviewer and approval context;
- acceptance decision and limitations.

## 3. Evidence integrity

Evidence shall identify the exact pushed candidate. Evidence paths shall be
repository-relative or point to a governed external evidence system with stable
identifiers, integrity metadata, access controls, retention, and redaction
rules.

Secrets, authentication material, protected operational data, and unnecessary
personal information shall not be embedded in committed evidence.

## 4. Maturity evidence

- `DOCUMENTED` requires a controlled requirement and design reference.
- `IMPLEMENTED` requires an implementation or procedural-control reference.
- `VALIDATED` requires retained validation evidence for the exact applicable
  boundary.
- `ACCEPTED` requires a formal acceptance record identifying the exact validated
  boundary.

Evidence for a higher maturity state shall include or reference evidence for all
lower states.

## 5. Evidence schemas

ISRAS v2 uses:

- `engineering-standards-impact-assessment-v1.schema.json` for release impact;
- `phase-standards-compliance-v1.schema.json` for phase reviews;
- `authority-boundary-record-v1.schema.json` for bounded-authority design and
  validation.

Schema validity is necessary but not sufficient. Semantic compliance is enforced
by `validate_engineering_standards_compliance.py`.
