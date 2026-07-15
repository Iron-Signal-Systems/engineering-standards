#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from urllib.parse import unquote

import jsonschema
import yaml

from common import ISRASError, executable_files, load_json, print_result, repository_root
from generate_source_manifest import digest_path, tracked_paths

REQUIRED_PATHS = (
    "REPOSITORY-ASSURANCE.json",
    "README.md",
    "GLOSSARY.md",
    "SUPPORT-AND-COMPATIBILITY.md",
    "SECURITY.md",
    "CONTRIBUTING.md",
    ".github/CODEOWNERS",
    ".github/pull_request_template.md",
    "docs/engineering/repository-assurance-adoption.md",
    "docs/engineering/secure-development-lifecycle.md",
    "docs/engineering/validation-environment-model.md",
    "docs/engineering/release-and-acceptance-model.md",
    "docs/acceptance/README.md",
    "schemas/repository-assurance-v1.schema.json",
    "schemas/environment-profile-v1.schema.json",
    "schemas/checkpoint-registry-v1.schema.json",
    "schemas/acceptance-evidence-v1.schema.json",
    "tools/requirements.txt",
    "tools/environment/profiles/portable.json",
    "tools/validation/checkpoints.json",
    "tools/validation/validate_portable.sh",
    "tools/validation/Validate-Portable.ps1",
    "tools/validation/validate_fresh_clone.sh",
    "tools/validation/validate_checkpoint.sh",
    "tools/isras/verify_source_manifest.py",
)

PERSONAL_PATHS = (
    re.compile(r"/home/[A-Za-z0-9._-]+/"),
    re.compile(r"/Users/[A-Za-z0-9._-]+/"),
    re.compile(r"[A-Za-z]:\\\\Users\\\\[A-Za-z0-9._-]+\\\\"),
)
FULL_SHA = re.compile(r"^[0-9a-f]{40}$")
SHA256 = re.compile(r"^[0-9a-f]{64}$")
MARKDOWN_LINK = re.compile(r"(?<!!)\[[^\]]+\]\(([^)]+)\)")
ADOPTION_RANK = {
    "RECORDED": 1,
    "REPRODUCIBLE": 2,
    "OBSERVED": 3,
    "ENFORCED": 4,
    "RELEASE_ASSURED": 5,
}


def schema_validate(instance: dict, schema: dict, label: str) -> list[str]:
    validator = jsonschema.Draft202012Validator(
        schema,
        format_checker=jsonschema.FormatChecker(),
    )
    return [
        f"{label}: {'/'.join(str(v) for v in error.absolute_path) or '<root>'}: {error.message}"
        for error in sorted(validator.iter_errors(instance), key=lambda e: list(e.absolute_path))
    ]


def validate_manifest(repo_root: Path) -> list[str]:
    data = load_json(repo_root / "REPOSITORY-ASSURANCE.json")
    schema = load_json(repo_root / "schemas/repository-assurance-v1.schema.json")
    errors = schema_validate(data, schema, "REPOSITORY-ASSURANCE.json")
    standard = data.get("standard", {})
    if standard.get("name") != "Iron Signal Repository Assurance Standard":
        errors.append("standard.name does not define the Iron Signal Repository Assurance Standard")
    if standard.get("acronym") != "ISRAS":
        errors.append("standard.acronym must be ISRAS")
    if "Information System Risk Assessment" not in (repo_root / "GLOSSARY.md").read_text(encoding="utf-8"):
        errors.append("GLOSSARY.md must distinguish ISRAS from Information System Risk Assessment")
    return errors


def validate_template_manifest(repo_root: Path) -> list[str]:
    path = repo_root / "templates/repository-baseline/REPOSITORY-ASSURANCE.json"
    if not path.is_file():
        return []
    text = path.read_text(encoding="utf-8")
    replacements = {
        "__REPOSITORY__": "Iron-Signal-Systems/example",
        "__CANONICAL_ORIGIN__": "git@github.com:Iron-Signal-Systems/example.git",
        "__DEVELOPMENT_BRANCH__": "dev",
        "__RELEASE_BRANCH__": "main",
        "__PROFILE__": "general",
        "UNPINNED-BOOTSTRAP": "0" * 40,
    }
    for token, value in replacements.items():
        text = text.replace(token, value)
    try:
        data = json.loads(text)
    except json.JSONDecodeError as exc:
        return [f"baseline repository assurance template is invalid JSON: {exc}"]
    schema = load_json(repo_root / "schemas/repository-assurance-v1.schema.json")
    return schema_validate(data, schema, "templates/repository-baseline/REPOSITORY-ASSURANCE.json")


