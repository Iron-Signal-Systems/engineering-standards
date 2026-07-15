#!/usr/bin/env python3
from __future__ import annotations

import argparse
import hashlib
import os
import sys
from pathlib import Path

from common import ISRASError, git, repository_root, run, sha256_file


def tracked_paths(repo_root: Path, output_relative: str) -> list[str]:
    raw = run(["git", "ls-files", "-z"], cwd=repo_root, capture=True).stdout
    paths = [value for value in raw.split("\0") if value]
    for value in paths:
        if "\n" in value or "\r" in value:
            raise ISRASError(f"tracked path contains a prohibited newline: {value!r}")
    return sorted(path for path in paths if path != output_relative)


def digest_path(path: Path) -> str:
    if path.is_symlink():
        return hashlib.sha256(os.readlink(path).encode("utf-8")).hexdigest()
    if not path.is_file():
        raise ISRASError(f"tracked source path is not a regular file: {path}")
    return sha256_file(path)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--output", default="SOURCE-SHA256SUMS.txt")
    parser.add_argument(
        "--require-clean",
        action="store_true",
        help="Refuse generation when the working tree contains changes other than the output file.",
    )
    args = parser.parse_args()
    repo_root = repository_root(args.repo_root)
    output = Path(args.output)
    output_relative = output.as_posix()
    if output.is_absolute():
        try:
            output_relative = output.relative_to(repo_root).as_posix()
        except ValueError as exc:
            raise ISRASError("source manifest output must be inside the repository") from exc
        output_path = output
    else:
        output_path = repo_root / output

    if args.require_clean:
        changed = [
            line for line in git(repo_root, "status", "--porcelain").splitlines()
            if line[3:].replace("\\", "/") != output_relative
        ]
        if changed:
            raise ISRASError("working tree is not clean enough for formal manifest generation")

    lines = []
    for relative in tracked_paths(repo_root, output_relative):
        path = repo_root / relative
        lines.append(f"{digest_path(path)}  {relative}")

    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(f"Wrote {len(lines)} tracked source hashes to {output_path}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ISRASError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
