#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

import jsonschema

from common import ISRASError, git, load_json, repository_root


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--name", required=True)
    parser.add_argument("--status", choices=["candidate", "accepted", "superseded", "withdrawn"], required=True)
    parser.add_argument("--gate", required=True)
    parser.add_argument("--environment-profile", required=True)
    parser.add_argument("--required-branch-name", default="dev")
    parser.add_argument("--tag")
    parser.add_argument("--expected-pass", type=int)
    parser.add_argument("--expected-fail", type=int, default=0)
    args = parser.parse_args()
    repo_root = repository_root(args.repo_root)
    if git(repo_root, "status", "--porcelain"):
        raise ISRASError("recording a checkpoint requires a clean working tree")
    commit = git(repo_root, "rev-parse", "HEAD")
    registry_path = repo_root / "tools/validation/checkpoints.json"
    registry = load_json(registry_path)
    expected = {"fail": args.expected_fail}
    if args.expected_pass is not None:
        expected["pass"] = args.expected_pass
    registry["checkpoints"][args.name] = {
        "status": args.status,
        "commit": commit,
        "tag": args.tag,
        "gate": args.gate,
        "environment_profile": args.environment_profile,
        "required_branch_name": args.required_branch_name,
        "expected_result": expected,
    }
    schema = load_json(repo_root / "schemas/checkpoint-registry-v1.schema.json")
    try:
        jsonschema.Draft202012Validator(schema).validate(registry)
    except jsonschema.ValidationError as exc:
        raise ISRASError(f"checkpoint registry would violate its schema: {exc.message}") from exc
    registry_path.write_text(json.dumps(registry, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print(f"Recorded {args.name} at {commit}. Commit the registry change separately.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ISRASError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
