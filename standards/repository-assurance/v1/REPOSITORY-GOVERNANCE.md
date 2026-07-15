# Repository Governance

## Development boundary

- `dev` is the active integration branch unless the repository explicitly
  declares another branch.
- Work occurs on purpose-named branches.
- Pull requests target `dev`.
- `main` identifies accepted or releasable state, not ordinary development.

## Rulesets

After observation-mode workflows are stable:

### Development branch

- require pull requests;
- require stable portable and policy checks;
- block force pushes and deletion;
- require linear history where compatible with the project;
- require resolved review conversations;
- invalidate approvals after material new commits;
- retain a documented emergency bypass.

### Release branch

- no ordinary development;
- accept only validated promotion;
- require release assurance;
- block force pushes and deletion.

### Tags

Protect accepted and release namespaces such as:

```text
phase-*
release-*
v*
```

## CODEOWNERS

Classify sensitive paths immediately. Required independent approval begins only
when a second qualified reviewer exists.

## Emergency bypass

A bypass requires:

- issue or incident identifier;
- reason;
- exact commit;
- person using the bypass;
- checks omitted;
- subsequent validation;
- corrective action.
