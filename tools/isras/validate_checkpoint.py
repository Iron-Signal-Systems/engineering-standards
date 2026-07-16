#!/usr/bin/env python3
from __future__ import annotations

import argparse
import os
import sys
import tempfile
from dataclasses import dataclass
from pathlib import Path

from common import ISRASError, load_json, print_result, repository_root, run


@dataclass(frozen=True)
class HistoricalToolEnvironment:
    bootstrap: Path
    venv: Path
    python: Path
    command: tuple[str, ...]


def historical_environment_spec(
    clone: Path,
    platform_name: str | None = None,
) -> HistoricalToolEnvironment:
    platform_name = os.name if platform_name is None else platform_name
    venv = clone / ".isras-tools-venv"
    if platform_name == "nt":
        bootstrap = clone / "tools/environment/Bootstrap-Tools.ps1"
        python = venv / "Scripts/python.exe"
        command = (
            "pwsh",
            "-NoProfile",
            "-File",
            str(bootstrap),
            "-VenvPath",
            str(venv),
        )
    else:
        bootstrap = clone / "tools/environment/bootstrap_tools.sh"
        python = venv / "bin/python"
        command = ("bash", str(bootstrap))
    return HistoricalToolEnvironment(bootstrap, venv, python, command)


def create_historical_environment(
    clone: Path,
    platform_name: str | None = None,
) -> HistoricalToolEnvironment:
    specification = historical_environment_spec(clone, platform_name)
    if not specification.bootstrap.is_file():
        raise ISRASError(
            "historical tool bootstrap is missing from accepted tree: "
            f"{specification.bootstrap.relative_to(clone)}"
        )

    environment = os.environ.copy()
    environment["ISRAS_TOOLS_VENV"] = str(specification.venv)
    run(
        list(specification.command),
        cwd=clone,
        env=environment,
    )
    if not specification.python.is_file():
        raise ISRASError(
            "historical tool bootstrap did not create its declared Python: "
            f"{specification.python}"
        )

    print_result(
        "Historical checkpoint tool environment created",
        True,
        str(specification.python),
    )
    return specification


def checkpoint_gate_environment(
    specification: HistoricalToolEnvironment,
) -> dict[str, str]:
    environment = os.environ.copy()
    environment["ISRAS_TOOLS_VENV"] = str(specification.venv)
    environment["ISRAS_PYTHON"] = str(specification.python)
    return environment


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--checkpoint", required=True)
    args = parser.parse_args()
    repo_root = repository_root(args.repo_root)

    manifest = load_json(repo_root / "REPOSITORY-ASSURANCE.json")
    registry_path = repo_root / manifest["checkpoint_registry"]
    registry = load_json(registry_path)
    checkpoints = registry.get("checkpoints", {})
    record = checkpoints.get(args.checkpoint)
    if not isinstance(record, dict):
        raise ISRASError(f"unknown checkpoint: {args.checkpoint}")

    commit = record["commit"]
    gate = record["gate"]
    branch = record["required_branch_name"]
    origin = manifest["canonical_origin"]

    with tempfile.TemporaryDirectory(prefix=f"isras-{args.checkpoint}-") as temporary:
        clone = Path(temporary) / "repository"
        run(["git", "clone", "--no-local", origin, str(clone)])
        run(["git", "checkout", "-B", branch, commit], cwd=clone)

        actual_commit = run(
            ["git", "rev-parse", "HEAD"],
            cwd=clone,
            capture=True,
        ).stdout.strip()
        if actual_commit != commit:
            raise ISRASError(
                "historical checkpoint checkout mismatch: "
                f"expected={commit} observed={actual_commit}"
            )
        actual_branch = run(
            ["git", "branch", "--show-current"],
            cwd=clone,
            capture=True,
        ).stdout.strip()
        if actual_branch != branch:
            raise ISRASError(
                "historical checkpoint branch mismatch: "
                f"expected={branch} observed={actual_branch}"
            )
        print_result("Historical checkpoint exact source checked out", True, commit)

        gate_path = clone / gate
        if not gate_path.exists():
            raise ISRASError(f"historical gate is missing from accepted tree: {gate}")

        tool_environment = create_historical_environment(clone)
        gate_environment = checkpoint_gate_environment(tool_environment)
        if gate_path.suffix.lower() == ".ps1":
            run(
                ["pwsh", "-NoProfile", "-File", str(gate_path)],
                cwd=clone,
                env=gate_environment,
            )
        else:
            run(
                ["bash", str(gate_path)],
                cwd=clone,
                env=gate_environment,
            )
        print_result(f"Historical checkpoint revalidates: {args.checkpoint}", True)

    print("\nHistorical checkpoint validation PASSED.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ISRASError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
