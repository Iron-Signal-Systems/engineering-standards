#!/usr/bin/env python3
from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import sys
from pathlib import Path

import jsonschema

from common import ISRASError, git, load_json, repository_root, run, sha256_file
from doctor import build_fingerprint

FULL_SHA_LENGTH = 40


def parse_time(value: str | None) -> dt.datetime:
    if value is None:
        return dt.datetime.now(dt.timezone.utc)
    try:
        parsed = dt.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError as exc:
        raise ISRASError(f"invalid ISO-8601 timestamp: {value}") from exc
    if parsed.tzinfo is None:
        raise ISRASError("timestamp must include a timezone offset")
    return parsed.astimezone(dt.timezone.utc)


def ensure_pushed(repo_root: Path, manifest: dict, source_commit: str) -> None:
    development = manifest["branches"]["development"]
    origin = manifest["canonical_origin"]
    result = run(
        ["git", "ls-remote", origin, f"refs/heads/{development}"],
        cwd=repo_root,
        capture=True,
        check=False,
    )
    if result.returncode != 0:
        raise ISRASError("cannot verify the canonical remote development branch")
    remote_sha = result.stdout.strip().split(maxsplit=1)[0] if result.stdout.strip() else ""
    if remote_sha != source_commit:
        raise ISRASError(
            f"source commit is not the canonical {development} head: "
            f"local={source_commit} remote={remote_sha or 'MISSING'}"
        )


def resolve_standard_commit(manifest: dict, source_commit: str) -> str:
    value = manifest["standard"]["commit"]
    if value == "SELF":
        return source_commit
    if value == "UNPINNED-BOOTSTRAP":
        raise ISRASError("formal acceptance evidence cannot use UNPINNED-BOOTSTRAP")
    if not isinstance(value, str) or len(value) != FULL_SHA_LENGTH:
        raise ISRASError("standard commit must resolve to a full 40-character SHA")
    return value


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--validator", required=True)
    parser.add_argument("--environment-profile", required=True)
    parser.add_argument("--runner-identity")
    parser.add_argument("--started-at")
    parser.add_argument("--acceptance-tag")
    parser.add_argument("--output", required=True)
    parser.add_argument("--artifact", action="append", default=[])
    parser.add_argument("--accepted-predecessor")
    parser.add_argument("--correctness-result", choices=["PASS", "FAIL"], required=True)
    parser.add_argument("--resource-observation", choices=["RECORDED", "NOT_RECORDED", "NOT_APPLICABLE"], default="NOT_APPLICABLE")
    parser.add_argument("--performance-budget", choices=["PASS", "FAIL", "NOT_EVALUATED", "NOT_APPLICABLE"], default="NOT_EVALUATED")
    parser.add_argument("--security-findings", choices=["NONE", "RECORDED", "NOT_EVALUATED"], default="NOT_EVALUATED")
    parser.add_argument("--operational-readiness", choices=["ACCEPTED", "NOT_ACCEPTED", "NOT_EVALUATED"], default="NOT_EVALUATED")
    parser.add_argument("--warning", action="append", default=[])
    parser.add_argument("--non-claim", action="append", default=[])
    parser.add_argument("--allow-unpushed", action="store_true")
    args = parser.parse_args()
    started = parse_time(args.started_at)
    repo_root = repository_root(args.repo_root)
    manifest = load_json(repo_root / "REPOSITORY-ASSURANCE.json")
    source_commit = git(repo_root, "rev-parse", "HEAD")
    source_branch = git(repo_root, "branch", "--show-current") or "DETACHED"
    runner_identity = args.runner_identity or os.environ.get("ISRAS_RUNNER_IDENTITY")
    if not runner_identity:
        raise ISRASError("runner identity is required through --runner-identity or ISRAS_RUNNER_IDENTITY")

    if args.correctness_result == "PASS":
        if git(repo_root, "status", "--porcelain"):
            raise ISRASError("PASS acceptance evidence requires a clean working tree")
        if not args.allow_unpushed:
            ensure_pushed(repo_root, manifest, source_commit)

    profile_path = repo_root / "tools/environment/profiles" / f"{args.environment_profile}.json"
    profile = load_json(profile_path)
    fingerprint = build_fingerprint(repo_root, profile)
    if fingerprint["repository_commit"] != source_commit:
        raise ISRASError("environment fingerprint source commit mismatch")

    artifacts = []
    for relative in args.artifact:
        path = repo_root / relative
        if not path.is_file():
            raise ISRASError(f"evidence artifact is missing: {relative}")
        artifacts.append({"path": relative, "sha256": sha256_file(path)})

    finished = dt.datetime.now(dt.timezone.utc)
    data = {
        "schema_version": "ISRAS-ACCEPTANCE-EVIDENCE-V1",
        "repository": manifest["repository"],
        "source_commit": source_commit,
        "source_branch": source_branch,
        "accepted_predecessor": args.accepted_predecessor,
        "acceptance_tag": args.acceptance_tag,
        "standard_commit": resolve_standard_commit(manifest, source_commit),
        "validator": args.validator,
        "runner_identity": runner_identity,
        "environment_profile": args.environment_profile,
        "environment_fingerprint": fingerprint,
        "started_at": started.isoformat(),
        "finished_at": finished.isoformat(),
        "correctness_result": args.correctness_result,
        "resource_observation": args.resource_observation,
        "performance_budget": args.performance_budget,
        "security_findings": args.security_findings,
        "operational_readiness": args.operational_readiness,
        "warnings": args.warning,
        "non_claims": args.non_claim,
        "artifacts": artifacts,
    }

    schema = load_json(repo_root / "schemas/acceptance-evidence-v1.schema.json")
    try:
        jsonschema.Draft202012Validator(schema, format_checker=jsonschema.FormatChecker()).validate(data)
    except jsonschema.ValidationError as exc:
        raise ISRASError(f"generated acceptance evidence does not satisfy its schema: {exc.message}") from exc

    output = repo_root / args.output
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(data, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print(output)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ISRASError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
