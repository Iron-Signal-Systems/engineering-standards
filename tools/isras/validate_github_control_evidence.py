#!/usr/bin/env python3
"""Validate exported GitHub rulesets and branch protections against ISRAS requirements."""
from __future__ import annotations

import argparse
import fnmatch
import hashlib
import json
import sys
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator, FormatChecker


def canonical_bytes(value: Any) -> bytes:
    return json.dumps(value, sort_keys=True, separators=(",", ":")).encode("utf-8")


def rule_types(ruleset: dict[str, Any]) -> set[str]:
    return {
        str(rule.get("type"))
        for rule in (ruleset.get("rules") or [])
        if isinstance(rule, dict)
    }


def expand_pattern(pattern: str, default_branch: str) -> str:
    if pattern == "~DEFAULT_BRANCH":
        return f"refs/heads/{default_branch}"
    if pattern == "~ALL":
        return "refs/*"
    return pattern


def pattern_matches(ref: str, pattern: str, default_branch: str) -> bool:
    expanded = expand_pattern(pattern, default_branch)
    return fnmatch.fnmatchcase(ref, expanded)


def targets(ruleset: dict[str, Any], ref: str, expected_target: str, default_branch: str) -> bool:
    if ruleset.get("target") != expected_target:
        return False
    conditions = ruleset.get("conditions")
    if not isinstance(conditions, dict):
        return False
    ref_name = conditions.get("ref_name")
    if not isinstance(ref_name, dict):
        return False
    includes = [str(value) for value in (ref_name.get("include") or [])]
    excludes = [str(value) for value in (ref_name.get("exclude") or [])]
    included = any(pattern_matches(ref, pattern, default_branch) for pattern in includes)
    excluded = any(pattern_matches(ref, pattern, default_branch) for pattern in excludes)
    return included and not excluded


def bypass_key(actor: dict[str, Any]) -> str:
    return (
        f"{actor.get('actor_type')}:{actor.get('actor_id')}:"
        f"{actor.get('bypass_mode')}"
    )


def classic_enabled(record: Any, key: str) -> bool:
    if not isinstance(record, dict):
        return False
    value = record.get(key)
    return isinstance(value, dict) and value.get("enabled") is True


def classic_branch_satisfies(protection: Any, requirement: str) -> bool:
    if not isinstance(protection, dict) or protection.get("status") == "NOT_CONFIGURED":
        return False
    if requirement == "deletion":
        return not classic_enabled(protection, "allow_deletions")
    if requirement == "non_fast_forward":
        return not classic_enabled(protection, "allow_force_pushes")
    if requirement == "pull_request":
        return isinstance(protection.get("required_pull_request_reviews"), dict)
    if requirement == "required_status_checks":
        checks = protection.get("required_status_checks")
        if not isinstance(checks, dict):
            return False
        return bool(checks.get("contexts") or checks.get("checks"))
    return False


def rule_checks(rule: dict[str, Any]) -> set[str]:
    if rule.get("type") != "required_status_checks":
        return set()
    parameters = rule.get("parameters") or {}
    values = parameters.get("required_status_checks") or []
    return {
        str(item.get("context"))
        for item in values
        if isinstance(item, dict) and item.get("context")
    }


