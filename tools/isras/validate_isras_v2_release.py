#!/usr/bin/env python3
# Validate the frozen ISRAS v2.0.0 release-source boundary.
from __future__ import annotations

import argparse
import json
import subprocess
import sys
from pathlib import Path

EXPECTED_VERSION = "2.0.0"
CANDIDATE_COMMIT = "4aff00dfdc88154390252898210abc336fa8b2fc"
EVIDENCE_COMMIT = "b0c982221acde7873307d010aca73ed2e386eb99"
ACCEPTANCE_COMMIT = "24e911b7c4a63735bcef9b4b84ab9b62ace10298"
ACCEPTED_V1_COMMIT = "c379417720faa595fa5cb89a1dfdb2259d6cb95e"


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
            print("ISRAS v2.0.0 release-source validation FAILED.")
            return 1
        print("ISRAS v2.0.0 release-source validation PASSED.")
        return 0


def read(root: Path, relative: str) -> str:
    path = root / relative
    return path.read_text(encoding="utf-8") if path.is_file() else ""


def normalized(content: str) -> str:
    return " ".join(content.split())


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", type=Path, default=Path.cwd())
    parser.add_argument("--skip-v1-git-diff", action="store_true")
    args = parser.parse_args()

    root = args.repo_root.resolve()
    results = Results()

    candidate_command = [
        sys.executable,
        str(root / "tools/isras/validate_isras_v2_candidate.py"),
        "--repo-root",
        str(root),
        "--release-finalization",
    ]
    if args.skip_v1_git_diff:
        candidate_command.append("--skip-v1-git-diff")
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
        "underlying ISRAS v2 release-mode candidate boundary passes",
    )

    results.check(
        read(root, "VERSION").strip() == EXPECTED_VERSION,
        "VERSION declares 2.0.0",
    )

    required = (
        "README.md",
        "CHANGELOG.md",
        "SUPPORT-AND-COMPATIBILITY.md",
        "templates/repository-baseline/SUPPORT-AND-COMPATIBILITY.md",
        "docs/acceptance/isras-v2.0.0-plan.md",
        "docs/acceptance/isras-v2.0.0-candidate-acceptance.md",
        "docs/acceptance/isras-v2.0.0-release-finalization.md",
        "standards/repository-assurance/v2/RELEASE-VERSIONING-SUPPORT-AND-DEPRECATION.md",
        "tools/validation/phase-gates/validate_isras_v2_release.sh",
    )
    for relative in required:
        results.check(
            (root / relative).is_file(),
            f"required v2 release-source file exists: {relative}",
        )

    root_support = read(root, "SUPPORT-AND-COMPATIBILITY.md")
    template_support = read(
        root,
        "templates/repository-baseline/SUPPORT-AND-COMPATIBILITY.md",
    )
    results.check(
        root_support == template_support,
        "root and baseline support policies are synchronized",
    )

    markers = {
        "README.md": (
            "current normative standard",
            "ISRAS v2",
            "isras-v2.0.0",
            "does not silently change",
        ),
        "CHANGELOG.md": (
            "## 2.0.0 — Governance and bounded authority — 2026-07-16",
            "bounded-authority",
            "isras-v2.0.0",
        ),
        "SUPPORT-AND-COMPATIBILITY.md": (
            "ISRAS v2.0.x",
            "ISRAS v1.0.1 remains",
            "Engineering Standards Impact Assessment",
        ),
        "docs/acceptance/isras-v2.0.0-plan.md": (
            "RELEASE SOURCE PREPARED FOR EXACT-COMMIT FINALIZATION",
            CANDIDATE_COMMIT,
            EVIDENCE_COMMIT,
            ACCEPTANCE_COMMIT,
        ),
        "docs/acceptance/isras-v2.0.0-candidate-acceptance.md": (
            "Accepted for release finalization",
            CANDIDATE_COMMIT,
            EVIDENCE_COMMIT,
            ACCEPTED_V1_COMMIT,
        ),
        "docs/acceptance/isras-v2.0.0-release-finalization.md": (
            "AUTHORIZED — COMPLETION REQUIRES SIGNED TAG AND BRANCH CONVERGENCE",
            "isras-v2.0.0",
            CANDIDATE_COMMIT,
            EVIDENCE_COMMIT,
            ACCEPTANCE_COMMIT,
            "cannot contain its own final object identity",
        ),
        "standards/repository-assurance/v2/RELEASE-VERSIONING-SUPPORT-AND-DEPRECATION.md": (
            "Release-source boundary",
            "isras-v2.0.0",
            CANDIDATE_COMMIT,
            EVIDENCE_COMMIT,
            ACCEPTANCE_COMMIT,
        ),
    }
    for relative, required_markers in markers.items():
        content = normalized(read(root, relative))
        for marker in required_markers:
            results.check(
                normalized(marker) in content,
                f"{relative} contains required release marker {marker!r}",
            )

    finalization = read(
        root,
        "docs/acceptance/isras-v2.0.0-release-finalization.md",
    )
    for marker in ("<RELEASE_COMMIT>", "TBD", "TO_BE_DETERMINED"):
        results.check(
            marker not in finalization,
            f"release-finalization record contains no {marker!r} placeholder",
        )

    checkpoint_data = json.loads(
        read(root, "tools/validation/checkpoints.json")
    )
    v1 = checkpoint_data.get("checkpoints", {}).get("isras-v1.0.1", {})
    results.check(
        v1.get("commit") == ACCEPTED_V1_COMMIT
        and v1.get("status") == "accepted"
        and v1.get("tag") == "isras-v1.0.1",
        "accepted v1.0.1 checkpoint remains exact",
    )

    return results.report()


if __name__ == "__main__":
    raise SystemExit(main())
