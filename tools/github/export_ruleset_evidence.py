#!/usr/bin/env python3
"""Export GitHub repository ruleset and branch-protection evidence through gh."""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import subprocess
import sys
from pathlib import Path
from typing import Any


def run_json(*args: str, allow_missing: bool = False) -> Any:
    result = subprocess.run(
        ["gh", *args],
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if result.returncode != 0:
        if allow_missing and "HTTP 404" in result.stderr:
            return {"status": "NOT_CONFIGURED", "http_status": 404}
        raise RuntimeError(result.stderr.strip() or f"gh {' '.join(args)} failed")
    text = result.stdout.strip()
    return json.loads(text) if text else None


def canonical_bytes(value: Any) -> bytes:
    return json.dumps(value, sort_keys=True, separators=(",", ":")).encode("utf-8")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repository", required=True, help="owner/name")
    parser.add_argument("--source-commit", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()

    if len(args.source_commit) != 40 or any(c not in "0123456789abcdef" for c in args.source_commit):
        raise RuntimeError("--source-commit must be a lowercase 40-character Git commit")

    repository = run_json("api", f"repos/{args.repository}")
    if repository.get("full_name") != args.repository:
        raise RuntimeError("GitHub repository identity does not match requested owner/name")
    commit = run_json("api", f"repos/{args.repository}/commits/{args.source_commit}")
    if commit.get("sha") != args.source_commit:
        raise RuntimeError("GitHub did not return the exact requested source commit")
    tree_sha = ((commit.get("commit") or {}).get("tree") or {}).get("sha")
    if not isinstance(tree_sha, str) or len(tree_sha) != 40:
        raise RuntimeError("GitHub commit response did not contain a valid tree SHA")

    listed_rulesets = run_json(
        "api", "--paginate", "--slurp", f"repos/{args.repository}/rulesets"
    )
    if (
        isinstance(listed_rulesets, list)
        and len(listed_rulesets) == 1
        and isinstance(listed_rulesets[0], list)
    ):
        listed_rulesets = listed_rulesets[0]
    if not isinstance(listed_rulesets, list):
        raise RuntimeError("GitHub ruleset listing did not return an array")
    rulesets = [
        run_json("api", f"repos/{args.repository}/rulesets/{item['id']}")
        for item in listed_rulesets
    ]

    branch_protection = {
        branch: run_json(
            "api",
            f"repos/{args.repository}/branches/{branch}/protection",
            allow_missing=True,
        )
        for branch in ("dev", "main")
    }
    raw = {
        "repository": args.repository,
        "source_commit": args.source_commit,
        "commit_tree_sha": tree_sha,
        "default_branch": repository.get("default_branch"),
        "rulesets": rulesets,
        "branch_protection": branch_protection,
    }
    digest = hashlib.sha512(canonical_bytes(raw)).hexdigest()
    actor = run_json("api", "user")["login"]
    gh_version = subprocess.run(
        ["gh", "--version"],
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=True,
    ).stdout.splitlines()[0].strip()

    record = {
        "schema_version": "ISRAS-GITHUB-CONTROL-EVIDENCE-V1",
        "repository": args.repository,
        "source_commit": args.source_commit,
        "commit_tree_sha": tree_sha,
        "default_branch": repository.get("default_branch"),
        "collected_at": dt.datetime.now(dt.timezone.utc).isoformat().replace("+00:00", "Z"),
        "collector": {
            "tool": "gh",
            "version": gh_version,
            "actor": actor,
        },
        "raw_configuration_sha512": digest,
        "rulesets": rulesets,
        "branch_protection": branch_protection,
    }
    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(record, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print(f"Wrote GitHub control evidence to {output}")
    print(f"Raw configuration SHA-512: {digest}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, RuntimeError, json.JSONDecodeError, subprocess.CalledProcessError) as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
