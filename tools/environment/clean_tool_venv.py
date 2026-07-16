#!/usr/bin/env python3
"""Remove bootstrap-only distributions so the release tool environment can be exact."""
from __future__ import annotations

import argparse
import importlib.metadata as metadata
import re
import subprocess
import sys


def normalized(value: str) -> str:
    return re.sub(r"[-_.]+", "-", value).lower()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--keep", action="append", default=["pip"])
    args = parser.parse_args()

    keep = {normalized(value) for value in args.keep}
    installed: dict[str, str] = {}
    for distribution in metadata.distributions():
        name = distribution.metadata.get("Name")
        if not name:
            continue
        installed[normalized(name)] = name

    remove = sorted(name for key, name in installed.items() if key not in keep)
    if not remove:
        print("PASS: no bootstrap-only distributions require removal")
        return 0

    result = subprocess.run(
        [
            sys.executable,
            "-I",
            "-m",
            "pip",
            "--isolated",
            "uninstall",
            "--yes",
            *remove,
        ],
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if result.returncode != 0:
        print(result.stdout, end="")
        print(result.stderr, end="", file=sys.stderr)
        return result.returncode
    print(f"Removed bootstrap-only distributions: {', '.join(remove)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
