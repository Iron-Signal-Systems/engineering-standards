#!/usr/bin/env python3
from __future__ import annotations

import argparse
import datetime as dt
import json
import sys
from pathlib import Path

from common import ISRASError, git, load_json, repository_root, sha256_file


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--validator", required=True)
    parser.add_argument("--environment-profile", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--artifact", action="append", default=[])
    parser.add_argument("--accepted-predecessor")
    parser.add_argument("--correctness-result", choices=["PASS", "FAIL"], required=True)
    parser.add_argument("--resource-observation", default="NOT_APPLICABLE")
    parser.add_argument("--performance-budget", default="NOT_EVALUATED")
    parser.add_argument("--warning", action="append", default=[])
    parser.add_argument("--non-claim", action="append", default=[])
    args = parser.parse_args()
    repo_root = repository_root(args.repo_root)

    manifest = load_json(repo_root / "REPOSITORY-ASSURANCE.json")
    now = dt.datetime.now(dt.timezone.utc).isoformat()
    artifacts = []
    for relative in args.artifact:
        path = repo_root / relative
        if not path.is_file():
            raise ISRASError(f"evidence artifact is missing: {relative}")
        artifacts.append({"path": relative, "sha256": sha256_file(path)})

    data = {
        "schema_version": "ISRAS-ACCEPTANCE-EVIDENCE-V1",
        "repository": manifest["repository"],
        "source_commit": git(repo_root, "rev-parse", "HEAD"),
        "accepted_predecessor": args.accepted_predecessor,
        "standard_commit": manifest["standard"]["commit"],
        "validator": args.validator,
        "runner_identity": "LOCAL-UNSET",
        "environment_profile": args.environment_profile,
        "started_at": now,
        "finished_at": now,
        "correctness_result": args.correctness_result,
        "resource_observation": args.resource_observation,
        "performance_budget": args.performance_budget,
        "security_findings": "NOT_EVALUATED",
        "operational_readiness": "NOT_EVALUATED",
        "warnings": args.warning,
        "non_claims": args.non_claim,
        "artifacts": artifacts,
    }
    output = repo_root / args.output
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
    print(output)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ISRASError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
