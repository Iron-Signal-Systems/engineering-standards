# ISRAS v2 Control Catalog

ISRAS v2 inherits the accepted v1 controls and adds mandatory governance,
phase-compliance, bounded-authority, hostile-testing, and maturity controls.

## Inherited controls

| ID | Requirement |
|---|---|
| ISRAS-GOV-001 | Repository ownership and branch roles are declared. |
| ISRAS-GOV-002 | Sensitive paths are classified in CODEOWNERS. |
| ISRAS-GOV-003 | Emergency bypass is documented and attributable. |
| ISRAS-REP-001 | Project-owned validation inputs are committed. |
| ISRAS-REP-002 | Exact pushed commit is required for acceptance. |
| ISRAS-REP-003 | Fresh-clone validation is available. |
| ISRAS-REP-004 | Historical checkpoints are revalidated from exact trees. |
| ISRAS-ENV-001 | Environment profiles declare required capabilities. |
| ISRAS-ENV-002 | Native, VM, and specialized requirements are explicit. |
| ISRAS-TST-001 | Portable validation runs without sensitive infrastructure. |
| ISRAS-TST-002 | Canonical and specialized results are distinct. |
| ISRAS-DOC-001 | Documentation is synchronized in the same change set. |
| ISRAS-SUP-001 | Dependencies and workflow references are integrity checked. |
| ISRAS-SUP-002 | Release artifacts receive hashes, SBOM, and provenance as applicable. |
| ISRAS-DAT-001 | Accepted migrations are hash-bound and immutable. |
| ISRAS-EVD-001 | Acceptance evidence identifies source, environment, and result. |
| ISRAS-EVD-002 | Evidence is redacted and retention-classified. |
| ISRAS-OPS-001 | Installation, rollback, restore, and recovery are validated before readiness claims. |
| ISRAS-PER-001 | Performance thresholds are governed only after representative baselines. |
| ISRAS-ID-001 | Specialized identity labs are isolated from public pull-request execution. |

## v2 controls

| ID | Requirement |
|---|---|
| ISRAS-GOV-004 | Accepted ISRAS requirements are mandatory governance. |
| ISRAS-GOV-005 | Repository controls may extend but shall not weaken inherited controls. |
| ISRAS-GOV-006 | Repositories remain pinned to an exact accepted ISRAS release. |
| ISRAS-GOV-007 | New accepted releases require an Engineering Standards Impact Assessment. |
| ISRAS-PHS-001 | Every phase requires a recorded phase-entry compliance review. |
| ISRAS-PHS-002 | Every phase requires a passing phase-exit compliance review. |
| ISRAS-PHS-003 | Engineering Standards Compliance Review failure fails phase acceptance. |
| ISRAS-AUT-001 | Unrestricted execution contexts are prohibited. |
| ISRAS-AUT-002 | Authority is limited to the current operation. |
| ISRAS-AUT-003 | Privilege does not implicitly propagate across boundaries. |
| ISRAS-AUT-004 | Administrative and ordinary identities are separated. |
| ISRAS-AUT-005 | Database and migration identities use bounded roles. |
| ISRAS-AUT-006 | Accumulated roles may not create unrestricted authority. |
| ISRAS-AUT-007 | Elevation and break-glass use are explicit, bounded, attributable, audited, and reviewed. |
| ISRAS-TST-003 | New or changed authority, trust, lifecycle, security, and operational boundaries receive applicable hostile-condition testing. |
| ISRAS-EVD-003 | Every accepted phase retains standards-compliance evidence. |
| ISRAS-EVD-004 | Applicable controls report DOCUMENTED, IMPLEMENTED, VALIDATED, or ACCEPTED maturity without overclaim. |

## Control interpretation

Control identifiers are stable references. Text may be clarified in compatible
minor releases, but a change that weakens an obligation, changes required
evidence, invalidates an accepted workflow, or changes validator behavior
incompatibly requires a major release.