def validate_profiles(repo_root: Path) -> list[str]:
    errors: list[str] = []
    schema = load_json(repo_root / "schemas/environment-profile-v1.schema.json")
    for base in [repo_root / "tools/environment/profiles", repo_root / "templates/repository-baseline/tools/environment/profiles"]:
        for path in sorted(base.glob("*.json")):
            data = load_json(path)
            errors.extend(schema_validate(data, schema, str(path.relative_to(repo_root))))
            if data.get("containers_required") is not False:
                errors.append(
                    f"{path.relative_to(repo_root)} requires containers; use a documented accepted exception"
                )
    return errors


def validate_checkpoints(repo_root: Path) -> list[str]:
    data = load_json(repo_root / "tools/validation/checkpoints.json")
    schema = load_json(repo_root / "schemas/checkpoint-registry-v1.schema.json")
    errors = schema_validate(data, schema, "tools/validation/checkpoints.json")
    for name, record in data.get("checkpoints", {}).items():
        gate = record.get("gate")
        if gate and not (repo_root / gate).is_file():
            errors.append(f"checkpoint {name} gate does not exist in the current tree: {gate}")
    return errors


def workflow_paths(repo_root: Path) -> list[Path]:
    roots = [
        repo_root / ".github/workflows",
        repo_root / "templates/workflows",
        repo_root / "templates/repository-baseline/.github/workflows",
    ]
    result: list[Path] = []
    for base in roots:
        if base.exists():
            result.extend(base.glob("*.yml"))
            result.extend(base.glob("*.yaml"))
    return sorted(set(result))


def validate_workflows(repo_root: Path) -> list[str]:
    errors: list[str] = []
    uses_pattern = re.compile(r"^\s*uses:\s*([^\s#]+)", re.MULTILINE)
    for path in workflow_paths(repo_root):
        relative = path.relative_to(repo_root).as_posix()
        text = path.read_text(encoding="utf-8")
        try:
            parsed = yaml.safe_load(text)
        except yaml.YAMLError as exc:
            errors.append(f"{relative} is invalid YAML: {exc}")
            continue
        if not isinstance(parsed, dict):
            errors.append(f"{relative} YAML root must be a mapping")
            continue
        if "pull_request_target:" in text:
            errors.append(f"{relative} uses pull_request_target")
        if "secrets: inherit" in text:
            errors.append(f"{relative} inherits all caller secrets")
        permissions = parsed.get("permissions")
        if permissions is None:
            errors.append(f"{relative} does not declare top-level permissions")
        elif permissions == "write-all":
            errors.append(f"{relative} grants write-all permissions")
        elif isinstance(permissions, dict):
            for key, value in permissions.items():
                if str(value).lower() == "write" and "release" not in relative and "canonical" not in relative:
                    errors.append(f"{relative} grants {key}: write outside a release/canonical workflow")
        has_pr = re.search(r"(?m)^\s{0,2}pull_request:\s*$", text) is not None
        if has_pr and re.search(r"runs-on:\s*(?:\n\s*-\s*)?self-hosted", text):
            errors.append(f"{relative} exposes a self-hosted runner to pull_request")
        if has_pr and "environment:" in text:
            errors.append(f"{relative} uses a protected deployment environment from pull_request")
        if ("portable" in path.name or "policy" in path.name) and re.search(r"runs-on:.*inputs\.", text):
            errors.append(f"{relative} permits caller-selected portable/policy runners")
        for reference in uses_pattern.findall(text):
            if reference.startswith("./"):
                continue
            if reference.startswith("docker://"):
                if "@sha256:" not in reference:
                    errors.append(f"{relative} uses mutable Docker action reference: {reference}")
                continue
            if "@" not in reference:
                errors.append(f"{relative} has unversioned uses reference: {reference}")
                continue
            ref = reference.rsplit("@", 1)[1]
            if ref == "STANDARD_FULL_COMMIT_SHA" and relative.startswith("templates/"):
                continue
            if not FULL_SHA.fullmatch(ref):
                errors.append(f"{relative} external action/workflow is not pinned to a full SHA: {reference}")
    return errors


def validate_yaml(repo_root: Path) -> list[str]:
    errors: list[str] = []
    bases = [repo_root / ".github", repo_root / "templates/repository-baseline/.github"]
    paths: set[Path] = set()
    for base in bases:
        if base.exists():
            paths.update(base.rglob("*.yml"))
            paths.update(base.rglob("*.yaml"))
    for path in sorted(paths):
        try:
            yaml.safe_load(path.read_text(encoding="utf-8"))
        except yaml.YAMLError as exc:
            errors.append(f"{path.relative_to(repo_root)} is invalid YAML: {exc}")
    return errors


