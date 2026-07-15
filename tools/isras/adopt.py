#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import re
import shutil
import stat
import subprocess
import sys
from pathlib import Path


TOKENS = {
    "__REPOSITORY__": "repository",
    "__CANONICAL_ORIGIN__": "canonical_origin",
    "__DEVELOPMENT_BRANCH__": "development_branch",
    "__RELEASE_BRANCH__": "release_branch",
    "__PROFILE__": "profile",
}


def transform(
    text: str,
    values: dict[str, str],
    suffix: str,
) -> str:
    escaped_string_suffixes = {".json", ".py"}

    for token, key in TOKENS.items():
        value = values[key]

        if suffix.lower() in escaped_string_suffixes:
            # Tokens in JSON and Python templates appear inside
            # double-quoted string literals. Encode their contents so
            # Windows backslashes, quotes, and control characters cannot
            # produce invalid generated files.
            value = json.dumps(value, ensure_ascii=False)[1:-1]

        text = text.replace(token, value)

    text = text.replace(
        "UNPINNED-BOOTSTRAP",
        values["standard_commit"],
    )
    return text


def resolve_standard_commit(standards_root: Path) -> str:
    result = subprocess.run(
        ["git", "rev-parse", "HEAD"],
        cwd=standards_root,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        check=False,
    )
    commit = result.stdout.strip()
    return commit if re.fullmatch(r"[0-9a-f]{40}", commit) else "UNPINNED-BOOTSTRAP"


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--target", required=True)
    parser.add_argument("--repository", required=True)
    parser.add_argument("--canonical-origin", required=True)
    parser.add_argument("--development-branch", default="dev")
    parser.add_argument("--release-branch", default="main")
    parser.add_argument("--profile", default="general")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument(
        "--skip-existing",
        action="store_true",
        help="write only missing baseline files and report existing paths for manual merge",
    )
    parser.add_argument("--force", action="store_true")
    args = parser.parse_args()

    standards_root = Path(__file__).resolve().parents[2]
    template_root = standards_root / "templates/repository-baseline"
    target = Path(args.target).expanduser().resolve()
    if not (target / ".git").exists():
        raise SystemExit(f"Target is not a Git repository: {target}")

    values = {
        "repository": args.repository,
        "canonical_origin": args.canonical_origin,
        "development_branch": args.development_branch,
        "release_branch": args.release_branch,
        "profile": args.profile,
        "standard_commit": resolve_standard_commit(standards_root),
    }

    sources: list[tuple[Path, Path]] = []
    for source in template_root.rglob("*"):
        if not source.is_file():
            continue
        if "__pycache__" in source.parts or source.suffix in {".pyc", ".pyo"}:
            continue
        sources.append((source, target / source.relative_to(template_root)))
    conflicts = [destination for _, destination in sources if destination.exists()]
    if args.dry_run:
        for source, destination in sources:
            state = "EXISTS" if destination.exists() else "PLAN"
            print(f"{state}: {destination.relative_to(target)}")
        if conflicts:
            print(
                "\nExisting paths require deliberate merge. Use --skip-existing to write "
                "only missing files, or --force only after full review."
            )
        return 0

    if conflicts and not args.force and not args.skip_existing:
        print("Refusing to overwrite existing files. Merge these paths deliberately:")
        for path in conflicts:
            print(f"  {path.relative_to(target)}")
        print(
            "\nUse --skip-existing to write only missing files. Use --force only "
            "after reviewing the existing content."
        )
        return 2

    for source, destination in sources:
        if destination.exists() and args.skip_existing and not args.force:
            print(f"SKIP: {destination.relative_to(target)}")
            continue
        print(f"WRITE: {destination.relative_to(target)}")
        destination.parent.mkdir(parents=True, exist_ok=True)
        text = source.read_text(encoding="utf-8")
        destination.write_text(
            transform(text, values, source.suffix),
            encoding="utf-8",
            newline="\n",
        )
        if source.stat().st_mode & stat.S_IXUSR:
            destination.chmod(destination.stat().st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)

    if not args.dry_run:
        print("\nBaseline applied. Review and customize every generated file before committing.")
        if values["standard_commit"] == "UNPINNED-BOOTSTRAP":
            print("The standard commit is UNPINNED-BOOTSTRAP because the standards source is not a committed Git tree.")
        else:
            print(f"The adopted standard is pinned to {values['standard_commit']}.")
        print("RECORDED adoption does not require a source manifest until the repository advances to REPRODUCIBLE.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
