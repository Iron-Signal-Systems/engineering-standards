# Proportional Change Governance

## Purpose

ISRAS proportionality preserves rigor without pretending an editorial correction
and a release or authority-boundary change need identical campaign depth.
Classification never removes an applicable mandatory control.

## Change classes

| Class | Name | Typical scope |
|---|---|---|
| C0 | Editorial | Spelling, formatting, link correction, or non-semantic prose |
| C1 | Tooling maintenance | Validator, dependency, test harness, or workflow maintenance without product behavior change |
| C2 | Normal implementation | Ordinary product, API, UI, service, or module behavior |
| C3 | Security or authority boundary | Authentication, authorization, secrets, trust, audit, privilege, queues, workers, or database authority |
| C4 | Schema or migration | Database schema, migration, durable data contract, compatibility, or destructive data operation |
| C5 | Acceptance boundary | Control semantics, evidence requirements, validator decision semantics, phase gates, or acceptance criteria |
| C6 | Release or recovery | Release promotion, signed tag, deployment, rollback, restore, disaster recovery, key compromise, revocation, or trusted rebuild |

## Minimum campaigns

C0 through C2 form a common cumulative foundation. C3 and C4 are **parallel
risk branches**, not a single linear ladder. A C4 data-contract change does not
inherit security campaigns merely because its number is higher; security
campaigns apply when the change also has a C3 impact. C5 and C6 add their own
acceptance or release campaigns while retaining every applicable C3 and C4
branch.

| Class | Minimum required campaigns |
|---|---|
| C0 | policy, documentation synchronization, links, whitespace |
| C1 | C0 plus portable, unit regression, tool-environment record, and fresh clone when completeness can change |
| C2 | C1 plus applicable integration, traceability, phase review, and exact pushed-source evidence |
| C3 | C2 plus threat/abuse analysis, authority record, hostile testing, findings separation, and revocation/retry/race coverage |
| C4 | C2 plus migration integrity, compatibility, rollback/restore, representative data, destructive safeguards, and historical migration validation |
| C5 | C2 plus ESIA, predecessor revalidation, approval independence, and evidence-relationship validation; add C3 and C4 campaigns only when those impacts are also present |
| C6 | C5 plus trusted build, artifact accounting, SBOM/provenance as applicable, signature verification, remote convergence, deployment/recovery, and checkpoint registration; retain applicable C3/C4 branches |

## Mandatory escalation

A change is classified at the highest applicable class. Labels, locations, or
author intent cannot lower it. The validator also inspects the changed path set
from the declared base commit and imposes a minimum class for governed paths.

- Authorization prose that changes obligation is C5, not C0.
- A migration validator changing acceptance outcomes is at least C5 and also
  carries C4 campaigns.
- A release-tag rule change is C6.
- Worker delegated privilege change is C3.
- A spelling correction without semantic change is C0.

Combined changes receive the highest class. Unrelated lower-risk work should be
split when practical. Emergency response retains classification and requires
post-event reconciliation.

The current development candidate is classified in
[`docs/acceptance/isras-v3.0.0-change-classification.json`](../acceptance/isras-v3.0.0-change-classification.json).
It is C5 with security and schema impacts, so both parallel campaign branches
remain applicable without falsely treating the work as a release operation.
