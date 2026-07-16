# ISRAS v2.0.0 Candidate and Acceptance Plan

## Status

`RELEASE SOURCE PREPARED FOR EXACT-COMMIT FINALIZATION`

The exact candidate source commit `4aff00dfdc88154390252898210abc336fa8b2fc` was formally accepted
for release finalization on 2026-07-16, with retained evidence committed at
`b0c982221acde7873307d010aca73ed2e386eb99` and formal authorization committed at
`24e911b7c4a63735bcef9b4b84ab9b62ace10298`. The release source now declares `VERSION` `2.0.0`.
Finalization is not complete until the exact pushed source passes the complete
campaign, the signed `isras-v2.0.0` tag verifies, and remote `dev`, remote
`main`, and the peeled tag target converge. Checkpoint registration and
repository adoption remain separate governed changes.

## Objective

Accept ISRAS v2.0.0 as the mandatory governing engineering contract for Iron
Signal Systems repositories that deliberately adopt the exact accepted release.

## Candidate scope

The candidate shall include:

- complete `standards/repository-assurance/v2/` normative tree;
- combined v1 and v2 control catalog;
- mandatory pinned inheritance and ESIA rules;
- phase-entry and phase-exit governance;
- bounded-authority and privilege-non-propagation invariant;
- control maturity model;
- hostile authority-boundary validation;
- minimum phase evidence model;
- three valid machine-readable schemas;
- four valid templates;
- compliance and candidate validators;
- regression tests;
- migration guidance, including Iron Atlas sequencing;
- release-finalization instructions that keep `VERSION` unchanged until the
  candidate is accepted.

## Candidate acceptance gates

1. **Tree completeness** — every required normative, schema, template, tool,
   test, migration, and acceptance artifact exists.
2. **v1 preservation** — the accepted v1 normative tree is unchanged from the
   accepted v1.0.1 source and its historical checkpoint remains exact.
3. **Schema validity** — all new schemas pass Draft 2020-12 schema checking.
4. **Template validity** — all templates conform structurally and remain DRAFT.
5. **Normative invariant** — the standard uses `unrestricted execution context`
   normatively and prohibits automatic privilege propagation.
6. **Control completeness** — all proposed v2 controls are unique and present.
7. **Validator behavior** — compliant records pass; unrestricted contexts,
   privilege propagation, maturity overclaim, missing hostile evidence, stale
   impact handling, and non-exact phase exit fail.
8. **Historical validation** — accepted v1.0.1 revalidates from its isolated
   historical tree.
9. **Portable and fresh-clone validation** — the exact pushed candidate passes.
10. **Source manifest** — the candidate source manifest is regenerated and
    verifies against the exact candidate tree.
11. **Acceptance evidence** — exact commit, environment, validator, results,
    warnings, limitations, and non-claims are retained.

## Candidate commands

```bash
python tools/isras/validate_isras_v2_candidate.py --repo-root .
python -m unittest -v tests.test_engineering_standards_compliance
python -m unittest -v tests.test_isras_tools
./tools/validation/validate_checkpoint.sh isras-v1.0.1
python tools/isras/generate_source_manifest.py --repo-root .
python tools/isras/verify_source_manifest.py --repo-root .
git diff --check
```

Run portable, fresh-clone, canonical, and applicable specialized validation
using the existing v1 acceptance model plus the v2 candidate validator.

## Required hostile validator tests

The test campaign shall prove rejection of:

- `unrestricted_execution_context_prohibited: false`;
- automatic privilege propagation;
- database owner or superuser runtime authority;
- service/user or admin/ordinary identity collapse;
- role accumulation that creates unrestricted authority;
- missing applicable hostile-test evidence at VALIDATED or ACCEPTED maturity;
- phase-exit maturity below the required state;
- phase-exit review against a placeholder or non-exact commit;
- required ESIA without a supplied assessment;
- phase acceptance with out-of-sync artifacts or open deviations.

## Formal acceptance boundary

Formal acceptance shall identify the exact pushed candidate commit and retained
results. Acceptance of the candidate authorizes a separate release-finalization
change; it does not itself mutate `VERSION`.

## Release finalization after candidate acceptance

### In-tree release-source change

1. Update `VERSION` from `1.0.1` to `2.0.0`.
2. Synchronize release, support, compatibility, acceptance, README, changelog,
   validator, test, and migration documentation.
3. Add the frozen ISRAS v2 release-source gate.
4. Regenerate and verify `SOURCE-SHA256SUMS.txt`.

### Exact pushed-source completion

5. Push the exact release-source commit to `dev`.
6. Rerun the complete portable, fresh-clone, regression, historical-checkpoint,
   release-source, and manifest campaign from that exact commit.
7. Create and verify the SSH-signed annotated `isras-v2.0.0` tag.
8. Fast-forward `main` without force to the exact same commit.
9. Verify remote `dev`, remote `main`, and the peeled tag target are identical.
10. Publish the exact commit, source-manifest digest, evidence digest, validation
    result, and signing identity.
11. Register the accepted immutable v2.0.0 checkpoint in a subsequent governed
    change.

## Adoption sequence

No repository silently inherits v2.0.0. Each repository performs an ESIA and
adopts the exact signed release through a reviewed change. Iron Atlas completes
its current pinned v1.0.1 `RECORDED` adoption before performing its v2 ESIA.
