#!/usr/bin/env python3
"""Prepare a reviewable, SHA-512-bound wheelhouse candidate for one platform."""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
import platform
import subprocess
import sys
import sysconfig
import tempfile
import urllib.request
from pathlib import Path
from typing import Any
from urllib.parse import urlparse, urlunparse

from jsonschema import Draft202012Validator, FormatChecker


def digest(path: Path, algorithm: str = "sha512") -> str:
    hasher = hashlib.new(algorithm)
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b""):
            hasher.update(block)
    return hasher.hexdigest()


def clean_environment() -> dict[str, str]:
    env = {
        key: value
        for key, value in os.environ.items()
        if not key.upper().startswith("PIP_")
        and key not in {"PYTHONPATH", "PYTHONHOME"}
    }
    env["PIP_CONFIG_FILE"] = os.devnull
    env["PYTHONNOUSERSITE"] = "1"
    return env


def active_pip_version() -> str:
    import pip  # type: ignore
    return pip.__version__


def run_report(requirement_args: list[str], report_path: Path) -> dict[str, Any]:
    command = [
        sys.executable,
        "-I",
        "-m",
        "pip",
        "--isolated",
        "install",
        "--disable-pip-version-check",
        "--dry-run",
        "--ignore-installed",
        "--only-binary=:all:",
        "--report",
        str(report_path),
        *requirement_args,
    ]
    result = subprocess.run(
        command,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=clean_environment(),
        check=False,
    )
    if result.returncode != 0:
        raise RuntimeError(result.stdout + result.stderr)
    return json.loads(report_path.read_text(encoding="utf-8"))


def source_hashes(download_info: dict[str, Any]) -> dict[str, str]:
    archive = download_info.get("archive_info") or {}
    hashes = archive.get("hashes") or {}
    if not isinstance(hashes, dict):
        hashes = {}
    normalized = {
        str(name).lower(): str(value).lower()
        for name, value in hashes.items()
        if isinstance(name, str) and isinstance(value, str)
    }
    if not normalized:
        legacy = archive.get("hash")
        if isinstance(legacy, str) and "=" in legacy:
            name, value = legacy.split("=", 1)
            normalized[name.lower()] = value.lower()
    if not normalized:
        raise RuntimeError("resolved artifact has no upstream archive hash")
    for name, value in normalized.items():
        try:
            hashlib.new(name)
        except ValueError as exc:
            raise RuntimeError(f"unsupported upstream hash algorithm: {name}") from exc
        if not value or any(character not in "0123456789abcdef" for character in value):
            raise RuntimeError(f"invalid upstream {name} digest")
    return normalized


def sanitized_url(value: str) -> str:
    parsed = urlparse(value)
    if parsed.scheme not in {"https", "file"}:
        raise RuntimeError(f"artifact URL must use https or file: {value!r}")
    if parsed.username or parsed.password:
        raise RuntimeError("artifact URL must not contain embedded credentials")
    host = parsed.hostname or ""
    if parsed.port:
        host = f"{host}:{parsed.port}"
    return urlunparse((parsed.scheme, host, parsed.path, "", "", ""))


def download(url: str, destination: Path, expected_hashes: dict[str, str]) -> None:
    request = urllib.request.Request(url, headers={"User-Agent": "ISRAS-wheelhouse-preparer/2"})
    with urllib.request.urlopen(request) as response, destination.open("wb") as output:
        while True:
            block = response.read(1024 * 1024)
            if not block:
                break
            output.write(block)
    for algorithm, expected in expected_hashes.items():
        actual = digest(destination, algorithm)
        if actual != expected:
            raise RuntimeError(
                f"downloaded artifact {destination.name} failed upstream {algorithm} verification"
            )


def report_artifacts(report: dict[str, Any]) -> list[dict[str, Any]]:
    install = report.get("install")
    if not isinstance(install, list) or not install:
        raise RuntimeError("pip resolution report contains no install artifacts")
    return install


