# Project Adoption

## Purpose

An Iron Signal Systems project adopts an accepted, immutable ISRAS release. The
project uses that release as the framework for repository governance,
documentation, testing, validation, evidence, release, and recovery while
retaining authority over its language and application design.

A project does not follow `engineering-standards/dev`, `main`, or the newest
available release automatically.

## Adoption model

A project created while `isras-v0.1.5` is the accepted baseline shall pin
`isras-v0.1.5`, its exact release commit, and the required artifact digests. It
remains on that release until an explicit upgrade is planned, reviewed,
validated, committed, and accepted.

The project commits its pin and project-specific declarations. It executes the
validator released by Engineering Standards rather than copying the validator
implementation into the project.

## What ISRAS governs

The pinned release identifies the applicable:

- core engineering requirements;
- repository-framework requirements;
- selected language and platform profiles;
- required project declarations;
- validation and evidence contracts;
- release and recovery expectations;
- exception rules;
- upgrade path.

## What the project governs

The project retains authority over:

- language selection;
- application architecture;
- frameworks and libraries;
- package and source layout;
- data model;
- deployment topology;
- project-specific commands and tests;
- additional security and operational controls.

A Go profile provides Go-specific implementation guidance. It does not prohibit a
different project from selecting Rust or another justified technology under a
supported profile.

## Project-owned adoption artifacts

The intended project boundary includes:

```text
.isras/project.json
.isras/adoption-verification.json
.isras/check-go-format
.github/workflows/isras-validation.yml
project-owned documentation
project-owned bounded exceptions
```

The first committed project-pin contract is
[`schemas/isras-project-v1.schema.json`](../schemas/isras-project-v1.schema.json).
A repository containing a candidate pin can run:

```bash
./.local/bin/isras-validate project-pin validate
./.local/bin/isras-validate project-pin inspect
```

These commands are read-only and do not acquire artifacts or execute the declared
project commands. The external validator may be launched from another directory:

```bash
isras-validator-linux-amd64 \
  --repo /src/example-project \
  project-pin validate
```

Target selection does not change the caller's working directory and does not
permit the project to replace the validator's embedded release identity.

After exact release, origin, and committed-pin identity match, one declared
project command may be executed by name:

```bash
isras-validator-linux-amd64 \
  --repo /src/example-project \
  project-command run test
```

Execution is governed by
[`standards/PROJECT-COMMAND-EXECUTION.md`](../standards/PROJECT-COMMAND-EXECUTION.md).
It does not treat a modified or staged pin as execution authority.

ISRAS shall not ordinarily copy its Go validator source or tests into the
project, and shall not add itself to the project's application dependency graph.

## New project

The accepted `0.1.2` release validator initializes one explicitly selected
release with:

```bash
isras-validator-linux-amd64 \
  --repo /src/example-project \
  project-pin initialize \
  --release isras-v0.1.2 \
  --go-defaults
```

Initialization is authorized only when the running executable is the exact
linker-bound validator artifact from the requested release. It verifies the signed
release, source commit, exact six assets, both digests, manifests, provenance, and
reusable workflow before modifying the target. It fixes runtime evidence to
untracked `.local/isras`, generates a stable project-owned adoption set, and
leaves reviewable changes without committing or pushing.

## Existing project

The first implementation supports a clean established Go repository that has no
prior ISRAS adoption paths. It preserves application source and layout. Partial,
conflicting, or hand-authored adoption state is refused rather than merged or
overwritten. Inventory-driven migration of such state is a later explicit
boundary.

Adoption is not permission to reorganize working application source merely to
match a reference layout.

## Local and CI consistency

Local validation and CI shall read the same committed project pin. They shall
execute the same ISRAS release identity and report the same target project
boundary.

CI may call an immutable reusable workflow from Engineering Standards, but that
workflow must verify its called-workflow identity, bootstrap-check the release,
digest-bind and execute the published validator artifact, run repository and
secret-protection checks, and retain `.local/isras` evidence.

## Upgrade

A project moves to a later ISRAS release only through the process defined by
[`standards/PROJECT-UPGRADE-CONTRACT.md`](../standards/PROJECT-UPGRADE-CONTRACT.md).

A newer release being available is information, not modification authority.

## Current implementation status

The `0.1.2-development` boundary implements first Go-project initialization and
reusable hosted validation on top of the accepted `0.1.1` release-consumption
foundation. It verifies an exact accepted release before target publication,
generates the canonical pin and caller workflow, installs a non-mutating format
checker and verification evidence, refuses unsafe or conflicting targets, and
proves idempotence and rollback through hostile tests.

This development branch is not adoption authority. Consuming-project adoption
becomes authorized only when this boundary passes acceptance and is published as
the exact signed `isras-v0.1.2` tag with the verified six-asset release set.
Upgrade application and migration of partial prior adoption remain outside this
boundary.

The existing `tools/export-project-validator.sh` source-copy model remains
deprecated and must not be used for new adoption.

## Related contracts

- [`standards/ISRAS-CORE-AND-LANGUAGE-PROFILES.md`](../standards/ISRAS-CORE-AND-LANGUAGE-PROFILES.md)
- [`standards/GO-REFERENCE-PROFILE.md`](../standards/GO-REFERENCE-PROFILE.md)
- [`standards/PINNED-PROJECT-FRAMEWORK.md`](../standards/PINNED-PROJECT-FRAMEWORK.md)
- [`standards/PROJECT-PIN-SCHEMA.md`](../standards/PROJECT-PIN-SCHEMA.md)
- [`standards/PROJECT-INITIALIZATION-AND-ADOPTION.md`](../standards/PROJECT-INITIALIZATION-AND-ADOPTION.md)
- [`standards/ISRAS-RELEASE-ARTIFACT-CONTRACT.md`](../standards/ISRAS-RELEASE-ARTIFACT-CONTRACT.md)
- [`standards/RELEASE-ARTIFACT-PRODUCTION.md`](../standards/RELEASE-ARTIFACT-PRODUCTION.md)
- [`standards/RELEASE-PUBLICATION.md`](../standards/RELEASE-PUBLICATION.md)
- [`standards/ARTIFACT-ACQUISITION-AND-VERIFICATION.md`](../standards/ARTIFACT-ACQUISITION-AND-VERIFICATION.md)
- [`standards/EXTERNAL-TARGET-ROOT.md`](../standards/EXTERNAL-TARGET-ROOT.md)
- [`standards/PROJECT-COMMAND-EXECUTION.md`](../standards/PROJECT-COMMAND-EXECUTION.md)
- [`standards/PROJECT-UPGRADE-CONTRACT.md`](../standards/PROJECT-UPGRADE-CONTRACT.md)
