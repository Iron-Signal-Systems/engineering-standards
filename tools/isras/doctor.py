#!/usr/bin/env python3
from __future__ import annotations

import argparse
import platform
import shutil
import sys
from pathlib import Path

from common import ISRASError, git, load_json, print_result, repository_root


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--profile", default="portable")
    args = parser.parse_args()
    repo_root = repository_root(args.repo_root)

    manifest = load_json(repo_root / "REPOSITORY-ASSURANCE.json")
    profile_path = repo_root / "tools/environment/profiles" / f"{args.profile}.json"
    profile = load_json(profile_path)

    failures = 0
    print(f"Repository: {manifest.get('repository')}")
    print(f"Host: {platform.node()}")
    print(f"Operating system: {platform.system()} {platform.release()}")
    print(f"Architecture: {platform.machine()}")
    print(f"Profile: {profile.get('profile')} ({profile.get('classification')})")

    origin = git(repo_root, "remote", "get-url", "origin")
    expected_origin = manifest.get("canonical_origin")
    origin_ok = origin == expected_origin
    print_result("Canonical origin", origin_ok, origin)
    failures += not origin_ok

    status = git(repo_root, "status", "--porcelain")
    clean = status == ""
    print_result("Working tree is clean", clean)
    if not clean:
        print("INFO: a dirty tree is allowed for development checks but not acceptance.")

    for command in profile.get("required_commands", []):
        ok = shutil.which(command) is not None
        print_result(f"Required command available: {command}", ok)
        failures += not ok

    for command in profile.get("optional_commands", []):
        ok = shutil.which(command) is not None
        state = "AVAILABLE" if ok else "NOT_AVAILABLE"
        print(f"INFO: Optional command {command}: {state}")

    if failures:
        print(f"\nEnvironment profile FAILED with {failures} missing or mismatched requirement(s).", file=sys.stderr)
        return 1

    print("\nEnvironment profile PASSED.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ISRASError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
