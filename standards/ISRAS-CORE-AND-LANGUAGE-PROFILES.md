# ISRAS Core and Language Profiles

## Purpose

The Iron Signal Repository Assurance Standard defines the engineering outcomes,
evidence, lifecycle controls, and repository governance expected of Iron Signal
Systems projects. It does not prescribe one programming language, application
architecture, framework, database, user-interface model, deployment topology, or
source-directory layout for every project.

ISRAS tooling is implemented in Go first. Go is also the first supported project
profile. That implementation choice does not convert Go-specific practice into a
universal requirement for every adopting repository.

## Core authority and additive profiles

ISRAS is language-neutral at its governing core. Core requirements define the
engineering outcome, decision rationale, validation method, retained evidence,
release boundary, and lifecycle control that a repository must demonstrate.

Language and platform profiles are additive implementation mappings. They may
supply ecosystem-specific commands, tools, file conventions, scanners, or
failure modes, but they do not redefine ISRAS, replace the core standard, or
convert one implementation language into a universal company requirement.

The Go implementation and Go reference profile are therefore the first supported
tooling and profile boundary, not the identity or complete scope of ISRAS.

## Authority boundaries

ISRAS has three distinct layers.

### Core standard

The core standard defines language-neutral requirements. Examples include:

- exact source and release identity;
- repeatable build and test entry points;
- declared and reviewable dependencies;
- secret protection and censored failure evidence;
- change control and signed acceptance boundaries;
- documented architecture and security-sensitive behavior;
- bounded exceptions;
- release provenance;
- recovery and rollback;
- explicit adoption and upgrade of an accepted ISRAS release.

A core requirement describes the outcome and evidence that must exist. It shall
not assume that every project uses the same toolchain.

### Language and platform profile

A profile translates applicable core requirements into reviewed implementation
guidance for a language or platform. A profile may identify conventional tools,
commands, manifests, lock files, layouts, or failure modes for that ecosystem.

Profiles do not replace the core standard. They explain how a project using the
profile normally demonstrates conformance.

### Project declaration

The project declares its actual implementation boundary, including:

- selected ISRAS release and profile;
- supported operating systems and deployment classes;
- project-owned format, test, analysis, build, packaging, and validation entry
  points;
- security-sensitive change boundaries;
- repository-specific required documentation;
- specialized scanners and tests;
- release, deployment, recovery, and evidence procedures;
- approved deviations and bounded exceptions.

The project declaration is reviewable source. ISRAS validates it against the
pinned release and selected profile.

## Project design authority

The adopting project retains authority to choose the technology appropriate for
its problem.

A project may use Go, Rust, Python, C#, SQL, shell, web technologies, or another
reviewed stack when that choice is justified by the project architecture and
operational needs. A project may use more than one language when the boundaries
are explicit and the resulting system remains maintainable and testable.

ISRAS shall not reject a project merely because it does not use Go. It may reject
a project that fails to declare, execute, or retain evidence for the controls
required by its pinned standard and selected profile.

## Profile lifecycle

A profile is versioned as part of an accepted ISRAS release. A project pins the
release containing the profile and does not silently inherit later profile
changes.

A future profile shall be added only when an actual project or supported
technology requires it. The initial implementation priority is:

1. language-neutral ISRAS core;
2. Go reference tooling;
3. Go project profile;
4. additional profiles when justified by real projects.

A profile change that alters project obligations requires a normal ISRAS release
and an explicit project upgrade.

## Equivalent evidence

Different profiles may use different tools while satisfying the same core
requirement.

For example, the core requirement may state:

> The project shall provide a repeatable automated test entry point and retain
> evidence of its result.

A Go profile may normally use `go test ./...`. A Rust profile may normally use
`cargo test --locked`. A project-owned wrapper may be acceptable when it
truthfully includes the intended test boundary and remains committed and
reviewable.

The commands differ. The required outcome and evidence remain consistent.

## Non-goals

The core standard does not:

- select the project's business architecture;
- require identical application directories across languages;
- require ISRAS packages in the project's runtime dependency graph;
- make Engineering Standards source part of the product;
- silently rewrite project code;
- claim that passing generic repository checks proves product correctness,
  security, production readiness, or regulatory compliance.

## Governing rule

ISRAS governs how a project proves its engineering discipline. It does not take
ownership of the project's technical design.
