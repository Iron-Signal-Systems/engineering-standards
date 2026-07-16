# Mandatory Governance and Pinned Inheritance

## 1. Governing contract

An accepted ISRAS release is the mandatory engineering contract for each
repository that formally adopts it. Adoption creates an explicit engineering
obligation; it is not a declaration of intent.

Repository owners remain responsible for applying the standard to the
repository's actual architecture, implementation, environments, operations, and
phase model. A repository shall not classify a control as not applicable merely
because compliance is inconvenient, incomplete, expensive, or deferred.

## 2. Exact adoption record

The adoption record shall contain:

- exact ISRAS version;
- signed release tag;
- exact 40-character commit;
- source-manifest SHA-256 digest;
- canonical standards repository;
- adoption decision artifact;
- adoption date;
- adopting repository commit;
- reviewer and approval context.

The signed tag and source commit shall resolve to the same accepted release
boundary. The source-manifest digest shall be verified before adoption.

## 3. No floating governance

The following are prohibited as governing baselines:

- `dev`, `main`, or another moving branch;
- an unsigned or moving tag;
- `latest` or equivalent aliases;
- a release page without an exact source commit;
- a copied standard whose origin and digest are not recorded.

Automated tooling may discover that a newer accepted release exists. It shall
not silently replace the pinned baseline.

## 4. Additive repository controls

Repository controls may:

- impose stricter maturity thresholds;
- add domain-specific evidence;
- add hostile test classes;
- require stronger separation of duties;
- reduce privilege or trust further;
- shorten review or exception periods.

Repository controls shall not:

- waive inherited controls without a governed exception permitted by ISRAS;
- substitute documentation for required implementation or validation;
- broaden authority prohibited by ISRAS;
- collapse distinct validation outcomes;
- weaken historical verification;
- permit a phase to bypass entry or exit review.

## 5. New release handling

When a newer accepted ISRAS release exists, the adopting repository shall create
an Engineering Standards Impact Assessment. The assessment determines whether
controls are already satisfied, require changes, are genuinely not applicable,
or must be governed as future work.

An in-progress phase remains governed by the exact baseline recorded at phase
entry unless the repository deliberately adopts the newer release during the
phase. A decision to adopt during a phase shall update requirements,
architecture, implementation, validation, sequencing, and acceptance criteria
in the same change set.

## 6. Exceptions and deferments

An exception or deferment shall be explicit, attributable, bounded, reviewed,
and linked from the applicable impact and phase-compliance records. It shall not
claim compliance for an unsatisfied control. A control with an open exception
cannot be classified ACCEPTED unless the standard explicitly permits acceptance
with that exact exception and the acceptance record states the limitation.
