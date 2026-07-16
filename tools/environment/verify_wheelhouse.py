#!/usr/bin/env python3
"""Verify a platform-specific ISRAS wheelhouse using only Python's standard library."""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import platform
import re
import sys
import sysconfig
from pathlib import Path
from typing import Any
from urllib.parse import urlparse

SHA512_RE = re.compile(r"^[0-9a-f]{128}$")
HEX_RE = re.compile(r"^[0-9a-f]+$")
LOCK_LINE_RE = re.compile(
    r"^([A-Za-z0-9][A-Za-z0-9._-]*)==([^\s]+) --hash=sha512:([0-9a-f]{128})$"
)


def digest(path: Path, algorithm: str = "sha512") -> str:
    hasher = hashlib.new(algorithm)
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b""):
            hasher.update(block)
    return hasher.hexdigest()


def normalized_name(value: str) -> str:
    return re.sub(r"[-_.]+", "-", value).lower()


def safe_relative(root: Path, value: str) -> Path:
    relative = Path(value)
    if relative.is_absolute() or "\\" in value or any(part in {"", ".", ".."} for part in relative.parts):
        raise ValueError(f"unsafe wheelhouse path: {value!r}")
    candidate = (root / relative).resolve()
    try:
        candidate.relative_to(root)
    except ValueError as exc:
        raise ValueError(f"wheelhouse path escapes root: {value!r}") from exc
    return candidate


def require_string(container: dict[str, Any], key: str, errors: list[str]) -> str | None:
    value = container.get(key)
    if not isinstance(value, str) or not value:
        errors.append(f"bootstrap lock field {key!r} must be a non-empty string")
        return None
    return value


def require_sha512(value: Any, label: str, errors: list[str]) -> str | None:
    if not isinstance(value, str) or SHA512_RE.fullmatch(value) is None:
        errors.append(f"{label} must be a lowercase SHA-512 digest")
        return None
    return value


def parse_requirement_lock(path: Path, errors: list[str]) -> dict[str, tuple[str, str]]:
    records: dict[str, tuple[str, str]] = {}
    for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        value = line.strip()
        if not value or value.startswith("#"):
            continue
        match = LOCK_LINE_RE.fullmatch(value)
        if match is None:
            errors.append(f"{path.name} line {number} is not an exact SHA-512 lock entry")
            continue
        name, version, hash_value = match.groups()
        key = normalized_name(name)
        if key in records:
            errors.append(f"{path.name} contains duplicate project {name!r}")
            continue
        records[key] = (version, hash_value)
    if not records:
        errors.append(f"{path.name} contains no locked distributions")
    return records