def validate_markdown_links(repo_root: Path) -> list[str]:
    errors: list[str] = []
    for path in sorted(repo_root.rglob("*.md")):
        if ".git" in path.parts or ".isras-tools-venv" in path.parts:
            continue
        text = path.read_text(encoding="utf-8", errors="replace")
        for raw in MARKDOWN_LINK.findall(text):
            target = raw.strip().split()[0].strip("<>")
            if target.startswith(("http://", "https://", "mailto:", "#")):
                continue
            target = unquote(target.split("#", 1)[0].split("?", 1)[0])
            if not target:
                continue
            resolved = (path.parent / target).resolve()
            try:
                resolved.relative_to(repo_root.resolve())
            except ValueError:
                errors.append(f"{path.relative_to(repo_root)} link escapes repository: {raw}")
                continue
            if not resolved.exists():
                errors.append(f"{path.relative_to(repo_root)} has broken relative link: {raw}")
    return errors


def validate_personal_paths(repo_root: Path) -> list[str]:
    errors: list[str] = []
    for path in executable_files(repo_root):
        text = path.read_text(encoding="utf-8", errors="replace")
        for pattern in PERSONAL_PATHS:
            if pattern.search(text):
                errors.append(f"{path.relative_to(repo_root)} contains a personal home path")
                break
    return errors


def validate_placeholders(repo_root: Path) -> list[str]:
    errors: list[str] = []
    marker = re.compile(r"__[A-Z][A-Z0-9_]+__")
    for path in executable_files(repo_root):
        relative = path.relative_to(repo_root).as_posix()
        if relative.startswith("templates/") or relative in {
            "tools/isras/adopt.py",
            "tools/isras/validate_policy.py",
        }:
            continue
        found = sorted(set(marker.findall(path.read_text(encoding="utf-8", errors="replace"))))
        if found:
            errors.append(f"{relative} contains unresolved placeholders: {', '.join(found)}")
    return errors


def validate_tool_requirements(repo_root: Path) -> list[str]:
    errors: list[str] = []
    path = repo_root / "tools/requirements.txt"
    for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        value = line.strip()
        if not value or value.startswith("#"):
            continue
        if "==" not in value or any(operator in value for operator in [">=", "<=", "~=", "!="]):
            errors.append(f"tools/requirements.txt line {number} is not exactly pinned: {value}")
    return errors


def validate_source_manifest(repo_root: Path) -> list[str]:
    assurance = load_json(repo_root / "REPOSITORY-ASSURANCE.json")
    if ADOPTION_RANK.get(assurance.get("adoption_level", "RECORDED"), 0) < ADOPTION_RANK["REPRODUCIBLE"]:
        return []
    relative = assurance.get("source_manifest", "SOURCE-SHA256SUMS.txt")
    path = repo_root / relative
    if not path.is_file():
        return [f"reproducible adoption requires source manifest: {relative}"]
    expected: dict[str, str] = {}
    for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        parts = line.split("  ", 1)
        if len(parts) != 2 or not SHA256.fullmatch(parts[0]):
            return [f"invalid source manifest line {number}: {line!r}"]
        if parts[1] in expected:
            return [f"duplicate source manifest path: {parts[1]}"]
        expected[parts[1]] = parts[0]
    actual = tracked_paths(repo_root, relative)
    if sorted(expected) != actual:
        return ["source manifest path set does not match tracked files"]
    mismatched = [name for name in actual if digest_path(repo_root / name) != expected[name]]
    if mismatched:
        return ["source manifest digest mismatch: " + ", ".join(mismatched)]
    return []


def validate_evidence(repo_root: Path) -> list[str]:
    errors: list[str] = []
    assurance = load_json(repo_root / "REPOSITORY-ASSURANCE.json")
    evidence_dir = repo_root / assurance.get("evidence_directory", "docs/acceptance/evidence")
    schema = load_json(repo_root / "schemas/acceptance-evidence-v1.schema.json")
    if not evidence_dir.exists():
        return errors
    for path in sorted(evidence_dir.rglob("*.json")):
        data = load_json(path)
        errors.extend(schema_validate(data, schema, str(path.relative_to(repo_root))))
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
        validators = (
            validate_manifest,
            validate_template_manifest,
            validate_profiles,
            validate_checkpoints,
            validate_workflows,
            validate_yaml,
            validate_markdown_links,
            validate_personal_paths,
            validate_placeholders,
            validate_tool_requirements,
            validate_source_manifest,
            validate_evidence,
        )
        for validator in validators:
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
