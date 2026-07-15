#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

from common import ISRASError, executable_files, load_json, print_result, repository_root


REQUIRED_PATHS = (
    "REPOSITORY-ASSURANCE.json",
    "SECURITY.md",
    "CONTRIBUTING.md",
    ".github/CODEOWNERS",
    ".github/pull_request_template.md",
    "docs/engineering/repository-assurance-adoption.md",
    "docs/engineering/secure-development-lifecycle.md",
    "docs/engineering/validation-environment-model.md",
    "docs/engineering/release-and-acceptance-model.md",
    "docs/acceptance/README.md",
    "tools/environment/profiles/portable.json",
    "tools/validation/checkpoints.json",
    "tools/validation/validate_portable.sh",
    "tools/validation/Validate-Portable.ps1",
    "tools/validation/validate_fresh_clone.sh",
    "tools/validation/validate_checkpoint.sh",
)

PERSONAL_PATHS = (
    re.compile(r"/home/[A-Za-z0-9._-]+/"),
    re.compile(r"/Users/[A-Za-z0-9._-]+/"),
    re.compile(r"[A-Za-z]:\\\\Users\\\\[A-Za-z0-9._-]+\\\\"),
)
FULL_SHA = re.compile(r"^[0-9a-f]{40}$")


def validate_manifest(repo_root: Path) -> list[str]:
    errors: list[str] = []
    data = load_json(repo_root / "REPOSITORY-ASSURANCE.json")
    if data.get("schema_version") != "ISRAS-REPOSITORY-V1":
        errors.append("repository assurance schema_version is not ISRAS-REPOSITORY-V1")
    if data.get("historical_gates_immutable") is not True:
        errors.append("historical_gates_immutable must be true")
    if data.get("native_first") is not True:
        errors.append("native_first must be true")
    repository = data.get("repository")
    if not isinstance(repository, str) or repository.count("/") != 1:
        errors.append("repository must use owner/name")
    standard = data.get("standard")
    if not isinstance(standard, dict):
        errors.append("standard must be an object")
    else:
        commit = standard.get("commit")
        if commit not in {"UNPINNED-BOOTSTRAP", "SELF"} and (
            not isinstance(commit, str) or not FULL_SHA.fullmatch(commit)
        ):
            errors.append("standard.commit must be a full SHA or UNPINNED-BOOTSTRAP")
    return errors


def validate_profiles(repo_root: Path) -> list[str]:
    errors: list[str] = []
    for path in sorted((repo_root / "tools/environment/profiles").glob("*.json")):
        data = load_json(path)
        if data.get("schema_version") != "ISRAS-ENVIRONMENT-V1":
            errors.append(f"{path.relative_to(repo_root)} has wrong schema version")
        if data.get("containers_required") is not False:
            errors.append(
                f"{path.relative_to(repo_root)} requires containers; document and accept "
                "that exception instead of using the baseline profile"
            )
        required = data.get("required_commands")
        if not isinstance(required, list):
            errors.append(f"{path.relative_to(repo_root)} required_commands must be a list")
    return errors


def validate_checkpoints(repo_root: Path) -> list[str]:
    errors: list[str] = []
    data = load_json(repo_root / "tools/validation/checkpoints.json")
    if data.get("schema_version") != "ISRAS-CHECKPOINTS-V1":
        errors.append("checkpoint registry has wrong schema version")
    checkpoints = data.get("checkpoints")
    if not isinstance(checkpoints, dict):
        return errors + ["checkpoint registry checkpoints must be an object"]
    for name, record in checkpoints.items():
        if not isinstance(record, dict):
            errors.append(f"checkpoint {name} must be an object")
            continue
        commit = record.get("commit", "")
        if not isinstance(commit, str) or not FULL_SHA.fullmatch(commit):
            errors.append(f"checkpoint {name} does not name a full commit SHA")
        gate = record.get("gate", "")
        if not isinstance(gate, str) or not gate:
            errors.append(f"checkpoint {name} has no gate")
    return errors


def validate_workflows(repo_root: Path) -> list[str]:
    errors: list[str] = []
    workflow_dir = repo_root / ".github/workflows"
    if not workflow_dir.exists():
        return errors
    uses_pattern = re.compile(r"^\s*uses:\s*([^\s#]+)", re.MULTILINE)
    for path in sorted(list(workflow_dir.glob("*.yml")) + list(workflow_dir.glob("*.yaml"))):
        text = path.read_text(encoding="utf-8")
        relative = path.relative_to(repo_root)
        if "pull_request_target:" in text:
            errors.append(f"{relative} uses pull_request_target")
        if not re.search(r"(?m)^permissions:\s*(?:\n|$)", text):
            errors.append(f"{relative} does not declare top-level permissions")
        for reference in uses_pattern.findall(text):
            if reference.startswith("./") or reference.startswith("docker://"):
                continue
            if "@" not in reference:
                errors.append(f"{relative} has unversioned uses reference: {reference}")
                continue
            ref = reference.rsplit("@", 1)[1]
            if not FULL_SHA.fullmatch(ref):
                errors.append(
                    f"{relative} external action/workflow is not pinned to a full SHA: "
                    f"{reference}"
                )
    return errors


def validate_personal_paths(repo_root: Path) -> list[str]:
    errors: list[str] = []
    for path in executable_files(repo_root):
        text = path.read_text(encoding="utf-8", errors="replace")
        for pattern in PERSONAL_PATHS:
            if pattern.search(text):
                errors.append(
                    f"{path.relative_to(repo_root)} contains a personal home path"
                )
                break
    return errors


def validate_placeholders(repo_root: Path) -> list[str]:
    errors: list[str] = []
    marker = re.compile(r"__[A-Z][A-Z0-9_]+__")
    for path in executable_files(repo_root):
        relative = path.relative_to(repo_root).as_posix()
        if "templates/" in relative or relative == "tools/isras/adopt.py":
            continue
        text = path.read_text(encoding="utf-8", errors="replace")
        found = sorted(set(marker.findall(text)))
        if found:
            errors.append(
                f"{path.relative_to(repo_root)} contains unresolved placeholders: "
                + ", ".join(found)
            )
    return errors


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    args = parser.parse_args()
    repo_root = repository_root(args.repo_root)

    errors: list[str] = []
    for relative in REQUIRED_PATHS:
        exists = (repo_root / relative).exists()
        print_result(f"Required assurance artifact exists: {relative}", exists)
        if not exists:
            errors.append(f"missing required path: {relative}")

    if not errors:
        for validator in (
            validate_manifest,
            validate_profiles,
            validate_checkpoints,
            validate_workflows,
            validate_personal_paths,
            validate_placeholders,
        ):
            errors.extend(validator(repo_root))

    for error in errors:
        print_result(error, False)

    if errors:
        print(f"\nISRAS policy validation FAILED with {len(errors)} error(s).", file=sys.stderr)
        return 1

    print("\nISRAS policy validation PASSED.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ISRASError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
