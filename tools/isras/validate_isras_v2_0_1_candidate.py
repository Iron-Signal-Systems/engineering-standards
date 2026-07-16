#!/usr/bin/env python3
# Validate the ISRAS v2.0.1 BSD-licensed patch candidate boundary.
from __future__ import annotations

import argparse
import json
import subprocess
from pathlib import Path

EXPECTED_VERSION = "2.0.1"
TARGET_VERSION = "2.0.1"
CANDIDATE_COMMIT = "6543a5a93f078f47d87aa3b8ed8ebd2024cec373"
EVIDENCE_COMMIT = "9dbe4d9696ff4a9838fd83cb0f6f652087710f98"
ACCEPTANCE_COMMIT = "57d23742e60d29bf6f46d15b8f64f0497bb260cd"
ACCEPTED_RELEASE_COMMIT = "781246e69f8a9a382c25040f94b62dfe3b25ba89"
CHECKPOINT_COMMIT = "a1861291110efccaad9c587a99aaaf2de6f21812"
BSD_BOUNDARY_COMMIT = "5c07b428b206e4f4e5d7e33d6f5811d7d4e6e739"
RELEASE_TAG = "isras-v2.0.0"
TAG_OBJECT = "a7a09a02798e2b2c905f2686820fd30890f62bc6"
SIGNING_FINGERPRINT = "SHA256:FiH+Jk7HHrNkvDEQTehI/aCfkmKpivtsqmkl5TmmMSE"

UNCHANGED_FROM_V2_RELEASE = (
    "standards/repository-assurance/v1",
    "standards/repository-assurance/v2",
    "schemas",
    "templates",
    "integration-guides",
    ".github/workflows",
)

REQUIRED_FILES = (
    "LICENSE",
    "LICENSING.md",
    "README.md",
    "CHANGELOG.md",
    "docs/acceptance/isras-v2.0.0-release-completion.md",
    "docs/acceptance/isras-v2.0.1-plan.md",
    "docs/acceptance/isras-v2.0.1-candidate-acceptance.md",
    "docs/acceptance/isras-v2.0.1-release-finalization.md",
    "tools/isras/validate_isras_v2_0_1_release.py",
    "tools/validation/phase-gates/"
    "validate_isras_v2_0_1_release.sh",
    "tools/isras/validate_isras_v2_0_1_candidate.py",
    "tools/validation/phase-gates/validate_isras_v2_0_1_candidate.sh",
)


class Results:
    def __init__(self) -> None:
        self.passes: list[str] = []
        self.failures: list[str] = []

    def check(self, condition: bool, message: str) -> None:
        (self.passes if condition else self.failures).append(message)

    def report(self) -> int:
        for message in self.passes:
            print(f"PASS: {message}")
        for message in self.failures:
            print(f"FAIL: {message}")
        print(f"PASS checks: {len(self.passes)}")
        print(f"FAIL checks: {len(self.failures)}")
        if self.failures:
            print("ISRAS v2.0.1 candidate validation FAILED.")
            return 1
        print("ISRAS v2.0.1 candidate validation PASSED.")
        return 0


def read(root: Path, relative: str) -> str:
    path = root / relative
    return path.read_text(encoding="utf-8") if path.is_file() else ""


def normalized(value: str) -> str:
    return " ".join(value.split())


