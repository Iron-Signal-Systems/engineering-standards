#!/usr/bin/env python3
# Validate the frozen ISRAS v2.0.1 release-source boundary.
from __future__ import annotations

import argparse
import json
import subprocess
import sys
from pathlib import Path

EXPECTED_VERSION = "2.0.1"
CANDIDATE_COMMIT = "6543a5a93f078f47d87aa3b8ed8ebd2024cec373"
EVIDENCE_COMMIT = "9dbe4d9696ff4a9838fd83cb0f6f652087710f98"
ACCEPTANCE_COMMIT = "57d23742e60d29bf6f46d15b8f64f0497bb260cd"
ACCEPTED_PREDECESSOR = "781246e69f8a9a382c25040f94b62dfe3b25ba89"
CHECKPOINT_COMMIT = "a1861291110efccaad9c587a99aaaf2de6f21812"
BSD_BOUNDARY_COMMIT = "5c07b428b206e4f4e5d7e33d6f5811d7d4e6e739"
EVIDENCE_SHA = "42d7dce7500929647af001f47bbbdf30ae7bef88c598d0aba8edd2424564d2b9"
CANDIDATE_MANIFEST_SHA = "e2b6488a7f670b0c81d873478154d03438a9c5f21a8bf05010863fbe1e4fd7e8"


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
            print("ISRAS v2.0.1 release-source validation FAILED.")
            return 1
        print("ISRAS v2.0.1 release-source validation PASSED.")
        return 0


def read(root: Path, relative: str) -> str:
    path = root / relative
    return path.read_text(encoding="utf-8") if path.is_file() else ""


def normalized(content: str) -> str:
    return " ".join(content.split())


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
    parser.add_argument("--skip-git-lineage", action="store_true")
    args = parser.parse_args()

    root = args.repo_root.resolve()
    results = Results()

    candidate_command = [
        sys.executable,
        str(root / "tools/isras/validate_isras_v2_0_1_candidate.py"),
        "--repo-root",
        str(root),
    ]
    if args.skip_git_lineage:
        candidate_command.append("--skip-git-diff")
    candidate = subprocess.run(
        candidate_command,
        cwd=root,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if candidate.stdout:
        print(candidate.stdout, end="")
    if candidate.stderr:
        print(candidate.stderr, end="", file=sys.stderr)
    results.check(
        candidate.returncode == 0,
        "underlying v2.0.1 accepted-candidate lineage passes",
    )

    results.check(
        read(root, "VERSION").strip() == EXPECTED_VERSION,
        "VERSION declares 2.0.1",
    )

    required = (
        "LICENSE",
        "LICENSING.md",
        "README.md",
        "CHANGELOG.md",
        "docs/acceptance/isras-v2.0.1-plan.md",
        "docs/acceptance/isras-v2.0.1-candidate-acceptance.md",
        "docs/acceptance/isras-v2.0.1-release-finalization.md",
        "docs/acceptance/evidence/isras-v2.0.1-candidate/"
        "acceptance-evidence.json",
        "tools/isras/validate_isras_v2_0_1_release.py",
        "tools/validation/phase-gates/"
        "validate_isras_v2_0_1_release.sh",
    )
    for relative in required:
        results.check(
            (root / relative).is_file(),
            f"required v2.0.1 release-source file exists: {relative}",
        )

    markers = {
        "README.md": (
            "Release-source finalization",
            "ISRAS v2.0.1",
            "isras-v2.0.1",
            "latest accepted release remains `isras-v2.0.0`",
        ),
        "CHANGELOG.md": (
            "## 2.0.1 — BSD-licensed patch release — 2026-07-16",
            "BSD-3-Clause",
            CANDIDATE_COMMIT,
            EVIDENCE_COMMIT,
            ACCEPTANCE_COMMIT,
        ),
        "docs/acceptance/isras-v2.0.1-plan.md": (
            "RELEASE SOURCE PREPARED — SIGNED TAG AND BRANCH CONVERGENCE PENDING",
            CANDIDATE_COMMIT,
            EVIDENCE_COMMIT,
            ACCEPTANCE_COMMIT,
            "Release-source `VERSION`: `2.0.1`",
        ),
        "docs/acceptance/isras-v2.0.1-candidate-acceptance.md": (
            "Accepted for release finalization",
            CANDIDATE_COMMIT,
            EVIDENCE_COMMIT,
            EVIDENCE_SHA,
            CANDIDATE_MANIFEST_SHA,
            "43 PASS and 0 FAIL",
        ),
        "docs/acceptance/isras-v2.0.1-release-finalization.md": (
            "AUTHORIZED — COMPLETION REQUIRES SIGNED TAG AND BRANCH CONVERGENCE",
            "isras-v2.0.1",
            CANDIDATE_COMMIT,
            EVIDENCE_COMMIT,
            ACCEPTANCE_COMMIT,
            ACCEPTED_PREDECESSOR,
            BSD_BOUNDARY_COMMIT,
            "cannot contain its own final object identity",
        ),
    }
    for relative, required_markers in markers.items():
        content = normalized(read(root, relative))
        for marker in required_markers:
            results.check(
                normalized(marker) in content,
                f"{relative} contains required release marker {marker!r}",
            )

    finalization_text = read(
        root,
        "docs/acceptance/isras-v2.0.1-release-finalization.md",
    )
    for marker in (
        "<RELEASE_COMMIT>",
        "TBD",
        "TO_BE_DETERMINED",
    ):
        results.check(
            marker not in finalization_text,
            f"release-finalization record contains no {marker!r} placeholder",
        )

    evidence = json.loads(
        read(
            root,
            "docs/acceptance/evidence/isras-v2.0.1-candidate/"
            "acceptance-evidence.json",
        )
    )
    results.check(
        evidence.get("source_commit") == CANDIDATE_COMMIT
        and evidence.get("correctness_result") == "PASS"
        and evidence.get("acceptance_tag") is None,
        "candidate evidence remains exact and non-release-claiming",
    )

    checkpoint_data = json.loads(
        read(root, "tools/validation/checkpoints.json")
    )
    predecessor = checkpoint_data.get("checkpoints", {}).get(
        "isras-v2.0.0",
        {},
    )
    results.check(
        predecessor
        == {
            "commit": ACCEPTED_PREDECESSOR,
            "environment_profile": "portable",
            "expected_result": {"fail": 0},
            "gate": "tools/validation/phase-gates/"
            "validate_isras_v2_release.sh",
            "required_branch_name": "dev",
            "status": "accepted",
            "tag": "isras-v2.0.0",
        },
        "accepted v2.0.0 checkpoint remains exact",
    )

    if not args.skip_git_lineage and (root / ".git").exists():
        for commit, label in (
            (CANDIDATE_COMMIT, "accepted v2.0.1 candidate source"),
            (EVIDENCE_COMMIT, "v2.0.1 candidate evidence"),
            (ACCEPTANCE_COMMIT, "v2.0.1 formal candidate acceptance"),
            (ACCEPTED_PREDECESSOR, "accepted v2.0.0 predecessor"),
            (CHECKPOINT_COMMIT, "v2.0.0 checkpoint registration"),
            (BSD_BOUNDARY_COMMIT, "BSD-3-Clause source boundary"),
        ):
            ancestor = git(root, "merge-base", "--is-ancestor", commit, "HEAD")
            results.check(
                ancestor.returncode == 0,
                f"release source descends from {label}",
            )

    return results.report()


if __name__ == "__main__":
    raise SystemExit(main())
