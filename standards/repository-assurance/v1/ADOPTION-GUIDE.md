# ISRAS v1 Adoption Guide

## 1. Adopt incrementally

Do not enable every enforcement rule on the first commit.

### Level 1 — Recorded

Add the assurance manifest, governance documents, environment profiles, and
checkpoint registry.

### Level 2 — Reproducible

Add portable, version-aware environment-doctor, source-manifest, fresh-clone,
and historical checkpoint validation. Stage the intended project-owned files,
generate `SOURCE-SHA256SUMS.txt`, verify it, and change `adoption_level` to
`REPRODUCIBLE` only after those checks pass:

```bash
git add -A
python3 tools/isras/generate_source_manifest.py --repo-root .
git add SOURCE-SHA256SUMS.txt
python3 tools/isras/verify_source_manifest.py --repo-root .
```

The manifest contains tracked files only. Caches, build output, virtual
environments, and other untracked state must not appear.

### Level 3 — Observed

Run GitHub workflows without making them required. Correct host assumptions,
missing assets, nondeterminism, and workflow failures.

### Level 4 — Enforced

Require pull requests and stable checks. Protect development, release, and
accepted tag boundaries.

### Level 5 — Release assured

Generate SBOMs, hashes, provenance, acceptance evidence, installation records,
rollback evidence, and recovery evidence.

## 2. Adoption command

From a clone of `engineering-standards`:

```bash
python3 tools/isras/adopt.py \
  --target /path/to/target \
  --repository Iron-Signal-Systems/project-name \
  --canonical-origin git@github.com:Iron-Signal-Systems/project-name.git \
  --development-branch dev \
  --release-branch main \
  --profile general \
  --dry-run
```

Review the plan, then rerun without `--dry-run`.

The adopter refuses to overwrite existing files unless `--force` is explicitly
provided. Existing project documentation and validation must be merged
deliberately, not replaced blindly.

## 3. Project customization

After copying the baseline:

- install the pinned validation-tool requirements from `tools/requirements.txt`;
- set exact environment requirements and command-version patterns;
- replace bootstrap heuristic checks with project-specific checks;
- populate accepted checkpoints;
- classify specialized environments;
- update CODEOWNERS;
- define required documentation synchronization;
- identify acceptance evidence and retention;
- define known unsupported systems and workflows.

## 4. Observation before enforcement

Run the workflow suite from normal work branches. Do not protect `dev` until:

- checks have stable names;
- tests do not depend on local state;
- the workflow does not require secrets for public pull requests;
- workflow permissions are minimal;
- false failures have been corrected;
- an emergency bypass is documented.

## 5. Acceptance

Repository assurance itself should receive a bounded implementation record and
formal acceptance. Record:

- the standard commit adopted;
- repository-specific deviations;
- exact required checks;
- environment profiles;
- fresh-clone results from approved development systems;
- historical checkpoint results;
- explicit non-claims.

## Safe existing-repository application

Preview all conflicts with `--dry-run`, then use `--skip-existing` to write only
missing baseline files. Existing validation, documentation, workflow, and
governance files must be merged deliberately. Do not use `--force` as a generic
upgrade mechanism.