def git(root: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args],
        cwd=root,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", type=Path, default=Path.cwd())
    parser.add_argument("--skip-git-diff", action="store_true")
    args = parser.parse_args()

    root = args.repo_root.resolve()
    results = Results()

    for relative in REQUIRED_FILES:
        results.check(
            (root / relative).is_file(),
            f"required v2.0.1 candidate file exists: {relative}",
        )

    results.check(
        read(root, "VERSION").strip() == EXPECTED_VERSION,
        "VERSION declares 2.0.1 for release-source finalization",
    )

    license_text = normalized(read(root, "LICENSE"))
    for marker in (
        "BSD 3-Clause License",
        "Copyright (c) 2026, Iron Signal Systems",
        "Redistribution and use in source and binary forms",
        "Neither the name of the copyright holder",
        'THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"',
    ):
        results.check(
            normalized(marker) in license_text,
            f"LICENSE contains required BSD-3-Clause marker {marker!r}",
        )

    licensing = normalized(read(root, "LICENSING.md"))
    for marker in (
        "BSD-3-Clause",
        BSD_BOUNDARY_COMMIT,
        ACCEPTED_RELEASE_COMMIT,
        "does not modify, replace, retag, or rewrite",
    ):
        results.check(
            normalized(marker) in licensing,
            f"LICENSING.md contains required boundary marker {marker!r}",
        )

    plan_text = read(root, "docs/acceptance/isras-v2.0.1-plan.md")
    plan = normalized(plan_text)
    for marker in (
        "RELEASE SOURCE PREPARED — SIGNED TAG AND BRANCH CONVERGENCE PENDING",
        TARGET_VERSION,
        "isras-v2.0.1",
        ACCEPTED_RELEASE_COMMIT,
        CHECKPOINT_COMMIT,
        BSD_BOUNDARY_COMMIT,
        "root `VERSION` remains `2.0.0`",
        CANDIDATE_COMMIT,
        EVIDENCE_COMMIT,
        ACCEPTANCE_COMMIT,
        "Release-source `VERSION`: `2.0.1`",
    ):
        results.check(
            normalized(marker) in plan,
            f"v2.0.1 candidate plan contains required marker {marker!r}",
        )
    for placeholder in ("<CANDIDATE_COMMIT>", "TBD", "TO_BE_DETERMINED"):
        results.check(
            placeholder not in plan_text,
            f"v2.0.1 candidate plan contains no {placeholder!r} placeholder",
        )

    completion = normalized(
        read(root, "docs/acceptance/isras-v2.0.0-release-completion.md")
    )
    for marker in (
        "**Status: COMPLETE**",
        ACCEPTED_RELEASE_COMMIT,
        TAG_OBJECT,
        SIGNING_FINGERPRINT,
    ):
        results.check(
            normalized(marker) in completion,
            f"v2.0.0 release completion retains marker {marker!r}",
        )

    checkpoint_data = json.loads(
        read(root, "tools/validation/checkpoints.json")
    )
    checkpoint = checkpoint_data.get("checkpoints", {}).get(RELEASE_TAG, {})
    results.check(
        checkpoint
        == {
            "commit": ACCEPTED_RELEASE_COMMIT,
            "environment_profile": "portable",
            "expected_result": {"fail": 0},
            "gate": "tools/validation/phase-gates/validate_isras_v2_release.sh",
            "required_branch_name": "dev",
            "status": "accepted",
            "tag": RELEASE_TAG,
        },
        "accepted v2.0.0 checkpoint remains exact",
    )

    if not args.skip_git_diff and (root / ".git").exists():
        for commit, label in (
            (ACCEPTED_RELEASE_COMMIT, "accepted v2.0.0 release"),
            (CHECKPOINT_COMMIT, "v2.0.0 checkpoint registration"),
            (BSD_BOUNDARY_COMMIT, "BSD-3-Clause source boundary"),
        ):
            ancestor = git(root, "merge-base", "--is-ancestor", commit, "HEAD")
            results.check(
                ancestor.returncode == 0,
                f"current candidate descends from {label}",
            )

        for relative in UNCHANGED_FROM_V2_RELEASE:
            diff = git(
                root,
                "diff",
                "--quiet",
                ACCEPTED_RELEASE_COMMIT,
                "--",
                relative,
            )
            results.check(
                diff.returncode == 0,
                f"{relative} is unchanged from accepted v2.0.0",
            )

    return results.report()


if __name__ == "__main__":
    raise SystemExit(main())
