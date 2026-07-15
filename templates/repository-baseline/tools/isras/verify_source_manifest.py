#!/usr/bin/env python3
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

from common import ISRASError, load_json, print_result, repository_root
from generate_source_manifest import digest_path, tracked_paths

LINE = re.compile(r"^([0-9a-f]{64})  (.+)$")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--manifest")
    args = parser.parse_args()
    repo_root = repository_root(args.repo_root)
    assurance = load_json(repo_root / "REPOSITORY-ASSURANCE.json")
    relative = args.manifest or assurance.get("source_manifest", "SOURCE-SHA256SUMS.txt")
    manifest_path = repo_root / relative
    if not manifest_path.is_file():
        raise ISRASError(f"source manifest is missing: {relative}")

    expected: dict[str, str] = {}
    for number, line in enumerate(manifest_path.read_text(encoding="utf-8").splitlines(), 1):
        match = LINE.fullmatch(line)
        if not match:
            raise ISRASError(f"invalid source manifest line {number}: {line!r}")
        digest, path = match.groups()
        if path in expected:
            raise ISRASError(f"duplicate source manifest path: {path}")
        expected[path] = digest

    actual_paths = tracked_paths(repo_root, relative)
    expected_paths = sorted(expected)
    if actual_paths != expected_paths:
        missing = sorted(set(actual_paths) - set(expected_paths))
        extra = sorted(set(expected_paths) - set(actual_paths))
        raise ISRASError(
            "source manifest path set does not match tracked files; "
            f"missing={missing} extra={extra}"
        )

    mismatches = []
    for path in actual_paths:
        actual = digest_path(repo_root / path)
        if actual != expected[path]:
            mismatches.append(path)
    if mismatches:
        raise ISRASError("source manifest digest mismatch: " + ", ".join(mismatches))

    print_result(f"Source manifest verifies {len(actual_paths)} tracked file(s)", True)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ISRASError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
