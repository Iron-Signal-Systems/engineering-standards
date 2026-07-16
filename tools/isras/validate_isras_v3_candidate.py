#!/usr/bin/env python3
"""Validate the development-only ISRAS v3 assurance-hardening candidate."""
from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator, FormatChecker

BASE_COMMIT = "08a0a514ec308f76dbf80ffdcb8caa70ce6e345f"
ACCEPTED_V2_COMMIT = "d34fad82781a4e8485f8907fbfd34f236fa79ad2"
ACCEPTED_V2_TAG = "isras-v2.0.1"
ACCEPTED_V2_MANIFEST = "8f54ed1e9bfee251bf89b4c5f12edf11ac1e25ef0d145ba745301f2d05787ef1"
REQUIRED_CONTROLS = {
    "ISRAS-BST-001", "ISRAS-BST-002", "ISRAS-BST-003", "ISRAS-BST-004",
    "ISRAS-DIG-001", "ISRAS-EVD-005", "ISRAS-EVD-006", "ISRAS-EVD-007",
    "ISRAS-EVD-008", "ISRAS-EVD-009", "ISRAS-MAP-001", "ISRAS-MAP-002",
    "ISRAS-GOV-008", "ISRAS-SCM-001", "ISRAS-CHG-001", "ISRAS-CHG-002",
    "ISRAS-CHG-003", "ISRAS-CHG-004",
}
REQUIRED_FILES = {
    "SOURCE-SHA512SUMS.txt",
    "docs/acceptance/isras-v3.0.0-plan.md",
    "docs/acceptance/isras-v3.0.0-change-classification.json",
    "docs/engineering/deterministic-tool-bootstrap.md",
    "docs/engineering/external-standards-crosswalk.md",
    "docs/engineering/external-standards-crosswalk.json",
    "docs/engineering/github-control-evidence.md",
    "docs/engineering/proportional-change-governance.md",
    "docs/engineering/repository-self-assurance.md",
    "standards/repository-assurance/v3/INDEX.md",
    "standards/repository-assurance/v3/STANDARD.md",
    "standards/repository-assurance/v3/CONTROL-CATALOG.md",
    "schemas/evidence-binding-v1.schema.json",
    "schemas/change-classification-v1.schema.json",
    "schemas/external-standards-crosswalk-v1.schema.json",
    "schemas/github-control-evidence-v1.schema.json",
    "schemas/tool-bootstrap-lock-v1.schema.json",
    "schemas/tool-environment-record-v1.schema.json",
    "tools/isras/generate_sha512_manifest.py",
    "tools/isras/verify_sha512_manifest.py",
    "tools/isras/validate_evidence_relationships.py",
    "tools/isras/validate_change_classification.py",
    "tools/isras/validate_external_standards_crosswalk.py",
    "tools/isras/validate_github_control_evidence.py",
    "tools/github/export_ruleset_evidence.py",
    "tools/environment/prepare_tool_wheelhouse.py",
    "tools/environment/verify_wheelhouse.py",
    "tools/environment/clean_tool_venv.py",
    "tools/environment/record_tool_environment.py",
    "tools/environment/bootstrap_tools_release.sh",
    "tools/environment/Bootstrap-Tools-Release.ps1",
}


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
            print("ISRAS v3 candidate validation FAILED.")
            return 1
        print("ISRAS v3 candidate validation PASSED.")
        return 0


def git(root: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args], cwd=root, text=True,
        stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False,
    )


def load(path: Path) -> Any:
    return json.loads(path.read_text(encoding="utf-8"))


def schema_check(root: Path, instance_path: str, schema_path: str, results: Results) -> None:
    schema = load(root / schema_path)
    try:
        Draft202012Validator.check_schema(schema)
    except Exception as exc:  # jsonschema exposes several schema error classes
        results.check(False, f"schema is valid Draft 2020-12: {schema_path}: {exc}")
        return
    results.check(True, f"schema is valid Draft 2020-12: {schema_path}")
    errors = list(Draft202012Validator(schema, format_checker=FormatChecker()).iter_errors(load(root / instance_path)))
    results.check(
        not errors,
        f"record conforms structurally: {instance_path}"
        + ("" if not errors else f": {errors[0].message}"),
    )


