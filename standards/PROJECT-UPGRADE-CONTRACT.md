# Project Upgrade Contract

## Purpose

A project upgrade changes the accepted ISRAS release governing that project. It
is a reviewed engineering change, not an automatic update.

A project pinned to `isras-v0.1.5` remains governed by that release until its
committed pin and required project artifacts are deliberately migrated to
another accepted release.

## Upgrade authority

Only an explicitly invoked upgrade operation may propose a new pin. Scheduled
automation may report that another release exists, but it shall not alter the
project, its workflows, or its accepted assurance boundary.

Upgrade tooling shall separate:

- inspection;
- migration planning;
- plan application;
- project validation;
- commit and publication.

No stage shall imply authority belonging to a later stage.

## Upgrade plan

A plan from one release to another shall identify:

- current and target release identities;
- target artifact and workflow digests;
- new, changed, deprecated, and removed core requirements;
- profile changes;
- schema changes;
- repository-framework changes;
- new documentation or evidence obligations;
- changed project commands;
- new platform requirements;
- exception compatibility;
- release and recovery changes;
- migration risks;
- exact files proposed for change;
- rollback instructions.

A plan is evidence and has no modification authority.

## Applying an upgrade

Applying a reviewed plan shall:

1. confirm the clean target repository and exact current pin;
2. reacquire and verify both current and target release contracts;
3. confirm that the plan still matches the working tree;
4. modify only declared project-framework and pin boundaries;
5. preserve project-owned application design;
6. run target-release validation;
7. restore the pre-upgrade state after any failed applied validation;
8. stage or report the resulting changes;
9. never commit, push, merge, tag, or release automatically.

## Compatibility

The target release shall declare which earlier releases it can assess and which
migration records are available. Upgrade tooling shall fail rather than infer an
unsupported migration path.

Multiple sequential migrations may be required. Tooling may combine them into
one plan only when every intermediate contract is available and the combined
effect remains reviewable.

## Exceptions

Existing project exceptions shall not be copied blindly into a new release.
Each exception must be:

- still applicable;
- mapped to a requirement in the target release;
- within its expiration and authority boundary;
- reviewed against any stronger target control.

An incompatible exception is an upgrade blocker until resolved or replaced
through the target release's governed exception process.

## Security advisories and revocation

A later release or signed advisory may identify a serious defect in a pinned
release. The project may then be required to upgrade within a stated remediation
window.

Even in that case, tooling shall not silently rewrite the project. It shall fail
with the exact advisory identity, affected release range, safe next actions, and
available migration targets.

Emergency policy may prevent release or deployment under a revoked validator
identity, but it shall preserve evidence of the project's previous pin.

## Rollback

Before applying an upgrade, tooling shall record enough local state to restore
the exact project boundary. Rollback shall not erase unrelated work.

After an upgrade has been committed and published, rollback is a new reviewed
change. It must restore a supported release and explain the reason.

## Upgrade acceptance

An upgrade is complete only when:

- the new project pin is committed;
- all required project-framework changes are committed;
- validation passes under the exact target release;
- project-specific tests and release checks pass;
- documentation describes the new boundary;
- evidence identifies the migration plan and result;
- the prior release remains reconstructable from history.

## Non-goal

An ISRAS upgrade does not authorize redesign of project application code merely
to resemble a template. Changes outside the assurance framework require their own
project justification and tests.
