#!/usr/bin/env python3
# Validate completeness and boundaries of the ISRAS v2 candidate or release source.
from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

from jsonschema import Draft202012Validator

ACCEPTED_V1_COMMIT = "c379417720faa595fa5cb89a1dfdb2259d6cb95e"
REQUIRED_NEW_CONTROLS = {
    "ISRAS-GOV-004", "ISRAS-GOV-005", "ISRAS-GOV-006", "ISRAS-GOV-007",
    "ISRAS-PHS-001", "ISRAS-PHS-002", "ISRAS-PHS-003",
    "ISRAS-AUT-001", "ISRAS-AUT-002", "ISRAS-AUT-003", "ISRAS-AUT-004",
    "ISRAS-AUT-005", "ISRAS-AUT-006", "ISRAS-AUT-007", "ISRAS-TST-003",
    "ISRAS-EVD-003", "ISRAS-EVD-004",
}
REQUIRED_FILES = {
    "standards/repository-assurance/v2/INDEX.md",
    "standards/repository-assurance/v2/STANDARD.md",
    "standards/repository-assurance/v2/CONTROL-CATALOG.md",
    "standards/repository-assurance/v2/MANDATORY-GOVERNANCE-AND-INHERITANCE.md",
    "standards/repository-assurance/v2/BOUNDED-AUTHORITY-AND-PRIVILEGE-NON-PROPAGATION.md",
    "standards/repository-assurance/v2/ENGINEERING-STANDARDS-IMPACT-ASSESSMENT.md",
    "standards/repository-assurance/v2/PHASE-ENTRY-AND-EXIT-COMPLIANCE.md",
    "standards/repository-assurance/v2/HOSTILE-AUTHORITY-VALIDATION.md",
    "standards/repository-assurance/v2/EVIDENCE-MODEL.md",
    "standards/repository-assurance/v2/VALIDATION-MODEL.md",
    "standards/repository-assurance/v2/RELEASE-VERSIONING-SUPPORT-AND-DEPRECATION.md",
    "standards/repository-assurance/v2/MIGRATION-GUIDE.md",
    "schemas/engineering-standards-impact-assessment-v1.schema.json",
    "schemas/phase-standards-compliance-v1.schema.json",
    "schemas/authority-boundary-record-v1.schema.json",
    "templates/engineering-standards/phase-entry-review.json",
    "templates/engineering-standards/phase-exit-review.json",
    "templates/engineering-standards/impact-assessment.json",
    "templates/engineering-standards/authority-boundary-record.json",
    "tools/isras/validate_engineering_standards_compliance.py",
    "tools/isras/validate_isras_v2_candidate.py",
    "tests/test_engineering_standards_compliance.py",
    "docs/acceptance/isras-v2.0.0-plan.md",
}
RELEASE_ONLY_FILES = {
    "docs/acceptance/isras-v2.0.0-release-finalization.md",
    "tools/isras/validate_isras_v2_release.py",
    "tools/validation/phase-gates/validate_isras_v2_release.sh",
}


class Results:
    def __init__(self) -> None:
        self.ok: list[str] = []
        self.fail: list[str] = []

    def check(self, condition: bool, message: str) -> None:
        (self.ok if condition else self.fail).append(message)

    def report(self, label: str) -> int:
        for item in self.ok:
            print(f"PASS: {item}")
        for item in self.fail:
            print(f"FAIL: {item}")
        print(f"PASS checks: {len(self.ok)}")
        print(f"FAIL checks: {len(self.fail)}")
        if self.fail:
            print(f"{label} validation FAILED.")
            return 1
        print(f"{label} validation PASSED.")
        return 0


