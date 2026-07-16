# Iron Signal Repository Assurance Standard — ISRAS v2

## 1. Status

This document is the normative candidate for ISRAS v2.0.0. It becomes an
accepted organization standard only after the complete candidate is validated,
formally accepted, signed, tagged, and released. Candidate development shall not
change the repository root `VERSION` from the currently accepted release.

## 2. Governing purpose

ISRAS is the organization-wide engineering contract governing repository
completeness, reproducibility, validation, change control, historical
verification, evidence, acceptance, release, deployment, recovery, and durable
maintenance.

For every Iron Signal Systems repository that has formally adopted an accepted
ISRAS release, accepted ISRAS requirements are mandatory governance. They are
not advisory guidance, optional documentation, or work that may be silently
deferred until production.

## 3. Normative language

The terms **shall**, **shall not**, **must**, **must not**, and **required** are
mandatory. **Should** identifies a recommended practice whose omission requires
an explicit engineering justification. **May** identifies a permitted option.

The normative term for a context possessing effectively unlimited authority is
**unrestricted execution context**. The explanatory phrases **God Access** and
**God Mode** may appear beside that term to make the prohibition unmistakable,
but they do not replace the normative term.

## 4. Mandatory governance invariant

An adopting repository shall:

1. identify the exact accepted ISRAS release it has adopted;
2. treat that pinned release as its governing engineering baseline;
3. review compliance at every phase entry and phase exit;
4. assess newer accepted releases through an Engineering Standards Impact
   Assessment before adoption;
5. retain the minimum evidence required for every accepted phase; and
6. preserve or strengthen every inherited control.

A repository-specific control may extend, specialize, or strengthen an inherited
ISRAS control. It shall not remove, weaken, bypass, reinterpret, or redefine the
inherited control.

## 5. Pinned inheritance

Every adoption record shall bind the repository to:

- an exact semantic version;
- a signed release tag;
- an exact 40-character source commit;
- the SHA-256 digest of the release source manifest;
- the adoption decision and date.

A repository is governed by that exact baseline until a newer accepted release
is deliberately adopted through a reviewed repository change. A floating
branch, moving tag, latest-release alias, or standards-development branch shall
not be a governing baseline.

Publication of a newer accepted ISRAS release triggers an Engineering Standards
Impact Assessment. It does not silently alter an in-progress phase and does not
by itself make that phase noncompliant with the baseline under which the phase
was entered.

## 6. Bounded authority invariant

No authentication event, identity, token, session, process, service, worker, API
handler, database connection, migration runner, background task, scheduler,
administrator account, delegated operation, or accumulated privilege set shall
create an unrestricted execution context (God Access / God Mode).

Every operation shall execute with only the minimum authority explicitly
required for that operation. Authority shall not be granted merely because an
upstream caller, identity, component, session, or process possessed it.

Privilege shall not automatically propagate across process, service, API,
queue, worker, database, module, administrative, deployment, lifecycle, or trust
boundaries. Authentication proves identity; it does not by itself grant
unrestricted authority. Authorization shall be evaluated independently at every
applicable boundary and shall be deny-by-default.

## 7. Control maturity

Every applicable control shall report exactly one cumulative maturity state:

- **DOCUMENTED** — the requirement and intended design exist;
- **IMPLEMENTED** — the applicable technical or procedural control exists;
- **VALIDATED** — retained evidence demonstrates required behavior;
- **ACCEPTED** — the exact validated boundary has received formal acceptance.

A documented intention shall not be represented as implemented, validated, or
accepted. An implementation without evidence shall not be represented as
validated. Validation of a different source, environment, identity, or boundary
shall not be represented as acceptance of the current candidate.

## 8. Phase-entry and phase-exit governance

A phase shall not begin until its phase-entry Engineering Standards Compliance
Review is complete and recorded. A phase shall not be accepted until its
phase-exit review passes against the exact pushed candidate commit.

Failure of the phase-exit Engineering Standards Compliance Review shall fail
phase acceptance. Acceptance records shall not override, suppress, or relabel a
standards-compliance failure.

## 9. Hostile-condition testing

Every new or changed authority, trust, security, lifecycle, or operational
boundary shall be evaluated for hostile-condition testing. Applicable tests
shall address privilege escalation, privilege accumulation, confused-deputy
behavior, authority propagation, replay, races, retries, duplicate execution,
partial failure, revocation, resource exhaustion, break-glass misuse, and other
credible abuse conditions.

Correctness outcomes, resource observations, performance-budget evaluations,
security findings, and operational-readiness outcomes remain separate results.

## 10. Minimum accepted-phase evidence

Every accepted phase shall retain evidence identifying the repository, phase,
exact repository commit, pinned ISRAS release, applicability decisions, impact
classifications, maturity classifications, implementation and validation
obligations, hostile tests, deviations, deferments, resource-observation status,
review context, and acceptance decision.

## 11. Inherited v1 requirements

ISRAS v2 inherits every accepted ISRAS v1 control unless a v2 control explicitly
strengthens or supersedes it. No inherited requirement is removed by omission
from a topic-specific v2 document. The v2 control catalog is the authoritative
combined catalog.

## 12. Definition of compliant phase acceptance

A phase is compliant only when:

- its governing ISRAS baseline is exact and pinned;
- phase-entry review is recorded;
- applicable controls and maturity requirements are explicit;
- documentation, requirements, architecture, implementation, validation, test
  campaigns, sequencing, and acceptance records are synchronized;
- required hostile-condition testing has passed;
- historical predecessor handling remains correct;
- deviations and deferments are governed;
- the exact pushed candidate commit was evaluated;
- phase-exit review passes; and
- minimum compliance evidence is retained.

## 13. Non-claims

ISRAS compliance alone does not establish absence of vulnerabilities, complete
regulatory compliance, production readiness, acceptable performance, high
availability, disaster recovery, or independent human review. Those claims
require their own applicable, validated, and accepted evidence.
