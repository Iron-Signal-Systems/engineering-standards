#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from common import ISRASError, git, repository_root, sha256_file


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--paths-file", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--repository")
    args = parser.parse_args()
    repo_root = repository_root(args.repo_root)

    manifest = json.loads((repo_root / "REPOSITORY-ASSURANCE.json").read_text())
    repository = args.repository or manifest["repository"]
    source_commit = git(repo_root, "rev-parse", "HEAD")
    listed = []
    for line in (repo_root / args.paths_file).read_text(encoding="utf-8").splitlines():
        text = line.strip()
        if not text or text.startswith("#"):
            continue
        listed.append(text)

    migrations = []
    for order, relative in enumerate(listed):
        path = repo_root / relative
        if not path.is_file():
            raise ISRASError(f"migration is missing: {relative}")
        migrations.append({
            "id": path.stem,
            "path": relative,
            "sha256": sha256_file(path),
            "order": order,
            "introduced_by": None,
            "transactional": None,
        })

    data = {
        "schema_version": "ISRAS-MIGRATIONS-V1",
        "repository": repository,
        "source_commit": source_commit,
        "migrations": migrations,
    }
    output = repo_root / args.output
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
    print(output)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ISRASError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
