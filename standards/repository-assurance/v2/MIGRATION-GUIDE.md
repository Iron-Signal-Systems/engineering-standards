# Migration Guide: ISRAS v1.0.1 to v2.0.0

## 1. Preconditions

An adopting repository shall first complete and record its current pinned ISRAS
v1.0.1 adoption. It shall not claim organization-wide v2 compliance while v2 is
a candidate.

A repository may adopt stricter local bounded-authority rules before v2 release,
but those rules are repository controls, not proof of accepted v2 adoption.

## 2. Required migration sequence

1. Confirm the current adoption record pins exact v1.0.1 version, signed tag,
   commit, and source-manifest digest.
2. Complete the current phase under its recorded baseline unless a deliberate
   mid-phase upgrade is approved.
3. After v2.0.0 is accepted and signed, create an ESIA comparing the pinned
   v1.0.1 release with the exact v2.0.0 release.
4. Classify every new or changed control.
5. Inventory authentication, session, service, API, worker, queue, background,
   administrative, deployment, migration, and database authority boundaries.
6. Create authority boundary records for applicable boundaries.
7. Update requirements and architecture for bounded authority and privilege
   non-propagation.
8. Implement required identity, authorization, role, revocation, audit, and
   break-glass controls.
9. Add hostile-condition campaigns.
10. Add phase-entry and phase-exit compliance records and validation.
11. Synchronize roadmap, sequencing, acceptance criteria, and evidence.
12. Validate the exact pushed adoption candidate.
13. Formally accept the pinned v2 adoption.

## 3. Iron Atlas sequence

Iron Atlas shall complete its current pinned ISRAS v1.0.1 `RECORDED` adoption
first. It may immediately establish bounded authority as an Atlas-specific
security axiom. After ISRAS v2.0.0 is accepted, Atlas shall perform an ESIA,
adopt the exact accepted v2.0.0 commit and digest, and then prove compliance at
each phase entry and exit.

## 4. Migration non-claims

Copying v2 templates, referencing the v2 candidate, or implementing selected v2
controls does not constitute accepted v2 adoption. Adoption requires the exact
accepted release and a reviewed repository change.
