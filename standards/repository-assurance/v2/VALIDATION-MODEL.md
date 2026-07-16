# ISRAS v2 Validation Model

## 1. Validation layers

ISRAS v2 retains the v1 validation layers and adds standards-compliance and
authority-boundary validation:

1. developer validation;
2. portable clean-runner validation;
3. fresh-clone and remote-completeness validation;
4. canonical environment validation;
5. specialized environment campaigns;
6. historical checkpoint revalidation;
7. release assurance;
8. deployment, rollback, restore, and operational validation;
9. phase-entry standards-compliance validation;
10. phase-exit standards-compliance validation;
11. hostile authority-boundary validation.

## 2. Structural and semantic validation

JSON Schema validation confirms record shape and field constraints. Semantic
validation additionally confirms:

- exact pinned baseline fields are present;
- control identifiers are unique;
- non-applicability has sufficient justification;
- actual maturity is not below required maturity at phase exit;
- no control maturity is overclaimed;
- required ESIA and authority records are linked;
- authority records prohibit unrestricted execution contexts;
- deny-by-default and independent authorization are affirmed;
- privilege non-propagation is explicit;
- database, administrative, worker, and break-glass authority is bounded;
- required hostile tests have passing evidence;
- the exact pushed candidate was evaluated.

## 3. Candidate validation

The ISRAS v2 candidate validator shall verify:

- the complete normative v2 tree exists;
- schemas are valid Draft 2020-12 schemas;
- templates validate structurally;
- required v2 controls exist without duplicates;
- normative unrestricted-execution-context language exists;
- the accepted v1 normative tree is unchanged from the accepted v1.0.1 source;
- the accepted v1.0.1 checkpoint remains registered;
- unit tests pass independently;
- root `VERSION` remains `1.0.1` until release finalization.

## 4. Outcome separation

Validators shall report correctness independently from resource observation,
performance-budget status, security findings, and operational readiness.
Warnings and non-claims shall not be counted as correctness PASS results.

## 5. Failure semantics

A schema error, missing required evidence, maturity shortfall, overclaim,
unauthorized propagation, unrestricted execution context, failed required
hostile test, stale baseline, or non-exact candidate shall produce a failing
validation exit status.
