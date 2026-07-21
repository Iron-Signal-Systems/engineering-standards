# ISRAS repository identity and language-neutrality record

**Status:** A5 WORKING-TREE CANDIDATE — NOT RELEASED OR ADOPTABLE

## Decision

The repository now states near the beginning of the README:

> ISRAS is the governing engineering authority for Iron Signal Systems repositories. It establishes consistent requirements, decision rationale, validation methods, evidence expectations, release boundaries, and lifecycle controls across company projects. Public use is permitted, but external adoption is not its primary design objective.

The same authority and audience boundary is synchronized with the vision,
language-neutral core, Go profile, and project-adoption documentation.

## Clarified boundaries

This change establishes that:

- ISRAS primarily governs Iron Signal Systems repositories;
- public visibility supports transparency and review;
- public use does not redefine ISRAS as a general-purpose public product;
- external adoption is permitted but is not the primary design objective;
- the core standard is language-neutral;
- language and platform profiles are additive;
- Go is the first implementation and profile, not the identity of ISRAS;
- adoption authority still requires an exact accepted release.

## Change isolation

This A5 step changes only the governed documentation and changelog paths listed
in its acceptance script. Runtime code, schemas, workflows, validation commands,
release state, project consumers, and repository history are not modified.

## Remaining work

A6 will add the documentation-impact gate and then the complete Workstream A
acceptance campaign will verify implementation, documentation, schemas, tests,
workflows, records, and evidence together.
