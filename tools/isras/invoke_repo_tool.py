#!/usr/bin/env python3
"""Invoke a repository-owned ISRAS Python tool under isolated Python.

Python's ``-I`` option intentionally removes the script directory from
``sys.path``. Repository tools import the sibling ``common.py`` module, so an
explicit, bounded bootstrap is required instead of relying on ambient import
paths.
"""
from __future__ import annotations

import argparse
import os
import runpy
import sys
from pathlib import Path

FAILURE_CODE = "ISRAS-REPO-TOOL-BOOTSTRAP-001"


def clean(value: object) -> str:
    return " ".join(str(value).split())


def fail(message: str, *, root: Path | None = None, tool: Path | None = None) -> None:
    print("FAIL: isolated repository tool bootstrap failed", file=sys.stderr)
    print(f"failure_code={FAILURE_CODE}", file=sys.stderr)
    print(f"message={clean(message)}", file=sys.stderr)
    if root is not None:
        print(f"repository_root={root}", file=sys.stderr)
    if tool is not None:
        print(f"tool={tool}", file=sys.stderr)
    raise SystemExit(1)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", required=True)
    parser.add_argument("--tool", required=True)
    parser.add_argument("tool_arguments", nargs=argparse.REMAINDER)
    args = parser.parse_args()

    root = Path(args.repo_root).expanduser().resolve()
    if not (root / ".git").exists() and not (root / "tools/isras").is_dir():
        fail("repository root is not recognizable", root=root)

    tool = (root / args.tool).resolve()
    try:
        tool.relative_to(root)
    except ValueError:
        fail("tool path escapes repository root", root=root, tool=tool)
    if not tool.is_file():
        fail("repository tool does not exist", root=root, tool=tool)

    isras_module_dir = (root / "tools/isras").resolve()
    if not isras_module_dir.is_dir():
        fail("tools/isras module directory is missing", root=root, tool=tool)

    # Bounded import path: only the repository-owned sibling-module directory is
    # added. Ambient PYTHONPATH and user-site paths remain disabled by Python -I.
    sys.path.insert(0, str(isras_module_dir))
    sys.dont_write_bytecode = True

    tool_arguments = list(args.tool_arguments)
    if tool_arguments[:1] == ["--"]:
        tool_arguments = tool_arguments[1:]
    sys.argv = [str(tool), *tool_arguments]
    os.chdir(root)
    runpy.run_path(str(tool), run_name="__main__")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except SystemExit:
        raise
    except (ImportError, OSError, RuntimeError, ValueError) as exc:
        print("FAIL: isolated repository tool bootstrap failed", file=sys.stderr)
        print(f"failure_code={FAILURE_CODE}", file=sys.stderr)
        print(f"exception_type={type(exc).__name__}", file=sys.stderr)
        print(f"exception={clean(exc)}", file=sys.stderr)
        raise SystemExit(1)
