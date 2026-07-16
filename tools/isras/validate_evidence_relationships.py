#!/usr/bin/env python3
"""Validate that assurance claims are bound to real, matching evidence."""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import subprocess
import sys
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator, FormatChecker

PASS_COUNT_RE = re.compile(r"(?m)^PASS checks:\s*([0-9]+)\s*$")
FAIL_COUNT_RE = re.compile(r"(?m)^FAIL checks:\s*([0-9]+)\s*$")
CONTROL_RE = re.compile(r"ISRAS-[A-Z]{2,4}-[0-9]{3}")


def load(path: Path) -> Any:
    with path.open(encoding="utf-8") as handle:
        return json.load(handle)


def sha512(path: Path) -> str:
    hasher = hashlib.sha512()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b""):
            hasher.update(block)
    return hasher.hexdigest()


def sha512_bytes(value: bytes) -> str:
    return hashlib.sha512(value).hexdigest()


def safe_path(root: Path, value: str) -> Path:
    relative = Path(value)
    if relative.is_absolute() or "\\" in value or any(part in {"", ".", ".."} for part in relative.parts):
        raise ValueError(f"unsafe repository-relative path: {value!r}")
    candidate = (root / relative).resolve()
    try:
        candidate.relative_to(root)
    except ValueError as exc:
        raise ValueError(f"path escapes repository: {value}") from exc
    return candidate


def git(root: Path, *args: str, binary: bool = False) -> subprocess.CompletedProcess[Any]:
    return subprocess.run(
        ["git", *args],
        cwd=root,
        text=not binary,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )


def tracked(root: Path, relative: str) -> bool:
    return git(root, "ls-files", "--error-unmatch", "--", relative).returncode == 0


def blob_at_commit(root: Path, commit: str, relative: str) -> bytes | None:
    result = git(root, "show", f"{commit}:{relative}", binary=True)
    return result.stdout if result.returncode == 0 else None


def json_value(document: Any, path: list[Any]) -> Any:
    value = document
    for segment in path:
        value = value[segment]
    return value


def validate_identity(
    path: Path,
    probe: dict[str, Any],
    candidate_commit: str,
    campaign_id: str,
    environment_sha512: str,
) -> list[str]:
    errors: list[str] = []
    if probe["format"] == "TEXT":
        text = path.read_text(encoding="utf-8", errors="replace")
        expected = {
            "candidate_commit_marker": candidate_commit,
            "campaign_id_marker": campaign_id,
            "environment_sha512_marker": environment_sha512,
        }
        for field, exact_value in expected.items():
            marker = probe[field]
            if marker not in text:
                errors.append(f"{path}: identity marker missing: {marker!r}")
            if exact_value not in marker:
                errors.append(f"{path}: {field} does not embed the exact bound identity")
    else:
        try:
            document = load(path)
            observed = {
                "candidate_commit": json_value(document, probe["candidate_commit_path"]),
                "campaign_id": json_value(document, probe["campaign_id_path"]),
                "environment_sha512": json_value(document, probe["environment_sha512_path"]),
            }
        except (KeyError, IndexError, TypeError, json.JSONDecodeError) as exc:
            errors.append(f"{path}: JSON identity probe failed: {exc}")
        else:
            expected = {
                "candidate_commit": candidate_commit,
                "campaign_id": campaign_id,
                "environment_sha512": environment_sha512,
            }
            for field, expected_value in expected.items():
                if observed[field] != expected_value:
                    errors.append(
                        f"{path}: JSON identity mismatch for {field}: "
                        f"expected {expected_value!r}, got {observed[field]!r}"
                    )
    return errors


def validate_outcome(path: Path, probe: dict[str, Any]) -> list[str]:
    errors: list[str] = []
    if probe["format"] == "TEXT_ISRAS_SUMMARY":
        text = path.read_text(encoding="utf-8", errors="replace")
        passes = [int(value) for value in PASS_COUNT_RE.findall(text)]
        failures = [int(value) for value in FAIL_COUNT_RE.findall(text)]
        if not passes or not failures:
            errors.append(f"{path}: ISRAS count summary is missing")
        else:
            if passes[-1] <= 0:
                errors.append(f"{path}: final PASS count is not positive")
            if failures[-1] != 0:
                errors.append(f"{path}: final FAIL count is not zero")
        pass_position = text.rfind(probe["pass_marker"])
        failed_position = text.rfind("validation FAILED")
        if pass_position < 0:
            errors.append(f"{path}: required terminal PASS marker is missing")
        elif failed_position > pass_position:
            errors.append(f"{path}: a later FAILED marker invalidates the PASS outcome")
    else:
        try:
            value = json_value(load(path), probe["json_path"])
        except (KeyError, IndexError, TypeError, json.JSONDecodeError) as exc:
            errors.append(f"{path}: JSON outcome probe failed: {exc}")
        else:
            if value != probe["expected"]:
                errors.append(
                    f"{path}: JSON outcome mismatch: expected {probe['expected']!r}, got {value!r}"
                )
    return errors