def materialize(item: dict[str, Any], wheels: Path) -> dict[str, Any]:
    metadata = item.get("metadata") or {}
    info = item.get("download_info") or {}
    url = info.get("url")
    name = metadata.get("name")
    version = metadata.get("version")
    if not all(isinstance(value, str) and value for value in (url, name, version)):
        raise RuntimeError("pip resolution report contains incomplete artifact metadata")
    parsed = urlparse(url)
    filename = Path(parsed.path).name
    if not filename.endswith(".whl") or Path(filename).name != filename:
        raise RuntimeError(f"non-wheel or unsafe dependency resolved: {filename!r}")
    hashes = source_hashes(info)
    destination = wheels / filename
    if destination.exists():
        raise RuntimeError(f"duplicate wheel filename resolved: {filename}")
    download(url, destination, hashes)
    return {
        "name": name,
        "version": version,
        "filename": filename,
        "sha512": digest(destination),
        "source_url": sanitized_url(url),
        "source_hashes": dict(sorted(hashes.items())),
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--requirements", default="tools/requirements.txt")
    parser.add_argument("--output", required=True)
    parser.add_argument("--pip-version", required=True)
    args = parser.parse_args()

    root = Path(args.repo_root).resolve()
    requirements = (root / args.requirements).resolve()
    try:
        requirements.relative_to(root)
    except ValueError as exc:
        raise RuntimeError("requirements file must remain inside the repository") from exc
    if not requirements.is_file():
        raise RuntimeError(f"requirements file does not exist: {requirements}")

    output = Path(args.output).resolve()
    output.mkdir(parents=True, exist_ok=True)
    if any(output.iterdir()):
        raise RuntimeError("wheelhouse output directory must be empty")
    wheels = output / "wheels"
    wheels.mkdir()

    active_pip = active_pip_version()
    if active_pip != args.pip_version:
        raise RuntimeError(
            f"active resolver pip is {active_pip}; required explicit version is {args.pip_version}"
        )

    with tempfile.TemporaryDirectory(prefix="isras-wheelhouse-report-") as temporary:
        report_path = Path(temporary) / "requirements-report.json"
        report = run_report(["-r", str(requirements)], report_path)
        artifacts = [materialize(item, wheels) for item in report_artifacts(report)]

        pip_report_path = Path(temporary) / "pip-report.json"
        pip_report = run_report(["--no-deps", f"pip=={args.pip_version}"], pip_report_path)
        pip_items = report_artifacts(pip_report)
        if len(pip_items) != 1:
            raise RuntimeError(f"expected one pip artifact, found {len(pip_items)}")
        pip_artifact = materialize(pip_items[0], wheels)
        if pip_artifact["name"].casefold() != "pip" or pip_artifact["version"] != args.pip_version:
            raise RuntimeError("resolved pip artifact does not match requested version")

    normalized_names: set[str] = set()
    for artifact in artifacts:
        normalized = artifact["name"].replace("_", "-").replace(".", "-").casefold()
        if normalized == "pip":
            raise RuntimeError("tools requirements must not declare pip; pip is bootstrapped separately")
        if normalized in normalized_names:
            raise RuntimeError(f"duplicate resolved distribution name: {artifact['name']}")
        normalized_names.add(normalized)

    artifacts = sorted(artifacts, key=lambda item: (item["name"].casefold(), item["version"]))
    requirements_lock = output / "requirements.lock"
    requirements_lock.write_text(
        "\n".join(
            [
                "# Generated wheelhouse candidate. Review, scan, and formally accept before release use.",
                "# SHA-512 is the primary artifact binding.",
                *[
                    f"{item['name']}=={item['version']} --hash=sha512:{item['sha512']}"
                    for item in artifacts
                ],
            ]
        )
        + "\n",
        encoding="utf-8",
    )
    bootstrap_pip_lock = output / "bootstrap-pip.lock"
    bootstrap_pip_lock.write_text(
        f"pip=={args.pip_version} --hash=sha512:{pip_artifact['sha512']}\n",
        encoding="utf-8",
    )

    executable = Path(sys.executable).resolve()
    lock = {
        "schema_version": "ISRAS-TOOL-BOOTSTRAP-LOCK-V1",
        "created_at": dt.datetime.now(dt.timezone.utc).isoformat().replace("+00:00", "Z"),
        "platform": {
            "system": platform.system(),
            "machine": platform.machine(),
            "python_implementation": platform.python_implementation(),
            "python_abi": sysconfig.get_config_var("SOABI") or "unknown",
        },
        "python": platform.python_version(),
        "python_executable_sha512": digest(executable),
        "requirements_source_sha512": digest(requirements),
        "resolver_pip_version": active_pip,
        "pip_wheel": pip_artifact,
        "requirements_lock_sha512": digest(requirements_lock),
        "artifacts": artifacts,
    }
    schema = json.loads(
        (root / "schemas/tool-bootstrap-lock-v1.schema.json").read_text(encoding="utf-8")
    )
    validation = list(
        Draft202012Validator(schema, format_checker=FormatChecker()).iter_errors(lock)
    )
    if validation:
        raise RuntimeError("; ".join(error.message for error in validation))

    lock_path = output / "bootstrap-lock.json"
    lock_path.write_text(json.dumps(lock, indent=2, sort_keys=True) + "\n", encoding="utf-8")

    manifest_entries = []
    for path in sorted(item for item in output.rglob("*") if item.is_file() and item.name != "SHA512SUMS"):
        manifest_entries.append(f"{digest(path)}  {path.relative_to(output).as_posix()}")
    (output / "SHA512SUMS").write_text("\n".join(manifest_entries) + "\n", encoding="utf-8")
    print(f"Prepared wheelhouse candidate in {output}")
    print("The directory contains no transient resolver report.")
    print("This output is not accepted merely because generation succeeded.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, RuntimeError, ValueError, json.JSONDecodeError) as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
