#!/usr/bin/env python3
from __future__ import annotations

import argparse
import importlib
import importlib.metadata
import json
import os
import platform
import re
import shutil
import sys
from pathlib import Path
from typing import Any
from urllib.parse import urlparse

from common import ISRASError, git, load_json, print_result, repository_root, run


def _command_spec(value: str | dict[str, Any]) -> dict[str, Any]:
    if isinstance(value, str):
        return {"command": value, "version_args": ["--version"]}
    if not isinstance(value, dict) or not isinstance(value.get("command"), str):
        raise ISRASError(f"invalid command requirement: {value!r}")
    return value


def _command_applies(spec: dict[str, Any]) -> bool:
    systems = spec.get("operating_systems")
    return not systems or platform.system() in systems


def _command_version(spec: dict[str, Any]) -> tuple[bool, str]:
    command = spec["command"]
    executable = shutil.which(command)
    if executable is None:
        return False, "NOT_AVAILABLE"
    args = [command, *spec.get("version_args", ["--version"])]
    result = run(args, check=False, capture=True)
    output = ((result.stdout or "") + (result.stderr or "")).strip().splitlines()
    version = output[0] if output else f"exit={result.returncode}"
    pattern = spec.get("version_pattern")
    if result.returncode != 0:
        return False, version
    if pattern and re.search(pattern, version) is None:
        return False, version
    return True, version



def _repository_identity(value: str) -> str:
    candidate = value.strip()

    # Git SCP-style SSH syntax:
    # git@github.com:Owner/repository.git
    if "://" not in candidate and not re.match(r"^[A-Za-z]:[\\/]", candidate):
        match = re.fullmatch(r"(?:[^@]+@)?([^:]+):(.+)", candidate)
        if match:
            host, path = match.groups()
            return f"{host.lower()}/{path.removesuffix('.git').strip('/')}"

    parsed = urlparse(candidate)
    if parsed.scheme in {"http", "https", "ssh", "git"} and parsed.hostname:
        path = parsed.path.removesuffix(".git").strip("/")
        return f"{parsed.hostname.lower()}/{path}"

    return candidate.rstrip("/")

