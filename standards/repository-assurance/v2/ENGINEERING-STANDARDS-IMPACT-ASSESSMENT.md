# Engineering Standards Impact Assessment

## 1. Trigger

An Engineering Standards Impact Assessment (ESIA) is required when:

- a newer accepted ISRAS release exists;
- an adopted ISRAS release is revised or corrected;
- an applicability decision changes;
- repository architecture creates a newly applicable control;
- an accepted exception or deferment expires;
- a phase-entry or phase-exit review identifies a standards gap.

Discovery of a development candidate alone does not require adoption. Only an
accepted release can become a governing baseline, but candidate review may be
performed for planning.

## 2. Assessment boundary

The ESIA shall compare the currently pinned release with the exact candidate
release. It shall identify both versions, signed tags, 40-character commits,
source-manifest digests, and the repository phase context.

## 3. Required classifications

Every evaluated control shall receive exactly one classification:

- `ALREADY_SATISFIED`
- `REQUIRES_DOCUMENTATION`
- `REQUIRES_IMPLEMENTATION`
- `REQUIRES_VALIDATION`
- `REQUIRES_ACCEPTANCE_UPDATE`
- `REQUIRES_FUTURE_WORK`
- `NOT_APPLICABLE`

A classification shall include the control identifier, justification, affected
artifacts, owner, target phase or milestone, validation impact, acceptance
impact, and any applicable exception or deferment record.

`NOT_APPLICABLE` requires an architecture-based justification. `REQUIRES_FUTURE_WORK`
requires a governed owner, target, and deferment or sequencing basis; it is not a
synonym for optional work.

## 4. Decision

The ESIA shall record one decision:

- `ADOPT_NOW`
- `ADOPT_NEXT_PHASE`
- `DEFER_WITH_RECORD`
- `NO_ADOPTION_REQUIRED`

The decision shall not contradict control classifications. Adoption shall not be
approved while mandatory adoption-blocking work remains ungoverned.

## 5. Synchronization

When adoption changes repository obligations, documentation, requirements,
architecture, implementation, validation, test campaigns, phase gates,
acceptance criteria, roadmaps, and evidence templates shall be updated as part
of the same governed change set.

## 6. Machine-readable record

The authoritative machine-readable ESIA shall conform to
`schemas/engineering-standards-impact-assessment-v1.schema.json`. Human-readable
summaries may accompany it but shall not replace required fields.
