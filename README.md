# Iron Signal Engineering Standards

> **Built on purpose. Backed by discipline. Engineered to endure.**

This repository defines the common engineering, secure-development, repository
assurance, validation, acceptance, release, deployment, and operational evidence
standards for Iron Signal Systems projects.

The first normative standard is the **Iron Signal Repository Assurance Standard
(ISRAS) v1**.

## What is ISRAS?

**ISRAS** stands for the **Iron Signal Repository Assurance Standard**. It is
the organization-wide Iron Signal Systems standard governing repository
reproducibility, validation, historical verification, change control, evidence,
acceptance, release, deployment verification, recovery, and long-term
maintainability.

ISRAS does **not** stand for “Information System Risk Assessment” and is not
itself a risk-assessment methodology. Projects may be required to maintain
information-system risk assessments, threat models, risk registers, findings,
and remediation evidence, but those remain separate assurance artifacts. ISRAS
governs how those artifacts and their related implementation and evidence are
versioned, validated, accepted, and maintained.

## Current status

ISRAS v1.0.0 remains an acceptance candidate until the exact merged `dev`
commit passes the candidate gate, native operating-system matrix, fresh-clone
campaign, evidence validation, and formal acceptance decision. Do not infer
formal acceptance from the target version number alone.

## Validation tool bootstrap

The policy validator uses exactly pinned Python tools declared in
`tools/requirements.txt`. Install them into a recreatable local virtual
environment without requiring Docker or Podman:

```bash
chmod +x tools/environment/bootstrap_tools.sh
./tools/environment/bootstrap_tools.sh
export ISRAS_PYTHON="$PWD/.isras-tools-venv/bin/python"
```

PowerShell users can run `tools/environment/Bootstrap-Tools.ps1` and set
`ISRAS_PYTHON` to the generated virtual-environment Python executable.

## Governing rule

A change is not complete merely because it works on the developer's current
system. It is complete only when its exact pushed commit can be reconstructed,
validated, and evidenced from the canonical repository using declared
environments and committed project-owned assets.

## Native-first portability

ISRAS does not require Docker or Podman as the universal answer to
reproducibility.

Projects must declare and validate their native host, virtual-machine, or
specialized-lab requirements. Containers may be used when useful, but they must
not hide undeclared dependencies or become the only validation path unless the
product's accepted deployment model is itself container-native.

## Repository contents

- `standards/repository-assurance/v1/` — normative ISRAS v1 documents; start with [the document index](standards/repository-assurance/v1/INDEX.md)
- `schemas/` — machine-readable assurance, environment, checkpoint, migration,
  and evidence schemas
- `templates/repository-baseline/` — files copied into adopting repositories
- `templates/workflows/` — GitHub Actions caller examples
- `.github/workflows/` — reusable organization workflows
- `tools/isras/` — adoption, policy, fresh-clone, checkpoint, manifest, and
  evidence tooling
- `integration-guides/` — project-specific adoption sequences
- `GLOSSARY.md` — authoritative terminology, including the ISRAS definition
- `SUPPORT-AND-COMPATIBILITY.md` — support, compatibility, and lifecycle policy

## Adoption sequence

1. Read `standards/repository-assurance/v1/ADOPTION-GUIDE.md`.
2. Create a work branch in the target repository.
3. Preview adoption:

   ```bash
   python3 tools/isras/adopt.py \
     --target /path/to/repository \
     --repository Iron-Signal-Systems/example \
     --canonical-origin git@github.com:Iron-Signal-Systems/example.git \
     --development-branch dev \
     --release-branch main \
     --profile general \
     --dry-run
   ```

4. Apply the baseline, review every generated file, and customize the
   project-specific commands.
5. Run portable and fresh-clone validation.
6. Introduce GitHub Actions in observation mode.
7. Enable repository rules only after the workflows are stable.
8. Formally accept the repository-assurance boundary.

## Standard versioning

Adopting repositories pin the standard and reusable workflows to an exact commit
SHA. A standard release tag is useful for humans, but exact commit identity is
the normative machine reference.