def build_fingerprint(repo_root: Path, profile: dict[str, Any]) -> dict[str, Any]:
    commands: dict[str, dict[str, Any]] = {}
    for raw in [*profile.get("required_commands", []), *profile.get("optional_commands", [])]:
        spec = _command_spec(raw)
        if not _command_applies(spec):
            commands[spec["command"]] = {"available": False, "version": "NOT_APPLICABLE"}
            continue
        ok, version = _command_version(spec)
        commands[spec["command"]] = {"available": ok, "version": version}

    modules: dict[str, dict[str, Any]] = {}
    for spec in profile.get("required_python_modules", []):
        module = spec["module"]
        distribution = spec["distribution"]
        try:
            importlib.import_module(module)
            actual = importlib.metadata.version(distribution)
            available = True
        except (ImportError, importlib.metadata.PackageNotFoundError):
            actual = "NOT_AVAILABLE"
            available = False
        modules[module] = {
            "distribution": distribution,
            "required_version": spec["version"],
            "actual_version": actual,
            "available": available,
        }

    return {
        "profile": profile.get("profile"),
        "classification": profile.get("classification"),
        "host": platform.node(),
        "operating_system": platform.system(),
        "operating_system_release": platform.release(),
        "operating_system_version": platform.version(),
        "architecture": platform.machine(),
        "python_executable": sys.executable,
        "python_version": platform.python_version(),
        "commands": commands,
        "python_modules": modules,
        "environment_variables_present": {
            name: bool(os.environ.get(name))
            for name in profile.get("required_environment_variables", [])
        },
        "repository_commit": git(repo_root, "rev-parse", "HEAD"),
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--profile", default="portable")
    parser.add_argument("--fingerprint-output")
    args = parser.parse_args()
    repo_root = repository_root(args.repo_root)

    manifest = load_json(repo_root / "REPOSITORY-ASSURANCE.json")
    profile_path = repo_root / "tools/environment/profiles" / f"{args.profile}.json"
    profile = load_json(profile_path)
    fingerprint = build_fingerprint(repo_root, profile)

    failures = 0
    print(f"Repository: {manifest.get('repository')}")
    print(f"Host: {fingerprint['host']}")
    print(
        "Operating system: "
        f"{fingerprint['operating_system']} {fingerprint['operating_system_release']}"
    )
    print(f"Architecture: {fingerprint['architecture']}")
    print(f"Profile: {profile.get('profile')} ({profile.get('classification')})")

    origin = git(repo_root, "remote", "get-url", "origin")
    expected_origin = manifest.get("canonical_origin")

    if profile.get("classification") == "portable":
        origin_ok = _repository_identity(origin) == _repository_identity(expected_origin)
        origin_label = "Canonical repository identity"
    else:
        origin_ok = origin == expected_origin
        origin_label = "Canonical origin"

    print_result(origin_label, origin_ok, origin)
    failures += int(not origin_ok)

    status = git(repo_root, "status", "--porcelain")
    clean = status == ""
    print_result("Working tree is clean", clean)
    if not clean:
        print("INFO: a dirty tree is allowed for development checks but not acceptance.")

    systems = profile.get("operating_systems", [])
    if systems and "PROJECT_DEFINED" not in systems:
        os_ok = fingerprint["operating_system"] in systems
        print_result("Operating system is permitted", os_ok, fingerprint["operating_system"])
        failures += int(not os_ok)
    elif "PROJECT_DEFINED" in systems:
        print_result("Operating system profile is finalized", False, "PROJECT_DEFINED remains")
        failures += 1

    architectures = profile.get("architectures", [])
    if architectures and "PROJECT_DEFINED" not in architectures:
        arch_ok = fingerprint["architecture"] in architectures
        print_result("Architecture is permitted", arch_ok, fingerprint["architecture"])
        failures += int(not arch_ok)
    elif "PROJECT_DEFINED" in architectures:
        print_result("Architecture profile is finalized", False, "PROJECT_DEFINED remains")
        failures += 1

    for raw in profile.get("required_commands", []):
        spec = _command_spec(raw)
        if not _command_applies(spec):
            print(f"INFO: Required command {spec['command']}: NOT_APPLICABLE_ON_{platform.system()}")
            continue
        ok, version = _command_version(spec)
        print_result(f"Required command and version: {spec['command']}", ok, version)
        failures += int(not ok)

    for raw in profile.get("optional_commands", []):
        spec = _command_spec(raw)
        if not _command_applies(spec):
            continue
        ok, version = _command_version(spec)
        state = "AVAILABLE" if ok else "NOT_AVAILABLE_OR_MISMATCHED"
        print(f"INFO: Optional command {spec['command']}: {state}: {version}")

    for spec in profile.get("required_python_modules", []):
        record = fingerprint["python_modules"][spec["module"]]
        ok = record["available"] and record["actual_version"] == spec["version"]
        detail = f"required={spec['version']} actual={record['actual_version']}"
        print_result(f"Required Python module: {spec['module']}", ok, detail)
        failures += int(not ok)

    for name, present in fingerprint["environment_variables_present"].items():
        print_result(f"Required environment variable is set: {name}", present)
        failures += int(not present)

    if args.fingerprint_output:
        output = Path(args.fingerprint_output)
        if not output.is_absolute():
            output = repo_root / output
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_text(json.dumps(fingerprint, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        print(f"Environment fingerprint: {output}")

    if failures:
        print(
            f"\nEnvironment profile FAILED with {failures} missing or mismatched requirement(s).",
            file=sys.stderr,
        )
        return 1

    print("\nEnvironment profile PASSED.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ISRASError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
