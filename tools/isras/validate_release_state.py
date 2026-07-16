#!/usr/bin/env python3
# Validate central ISRAS v2.0.0 release-state and documentation consistency.
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

EXPECTED_VERSION = "2.0.0"
CANDIDATE_COMMIT = "4aff00dfdc88154390252898210abc336fa8b2fc"
EVIDENCE_COMMIT = "b0c982221acde7873307d010aca73ed2e386eb99"
ACCEPTANCE_COMMIT = "24e911b7c4a63735bcef9b4b84ab9b62ace10298"
RELEASE_COMMIT = "781246e69f8a9a382c25040f94b62dfe3b25ba89"
RELEASE_TAG = "isras-v2.0.0"
TAG_OBJECT = "a7a09a02798e2b2c905f2686820fd30890f62bc6"
MANIFEST_SHA = "262e275e63f1c7d104bb77c8799633121bad43d2fc58edf54594e5eda61555b7"
EVIDENCE_SHA = "0e4516f76032008075a844ddc43cb44fdb90ae09ab31b9af113b32923f082cd7"
SIGNING_FINGERPRINT = "SHA256:FiH+Jk7HHrNkvDEQTehI/aCfkmKpivtsqmkl5TmmMSE"

REQUIRED_FILES = (
    "LICENSE",
    "docs/acceptance/isras-v1.0.0-release-finalization.md",
    "docs/acceptance/isras-v1.0.1-plan.md",
    "docs/acceptance/isras-v2.0.0-plan.md",
    "docs/acceptance/isras-v2.0.0-candidate-acceptance.md",
    "docs/acceptance/isras-v2.0.0-release-finalization.md",
    "docs/acceptance/isras-v2.0.0-release-completion.md",
    "docs/acceptance/isras-v2.0.1-plan.md",
    "docs/engineering/adopter-quick-start.md",
    "docs/engineering/github-release-rulesets.md",
    "standards/repository-assurance/v2/RELEASE-VERSIONING-SUPPORT-AND-DEPRECATION.md",
    "tools/isras/validate_isras_v2_release.py",
    "tools/validation/phase-gates/validate_isras_v2_release.sh",
)

STALE_CURRENT_PHRASES = (
    "The first normative standard is the **Iron Signal Repository Assurance Standard (ISRAS) v1**",
    "ISRAS v1.0.x releases are supported when their exact source commit is",
    "`CANDIDATE DEVELOPMENT`",
)

RELEASE_MARKERS = (
    "signed annotated tag",
    "authoritative acceptance-decision object",
    "dev",
    "main",
    "same exact commit",
)


def fail(message: str) -> None:
    print(f"FAIL: {message}", file=sys.stderr)
    raise SystemExit(1)


def normalize(value: str) -> str:
    return " ".join(value.split())


def read(repo_root: Path, relative: str) -> str:
    path = repo_root / relative
    if not path.is_file():
        fail(f"required file is missing: {relative}")
    return path.read_text(encoding="utf-8")