def load(path: Path):
    with path.open(encoding="utf-8") as handle:
        return json.load(handle)


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
    parser.add_argument("--skip-v1-git-diff", action="store_true")
    parser.add_argument(
        "--release-finalization",
        action="store_true",
        help="Validate the frozen 2.0.0 release-source boundary.",
    )
    args = parser.parse_args()

    root = args.repo_root.resolve()
    results = Results()
    required_files = set(REQUIRED_FILES)
    if args.release_finalization:
        required_files.update(RELEASE_ONLY_FILES)

    boundary = "release" if args.release_finalization else "candidate"
    for relative in sorted(required_files):
        results.check(
            (root / relative).is_file(),
            f"required {boundary} file exists: {relative}",
        )

    version = (
        (root / "VERSION").read_text(encoding="utf-8").strip()
        if (root / "VERSION").exists()
        else ""
    )
    expected_version = "2.0.0" if args.release_finalization else "1.0.1"
    results.check(
        version == expected_version,
        f"VERSION declares required {boundary} value {expected_version}",
    )

    standard = (
        root / "standards/repository-assurance/v2/STANDARD.md"
    ).read_text(encoding="utf-8")
    authority = (
        root
        / "standards/repository-assurance/v2/"
        "BOUNDED-AUTHORITY-AND-PRIVILEGE-NON-PROPAGATION.md"
    ).read_text(encoding="utf-8")
    results.check(
        "unrestricted execution context" in standard.lower(),
        "normative unrestricted execution context term is present",
    )
    results.check(
        "God Access / God Mode" in standard,
        "explanatory God Access / God Mode phrase accompanies the normative term",
    )
    results.check(
        bool(
            re.search(
                r"shall\s+create an \*\*unrestricted execution context\*\*",
                authority,
                re.IGNORECASE,
            )
        ),
        "bounded-authority invariant is explicit",
    )

    catalog = (
        root / "standards/repository-assurance/v2/CONTROL-CATALOG.md"
    ).read_text(encoding="utf-8")
    identifiers = re.findall(r"ISRAS-[A-Z]{3}-[0-9]{3}", catalog)
    results.check(
        REQUIRED_NEW_CONTROLS.issubset(set(identifiers)),
        "all required v2 controls are cataloged",
    )
    results.check(
        len(identifiers) == len(set(identifiers)),
        "control catalog identifiers are unique",
    )

    schemas = {
        "templates/engineering-standards/impact-assessment.json":
            "schemas/engineering-standards-impact-assessment-v1.schema.json",
        "templates/engineering-standards/phase-entry-review.json":
            "schemas/phase-standards-compliance-v1.schema.json",
        "templates/engineering-standards/phase-exit-review.json":
            "schemas/phase-standards-compliance-v1.schema.json",
        "templates/engineering-standards/authority-boundary-record.json":
            "schemas/authority-boundary-record-v1.schema.json",
    }
    for template, schema_path in schemas.items():
        schema = load(root / schema_path)
        try:
            Draft202012Validator.check_schema(schema)
            valid_schema = True
        except Exception:
            valid_schema = False
        results.check(
            valid_schema,
            f"schema is valid Draft 2020-12: {schema_path}",
        )
        errors = (
            list(Draft202012Validator(schema).iter_errors(load(root / template)))
            if valid_schema
            else [1]
        )
        results.check(
            not errors,
            f"template conforms structurally: {template}",
        )

    checkpoints = (
        load(root / "tools/validation/checkpoints.json")
        if (root / "tools/validation/checkpoints.json").exists()
        else {}
    )
    checkpoint = checkpoints.get("checkpoints", {}).get("isras-v1.0.1", {})
    results.check(
        checkpoint.get("commit") == ACCEPTED_V1_COMMIT
        and checkpoint.get("status") == "accepted"
        and checkpoint.get("tag") == "isras-v1.0.1",
        "accepted v1.0.1 checkpoint remains exact",
    )

    v1 = root / "standards/repository-assurance/v1"
    results.check(
        v1.is_dir()
        and (v1 / "STANDARD.md").is_file()
        and (v1 / "CONTROL-CATALOG.md").is_file(),
        "accepted v1 normative tree remains present",
    )
    if not args.skip_v1_git_diff and (root / ".git").exists():
        diff = git(
            root,
            "diff",
            "--quiet",
            ACCEPTED_V1_COMMIT,
            "--",
            "standards/repository-assurance/v1",
        )
        results.check(
            diff.returncode == 0,
            "accepted v1 normative tree is unchanged from v1.0.1",
        )

    label = (
        "ISRAS v2 release-source"
        if args.release_finalization
        else "ISRAS v2 candidate"
    )
    return results.report(label)


if __name__ == "__main__":
    raise SystemExit(main())
