#!/usr/bin/env python3
"""Verify a SHA-512 manifest against the Git index or an exact commit tree."""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

from generate_sha512_manifest import (
    EXCLUDED_DEFAULTS,
    blob_sha512,
    repository_root,
    run,
    tracked_commit_entries,
    tracked_index_entries,
)

SHA512 = re.compile(r"^[0-9a-f]{128}$")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--manifest", default="SOURCE-SHA512SUMS.txt")
    parser.add_argument("--source", choices=["index", "commit"], default="index")
    parser.add_argument("--commit")
    args = parser.parse_args()

    root = repository_root(args.repo_root)
    manifest_path = Path(args.manifest)
    if not manifest_path.is_absolute():
        manifest_path = root / manifest_path
    if not manifest_path.is_file():
        raise RuntimeError(f"SHA-512 manifest does not exist: {manifest_path}")
    try:
        manifest_relative = manifest_path.relative_to(root).as_posix()
    except ValueError as exc:
        raise RuntimeError("manifest must remain inside the repository") from exc

    expected: dict[str, str] = {}
    for number, line in enumerate(manifest_path.read_text(encoding="utf-8").splitlines(), 1):
        parts = line.split("  ", 1)
        if len(parts) != 2 or not SHA512.fullmatch(parts[0]):
            raise RuntimeError(f"invalid SHA-512 manifest line {number}: {line!r}")
        path = parts[1]
        if path.startswith("/") or "\\" in path or any(part in {"", ".", ".."} for part in Path(path).parts):
            raise RuntimeError(f"unsafe SHA-512 manifest path on line {number}: {path!r}")
        if path in expected:
            raise RuntimeError(f"duplicate SHA-512 manifest path: {path}")
        expected[path] = parts[0]

    excluded = set(EXCLUDED_DEFAULTS)
    excluded.add(manifest_relative)

    if args.source == "commit":
        if not args.commit:
            raise RuntimeError("--commit is required with --source=commit")
        commit = str(run(root, "rev-parse", "--verify", f"{args.commit}^{{commit}}")).strip()
        entries = tracked_commit_entries(root, commit, excluded)
        source_label = commit
    else:
        if args.commit:
            raise RuntimeError("--commit is only valid with --source=commit")
        entries = tracked_index_entries(root, excluded)
        source_label = "Git index"

    actual_paths = [path for path, _object_id in entries]
    if sorted(expected) != actual_paths:
        missing = sorted(set(actual_paths) - set(expected))
        extra = sorted(set(expected) - set(actual_paths))
        raise RuntimeError(
            f"SHA-512 manifest path set mismatch; missing={missing!r}; extra={extra!r}"
        )

    mismatched = [
        path
        for path, object_id in entries
        if blob_sha512(root, object_id) != expected[path]
    ]
    if mismatched:
        raise RuntimeError("SHA-512 digest mismatch: " + ", ".join(mismatched))

    print(f"PASS: SHA-512 manifest verified for {len(entries)} tracked files from {source_label}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, RuntimeError, UnicodeDecodeError, ValueError) as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
