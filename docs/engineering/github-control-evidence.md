# GitHub Control Evidence

Repository rulesets and branch protections are security controls. Documentation
of intended settings is not evidence of current enforcement.

`tools/github/export_ruleset_evidence.py` collects repository identity, the
exact source commit and tree, default branch, repository rulesets, `dev` and
`main` branch protection, collector identity/version, timestamp, and a SHA-512
digest of the canonical raw configuration. The collector independently resolves
the supplied commit through the GitHub API and rejects a repository mismatch.

The collector requires authenticated `gh` access authorized to read repository
administration metadata.

The offline validator evaluates the effective target, include patterns, and
exclude patterns for each ruleset. A rule cannot satisfy a requirement when its
target is wrong or the protected ref is excluded. Evidence must demonstrate:

- `dev` blocks deletion and non-fast-forward changes, requires pull-request
  integration, and requires the exact reviewed policy, portable, integration,
  and native operating-system check names;
- `main` blocks deletion and non-fast-forward changes and remains subject to the
  controlled exact-commit promotion procedure;
- the `isras-*` tag namespace restricts ordinary creation, update, and deletion;
- every bypass actor and bypass mode is explicitly authorized by a reviewed
  release or recovery record.

Collection failure is not a PASS and is reported separately from control
failure. Sensitive administrative metadata may only be redacted through an
approved process preserving every field needed to evaluate effective authority.

## Bypass authorization

Every exported ruleset bypass actor is denied unless the validator receives an
explicit reviewed `actor_type:actor_id:bypass_mode` allowlist entry. An
organization role, administrator label, or repository ownership alone is not
sufficient evidence of release or recovery authorization.

Example:

```bash
python3 tools/isras/validate_github_control_evidence.py \
  --repo-root . \
  --record docs/acceptance/evidence/<campaign>/github-controls.json \
  --expected-repository Iron-Signal-Systems/engineering-standards \
  --expected-commit "$(git rev-parse HEAD)" \
  --required-dev-check policy \
  --required-dev-check portable \
  --required-dev-check integration-tools \
  --required-dev-check native-os-matrix \
  --allowed-bypass-actor RepositoryRole:5:pull_request
```

Exact check names and bypass identities are acceptance inputs. They must come
from the repository workflows and applicable approved authority record rather
than being inferred by the validator.
