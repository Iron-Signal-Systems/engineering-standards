#!/usr/bin/env python3
"""Validate central ISRAS release-state and documentation consistency."""

from __future__ import annotations

import argparse
import sys
from pathlib import Path


EXPECTED_VERSION = "1.0.1"

REQUIRED_FILES = (
    "docs/acceptance/isras-v1.0.0-release-finalization.md",
    "docs/acceptance/isras-v1.0.1-plan.md",
    "docs/engineering/adopter-quick-start.md",
    "docs/engineering/github-release-rulesets.md",
)

STALE_PHRASES = (
    "ISRAS v1.0.0 remains an acceptance candidate",
    "Until the first formal v1 acceptance",
    "No response-time promise is made until",
    "Authorized for controlled replacement; remote replacement pending",
    "main remains blocked from promotion",
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
    """Collapse formatting whitespace for prose-marker comparisons."""
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

    selected_files = (
        "README.md",
        "SUPPORT-AND-COMPATIBILITY.md",
        "SECURITY.md",
        "templates/repository-baseline/SUPPORT-AND-COMPATIBILITY.md",
        "templates/repository-baseline/SECURITY.md",
        "docs/acceptance/isras-v1.0.0-tag-correction.md",
    )

    for relative in selected_files:
        content = read(repo_root, relative)
        normalized_content = normalize(content)
        for phrase in STALE_PHRASES:
            if normalize(phrase) in normalized_content:
                fail(f"stale release wording in {relative}: {phrase!r}")
    print("PASS: stale release-state wording is absent")

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

    correction = read(
        repo_root,
        "docs/acceptance/isras-v1.0.0-tag-correction.md",
    )
    for marker in (
        "Controlled replacement completed and verified",
        "3f7d4e7f5b340c65cfe74f757ba0a24b2f94cc2b",
        "f9655ddbbf04430fc468aab405f2ed880df3e97d",
        "Signature verification: `PASS`",
    ):
        require_marker(correction, marker, "v1.0.0 tag-correction record")
    print("PASS: v1.0.0 tag correction is recorded as complete")

    finalization = read(
        repo_root,
        "docs/acceptance/isras-v1.0.0-release-finalization.md",
    )
    for marker in (
        "**Status: COMPLETE**",
        "isras-v1.0.0",
        "3f7d4e7f5b340c65cfe74f757ba0a24b2f94cc2b",
        "f9655ddbbf04430fc468aab405f2ed880df3e97d",
    ):
        require_marker(finalization, marker, "v1.0.0 release-finalization record")
    print("PASS: v1.0.0 release finalization record is complete")

    plan = read(repo_root, "docs/acceptance/isras-v1.0.1-plan.md")
    for marker in (
        "Candidate plan",
        "isras-v1.0.1",
        "authoritative acceptance-decision object",
        "No later source commit is required",
    ):
        require_marker(plan, marker, "v1.0.1 acceptance plan")
    print("PASS: v1.0.1 acceptance plan is predeclared")

    rulesets = read(repo_root, "docs/engineering/github-release-rulesets.md")
    require_marker(rulesets, "isras-*", "GitHub release-ruleset requirements")
    print("PASS: isras-* tag namespace protection is documented")

    changelog = read(repo_root, "CHANGELOG.md")
    require_marker(changelog, "## 1.0.1 — Release hardening", "CHANGELOG")
    print("PASS: v1.0.1 release notes exist")

    licensing = read(repo_root, "LICENSING.md")
    require_marker(licensing, "**All rights reserved.**", "LICENSING.md")
    print("PASS: licensing decision is explicit")

    print("\nRelease-state validation PASSED.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
