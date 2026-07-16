# Iron Signal Engineering Standards

> **Built on purpose. Backed by discipline. Engineered to endure.**

This repository defines the common engineering, secure-development, repository
assurance, validation, acceptance, release, deployment, and operational evidence
standards for Iron Signal Systems projects.

The current accepted normative standard is the **Iron Signal Repository Assurance
Standard (ISRAS) v2.0.1**. The accepted ISRAS v1 normative tree remains retained
and immutable for historical verification and repositories that are still
deliberately pinned to an accepted v1 release.

## What is ISRAS?

**ISRAS** stands for the **Iron Signal Repository Assurance Standard**. It is
the organization-wide Iron Signal Systems standard governing repository
reproducibility, validation, historical verification, change control, evidence,
acceptance, release, deployment verification, recovery, long-term
maintainability, engineering-standard inheritance, phase compliance, and
bounded authority.

ISRAS does **not** stand for “Information System Risk Assessment” and is not
itself a risk-assessment methodology. Projects may be required to maintain
information-system risk assessments, threat models, risk registers, findings,
and remediation evidence, but those remain separate assurance artifacts. ISRAS
governs how those artifacts and their related implementation and evidence are
versioned, validated, accepted, and maintained.

## Current status

The repository's declared source version is recorded in `VERSION`. A version
number alone is not acceptance evidence.

The latest accepted release is **ISRAS v2.0.1**, jointly identified by:

- signed annotated tag `isras-v2.0.1`;
- exact release commit `d34fad82781a4e8485f8907fbfd34f236fa79ad2`;
- remote `main`;
- source-manifest SHA-256 `8f54ed1e9bfee251bf89b4c5f12edf11ac1e25ef0d145ba745301f2d05787ef1`.

ISRAS v2.0.1 carries the BSD 3-Clause licensing decision into the signed release
line without changing normative ISRAS v2 controls. The later release-completion
and checkpoint record is retained on `dev` at `08a0a514ec308f76dbf80ffdcb8caa70ce6e345f` while `main` and the
signed tag remain fixed at the immutable release source.

`dev` may advance with governed candidate work after a release. The signed
annotated tag remains the authoritative acceptance-decision object.

## ISRAS v3 assurance-hardening candidate

The `standards/repository-assurance/v3/` tree is a **development candidate** for
the next major assurance boundary. It is not accepted, released, or inherited
by adopters.

The v3 candidate addresses:

- clean-room, isolated, offline-capable release tool bootstrap bound to an exact Python executable and accepted wheelhouse;
- SHA-512-first evidence relationships while retaining SHA-256 where required
  for accepted history or ecosystem compatibility;
- tracked evidence-file existence, digest, artifact-internal source, campaign, environment, validator, control, test, and extracted-outcome binding;
- a machine-readable control-level external framework crosswalk that does not claim certification or equivalence;
- explicit self-assurance for this standards repository;
- exported and validated GitHub ruleset evidence; and
- risk-proportionate change classes with parallel security and schema branches that preserve mandatory controls without imposing irrelevant campaigns.

Start with [the v3 candidate index](standards/repository-assurance/v3/INDEX.md)
and [the candidate plan](docs/acceptance/isras-v3.0.0-plan.md).

## Validation tool bootstrap

### Developer bootstrap

The compatibility developer bootstrap installs the exactly versioned top-level
Python requirements and records Python, pip, platform, and installed
distributions. It no longer upgrades pip implicitly.

```bash
chmod +x tools/environment/bootstrap_tools.sh
./tools/environment/bootstrap_tools.sh
export ISRAS_PYTHON="$PWD/.isras-tools-venv/bin/python"
```

PowerShell users can run `tools/environment/Bootstrap-Tools.ps1`.

Developer bootstrap output is not sufficient release-bootstrap evidence because
network package resolution may still occur.

### Deterministic release bootstrap

Release and acceptance campaigns use a governed environment-specific wheelhouse
containing a pinned pip wheel, all resolved tool wheels, a hash-locked
`requirements.lock`, `bootstrap-lock.json`, and `SHA512SUMS`.

```bash
chmod +x tools/environment/bootstrap_tools_release.sh
ISRAS_WHEELHOUSE=/approved/wheelhouse \
  ./tools/environment/bootstrap_tools_release.sh
```

The release bootstrap refuses an existing destination environment, performs no package-index access, isolates Python and pip configuration, verifies the complete wheelhouse and upstream provenance before installation, and proves the final installed distribution set exactly matches the accepted lock.

## Governing rule

A change is not complete merely because it works on the developer's current
system. It is complete only when its exact pushed commit can be reconstructed,
validated, and evidenced from the canonical repository using declared
environments and committed or governed project-owned assets.

ISRAS v2 additionally requires exact standards inheritance, non-weakening
governance, Engineering Standards Impact Assessments where applicable, phase
entry and exit reviews, maturity-accurate evidence, hostile authority testing,
and prevention of unrestricted execution contexts.

The v3 candidate adds proportional change classification. Proportionality
changes campaign depth, not whether applicable mandatory controls still apply.

## Native-first portability

ISRAS does not require Docker or Podman as the universal answer to
reproducibility.

Projects must declare and validate native host, virtual-machine, or
specialized-lab requirements. Containers may be used when useful, but they must
not hide undeclared dependencies or become the only validation path unless the
accepted deployment model is container-native.

## Repository contents

- `standards/repository-assurance/v3/` — development-only v3 candidate
- `standards/repository-assurance/v2/` — current accepted normative ISRAS v2
- `standards/repository-assurance/v1/` — retained accepted ISRAS v1 history
- `schemas/` — machine-readable assurance and evidence contracts
- `templates/repository-baseline/` — adopting-repository baseline
- `templates/engineering-standards/` — governed record templates
- `.github/workflows/` and `templates/workflows/` — reusable workflows
- `tools/isras/` — assurance and validation tooling
- `tools/github/` — GitHub configuration-evidence collection
- `integration-guides/` — project-specific adoption sequences
- `GLOSSARY.md` — authoritative terminology
- `SUPPORT-AND-COMPATIBILITY.md` — lifecycle policy

## ISRAS v2 adoption sequence

1. Read `standards/repository-assurance/v2/MIGRATION-GUIDE.md`.
2. Confirm the target repository's pinned ISRAS release.
3. Complete an Engineering Standards Impact Assessment when required.
4. Create a reviewed work branch.
5. Pin exact accepted release commit `d34fad82781a4e8485f8907fbfd34f236fa79ad2` and source-manifest
   digest `8f54ed1e9bfee251bf89b4c5f12edf11ac1e25ef0d145ba745301f2d05787ef1`.
6. Apply and customize the baseline without weakening controls.
7. Complete phase-entry review.
8. Run portable, fresh-clone, policy, compliance, and specialized validation.
9. Complete phase-exit review with exact pushed-source evidence.
10. Record adoption only after acceptance passes.

Publication of a newer release does not silently change a pinned repository.

## License

Repository-authored materials in revisions containing the root `LICENSE` file
are licensed under the BSD 3-Clause License (`BSD-3-Clause`), except where a
different license is explicitly identified.

See [`LICENSE`](LICENSE) and [`LICENSING.md`](LICENSING.md).

## Standard versioning

Adopting repositories pin the standard and reusable workflows to an exact commit
SHA. A signed release tag provides the authoritative acceptance decision, while
the exact commit remains the normative machine reference.
