# ISRAS v3 Candidate Control Catalog

ISRAS v3 inherits all accepted v2 controls and adds these candidate controls.

| ID | Candidate requirement |
|---|---|
| ISRAS-BST-001 | Release bootstrap uses a governed environment-specific offline wheelhouse and an absent destination environment. |
| ISRAS-BST-002 | Bootstrap shall not perform an implicit unpinned pip self-upgrade or consult undeclared package/configuration sources. |
| ISRAS-BST-003 | Base Python, pip, platform, requirements, lock, wheelhouse, and exact installed-distribution identities are recorded. |
| ISRAS-BST-004 | Pip and every resolved tool artifact are pinned, provenance-recorded, and hash-verified before installation. |
| ISRAS-DIG-001 | SHA-512 is primary for new evidence and bootstrap relationships while accepted compatibility identities remain preserved. |
| ISRAS-EVD-005 | Every referenced repository-relative evidence artifact exists, is tracked when used for acceptance, and cannot escape the repository boundary. |
| ISRAS-EVD-006 | Every applicable evidence artifact matches its recorded SHA-512 digest. |
| ISRAS-EVD-007 | Evidence is bound internally and externally to exact repository, source, campaign, environment, validator, control, and test identities. |
| ISRAS-EVD-008 | Claimed PASS outcomes are extracted from referenced artifacts using governed probes. |
| ISRAS-EVD-009 | Evidence reuse across incompatible source, campaign, or environment boundaries is prohibited. |
| ISRAS-MAP-001 | A control-level, baseline-identified external standards crosswalk is maintained. |
| ISRAS-MAP-002 | Crosswalk states are explicit and do not claim certification or equivalence. |
| ISRAS-GOV-008 | The standards repository is governed by an exact accepted self-pinned release. |
| ISRAS-SCM-001 | Effective GitHub ruleset and branch-protection configuration is collected and validated for acceptance and release. |
| ISRAS-CHG-001 | Every change receives the highest applicable C0 through C6 classification based on impact and changed paths. |
| ISRAS-CHG-002 | Each change class has a mandatory minimum campaign, with C3 and C4 treated as parallel impact branches. |
| ISRAS-CHG-003 | Proportionality shall not waive an applicable inherited or project control. |
| ISRAS-CHG-004 | Security, schema, acceptance, release, and recovery triggers force escalation and applicable branch campaigns. |

These controls are candidates only and do not rewrite accepted v1 or v2.
