#!/usr/bin/env python3
from __future__ import annotations

import argparse
import sys
from pathlib import Path

from common import ISRASError, repository_root, sha256_file


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--output", default="SOURCE-SHA256SUMS.txt")
    args = parser.parse_args()
    repo_root = repository_root(args.repo_root)
    output = (repo_root / args.output).resolve()
    lines = []
    for path in sorted(repo_root.rglob("*")):
        if not path.is_file() or ".git" in path.parts or "__pycache__" in path.parts:
            continue
        if path.resolve() == output or path.suffix in {".pyc", ".pyo"}:
            continue
        relative = path.relative_to(repo_root).as_posix()
        lines.append(f"{sha256_file(path)}  {relative}")
    output.write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(output)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ISRASError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
