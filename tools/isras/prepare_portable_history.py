#!/usr/bin/env python3
"""Acquire and verify Git history required by portable ISRAS validation."""
from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import Iterable

SHA_RE = re.compile(r"^[0-9a-f]{40}$")
FAILURE_CODE = "ISRAS-CI-HISTORY-001"


@dataclass
class Requirement:
    commit: str
    purposes: set[str] = field(default_factory=set)
    fetch_refs: set[str] = field(default_factory=set)


def git(
    root: Path,
    *args: str,
    extra_header: str | None = None,
) -> subprocess.CompletedProcess[str]:
    command = ["git"]
    if extra_header:
        command.extend(["-c", f"http.extraheader={extra_header}"])
    command.extend(args)
    return subprocess.run(
        command,
        cwd=root,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )


def load_object(path: Path) -> dict:
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise ValueError(f"JSON root must be an object: {path}")
    return data


def add_requirement(
    result: dict[str, Requirement],
    commit: str,
    purpose: str,
    fetch_ref: str | None = None,
) -> None:
    if not SHA_RE.fullmatch(commit):
        raise ValueError(f"invalid required commit in {purpose}: {commit!r}")
    requirement = result.setdefault(commit, Requirement(commit=commit))
    requirement.purposes.add(purpose)
    requirement.fetch_refs.add(fetch_ref or commit)


def discover_requirements(root: Path) -> list[Requirement]:
    result: dict[str, Requirement] = {}

    registry_path = root / "tools/validation/checkpoints.json"
    registry = load_object(registry_path)
    checkpoints = registry.get("checkpoints")
    if not isinstance(checkpoints, dict):
        raise ValueError("checkpoint registry lacks a checkpoints object")
    for tag, value in sorted(checkpoints.items()):
        if not isinstance(value, dict) or value.get("status") != "accepted":
            continue
        commit = value.get("commit")
        if not isinstance(commit, str):
            raise ValueError(f"accepted checkpoint {tag!r} lacks a commit")
        declared_tag = value.get("tag")
        fetch_ref = f"refs/tags/{declared_tag}" if isinstance(declared_tag, str) else commit
        add_requirement(result, commit, f"accepted checkpoint {tag}", fetch_ref)

    for path in sorted((root / "docs/acceptance").glob("*change-classification.json")):
        record = load_object(path)
        base = record.get("base_commit")
        if isinstance(base, str):
            add_requirement(
                result,
                base,
                f"classification base {path.relative_to(root).as_posix()}",
            )

    return [result[key] for key in sorted(result)]


def commit_exists(root: Path, commit: str) -> bool:
    return git(root, "cat-file", "-e", f"{commit}^{{commit}}").returncode == 0


def single_line(value: str) -> str:
    return " ".join(value.replace("\r", " ").replace("\n", " ").split())


def context(root: Path) -> dict[str, str]:
    head = git(root, "rev-parse", "HEAD")
    remote = git(root, "remote", "get-url", "origin")
    return {
        "tested_commit": head.stdout.strip() if head.returncode == 0 else "UNRESOLVED",
        "remote_url": remote.stdout.strip() if remote.returncode == 0 else "UNRESOLVED",
        "workflow": os.environ.get("GITHUB_WORKFLOW", "LOCAL"),
        "job": os.environ.get("GITHUB_JOB", "LOCAL"),
        "runner_os": os.environ.get("RUNNER_OS", os.name),
    }


def print_pass(requirement: Requirement, acquisition: str) -> None:
    print("PASS: required historical commit available")
    print(f"commit={requirement.commit}")
    print(f"purpose={' | '.join(sorted(requirement.purposes))}")
    print(f"acquisition={acquisition}")
    print("object_type=commit")


