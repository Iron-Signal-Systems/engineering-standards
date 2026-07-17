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
tools/isras
project-owned documentation
project-owned command declarations
project-owned bounded exceptions
CI integration pinned to an immutable workflow commit
```

The first committed project-pin contract is
[`schemas/isras-project-v1.schema.json`](../schemas/isras-project-v1.schema.json).
A repository containing a candidate pin can run:

```bash
./.local/bin/isras-validate project-pin validate
./.local/bin/isras-validate project-pin inspect
```

These commands are read-only and do not acquire artifacts or execute the declared
project commands.

ISRAS shall not ordinarily copy its Go validator source or tests into the
project, and shall not add itself to the project's application dependency graph.

## New project

New-project initialization will:

1. require an exact accepted release;
2. verify the signed tag, source commit, manifests, and artifacts;
3. select applicable profiles;
4. prepare the repository framework;
5. preserve project authority over application design;
6. validate the initialized result;
7. leave reviewable changes without committing or pushing.

## Existing project

Existing-project adoption begins with an inventory. It maps existing project
artifacts to the pinned release and prepares only the missing or incompatible
framework changes.

Adoption is not permission to reorganize working application source merely to
match a reference layout.

## Local and CI consistency

Local validation and CI shall read the same committed project pin. They shall
execute the same ISRAS release identity and report the same target project
boundary.

CI may call an immutable reusable workflow from Engineering Standards, but that
workflow must verify that its release identity corresponds to the project pin.

## Upgrade

A project moves to a later ISRAS release only through the process defined by
[`standards/PROJECT-UPGRADE-CONTRACT.md`](../standards/PROJECT-UPGRADE-CONTRACT.md).

A newer release being available is information, not modification authority.

## Current implementation status

The pinned project framework is the accepted architectural direction for the
`0.1.1-development` cycle. The strict v1 pin schema and read-only parser are now
implemented. Artifact acquisition, release verification, command execution,
project initialization, and upgrade application are not yet implemented.

The existing `tools/export-project-validator.sh` source-copy model remains
deprecated for new adoption. It must not be used to initialize another project.

No consuming project should be modified until the complete replacement boundary
has passed its own tests and acceptance gates.

## Related contracts

- [`standards/ISRAS-CORE-AND-LANGUAGE-PROFILES.md`](../standards/ISRAS-CORE-AND-LANGUAGE-PROFILES.md)
- [`standards/GO-REFERENCE-PROFILE.md`](../standards/GO-REFERENCE-PROFILE.md)
- [`standards/PINNED-PROJECT-FRAMEWORK.md`](../standards/PINNED-PROJECT-FRAMEWORK.md)
- [`standards/PROJECT-PIN-SCHEMA.md`](../standards/PROJECT-PIN-SCHEMA.md)
- [`standards/ISRAS-RELEASE-ARTIFACT-CONTRACT.md`](../standards/ISRAS-RELEASE-ARTIFACT-CONTRACT.md)
- [`standards/PROJECT-UPGRADE-CONTRACT.md`](../standards/PROJECT-UPGRADE-CONTRACT.md)