def require_marker(content: str, marker: str, context: str) -> None:
    if normalize(marker) not in normalize(content):
        fail(f"{context} lacks required marker {marker!r}")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", required=True)
    args = parser.parse_args()
    repo_root = Path(args.repo_root).resolve()

    version = read(repo_root, "VERSION").strip()
    if version != EXPECTED_VERSION:
        fail(f"VERSION must be {EXPECTED_VERSION}, found {version!r}")
    print(f"PASS: VERSION declares {version}")

    for relative in REQUIRED_FILES:
        read(repo_root, relative)
        print(f"PASS: required release file exists: {relative}")

    selected_current_files = (
        "README.md",
        "SUPPORT-AND-COMPATIBILITY.md",
        "templates/repository-baseline/SUPPORT-AND-COMPATIBILITY.md",
        "docs/acceptance/isras-v2.0.0-plan.md",
        "standards/repository-assurance/v2/RELEASE-VERSIONING-SUPPORT-AND-DEPRECATION.md",
    )
    for relative in selected_current_files:
        content = normalize(read(repo_root, relative))
        for phrase in STALE_CURRENT_PHRASES:
            if normalize(phrase) in content:
                fail(f"stale current release wording in {relative}: {phrase!r}")
    print("PASS: stale current release-state wording is absent")

    root_support = read(repo_root, "SUPPORT-AND-COMPATIBILITY.md")
    template_support = read(
        repo_root,
        "templates/repository-baseline/SUPPORT-AND-COMPATIBILITY.md",
    )
    if root_support != template_support:
        fail("root and baseline support policies differ")
    print("PASS: support policies are synchronized")

    root_security = read(repo_root, "SECURITY.md")
    template_security = read(
        repo_root,
        "templates/repository-baseline/SECURITY.md",
    )
    if root_security != template_security:
        fail("root and baseline security policies differ")
    print("PASS: security policies are synchronized")

    for relative in (
        "docs/engineering/release-and-acceptance-model.md",
        "templates/repository-baseline/docs/engineering/"
        "release-and-acceptance-model.md",
    ):
        content = read(repo_root, relative)
        for marker in RELEASE_MARKERS:
            require_marker(content, marker, relative)
    print("PASS: release models enforce exact-boundary convergence")

    v1_finalization = read(
        repo_root,
        "docs/acceptance/isras-v1.0.0-release-finalization.md",
    )
    for marker in (
        "**Status: COMPLETE**",
        "isras-v1.0.0",
        "3f7d4e7f5b340c65cfe74f757ba0a24b2f94cc2b",
        "f9655ddbbf04430fc468aab405f2ed880df3e97d",
    ):
        require_marker(v1_finalization, marker, "v1.0.0 release-finalization record")
    print("PASS: v1.0.0 release finalization record remains complete")

    candidate_acceptance = read(
        repo_root,
        "docs/acceptance/isras-v2.0.0-candidate-acceptance.md",
    )
    for marker in (
        "Accepted for release finalization",
        CANDIDATE_COMMIT,
        EVIDENCE_COMMIT,
        "40 PASS and 0 FAIL",
    ):
        require_marker(
            candidate_acceptance,
            marker,
            "v2.0.0 candidate-acceptance record",
        )
    print("PASS: v2.0.0 candidate acceptance remains exact")

    finalization = read(
        repo_root,
        "docs/acceptance/isras-v2.0.0-release-finalization.md",
    )
    for marker in (
        "AUTHORIZED — COMPLETION REQUIRES SIGNED TAG AND BRANCH CONVERGENCE",
        "isras-v2.0.0",
        CANDIDATE_COMMIT,
        EVIDENCE_COMMIT,
        ACCEPTANCE_COMMIT,
        "remote `dev`",
        "remote `main`",
    ):
        require_marker(finalization, marker, "v2.0.0 release-finalization record")
    print("PASS: v2.0.0 release finalization boundary is predeclared")

    completion = read(
        repo_root,
        "docs/acceptance/isras-v2.0.0-release-completion.md",
    )
    for marker in (
        "**Status: COMPLETE**",
        RELEASE_TAG,
        RELEASE_COMMIT,
        TAG_OBJECT,
        MANIFEST_SHA,
        EVIDENCE_SHA,
        SIGNING_FINGERPRINT,
        "45 PASS",
        "29 tests",
        "remote `dev`",
        "remote `main`",
    ):
        require_marker(completion, marker, "v2.0.0 release-completion record")
    print("PASS: v2.0.0 signed release completion is recorded exactly")

    plan = read(repo_root, "docs/acceptance/isras-v2.0.0-plan.md")
    for marker in (
        "RELEASE COMPLETE — IMMUTABLE CHECKPOINT REGISTERED",
        "isras-v2.0.0",
        CANDIDATE_COMMIT,
        EVIDENCE_COMMIT,
        ACCEPTANCE_COMMIT,
    ):
        require_marker(plan, marker, "v2.0.0 acceptance plan")
    print("PASS: v2.0.0 exact-commit finalization plan is synchronized")

    checkpoint_registry = json.loads(
        read(repo_root, "tools/validation/checkpoints.json")
    )
    checkpoint = checkpoint_registry.get("checkpoints", {}).get(
        RELEASE_TAG,
        {},
    )
    expected_checkpoint = {
        "commit": RELEASE_COMMIT,
        "environment_profile": "portable",
        "expected_result": {"fail": 0},
        "gate": "tools/validation/phase-gates/validate_isras_v2_release.sh",
        "required_branch_name": "dev",
        "status": "accepted",
        "tag": RELEASE_TAG,
    }
    if checkpoint != expected_checkpoint:
        fail("isras-v2.0.0 checkpoint registration is not exact")
    print("PASS: isras-v2.0.0 checkpoint registration is exact")

    patch_candidate = read(
        repo_root,
        "docs/acceptance/isras-v2.0.1-plan.md",
    )
    for marker in (
        "CANDIDATE PREPARATION — NOT FORMALLY ACCEPTED",
        "2.0.1",
        "isras-v2.0.1",
        RELEASE_COMMIT,
        "a1861291110efccaad9c587a99aaaf2de6f21812",
        "5c07b428b206e4f4e5d7e33d6f5811d7d4e6e739",
        "Current repository `VERSION` during candidate preparation: `2.0.0`",
    ):
        require_marker(
            patch_candidate,
            marker,
            "v2.0.1 candidate and acceptance plan",
        )
    print("PASS: v2.0.1 candidate preparation is synchronized")

    rulesets = read(repo_root, "docs/engineering/github-release-rulesets.md")
    require_marker(rulesets, "isras-*", "GitHub release-ruleset requirements")
    print("PASS: isras-* tag namespace protection is documented")

    changelog = read(repo_root, "CHANGELOG.md")
    require_marker(
        changelog,
        "## 2.0.0 — Governance and bounded authority — 2026-07-16",
        "CHANGELOG",
    )
    print("PASS: v2.0.0 release notes exist")

    license_text = read(repo_root, "LICENSE")
    for marker in (
        "BSD 3-Clause License",
        "Copyright (c) 2026, Iron Signal Systems",
        "Redistribution and use in source and binary forms",
        "Neither the name of the copyright holder",
        'THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"',
    ):
        require_marker(license_text, marker, "LICENSE")

    licensing = read(repo_root, "LICENSING.md")
    for marker in (
        "BSD 3-Clause",
        "BSD-3-Clause",
        "5c07b428b206e4f4e5d7e33d6f5811d7d4e6e739",
        "781246e69f8a9a382c25040f94b62dfe3b25ba89",
        "does not modify, replace, retag, or rewrite",
    ):
        require_marker(licensing, marker, "LICENSING.md")

    print("PASS: BSD-3-Clause licensing decision is explicit")

    print("\nRelease-state validation PASSED.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