def declared_controls(root: Path) -> set[str]:
    controls: set[str] = set()
    for path in sorted((root / "standards/repository-assurance").glob("v*/CONTROL-CATALOG.md")):
        controls.update(CONTROL_RE.findall(path.read_text(encoding="utf-8")))
    return controls


def normalize_origin(value: str) -> str | None:
    value = value.strip()
    match = re.search(r"github\.com[:/]([^/]+)/([^/]+?)(?:\.git)?$", value)
    return f"{match.group(1)}/{match.group(2)}" if match else None


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--binding", required=True)
    parser.add_argument("--expected-commit", required=True)
    parser.add_argument("--expected-repository")
    parser.add_argument("--prior-binding", action="append", default=[])
    parser.add_argument(
        "--allow-untracked-fixtures",
        action="store_true",
        help="Test-only escape hatch; prohibited for acceptance evidence.",
    )
    args = parser.parse_args()

    root = Path(args.repo_root).resolve()
    binding_path = safe_path(root, args.binding)
    schema = load(root / "schemas/evidence-binding-v1.schema.json")
    binding = load(binding_path)
    errors = [
        f"schema: {'/'.join(map(str, error.absolute_path)) or '<root>'}: {error.message}"
        for error in Draft202012Validator(
            schema, format_checker=FormatChecker()
        ).iter_errors(binding)
    ]
    if errors:
        for error in sorted(errors):
            print(f"FAIL: {error}")
        return 1

    head = git(root, "rev-parse", "HEAD")
    if head.returncode != 0:
        errors.append("unable to resolve repository HEAD")
        current_head = ""
    else:
        current_head = head.stdout.strip()
    if args.expected_commit != current_head:
        errors.append(
            f"expected commit is not current HEAD: expected={args.expected_commit}; HEAD={current_head}"
        )
    if binding["candidate_commit"] != args.expected_commit:
        errors.append("binding candidate_commit does not match --expected-commit")
    if git(root, "cat-file", "-e", f"{args.expected_commit}^{{commit}}").returncode != 0:
        errors.append("expected commit does not resolve to a Git commit")

    assurance = load(root / "REPOSITORY-ASSURANCE.json")
    expected_repository = args.expected_repository or assurance.get("repository")
    if binding["repository"] != expected_repository:
        errors.append("binding repository does not match expected repository")
    origin = git(root, "remote", "get-url", "origin")
    if origin.returncode == 0:
        observed_repository = normalize_origin(origin.stdout)
        if observed_repository and observed_repository != expected_repository:
            errors.append(
                f"origin repository mismatch: expected={expected_repository}; observed={observed_repository}"
            )

    environment = binding["environment_artifact"]
    environment_path = safe_path(root, environment["path"])
    if not environment_path.is_file():
        errors.append(f"environment artifact does not exist: {environment_path}")
    elif sha512(environment_path) != environment["sha512"]:
        errors.append("environment artifact SHA-512 mismatch")
    if not args.allow_untracked_fixtures and not tracked(root, environment["path"]):
        errors.append("environment artifact is not tracked by Git")

    validators = {item["validator_id"]: item for item in binding["validators"]}
    if len(validators) != len(binding["validators"]):
        errors.append("validator identifiers are not unique")
    artifacts = {item["artifact_id"]: item for item in binding["artifacts"]}
    if len(artifacts) != len(binding["artifacts"]):
        errors.append("artifact identifiers are not unique")
    claims = {item["claim_id"]: item for item in binding["claims"]}
    if len(claims) != len(binding["claims"]):
        errors.append("claim identifiers are not unique")

    for validator_id, validator in validators.items():
        relative = validator["executable_path"]
        path = safe_path(root, relative)
        if not path.is_file():
            errors.append(f"{validator_id}: validator executable does not exist: {path}")
            continue
        if sha512(path) != validator["executable_sha512"]:
            errors.append(f"{validator_id}: working validator executable SHA-512 mismatch")
        if validator["source_commit"] != binding["candidate_commit"]:
            errors.append(f"{validator_id}: validator source commit does not match candidate")
        if not args.allow_untracked_fixtures and not tracked(root, relative):
            errors.append(f"{validator_id}: validator executable is not tracked by Git")
        committed = blob_at_commit(root, validator["source_commit"], relative)
        if committed is None:
            if not args.allow_untracked_fixtures:
                errors.append(f"{validator_id}: validator executable is absent from source commit")
        elif sha512_bytes(committed) != validator["executable_sha512"]:
            errors.append(f"{validator_id}: committed validator executable SHA-512 mismatch")

    declared_tests = set(binding["declared_test_ids"])
    observed_tests: set[str] = set()
    seen_paths: set[str] = set()
    seen_digests: set[str] = set()
    for artifact_id, artifact in artifacts.items():
        relative = artifact["path"]
        path = safe_path(root, relative)
        if relative in seen_paths:
            errors.append(f"{artifact_id}: duplicate artifact path")
        seen_paths.add(relative)
        if artifact["sha512"] in seen_digests:
            errors.append(f"{artifact_id}: duplicate artifact digest under multiple identifiers")
        seen_digests.add(artifact["sha512"])
        if not path.is_file():
            errors.append(f"{artifact_id}: evidence artifact does not exist: {path}")
            continue
        if sha512(path) != artifact["sha512"]:
            errors.append(f"{artifact_id}: evidence artifact SHA-512 mismatch")
        if not args.allow_untracked_fixtures and not tracked(root, relative):
            errors.append(f"{artifact_id}: evidence artifact is not tracked by Git")
        if artifact["validator_id"] not in validators:
            errors.append(f"{artifact_id}: unknown validator_id {artifact['validator_id']!r}")
        observed_tests.update(artifact["test_ids"])
        undeclared = sorted(set(artifact["test_ids"]) - declared_tests)
        if undeclared:
            errors.append(f"{artifact_id}: artifacts reference undeclared tests: {undeclared}")
        errors.extend(
            validate_identity(
                path,
                artifact["identity_probe"],
                binding["candidate_commit"],
                binding["campaign_id"],
                environment["sha512"],
            )
        )
        errors.extend(validate_outcome(path, artifact["outcome_probe"]))

    controls = declared_controls(root)
    for claim_id, claim in claims.items():
        if claim["control_id"] not in controls:
            errors.append(f"{claim_id}: unknown control identifier {claim['control_id']}")
        missing_artifacts = sorted(set(claim["artifact_ids"]) - set(artifacts))
        if missing_artifacts:
            errors.append(f"{claim_id}: unknown artifact identifiers: {missing_artifacts}")
        covered_tests: set[str] = set()
        for artifact_id in claim["artifact_ids"]:
            if artifact_id in artifacts:
                covered_tests.update(artifacts[artifact_id]["test_ids"])
        missing_tests = sorted(set(claim["required_test_ids"]) - covered_tests)
        if missing_tests:
            errors.append(f"{claim_id}: required tests lack linked evidence: {missing_tests}")
        undeclared = sorted(set(claim["required_test_ids"]) - declared_tests)
        if undeclared:
            errors.append(f"{claim_id}: required test identifiers are undeclared: {undeclared}")

    for prior_value in args.prior_binding:
        prior_path = safe_path(root, prior_value)
        prior = load(prior_path)
        prior_boundary = (
            prior.get("candidate_commit"),
            prior.get("campaign_id"),
            (prior.get("environment_artifact") or {}).get("sha512"),
        )
        current_boundary = (
            binding["candidate_commit"],
            binding["campaign_id"],
            environment["sha512"],
        )
        if prior_boundary == current_boundary:
            continue
        prior_digests = {
            item.get("sha512")
            for item in prior.get("artifacts", [])
            if isinstance(item, dict)
        }
        reused = sorted(seen_digests & prior_digests)
        if reused:
            errors.append(
                f"evidence digests are reused across incompatible boundaries: {reused}"
            )

    if errors:
        for error in sorted(set(errors)):
            print(f"FAIL: {error}")
        print(f"Evidence relationship validation FAILED with {len(set(errors))} error(s).")
        return 1
    print(
        "Evidence relationship validation PASSED: "
        f"{len(validators)} validators, {len(artifacts)} artifacts, {len(claims)} claims."
    )
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
