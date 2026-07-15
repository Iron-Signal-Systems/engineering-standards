# Iron Signal Repository Assurance Standard — ISRAS v1

## 1. Status

This document is the normative repository assurance standard for Iron Signal
Systems projects adopting ISRAS v1.

## 2. Governing rule

A change is not complete merely because it works on the developer's current
system. It is complete only when its exact pushed commit can be reconstructed,
validated, and evidenced from the canonical repository using declared
environments and committed project-owned assets.

## 3. Scope

ISRAS applies to:

- architecture and requirements;
- source code and migrations;
- tests, hostile fixtures, generators, and expected results;
- validation and phase-gate logic;
- documentation and traceability;
- dependencies and build toolchains;
- acceptance records and historical checkpoints;
- release artifacts and provenance;
- deployment, rollback, recovery, and operational evidence;
- specialized labs, including Windows Active Directory trust environments.

## 4. Mandatory principles

### 4.1 Canonical repository completeness

The canonical repository must contain every project-owned input required to
build, test, validate, document, release, deploy, or reconstruct an accepted
checkpoint.

No accepted result may depend on an untracked local file, ignored project input,
personal path, prior database state, retained credential, compiler cache, or
developer memory.

### 4.2 Exact pushed commit

Acceptance evidence must identify the exact commit. The commit must exist in the
canonical remote before acceptance is claimed.

### 4.3 Documentation synchronization

Documentation changes are part of the same change set as the architecture,
implementation, validation, sequencing, environment, or acceptance change they
describe.

### 4.4 Historical checkpoint immutability

An accepted historical gate is run from its accepted historical tree. Later
gates must not weaken earlier gates merely to permit later artifacts.

A discovered validator defect requires an explicit erratum and revalidation
record rather than silent historical rewriting.

### 4.5 Separation of correctness and observation

Correctness, resource observation, performance-budget evaluation, security
findings, and operational readiness are separate outcomes.

A resource report cannot conceal a correctness failure. An unevaluated
performance budget cannot be described as a performance pass.

### 4.6 Native-first validation

The standard does not mandate Docker or Podman for every project.

A repository must declare the host, VM, or specialized environment required by
each validation profile. Containers are optional unless the accepted product
deployment is container-native. Container convenience must not hide undeclared
host or runtime dependencies.

### 4.7 Least authority

Validation, CI, release, deployment, and specialized-lab identities receive only
the authority required for their exact operation.

Public pull-request jobs must not run on sensitive self-hosted infrastructure or
receive production, lab-administration, signing, or deployment secrets.

### 4.8 Machine enforcement before human independence

Automated required checks may be established while a project has one
maintainer. Genuine separation of duties is claimed only after a second
qualified and independent reviewer exists.

A second account controlled by the same person does not constitute independent
review.

## 5. Validation layers

Every adopting repository defines applicable layers:

1. Developer validation.
2. Portable clean-runner validation.
3. Fresh-clone and remote-completeness validation.
4. Canonical environment validation.
5. Specialized environment campaigns.
6. Historical checkpoint revalidation.
7. Release assurance.
8. Deployment, rollback, restore, and operational readiness validation.

## 6. Definition of complete

A change is complete only when all applicable conditions are satisfied:

- requirements and architecture are synchronized;
- implementation is committed;
- tests, fixtures, generators, and expected outcomes are committed;
- validation logic is committed;
- environment and toolchain requirements are declared;
- no machine-specific or untracked project input is required;
- the exact commit is pushed;
- portable validation passes;
- fresh-clone validation passes;
- applicable canonical and specialized campaigns pass;
- accepted predecessors revalidate in isolation;
- the working tree is clean;
- acceptance evidence identifies exact source and result boundaries;
- warnings, limitations, exceptions, and non-claims are recorded.

## 7. Prohibited claims

ISRAS adoption alone does not establish:

- absence of vulnerabilities;
- production readiness;
- complete regulatory compliance;
- independent human review;
- high availability;
- disaster recovery;
- reproducible binary identity;
- tamper-proof history;
- acceptable performance.

Those claims require their own accepted evidence.
