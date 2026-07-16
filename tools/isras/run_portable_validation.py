#!/usr/bin/env python3
"""Run portable validation with stage-level, machine-readable diagnostics."""
from __future__ import annotations

import argparse
import os
import shlex
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class Stage:
    name: str
    failure_code: str
    relative_validator: str
    arguments: tuple[str, ...] = ()


STAGES = (
    Stage("history-preflight", "ISRAS-PORTABLE-HISTORY-001", "tools/isras/prepare_portable_history.py"),
    Stage("environment-profile", "ISRAS-PORTABLE-ENVIRONMENT-001", "tools/isras/doctor.py", ("--profile", "portable")),
    Stage("policy", "ISRAS-PORTABLE-POLICY-001", "tools/isras/validate_policy.py"),
    Stage("release-state", "ISRAS-PORTABLE-RELEASE-STATE-001", "tools/isras/validate_release_state.py"),
    Stage("project-checks", "ISRAS-PORTABLE-PROJECT-001", "tools/isras/portable_project_checks.py"),
)


def git_head(root: Path) -> str:
    result = subprocess.run(
        ["git", "rev-parse", "HEAD"],
        cwd=root,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    return result.stdout.strip() if result.returncode == 0 else "UNRESOLVED"


def annotation_escape(value: str) -> str:
    return value.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")


def build_command(stage: Stage, root: Path) -> list[str]:
    bootstrap = root / "tools/isras/invoke_repo_tool.py"
    return [
        sys.executable,
        "-I",
        str(bootstrap),
        "--repo-root",
        str(root),
        "--tool",
        stage.relative_validator,
        "--",
        "--repo-root",
        str(root),
        *stage.arguments,
    ]


def print_context(stage: Stage, command: list[str], root: Path) -> None:
    print("BEGIN: portable validation stage")
    print(f"stage={stage.name}")
    print(f"validator={stage.relative_validator}")
    print(f"tested_commit={git_head(root)}")
    print(f"workflow={os.environ.get('GITHUB_WORKFLOW', 'LOCAL')}")
    print(f"job={os.environ.get('GITHUB_JOB', 'LOCAL')}")
    print(f"runner_os={os.environ.get('RUNNER_OS', os.name)}")
    print(f"command={shlex.join(command)}")


def run_stage(stage: Stage, root: Path) -> int:
    validator = root / stage.relative_validator
    bootstrap = root / "tools/isras/invoke_repo_tool.py"
    command = build_command(stage, root)
    in_actions = os.environ.get("GITHUB_ACTIONS") == "true"
    if in_actions:
        print(f"::group::ISRAS portable stage: {stage.name}")
    print_context(stage, command, root)

    missing = None
    if not bootstrap.is_file():
        missing = "tools/isras/invoke_repo_tool.py"
    elif not validator.is_file():
        missing = stage.relative_validator
    if missing is not None:
        return_code = 1
        print("FAIL: portable validation stage prerequisite is missing", file=sys.stderr)
        print(f"failure_code={stage.failure_code}", file=sys.stderr)
        print(f"stage={stage.name}", file=sys.stderr)
        print(f"missing_path={missing}", file=sys.stderr)
    else:
        with subprocess.Popen(
            command,
            cwd=root,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            bufsize=1,
        ) as process:
            assert process.stdout is not None
            for line in process.stdout:
                print(line, end="")
            return_code = process.wait()

    if return_code == 0:
        print("PASS: portable validation stage")
        print(f"stage={stage.name}")
        print("exit_code=0")
        if in_actions:
            print("::endgroup::")
        return 0

    print("FAIL: portable validation stage failed", file=sys.stderr)
    print(f"failure_code={stage.failure_code}", file=sys.stderr)
    print(f"stage={stage.name}", file=sys.stderr)
    print(f"validator={stage.relative_validator}", file=sys.stderr)
    print("bootstrap=tools/isras/invoke_repo_tool.py", file=sys.stderr)
    print(f"tested_commit={git_head(root)}", file=sys.stderr)
    print(f"workflow={os.environ.get('GITHUB_WORKFLOW', 'LOCAL')}", file=sys.stderr)
    print(f"job={os.environ.get('GITHUB_JOB', 'LOCAL')}", file=sys.stderr)
    print(f"runner_os={os.environ.get('RUNNER_OS', os.name)}", file=sys.stderr)
    print(f"command={shlex.join(command)}", file=sys.stderr)
    print(f"exit_code={return_code}", file=sys.stderr)
    if in_actions:
        message = (
            f"stage={stage.name}; validator={stage.relative_validator}; "
            f"tested_commit={git_head(root)}; exit_code={return_code}; "
            f"failure_code={stage.failure_code}"
        )
        print(
            f"::error title=ISRAS portable validation failed::{annotation_escape(message)}",
            file=sys.stderr,
        )
        print("::endgroup::")
    return return_code


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    args = parser.parse_args()
    root = Path(args.repo_root).resolve()

    print("ISRAS structured portable validation")
    print(f"repository_root={root}")
    print(f"python={sys.executable}")
    print(f"tested_commit={git_head(root)}")
    print("bootstrap=tools/isras/invoke_repo_tool.py")

    for stage in STAGES:
        result = run_stage(stage, root)
        if result != 0:
            return result

    print("Portable validation PASSED.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except OSError as exc:
        print("FAIL: portable validation runner could not execute", file=sys.stderr)
        print("failure_code=ISRAS-PORTABLE-RUNNER-001", file=sys.stderr)
        print("validator=tools/isras/run_portable_validation.py", file=sys.stderr)
        print(f"exception_type={type(exc).__name__}", file=sys.stderr)
        print(f"exception={' '.join(str(exc).split())}", file=sys.stderr)
        raise SystemExit(1)
