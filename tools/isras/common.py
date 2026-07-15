from __future__ import annotations

import hashlib
import json
import os
import subprocess
import sys
from pathlib import Path
from typing import Iterable, Sequence


class ISRASError(RuntimeError):
    pass


def load_json(path: Path) -> dict:
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as exc:
        raise ISRASError(f"required JSON file is missing: {path}") from exc
    except json.JSONDecodeError as exc:
        raise ISRASError(f"invalid JSON in {path}: {exc}") from exc
    if not isinstance(data, dict):
        raise ISRASError(f"JSON root must be an object: {path}")
    return data


def run(
    args: Sequence[str],
    *,
    cwd: Path | None = None,
    check: bool = True,
    capture: bool = False,
    env: dict[str, str] | None = None,
) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(
        list(args),
        cwd=str(cwd) if cwd else None,
        check=False,
        text=True,
        stdout=subprocess.PIPE if capture else None,
        stderr=subprocess.PIPE if capture else None,
        env=env,
    )
    if check and result.returncode != 0:
        detail = ""
        if capture:
            detail = f"\nstdout:\n{result.stdout}\nstderr:\n{result.stderr}"
        raise ISRASError(
            f"command failed ({result.returncode}): {' '.join(args)}{detail}"
        )
    return result


def git(repo_root: Path, *args: str, capture: bool = True) -> str:
    result = run(["git", *args], cwd=repo_root, capture=capture)
    return result.stdout.strip() if capture else ""


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def repository_root(value: str | Path) -> Path:
    path = Path(value).expanduser().resolve()
    if not (path / ".git").exists():
        try:
            top = run(
                ["git", "rev-parse", "--show-toplevel"],
                cwd=path,
                capture=True,
            ).stdout.strip()
        except Exception as exc:
            raise ISRASError(f"not a Git repository: {path}") from exc
        path = Path(top).resolve()
    return path


def print_result(label: str, ok: bool, detail: str = "") -> None:
    state = "PASS" if ok else "FAIL"
    suffix = f": {detail}" if detail else ""
    stream = sys.stdout if ok else sys.stderr
    print(f"{state}: {label}{suffix}", file=stream)


def executable_files(repo_root: Path) -> Iterable[Path]:
    for base in ("tools", ".github"):
        start = repo_root / base
        if not start.exists():
            continue
        for path in start.rglob("*"):
            if path.is_file() and path.suffix.lower() in {
                ".sh", ".ps1", ".py", ".yml", ".yaml", ".json"
            }:
                yield path
