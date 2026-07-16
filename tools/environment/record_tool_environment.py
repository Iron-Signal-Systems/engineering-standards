#!/usr/bin/env python3
"""Record and, for release mode, verify the exact Python validation-tool environment."""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import importlib.metadata as metadata
import json
import platform
import re
import sys
import sysconfig
from pathlib import Path

from jsonschema import Draft202012Validator, FormatChecker


def normalized(value: str) -> str:
    return re.sub(r"[-_.]+", "-", value).lower()


def file_sha512(path: Path | None) -> str | None:
    if path is None or not path.is_file():
        return None
    hasher = hashlib.sha512()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b""):
            hasher.update(block)
    return hasher.hexdigest()


def distributions() -> dict[str, tuple[str, str]]:
    result: dict[str, tuple[str, str]] = {}
    for distribution in metadata.distributions():
        name = distribution.metadata.get("Name")
        if not name:
            continue
        key = normalized(name)
        if key in result:
            raise RuntimeError(f"duplicate installed distribution identity: {name}")
        result[key] = (name, distribution.version)
    return result


def distribution_tree_sha512(name: str) -> str | None:
    try:
        distribution = metadata.distribution(name)
    except metadata.PackageNotFoundError:
        return None
    files = distribution.files or []
    hasher = hashlib.sha512()
    count = 0
    for relative in sorted(files, key=lambda item: str(item).casefold()):
        path = distribution.locate_file(relative)
        if not path.is_file():
            continue
        count += 1
        encoded = str(relative).replace("\\", "/").encode("utf-8")
        hasher.update(len(encoded).to_bytes(8, "big"))
        hasher.update(encoded)
        with path.open("rb") as handle:
            for block in iter(lambda: handle.read(1024 * 1024), b""):
                hasher.update(block)
    return hasher.hexdigest() if count else None


def identity(path_value: str | None) -> dict[str, str | None]:
    path = Path(path_value).resolve() if path_value else None
    return {
        "path": str(path) if path else None,
        "sha512": file_sha512(path),
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", required=True)
    parser.add_argument("--bootstrap-mode", choices=["developer", "release"], required=True)
    parser.add_argument("--requirements")
    parser.add_argument("--bootstrap-lock")
    parser.add_argument("--wheelhouse-manifest")
    args = parser.parse_args()

    installed = distributions()
    bootstrap_lock = Path(args.bootstrap_lock).resolve() if args.bootstrap_lock else None

    if args.bootstrap_mode == "release":
        if bootstrap_lock is None or not bootstrap_lock.is_file():
            raise RuntimeError("release environment recording requires --bootstrap-lock")
        lock = json.loads(bootstrap_lock.read_text(encoding="utf-8"))
        expected = {"pip": lock["pip_wheel"]["version"]}
        for artifact in lock["artifacts"]:
            key = normalized(artifact["name"])
            if key in expected:
                raise RuntimeError(f"duplicate expected distribution identity: {artifact['name']}")
            expected[key] = artifact["version"]
        actual = {key: version for key, (_name, version) in installed.items()}
        if actual != expected:
            missing = sorted(set(expected) - set(actual))
            extra = sorted(set(actual) - set(expected))
            changed = sorted(
                key for key in set(actual) & set(expected) if actual[key] != expected[key]
            )
            raise RuntimeError(
                "installed release distributions do not exactly match bootstrap lock: "
                f"missing={missing}; extra={extra}; changed={changed}"
            )
        manifest = Path(args.wheelhouse_manifest).resolve() if args.wheelhouse_manifest else None
        if manifest is None or not manifest.is_file():
            raise RuntimeError("release environment recording requires --wheelhouse-manifest")
        base_executable = Path(getattr(sys, "_base_executable", sys.executable)).resolve()
        if file_sha512(base_executable) != lock["python_executable_sha512"]:
            raise RuntimeError("release base Python executable does not match bootstrap lock")
        if platform.python_version() != lock["python"]:
            raise RuntimeError("release Python version does not match bootstrap lock")
        expected_platform = lock["platform"]
        observed_platform = {
            "system": platform.system(),
            "machine": platform.machine(),
            "python_implementation": platform.python_implementation(),
            "python_abi": sysconfig.get_config_var("SOABI") or "unknown",
        }
        if observed_platform != expected_platform:
            raise RuntimeError(
                f"release Python platform does not match bootstrap lock: expected={expected_platform}; observed={observed_platform}"
            )
        if distribution_tree_sha512("pip") is None:
            raise RuntimeError("release pip distribution tree cannot be hashed")

    records = [
        {"name": installed[key][0], "version": installed[key][1]}
        for key in sorted(installed)
    ]
    executable = Path(sys.executable).resolve()
    base_executable = Path(getattr(sys, "_base_executable", sys.executable)).resolve()
    record = {
        "schema_version": "ISRAS-TOOL-ENVIRONMENT-V1",
        "recorded_at": dt.datetime.now(dt.timezone.utc).isoformat().replace("+00:00", "Z"),
        "bootstrap_mode": args.bootstrap_mode,
        "python": {
            "executable": str(executable),
            "executable_sha512": file_sha512(executable),
            "base_executable": str(base_executable),
            "base_executable_sha512": file_sha512(base_executable),
            "abi": sysconfig.get_config_var("SOABI") or "unknown",
            "version": platform.python_version(),
            "implementation": platform.python_implementation(),
            "compiler": platform.python_compiler(),
        },
        "platform": {
            "system": platform.system(),
            "release": platform.release(),
            "machine": platform.machine(),
        },
        "pip": {
            "version": installed.get("pip", ("pip", "NOT_INSTALLED"))[1],
            "distribution_sha512": distribution_tree_sha512("pip"),
        },
        "distributions": records,
        "requirements": identity(args.requirements),
        "bootstrap_lock": identity(args.bootstrap_lock),
        "wheelhouse_manifest": identity(args.wheelhouse_manifest),
    }
    schema_path = Path(__file__).resolve().parents[2] / "schemas" / "tool-environment-record-v1.schema.json"
    schema = json.loads(schema_path.read_text(encoding="utf-8"))
    validation = list(
        Draft202012Validator(schema, format_checker=FormatChecker()).iter_errors(record)
    )
    if validation:
        raise RuntimeError("; ".join(error.message for error in validation))

    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(record, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print(f"Wrote exact tool-environment record to {output}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, RuntimeError, KeyError, json.JSONDecodeError) as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