def classic_checks(protection: Any) -> set[str]:
    if not isinstance(protection, dict):
        return set()
    checks = protection.get("required_status_checks")
    if not isinstance(checks, dict):
        return set()
    result = {str(value) for value in (checks.get("contexts") or [])}
    result.update(
        str(item.get("context"))
        for item in (checks.get("checks") or [])
        if isinstance(item, dict) and item.get("context")
    )
    return result


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--record", required=True)
    parser.add_argument("--expected-commit", required=True)
    parser.add_argument("--expected-repository", required=True)
    parser.add_argument(
        "--required-dev-check",
        action="append",
        default=[],
        help="Exact required status-check context expected on dev.",
    )
    parser.add_argument(
        "--allowed-bypass-actor",
        action="append",
        default=[],
        help="Explicit actor_type:actor_id:bypass_mode allowed by reviewed governance.",
    )
    args = parser.parse_args()

    root = Path(args.repo_root).resolve()
    path = (root / args.record).resolve()
    try:
        path.relative_to(root)
    except ValueError as exc:
        raise ValueError("GitHub evidence record must remain inside the repository") from exc
    data = json.loads(path.read_text(encoding="utf-8"))
    schema = json.loads(
        (root / "schemas/github-control-evidence-v1.schema.json").read_text(encoding="utf-8")
    )
    errors = [
        f"schema: {'/'.join(map(str, error.absolute_path)) or '<root>'}: {error.message}"
        for error in Draft202012Validator(
            schema, format_checker=FormatChecker()
        ).iter_errors(data)
    ]

    if data.get("source_commit") != args.expected_commit:
        errors.append("GitHub control evidence source commit does not match expected commit")
    if data.get("repository") != args.expected_repository:
        errors.append("GitHub control evidence repository does not match expected repository")
    if not args.required_dev_check:
        errors.append("at least one --required-dev-check must be supplied")

    raw = {
        "repository": data.get("repository"),
        "source_commit": data.get("source_commit"),
        "commit_tree_sha": data.get("commit_tree_sha"),
        "default_branch": data.get("default_branch"),
        "rulesets": data.get("rulesets"),
        "branch_protection": data.get("branch_protection"),
    }
    actual_digest = hashlib.sha512(canonical_bytes(raw)).hexdigest()
    if actual_digest != data.get("raw_configuration_sha512"):
        errors.append("raw GitHub configuration SHA-512 mismatch")

    rulesets = [
        item for item in data.get("rulesets", [])
        if isinstance(item, dict) and item.get("enforcement") == "active"
    ]
    default_branch = str(data.get("default_branch") or "")
    allowed_bypass = set(args.allowed_bypass_actor)
    for ruleset in rulesets:
        for actor in ruleset.get("bypass_actors") or []:
            if not isinstance(actor, dict):
                errors.append(f"ruleset {ruleset.get('name')!r} has malformed bypass actor")
                continue
            key = bypass_key(actor)
            if key not in allowed_bypass:
                errors.append(
                    f"ruleset {ruleset.get('name')!r} has unapproved bypass actor {key}"
                )

    def requirements_for(
        ref: str,
        target: str,
        required: set[str],
        classic_name: str | None = None,
        required_checks: set[str] | None = None,
    ) -> None:
        matching = [
            item
            for item in rulesets
            if targets(item, ref, target, default_branch)
        ]
        combined: set[str] = set()
        named_checks: set[str] = set()
        for item in matching:
            combined.update(rule_types(item))
            for rule in item.get("rules") or []:
                if isinstance(rule, dict):
                    named_checks.update(rule_checks(rule))
        classic = (
            data.get("branch_protection", {}).get(classic_name)
            if classic_name
            else None
        )
        named_checks.update(classic_checks(classic))
        missing = sorted(
            requirement
            for requirement in required
            if requirement not in combined
            and not classic_branch_satisfies(classic, requirement)
        )
        if missing:
            errors.append(f"{ref} lacks effective protection requirements: {missing}")

        if required_checks is not None:
            missing_checks = sorted(required_checks - named_checks)
            if missing_checks:
                errors.append(f"{ref} lacks required named checks: {missing_checks}")

    branch_required = {
        "deletion",
        "non_fast_forward",
        "pull_request",
        "required_status_checks",
    }
    requirements_for(
        "refs/heads/dev",
        "branch",
        branch_required,
        "dev",
        set(args.required_dev_check),
    )
    requirements_for(
        "refs/heads/main",
        "branch",
        {"deletion", "non_fast_forward"},
        "main",
    )
    requirements_for(
        "refs/tags/isras-v0.0.0",
        "tag",
        {"creation", "update", "deletion"},
    )

    if errors:
        for error in sorted(set(errors)):
            print(f"FAIL: {error}")
        print(f"GitHub control evidence validation FAILED with {len(set(errors))} error(s).")
        return 1
    print("GitHub control evidence validation PASSED.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
