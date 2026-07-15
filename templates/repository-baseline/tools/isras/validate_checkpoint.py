#!/usr/bin/env python3
from __future__ import annotations

import argparse
import os
import sys
import tempfile
from pathlib import Path

from common import ISRASError, load_json, print_result, repository_root, run


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--checkpoint", required=True)
    args = parser.parse_args()
    repo_root = repository_root(args.repo_root)

    manifest = load_json(repo_root / "REPOSITORY-ASSURANCE.json")
    registry_path = repo_root / manifest["checkpoint_registry"]
    registry = load_json(registry_path)
    checkpoints = registry.get("checkpoints", {})
    record = checkpoints.get(args.checkpoint)
    if not isinstance(record, dict):
        raise ISRASError(f"unknown checkpoint: {args.checkpoint}")

    commit = record["commit"]
    gate = record["gate"]
    branch = record["required_branch_name"]
    origin = manifest["canonical_origin"]

    with tempfile.TemporaryDirectory(prefix=f"isras-{args.checkpoint}-") as temporary:
        clone = Path(temporary) / "repository"
        run(["git", "clone", "--no-local", origin, str(clone)])
        run(["git", "checkout", "-B", branch, commit], cwd=clone)
        gate_path = clone / gate
        if not gate_path.exists():
            raise ISRASError(f"historical gate is missing from accepted tree: {gate}")
        if gate_path.suffix.lower() == ".ps1":
            if os.name != "nt":
                executable = "pwsh"
            else:
                executable = "pwsh"
            run([executable, "-NoProfile", "-File", str(gate_path)], cwd=clone)
        else:
            run(["bash", str(gate_path)], cwd=clone)
        print_result(f"Historical checkpoint revalidates: {args.checkpoint}", True)

    print("\nHistorical checkpoint validation PASSED.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ISRASError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
