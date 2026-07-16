# ISRAS v3.0.0 Candidate and Acceptance Plan

## Status

**Development candidate only.** ISRAS v2.0.1 remains the latest accepted release.

## Candidate boundary

- Development base: `08a0a514ec308f76dbf80ffdcb8caa70ce6e345f`
- Governing accepted release: `isras-v2.0.1`
- Governing accepted source: `d34fad82781a4e8485f8907fbfd34f236fa79ad2`
- Classification: [`isras-v3.0.0-change-classification.json`](isras-v3.0.0-change-classification.json), C5 with applicable C3 and C4 campaigns

## Scope

The candidate addresses deterministic validation-tool bootstrap, SHA-512
relationships, evidence binding, external standards translation, repository
self-assurance, effective GitHub control evidence, and proportional change
governance.

## Phase-entry blockers

Formal phase entry shall not occur until:

- every external crosswalk baseline has an immutable reviewed pin;
- the deterministic release wheelhouse model has passed clean-room positive and
  hostile negative validation on every supported operating-system profile;
- the evidence-binding validator has campaign evidence proving source,
  environment, validator, control, test, and outcome relationships;
- current GitHub rulesets and branch protections have been exported and their
  exact required check names and bypass authority reviewed;
- predecessor checkpoints have been revalidated from isolated exact trees using each accepted tree's own declared tool bootstrap and interpreter; and
- approval independence and reviewer authority are recorded.

## Candidate validation

The development candidate gate shall verify:

- accepted v1/v2 normative trees are unchanged;
- the base commit is an ancestor of the candidate;
- no unstaged source drift exists;
- repository self-assurance is internally consistent;
- schemas and templates are valid;
- source SHA-512 is generated from the Git index or exact commit;
- the actual C5 classification matches changed paths and impacts;
- the control-level external crosswalk covers every catalog control without
  premature `COVERED` claims;
- release bootstrap is clean-room, isolated, hash-locked, and exact-set checked;
- portable CI acquires and verifies every accepted checkpoint and active classification-base commit before regression execution;
- portable failures identify the exact stage, validator, tested commit, expected or required object, observed result, command, and exit code;
- portable diagnostic regressions isolate ambient GitHub context, use platform-native path semantics, and prove streamed subprocess handles close;
- historical checkpoint validation provisions the accepted tree's own tool environment before invoking its frozen gate and binds that interpreter through `ISRAS_PYTHON`; and
- unit, portable, fresh-clone, and applicable specialized campaigns pass.

## Formal acceptance sequence

1. Freeze an exact pushed candidate source commit.
2. Generate and accept environment-specific wheelhouses separately.
3. Collect exact environment, GitHub-control, and evidence-relationship records.
4. Complete all C5 campaigns and applicable C3/C4 campaigns.
5. Revalidate accepted predecessors in isolated clones.
6. Complete independent review and formal candidate acceptance.
7. Prepare a separate release-source commit.
8. Pass the frozen exact-source release gate.
9. Create and verify an SSH-signed annotated `isras-v3.0.0` tag.
10. Promote `main` without force and prove branch/tag convergence.
11. Record release completion and register the immutable checkpoint.

No candidate validation, plan, or development record is itself acceptance.