def parse_artifact(item: Any, label: str, errors: list[str]) -> tuple[str, str, str, str, dict[str, str]] | None:
    if not isinstance(item, dict):
        errors.append(f"{label} must be an object")
        return None
    name = require_string(item, "name", errors)
    version = require_string(item, "version", errors)
    filename = require_string(item, "filename", errors)
    hash_value = require_sha512(item.get("sha512"), f"{label}.sha512", errors)
    source_url = require_string(item, "source_url", errors)
    source_hashes = item.get("source_hashes")
    if not isinstance(source_hashes, dict) or not source_hashes:
        errors.append(f"{label}.source_hashes must be a non-empty object")
    else:
        for algorithm, value in source_hashes.items():
            if not isinstance(algorithm, str) or not algorithm:
                errors.append(f"{label}.source_hashes has an invalid algorithm name")
                continue
            try:
                hashlib.new(algorithm)
            except ValueError:
                errors.append(f"{label}.source_hashes uses unsupported algorithm {algorithm!r}")
            if not isinstance(value, str) or HEX_RE.fullmatch(value) is None:
                errors.append(f"{label}.source_hashes.{algorithm} is not lowercase hexadecimal")
    if source_url:
        parsed = urlparse(source_url)
        if parsed.scheme not in {"https", "file"} or parsed.username or parsed.password or parsed.query or parsed.fragment:
            errors.append(f"{label}.source_url is not a sanitized https/file URL")
    if not all((name, version, filename, hash_value)) or not isinstance(source_hashes, dict):
        return None
    if Path(filename).name != filename or not filename.endswith(".whl"):
        errors.append(f"{label}.filename is not a safe wheel filename: {filename!r}")
        return None
    return name, version, filename, hash_value, {str(k): str(v) for k, v in source_hashes.items()}


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--wheelhouse", required=True)
    parser.add_argument("--requirements", default="tools/requirements.txt")
    args = parser.parse_args()

    root = Path(args.repo_root).resolve()
    requirements_source = (root / args.requirements).resolve()
    try:
        requirements_source.relative_to(root)
    except ValueError as exc:
        raise ValueError("requirements file must remain inside the repository") from exc
    if not requirements_source.is_file():
        raise ValueError(f"requirements source does not exist: {requirements_source}")

    wheelhouse = Path(args.wheelhouse).resolve()
    errors: list[str] = []
    if not wheelhouse.is_dir():
        raise ValueError(f"wheelhouse directory does not exist: {wheelhouse}")

    lock_path = wheelhouse / "bootstrap-lock.json"
    if not lock_path.is_file():
        raise ValueError("bootstrap-lock.json is missing")
    lock = json.loads(lock_path.read_text(encoding="utf-8"))
    if not isinstance(lock, dict):
        raise ValueError("bootstrap-lock.json root must be an object")

    if lock.get("schema_version") != "ISRAS-TOOL-BOOTSTRAP-LOCK-V1":
        errors.append("bootstrap lock schema_version is not ISRAS-TOOL-BOOTSTRAP-LOCK-V1")

    created_at = require_string(lock, "created_at", errors)
    if created_at:
        try:
            parsed_time = dt.datetime.fromisoformat(created_at.replace("Z", "+00:00"))
            if parsed_time.tzinfo is None:
                errors.append("bootstrap lock created_at must include a timezone")
        except ValueError:
            errors.append("bootstrap lock created_at is not a valid ISO-8601 timestamp")

    locked_platform = lock.get("platform")
    if not isinstance(locked_platform, dict):
        errors.append("bootstrap lock platform must be an object")
        locked_platform = {}
    current_platform = {
        "system": platform.system(),
        "machine": platform.machine(),
        "python_implementation": platform.python_implementation(),
        "python_abi": sysconfig.get_config_var("SOABI") or "unknown",
    }
    for key, current_value in current_platform.items():
        locked_value = locked_platform.get(key)
        if not isinstance(locked_value, str) or not locked_value:
            errors.append(f"bootstrap lock platform.{key} must be a non-empty string")
        elif locked_value != current_value:
            errors.append(
                f"wheelhouse platform mismatch for {key}: lock={locked_value!r}; current={current_value!r}"
            )

    locked_python = require_string(lock, "python", errors)
    if locked_python and locked_python != platform.python_version():
        errors.append(
            f"wheelhouse Python mismatch: lock={locked_python}; current={platform.python_version()}"
        )
    executable_digest = require_sha512(
        lock.get("python_executable_sha512"), "python_executable_sha512", errors
    )
    if executable_digest and digest(Path(sys.executable).resolve()) != executable_digest:
        errors.append("bootstrap Python executable SHA-512 mismatch")

    requirements_source_digest = require_sha512(
        lock.get("requirements_source_sha512"), "requirements_source_sha512", errors
    )
    if requirements_source_digest and digest(requirements_source) != requirements_source_digest:
        errors.append("repository requirements source SHA-512 mismatch")

    require_string(lock, "resolver_pip_version", errors)
    requirements_digest = require_sha512(
        lock.get("requirements_lock_sha512"), "requirements_lock_sha512", errors
    )

    pip_record = parse_artifact(lock.get("pip_wheel"), "pip_wheel", errors)
    artifacts_value = lock.get("artifacts")
    if not isinstance(artifacts_value, list) or not artifacts_value:
        errors.append("bootstrap lock artifacts must be a non-empty array")
        artifacts_value = []

    artifact_records: dict[str, tuple[str, str, str, dict[str, str]]] = {}
    filenames: set[str] = set()
    for index, item in enumerate(artifacts_value):
        parsed = parse_artifact(item, f"artifacts[{index}]", errors)
        if parsed is None:
            continue
        name, version, filename, hash_value, upstream_hashes = parsed
        normalized = normalized_name(name)
        if normalized == "pip":
            errors.append("pip must be represented only by pip_wheel")
        if normalized in artifact_records:
            errors.append(f"duplicate locked distribution name: {name!r}")
        else:
            artifact_records[normalized] = (version, filename, hash_value, upstream_hashes)
        if filename in filenames:
            errors.append(f"duplicate locked wheel filename: {filename!r}")
        filenames.add(filename)

    if pip_record:
        pip_name, pip_version, pip_filename, pip_digest, pip_upstream_hashes = pip_record
        if normalized_name(pip_name) != "pip":
            errors.append("pip_wheel.name must be pip")
        if pip_filename in filenames:
            errors.append("pip wheel filename duplicates a dependency wheel")
    else:
        pip_version = pip_filename = pip_digest = None
        pip_upstream_hashes = {}

    requirements_lock = wheelhouse / "requirements.lock"
    bootstrap_pip_lock = wheelhouse / "bootstrap-pip.lock"
    for required_path in (requirements_lock, bootstrap_pip_lock):
        if not required_path.is_file():
            errors.append(f"required wheelhouse file is missing: {required_path.name}")

    requirement_records: dict[str, tuple[str, str]] = {}
    if requirements_lock.is_file():
        if requirements_digest and digest(requirements_lock) != requirements_digest:
            errors.append("requirements.lock SHA-512 mismatch")
        requirement_records = parse_requirement_lock(requirements_lock, errors)

    expected_requirements = {
        name: (version, hash_value)
        for name, (version, _filename, hash_value, _upstream_hashes) in artifact_records.items()
    }
    if requirement_records != expected_requirements:
        missing = sorted(set(expected_requirements) - set(requirement_records))
        extra = sorted(set(requirement_records) - set(expected_requirements))
        changed = sorted(
            name
            for name in set(requirement_records) & set(expected_requirements)
            if requirement_records[name] != expected_requirements[name]
        )
        errors.append(
            "requirements.lock does not exactly match bootstrap artifacts: "
            f"missing={missing}; extra={extra}; changed={changed}"
        )

    if bootstrap_pip_lock.is_file():
        pip_records = parse_requirement_lock(bootstrap_pip_lock, errors)
        expected_pip = (
            {"pip": (pip_version, pip_digest)}
            if pip_version is not None and pip_digest is not None
            else {}
        )
        if pip_records != expected_pip:
            errors.append("bootstrap-pip.lock does not exactly match the pinned pip wheel")

    expected_files = {"bootstrap-lock.json", "bootstrap-pip.lock", "requirements.lock"}
    expected_wheels: dict[str, tuple[str, dict[str, str]]] = {}
    if pip_filename and pip_digest:
        expected_wheels[pip_filename] = (pip_digest, pip_upstream_hashes)
    for _name, (_version, filename, hash_value, upstream_hashes) in artifact_records.items():
        expected_wheels[filename] = (hash_value, upstream_hashes)
    expected_files.update(f"wheels/{filename}" for filename in expected_wheels)

    for filename, (hash_value, upstream_hashes) in expected_wheels.items():
        path = wheelhouse / "wheels" / filename
        if not path.is_file():
            errors.append(f"wheel missing: {filename}")
            continue
        if digest(path) != hash_value:
            errors.append(f"wheel SHA-512 mismatch: {filename}")
        for algorithm, expected_upstream in upstream_hashes.items():
            if digest(path, algorithm) != expected_upstream:
                errors.append(f"wheel upstream {algorithm} mismatch: {filename}")

    manifest = wheelhouse / "SHA512SUMS"
    manifest_paths: dict[str, str] = {}
    if not manifest.is_file():
        errors.append("wheelhouse SHA512SUMS is missing")
    else:
        for number, line in enumerate(manifest.read_text(encoding="utf-8").splitlines(), 1):
            parts = line.split("  ", 1)
            if len(parts) != 2 or SHA512_RE.fullmatch(parts[0]) is None:
                errors.append(f"invalid SHA512SUMS line {number}")
                continue
            relative = parts[1]
            if relative in manifest_paths:
                errors.append(f"duplicate SHA512SUMS path: {relative}")
                continue
            try:
                path = safe_relative(wheelhouse, relative)
            except ValueError as exc:
                errors.append(str(exc))
                continue
            manifest_paths[relative] = parts[0]
            if not path.is_file() or digest(path) != parts[0]:
                errors.append(f"SHA512SUMS mismatch: {relative}")

    actual_files = {
        path.relative_to(wheelhouse).as_posix()
        for path in wheelhouse.rglob("*")
        if path.is_file() and path.name != "SHA512SUMS"
    }
    if actual_files != expected_files:
        errors.append(
            "wheelhouse file set mismatch: "
            f"missing={sorted(expected_files - actual_files)}; "
            f"extra={sorted(actual_files - expected_files)}"
        )
    if set(manifest_paths) != actual_files:
        errors.append(
            "wheelhouse manifest path set mismatch: "
            f"missing={sorted(actual_files - set(manifest_paths))}; "
            f"extra={sorted(set(manifest_paths) - actual_files)}"
        )

    if errors:
        for error in sorted(set(errors)):
            print(f"FAIL: {error}")
        print(f"Wheelhouse verification FAILED with {len(set(errors))} error(s).")
        return 1
    print(f"PASS: wheelhouse verified for {len(expected_wheels)} wheel artifacts")
    print("PASS: Python executable and repository requirements source match the accepted lock")
    print("PASS: pre-bootstrap verification used only the Python standard library")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
