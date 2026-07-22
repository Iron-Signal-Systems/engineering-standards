# Project Adoption

## Purpose

An Iron Signal Systems project adopts an accepted, immutable ISRAS release. The
project uses that release as the framework for repository governance,
documentation, testing, validation, evidence, release, and recovery while
retaining authority over its language and application design.

A project does not follow `engineering-standards/dev`, `main`, or the newest
available release automatically.

## Adoption model

After publication and post-publication acceptance, a project created while
`isras-v0.1.6` is the accepted baseline shall pin `isras-v0.1.6`, its exact
release commit, and the required artifact digests. It remains on that release
until an explicit upgrade is planned, reviewed, validated, committed, and
accepted. The immutable `isras-v0.1.5` tag is unpublished and non-adoptable.

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

After publication and post-publication acceptance, the accepted `0.1.6`
release validator initializes one explicitly selected release with:

```bash
isras-validator-linux-amd64 \
  --repo /src/example-project \
  project-pin initialize \
  --release isras-v0.1.6 \
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
workflow must verify its called-workflow identity, establish release-bound SSH
signer trust, bootstrap-check the release, digest-bind and execute the published
validator artifact, run repository and secret-protection checks, and retain both
`.local/isras` and `.local/validation` evidence.

## Upgrade

A project moves to a later ISRAS release only through the process defined by
[`standards/PROJECT-UPGRADE-CONTRACT.md`](../standards/PROJECT-UPGRADE-CONTRACT.md).

A newer release being available is information, not modification authority.

## Current implementation status

ISRAS `0.1.2` remains an immutable published release, but its reusable workflow
does not establish SSH signer trust on a clean runner and does not retain the
validator's `.local/validation` failure logs. A failed required hosted run under
that release is not formal project adoption.

The signed `isras-v0.1.3` tag is immutable but was not published. Its first
publication attempt exposed an invalid upload-host invocation and a cleanup path
that could not observe the empty draft it had created. The exact failed draft was
independently verified and deleted. `0.1.3` is not adoption authority.

ISRAS `0.1.4` remains an immutable published release and the current adoption
authority until the corrective 0.1.6 release completes publication and
post-publication acceptance.

The signed `isras-v0.1.5` tag is immutable but unpublished and non-adoptable. Its
release-artifact producer incorrectly required exact Go compiler equality even
though the accepted standard defines the `go.mod` `go` directive as a minimum.
No canonical 0.1.5 asset set or GitHub Release exists.

ISRAS `0.1.6` is the corrective release candidate. It preserves Workstream A and
corrects release-artifact production to accept later compatible releases and
valid custom-suffix toolchains while rejecting toolchains below the minimum.

A consuming project may pin `isras-v0.1.6` only after the exact signed tag and
verified six-asset release are published and that repository's required
adoption or upgrade validation passes.

The existing `tools/export-project-validator.sh` source-copy model remains
deprecated and must not be used for new adoption.

## Related contracts

- [`standards/ISRAS-CORE-AND-LANGUAGE-PROFILES.md`](../standards/ISRAS-CORE-AND-LANGUAGE-PROFILES.md)
- [`standards/GO-REFERENCE-PROFILE.md`](../standards/GO-REFERENCE-PROFILE.md)
- [`standards/HOSTED-SSH-SIGNER-TRUST.md`](../standards/HOSTED-SSH-SIGNER-TRUST.md)
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
