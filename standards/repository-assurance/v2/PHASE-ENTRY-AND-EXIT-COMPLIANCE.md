# Phase-Entry and Phase-Exit Compliance

## 1. Phase-entry review

Before work begins on a phase, the repository shall complete and record an
Engineering Standards Compliance Review of type `ENTRY`.

The review shall determine:

- exact adopted ISRAS version, signed tag, commit, and source-manifest digest;
- applicable and non-applicable controls;
- whether a newer accepted release exists;
- whether an ESIA is required;
- documentation, requirements, architecture, implementation, validation,
  acceptance, historical-validation, hostile-testing, resource-observation,
  roadmap, and sequencing impacts;
- remaining exceptions, deferments, and predecessor obligations;
- required maturity at phase exit and the plan for closing maturity gaps.

A phase shall not begin until the entry review is complete and recorded. Entry
approval confirms that the standards boundary and required work are understood;
it does not falsely claim that phase-exit maturity has already been achieved.

## 2. Phase-exit review

Before a phase may be accepted, the repository shall complete a review of type
`EXIT` against the exact pushed candidate commit.

The review shall confirm:

- applicable controls meet required maturity;
- non-applicability decisions remain valid;
- documentation, requirements, architecture, implementation, validation, test
  campaigns, acceptance criteria, and records are synchronized;
- hostile-condition testing was completed where required;
- correctness and resource observations remain distinct;
- accepted predecessor handling remains correct;
- deviations and deferments are explicit and governed;
- future mandatory work is identified;
- reviewer and approval context are recorded.

A failed or incomplete exit review shall fail phase acceptance.

## 3. Control maturity at phase gates

Each applicable control shall identify required and actual maturity. The order
is:

`DOCUMENTED < IMPLEMENTED < VALIDATED < ACCEPTED`

An entry review may approve a phase with planned maturity work when the plan,
owner, target, and validation impact are explicit. An exit review shall not pass
when actual maturity is below required maturity.

## 4. Exact candidate requirement

A passing exit review shall identify a non-placeholder 40-character commit that
exists in the canonical remote and shall affirm that the exact pushed candidate
was evaluated. Evidence from an earlier or locally modified tree shall not be
substituted.

## 5. Machine-readable record

Both reviews shall conform to
`schemas/phase-standards-compliance-v1.schema.json`. The review type determines
additional semantic requirements enforced by the compliance validator.
