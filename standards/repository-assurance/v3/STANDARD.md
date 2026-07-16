# Iron Signal Repository Assurance Standard v3 Candidate

## 1. Status and inheritance

ISRAS v3 is a development candidate. It inherits every accepted ISRAS v2
control without weakening or rewriting the immutable v1 or v2 normative trees.
No repository may claim v3 adoption or acceptance until a signed v3 release is
formally completed.

## 2. Deterministic validation-tool bootstrap

Release validation tooling shall be reconstructed from an accepted,
environment-specific offline wheelhouse. The accepted boundary shall include
the base Python executable identity, pinned pip wheel, exact requirements lock,
wheel provenance and hashes, exact wheelhouse file set, and exact final
installed-distribution set.

Release bootstrap shall start from an absent destination environment, run in
isolated mode, reject package-index access and external pip configuration, and
fail on missing, extra, changed, or incompatible artifacts. A connected
wheelhouse-preparation process produces only a candidate until review, scanning,
validation, and acceptance are complete.

## 3. Digest policy

SHA-512 is primary for new v3 source, bootstrap, and evidence relationships.
Accepted SHA-256 identities from v1 and v2 remain authoritative and are not
silently rewritten. Hash algorithms identify bytes; they do not replace source,
signer, authority, environment, or acceptance decisions.

Tracked-source manifests shall be generated and verified from the Git index or
an exact commit tree, not from an ambiguous working-tree view.

## 4. Evidence relationships

Evidence shall be bound to the exact repository, commit, campaign, environment
artifact, validator source, test identifiers, and outcome. Validators shall:

- resolve the expected commit and repository independently;
- verify referenced paths are safe, present, tracked, and SHA-512 matched;
- compare validator bytes with both the working source and exact committed blob;
- verify artifact-internal source, campaign, and environment identities;
- extract the claimed PASS result from the artifact rather than trust a copied
  Boolean or narrative assertion;
- verify control and test references against declared obligations; and
- reject reuse across incompatible source, campaign, or environment boundaries.

## 5. External standards translation

ISRAS shall maintain a machine-readable, control-level external standards
crosswalk. Mapping states shall distinguish partial contribution, project
responsibility, and non-applicability without claiming certification or
equivalence. Formal phase entry requires immutable baseline identification and
review of every mapping.

## 6. Repository self-assurance

The engineering-standards repository shall state the exact accepted ISRAS
release governing its own work. A later development tree does not retroactively
claim acceptance under itself. `RELEASE_ASSURED` requires an exact accepted tag,
commit, manifest identity, and consistent self-assurance record.

## 7. Effective GitHub controls

Acceptance and release evidence shall include exported GitHub ruleset and branch
protection configuration. Validation shall evaluate rule target, include and
exclude conditions, exact required check names, ordinary branch/tag mutation
restrictions, and explicitly authorized bypass actors and modes.

## 8. Proportional change governance

Every change shall receive the highest applicable C0 through C6 class. C0–C2
form a common foundation. C3 security/authority and C4 schema/migration are
parallel impact branches. C5 and C6 add acceptance or release campaigns while
retaining every applicable branch. Classification shall be checked against the
actual changed path set and shall never waive an inherited or project control.

## 9. Candidate non-claims

Passing a development-candidate gate does not establish formal ISRAS v3
acceptance, production readiness, certification, regulatory compliance,
independent assurance, or absence of vulnerabilities. Those claims require the
applicable reviewed evidence and acceptance boundary.
