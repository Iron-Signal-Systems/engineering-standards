# ISRAS Repository Identity and Public Use

**Status:** GOVERNING SCOPE CANDIDATE — NOT RELEASED OR ADOPTABLE

## Authoritative identity

**ISRAS is the governing engineering authority for Iron Signal Systems repositories.** It establishes consistent requirements, decision rationale, validation methods, evidence expectations, release boundaries, and lifecycle controls across company projects. Public use is permitted, but external adoption is not its primary design objective.

## Primary audience

ISRAS exists first to govern Iron Signal Systems repositories. Its standards,
profiles, validators, evidence contracts, release controls, and lifecycle rules
are designed around the company's need for consistent, reviewable, repeatable,
and durable engineering authority across its projects.

This authority includes:

- engineering requirements and decision rationale;
- validation methods and evidence expectations;
- source, release, deployment, recovery, and upgrade boundaries;
- bounded exceptions and accountable approvals;
- profile inheritance and repository-specific declarations;
- lifecycle maintenance and historical reconstruction.

## Public visibility

The repository is public to support transparency, technical review, durable
reference, and reuse where appropriate. Public visibility is a distribution and
review property; it is not the repository's governing product identity.

A public repository is not automatically:

- a general-purpose public product;
- a promise of universal compatibility;
- a public support commitment;
- a guarantee that every external use case will shape the roadmap;
- authority for an Iron Signal Systems project to adopt unreleased source.

## External use

Public use is permitted, but external adoption is not ISRAS's primary design
objective. External users may evaluate or reuse the published material where
appropriate, but they must not infer company governance, support, certification,
compatibility, or release authority that the accepted ISRAS release does not
expressly establish.

External use does not weaken the controls required for Iron Signal Systems
repositories and does not require the core standard to become a least-common-
denominator public framework.

## Language-neutral core

ISRAS is language-neutral at its core. The core standard defines required
outcomes, rationale, evidence, authority, and lifecycle controls without assuming
one programming language, framework, database, deployment topology, or source
layout.

Language and platform profiles are additive mappings. They explain how a
particular ecosystem normally demonstrates conformance and may define specialized
tools or evidence. A profile does not replace the core standard or redefine the
identity of ISRAS.

Go is the first implementation language for repository-owned tooling and the
first supported project profile. That implementation priority does not make
ISRAS a Go-only standard or a general-purpose Go product.

## Adoption authority

Public source, a development branch, an open pull request, or a copied framework
directory is not adoption authority. An Iron Signal Systems project adopts only
an exact accepted and published ISRAS release through the governed pin, profile,
validation, evidence, and upgrade boundaries.

## Review rule

Repository and release review must distinguish these concepts:

1. **Public visibility** permits transparent access and review.
2. **Public use** permits appropriate external reference or reuse.
3. **Company governance** is the primary ISRAS design objective.
4. **Adoption authority** comes only from an exact accepted release and explicit
   project adoption.
5. **Profiles are additive** and do not redefine the language-neutral core.

Reviewers must not treat public visibility as proof that ISRAS is intended to be
a general-purpose public product.
