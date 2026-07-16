#!/usr/bin/env python3
"""Generate a SHA-512 manifest from the Git index or an exact commit tree."""
from __future__ import annotations

import argparse
import hashlib
import subprocess
import sys
from pathlib import Path

EXCLUDED_DEFAULTS = {"SOURCE-SHA256SUMS.txt", "SOURCE-SHA512SUMS.txt"}


def run(root: Path, *args: str, binary: bool = False) -> bytes | str:
    result = subprocess.run(
        ["git", *args],
        cwd=root,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=not binary,
        check=False,
    )
    if result.returncode != 0:
        stderr = result.stderr.decode("utf-8", errors="replace") if binary else result.stderr
        raise RuntimeError(stderr.strip() or f"git {' '.join(args)} failed")
    return result.stdout


def repository_root(value: str) -> Path:
    root = Path(value).resolve()
    result = run(root, "rev-parse", "--show-toplevel")
    return Path(str(result).strip()).resolve()


def tracked_index_entries(root: Path, excluded: set[str]) -> list[tuple[str, str]]:
    raw = run(root, "ls-files", "-s", "-z", binary=True)
    assert isinstance(raw, bytes)
    entries: list[tuple[str, str]] = []
    for record in raw.split(b"\0"):
        if not record:
            continue
        metadata, raw_path = record.split(b"\t", 1)
        mode, object_id, stage = metadata.decode("ascii").split()
        if stage != "0":
            raise RuntimeError(f"unmerged index entry is not allowed: {raw_path!r}")
        path = raw_path.decode("utf-8", errors="strict").replace("\\", "/")
        if "\n" in path or "\r" in path:
            raise RuntimeError(f"tracked path contains a prohibited newline: {path!r}")
        if path not in excluded:
            entries.append((path, object_id))
    return sorted(entries)


def tracked_commit_entries(root: Path, commit: str, excluded: set[str]) -> list[tuple[str, str]]:
    raw = run(root, "ls-tree", "-r", "-z", commit, binary=True)
    assert isinstance(raw, bytes)
    entries: list[tuple[str, str]] = []
    for record in raw.split(b"\0"):
        if not record:
            continue
        metadata, raw_path = record.split(b"\t", 1)
        _mode, kind, object_id = metadata.decode("ascii").split()
        if kind != "blob":
            continue
        path = raw_path.decode("utf-8", errors="strict").replace("\\", "/")
        if "\n" in path or "\r" in path:
            raise RuntimeError(f"tracked path contains a prohibited newline: {path!r}")
        if path not in excluded:
            entries.append((path, object_id))
    return sorted(entries)


def blob_sha512(root: Path, object_id: str) -> str:
    data = run(root, "cat-file", "blob", object_id, binary=True)
    assert isinstance(data, bytes)
    return hashlib.sha512(data).hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--output", default="SOURCE-SHA512SUMS.txt")
    parser.add_argument("--exclude", action="append", default=[])
    parser.add_argument(
        "--source",
        choices=["index", "commit"],
        default="index",
        help="Hash staged index blobs or an exact commit tree.",
    )
    parser.add_argument("--commit", help="Commit to hash when --source=commit.")
    args = parser.parse_args()

    root = repository_root(args.repo_root)
    output = Path(args.output)
    output_path = output if output.is_absolute() else root / output
    try:
        output_relative = output_path.relative_to(root).as_posix()
    except ValueError as exc:
        raise RuntimeError("manifest output must remain inside the repository") from exc

    excluded = set(EXCLUDED_DEFAULTS)
    excluded.add(output_relative)
    excluded.update(value.replace("\\", "/") for value in args.exclude)

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

    lines = [f"{blob_sha512(root, object_id)}  {path}" for path, object_id in entries]
    output_path.write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(f"Wrote {len(lines)} tracked SHA-512 hashes from {source_label} to {output_path}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, RuntimeError, UnicodeDecodeError, ValueError) as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
