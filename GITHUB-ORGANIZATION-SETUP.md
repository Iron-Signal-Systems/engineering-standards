# GitHub Organization Setup

Repository files cannot fully configure organization or repository settings.

## Organization Actions policy

After reusable workflows are published and pinned:

- allow GitHub-authored actions and approved Iron Signal Systems reusable
  workflows;
- require full commit SHA references where supported;
- minimize default workflow token permissions;
- restrict self-hosted runner groups to exact repositories;
- keep public pull requests away from sensitive runners.

## Repository rulesets

Create observation-mode rulesets first, then activate them after stable workflow
runs.

### Development branch

- require pull requests;
- require stable policy and portable checks;
- block deletion and force pushes;
- require resolved conversations;
- use linear history where appropriate;
- retain a documented emergency bypass.

### Release branch

- no ordinary development;
- require release assurance;
- accept only validated promotion;
- block deletion and force pushes.

### Tags

Protect:

```text
v*
phase-*
release-*
```

## Private vulnerability reporting

Enable GitHub private vulnerability reporting where available and make the
configured private channel match `SECURITY.md`.

## Future teams

Create teams when additional qualified reviewers exist:

- platform maintainers;
- architecture reviewers;
- security reviewers;
- database security reviewers;
- assurance reviewers;
- module or product reviewers.

Do not claim separation of duties while all teams are controlled by one person.