def run_tool(root: Path, relative: str, *arguments: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [sys.executable, str(root / relative), "--repo-root", str(root), *arguments],
        cwd=root, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False,
    )


def tool_result(results: Results, process: subprocess.CompletedProcess[str], message: str) -> None:
    detail = "" if process.returncode == 0 else f": {process.stdout}{process.stderr}"
    results.check(process.returncode == 0, message + detail)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", type=Path, default=Path.cwd())
    parser.add_argument("--skip-git-diff", action="store_true", help="Unit-test-only escape hatch.")
    args = parser.parse_args()

    root = args.repo_root.resolve()
    results = Results()

    for relative in sorted(REQUIRED_FILES):
        results.check((root / relative).is_file(), f"required v3 candidate file exists: {relative}")

    results.check((root / "VERSION").read_text(encoding="utf-8").strip() == "2.0.1", "VERSION remains at accepted v2.0.1")

    if not args.skip_git_diff:
        results.check(git(root, "cat-file", "-e", f"{BASE_COMMIT}^{{commit}}").returncode == 0, "v3 base commit resolves")
        results.check(git(root, "merge-base", "--is-ancestor", BASE_COMMIT, "HEAD").returncode == 0, "v3 base commit is an ancestor of HEAD")
        results.check(git(root, "diff", "--quiet").returncode == 0, "no unstaged source drift exists")
        for tree in ("standards/repository-assurance/v1", "standards/repository-assurance/v2"):
            results.check(git(root, "diff", "--quiet", BASE_COMMIT, "--", tree).returncode == 0, f"accepted normative tree is unchanged from base: {tree}")

    assurance = load(root / "REPOSITORY-ASSURANCE.json")
    standard = assurance.get("standard", {})
    self_assurance = assurance.get("self_assurance", {})
    exact_standard = {
        "version": "2.0.1",
        "tag": ACCEPTED_V2_TAG,
        "commit": ACCEPTED_V2_COMMIT,
        "source_manifest_sha256": ACCEPTED_V2_MANIFEST,
    }
    for field, expected in exact_standard.items():
        results.check(standard.get(field) == expected, f"self-governing standard {field} is exact")
    results.check(assurance.get("adoption_level") == "RELEASE_ASSURED", "repository adoption level reflects accepted release boundary")
    results.check(self_assurance.get("mode") == "PINNED_ACCEPTED_RELEASE", "repository self-assurance mode is explicit")
    results.check(self_assurance.get("governing_release") == standard.get("tag"), "self-assurance release matches standard tag")
    results.check(self_assurance.get("governing_commit") == standard.get("commit"), "self-assurance commit matches standard commit")
    results.check(self_assurance.get("governing_source_manifest_sha256") == standard.get("source_manifest_sha256"), "self-assurance manifest matches standard manifest")
    record_path = root / str(self_assurance.get("record", ""))
    results.check(record_path.is_file(), "self-assurance explanatory record exists")

    catalog = (root / "standards/repository-assurance/v3/CONTROL-CATALOG.md").read_text(encoding="utf-8")
    identifiers = re.findall(r"ISRAS-[A-Z]{2,4}-[0-9]{3}", catalog)
    results.check(REQUIRED_CONTROLS.issubset(set(identifiers)), "all required v3 controls are cataloged")
    results.check(len(identifiers) == len(set(identifiers)), "v3 candidate control identifiers are unique")

    schema_templates = {
        "templates/engineering-standards/evidence-binding.json": "schemas/evidence-binding-v1.schema.json",
        "templates/engineering-standards/change-classification.json": "schemas/change-classification-v1.schema.json",
        "templates/engineering-standards/github-control-evidence.json": "schemas/github-control-evidence-v1.schema.json",
        "templates/engineering-standards/tool-bootstrap-lock.json": "schemas/tool-bootstrap-lock-v1.schema.json",
        "docs/acceptance/isras-v3.0.0-change-classification.json": "schemas/change-classification-v1.schema.json",
        "docs/engineering/external-standards-crosswalk.json": "schemas/external-standards-crosswalk-v1.schema.json",
        "REPOSITORY-ASSURANCE.json": "schemas/repository-assurance-v1.schema.json",
    }
    for instance, schema in schema_templates.items():
        schema_check(root, instance, schema, results)
    for schema_path in sorted((root / "schemas").glob("*-v1.schema.json")):
        try:
            Draft202012Validator.check_schema(load(schema_path))
        except Exception as exc:
            results.check(False, f"schema is valid Draft 2020-12: {schema_path.relative_to(root)}: {exc}")
        else:
            results.check(True, f"schema is valid Draft 2020-12: {schema_path.relative_to(root)}")

    all_bootstrap = "\n".join((root / relative).read_text(encoding="utf-8") for relative in (
        "tools/environment/bootstrap_tools.sh",
        "tools/environment/Bootstrap-Tools.ps1",
        "tools/environment/bootstrap_tools_release.sh",
        "tools/environment/Bootstrap-Tools-Release.ps1",
    ))
    results.check("pip install --upgrade pip" not in all_bootstrap and "-m pip install --upgrade pip" not in all_bootstrap, "bootstrap performs no implicit unpinned pip self-upgrade")

    release_shell = (root / "tools/environment/bootstrap_tools_release.sh").read_text(encoding="utf-8")
    for marker in ("release tool environment path already exists", "-I", "--isolated", "--no-index", "--no-cache-dir", "--only-binary=:all:", "--require-hashes", "--force-reinstall", "verify_wheelhouse.py", "clean_tool_venv.py", "record_tool_environment.py"):
        results.check(marker in release_shell, f"release bootstrap contains required marker {marker!r}")
    preparer = (root / "tools/environment/prepare_tool_wheelhouse.py").read_text(encoding="utf-8")
    results.check("wheelhouse output directory must be empty" in preparer, "wheelhouse preparation refuses stale output")
    results.check("resolution-report.json" not in preparer, "transient resolver reports are excluded from wheelhouse")
    results.check("source_hashes" in preparer and "source_url" in preparer, "wheel provenance is retained")

    verifier = (root / "tools/environment/verify_wheelhouse.py").read_text(encoding="utf-8")
    results.check(all(name not in verifier for name in ("jsonschema", "yaml", "requests")), "pre-bootstrap wheelhouse verification has no third-party dependency")
    results.check("python_executable_sha512" in verifier and "requirements_source_sha512" in verifier, "pre-bootstrap verifier binds Python and repository requirements")

    tool_result(results, run_tool(root, "tools/isras/verify_sha512_manifest.py"), "Git-index SHA-512 manifest verifies")
    tool_result(results, run_tool(root, "tools/isras/validate_change_classification.py", "--record", "templates/engineering-standards/change-classification.json", "--skip-diff-floor"), "change-classification template passes semantic validation")
    tool_result(results, run_tool(root, "tools/isras/validate_change_classification.py", "--record", "docs/acceptance/isras-v3.0.0-change-classification.json"), "actual v3 candidate classification passes changed-path validation")
    tool_result(results, run_tool(root, "tools/isras/validate_external_standards_crosswalk.py", "--record", "docs/engineering/external-standards-crosswalk.json"), "control-level external crosswalk passes candidate validation")

    crosswalk_doc = (root / "docs/engineering/external-standards-crosswalk.md").read_text(encoding="utf-8")
    for marker in ("NIST SSDF", "NIST CSF 2.0", "NIST SP 800-53", "SLSA", "OpenSSF Scorecard", "OWASP SAMM", "OWASP ASVS", "CIS Software Supply Chain Security", "CJIS Security Policy", "does not claim certification"):
        results.check(marker in crosswalk_doc, f"external crosswalk identifies {marker}")

    return results.report()


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