def print_failure(
    requirement: Requirement,
    details: dict[str, str],
    fetch_ref: str,
    process: subprocess.CompletedProcess[str] | None,
) -> None:
    print("FAIL: required historical commit unavailable", file=sys.stderr)
    print(f"failure_code={FAILURE_CODE}", file=sys.stderr)
    print("validator=tools/isras/prepare_portable_history.py", file=sys.stderr)
    print(f"workflow={details['workflow']}", file=sys.stderr)
    print(f"job={details['job']}", file=sys.stderr)
    print(f"runner_os={details['runner_os']}", file=sys.stderr)
    print(f"tested_commit={details['tested_commit']}", file=sys.stderr)
    print(f"remote_url={details['remote_url']}", file=sys.stderr)
    print(f"required_commit={requirement.commit}", file=sys.stderr)
    print(f"purpose={' | '.join(sorted(requirement.purposes))}", file=sys.stderr)
    print(f"fetch_ref={fetch_ref}", file=sys.stderr)
    print("observed=required commit object is unavailable", file=sys.stderr)
    if process is None:
        print("fetch_attempted=false", file=sys.stderr)
        print("fetch_exit_code=NOT_RUN", file=sys.stderr)
    else:
        print("fetch_attempted=true", file=sys.stderr)
        print(f"fetch_exit_code={process.returncode}", file=sys.stderr)
        print(f"fetch_stdout={single_line(process.stdout) or '<empty>'}", file=sys.stderr)
        print(f"fetch_stderr={single_line(process.stderr) or '<empty>'}", file=sys.stderr)


def acquire(
    root: Path,
    requirement: Requirement,
    remote: str,
    extra_header: str | None,
    allow_fetch: bool,
) -> bool:
    if commit_exists(root, requirement.commit):
        print_pass(requirement, "already_present")
        return True

    if not allow_fetch:
        print_failure(requirement, context(root), requirement.commit, None)
        return False

    last_process: subprocess.CompletedProcess[str] | None = None
    last_ref = requirement.commit
    for fetch_ref in sorted(requirement.fetch_refs):
        last_ref = fetch_ref
        process = git(
            root,
            "fetch",
            "--no-tags",
            "--depth=1",
            remote,
            fetch_ref,
            extra_header=extra_header,
        )
        last_process = process
        if process.returncode == 0 and commit_exists(root, requirement.commit):
            print_pass(requirement, f"fetched_exact:{fetch_ref}")
            return True

    print_failure(requirement, context(root), last_ref, last_process)
    return False


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--remote", default="origin")
    parser.add_argument("--no-fetch", action="store_true")
    args = parser.parse_args()

    root = Path(args.repo_root).resolve()
    if git(root, "rev-parse", "--show-toplevel").returncode != 0:
        raise ValueError(f"not a Git repository: {root}")

    details = context(root)
    requirements = discover_requirements(root)
    extra_header = os.environ.get("ISRAS_GIT_HTTP_EXTRAHEADER")
    if extra_header and any(character in extra_header for character in "\r\n"):
        raise ValueError("ISRAS_GIT_HTTP_EXTRAHEADER contains a prohibited newline")

    print("ISRAS portable Git history preflight")
    for key in ("tested_commit", "remote_url", "workflow", "job", "runner_os"):
        print(f"{key}={details[key]}")
    print(f"required_commit_count={len(requirements)}")

    failures = 0
    for requirement in requirements:
        if not acquire(
            root,
            requirement,
            args.remote,
            extra_header,
            not args.no_fetch,
        ):
            failures += 1

    if failures:
        print(f"Portable Git history preflight FAILED: failures={failures}.", file=sys.stderr)
        return 1
    print("Portable Git history preflight PASSED.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print("FAIL: portable Git history preflight could not execute", file=sys.stderr)
        print(f"failure_code={FAILURE_CODE}", file=sys.stderr)
        print("validator=tools/isras/prepare_portable_history.py", file=sys.stderr)
        print(f"exception_type={type(exc).__name__}", file=sys.stderr)
        print(f"exception={single_line(str(exc))}", file=sys.stderr)
        raise SystemExit(1)
