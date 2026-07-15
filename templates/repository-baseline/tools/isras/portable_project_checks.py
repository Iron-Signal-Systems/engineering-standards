#!/usr/bin/env python3
from __future__ import annotations

import argparse
import os
import ast
import shutil
import subprocess
import sys
from pathlib import Path

from common import ISRASError, print_result, repository_root, run


EXCLUDED_PARTS = {".git", ".venv", "venv", "node_modules", "bin", "obj"}


def paths(repo_root: Path, suffix: str) -> list[Path]:
    return [
        p for p in repo_root.rglob(f"*{suffix}")
        if p.is_file() and not any(part in EXCLUDED_PARTS for part in p.parts)
    ]


def command_available(name: str) -> bool:
    return shutil.which(name) is not None


def check_shell(repo_root: Path) -> None:
    files = paths(repo_root, ".sh")
    if not files:
        return
    if not command_available("bash"):
        raise ISRASError("Bash scripts exist but bash is unavailable")
    for path in files:
        run(["bash", "-n", str(path)], cwd=repo_root)
    print_result(f"Bash syntax valid for {len(files)} script(s)", True)


def check_go(repo_root: Path) -> None:
    modules = paths(repo_root, "go.mod")
    if not modules:
        return
    if not command_available("go"):
        raise ISRASError("Go module exists but go is unavailable")
    for go_mod in modules:
        module_root = go_mod.parent
        go_files = paths(module_root, ".go")
        if go_files:
            result = run(["gofmt", "-l", *[str(p) for p in go_files]], cwd=module_root, capture=True)
            if result.stdout.strip():
                raise ISRASError(f"gofmt required:\n{result.stdout}")
        run(["go", "mod", "verify"], cwd=module_root)
        run(["go", "vet", "./..."], cwd=module_root)
        run(["go", "test", "./..."], cwd=module_root)
        cgo = run(["go", "env", "CGO_ENABLED"], cwd=module_root, capture=True).stdout.strip()
        if cgo == "1":
            run(["go", "test", "-race", "./..."], cwd=module_root)
        else:
            print(f"INFO: Race tests skipped for {module_root}: CGO_ENABLED={cgo}")
        print_result(f"Go portable checks pass: {module_root.relative_to(repo_root) or '.'}", True)


def check_dotnet(repo_root: Path) -> None:
    projects = paths(repo_root, ".sln") + paths(repo_root, ".csproj")
    if not projects:
        return
    if not command_available("dotnet"):
        raise ISRASError(".NET project exists but dotnet is unavailable")
    target = next((p for p in projects if p.suffix == ".sln"), projects[0])
    locked = bool(list(repo_root.rglob("packages.lock.json")))
    restore = ["dotnet", "restore", str(target)]
    if locked:
        restore.append("--locked-mode")
    else:
        print("WARN: No packages.lock.json found; locked restore is not enforced.")
    run(restore, cwd=repo_root)
    run(["dotnet", "build", str(target), "--no-restore"], cwd=repo_root)
    run(["dotnet", "test", str(target), "--no-build"], cwd=repo_root)
    print_result(".NET portable checks pass", True)


def check_python(repo_root: Path) -> None:
    files = paths(repo_root, ".py")
    if not files:
        return
    for path in files:
        try:
            ast.parse(path.read_text(encoding="utf-8"), filename=str(path))
        except (SyntaxError, UnicodeDecodeError) as exc:
            raise ISRASError(f"Python syntax failed for {path}: {exc}") from exc
    print_result(f"Python syntax valid for {len(files)} file(s)", True)
    pytest_configured = (repo_root / "pytest.ini").exists() or (repo_root / "tox.ini").exists()
    pyproject = repo_root / "pyproject.toml"
    if pyproject.exists() and "[tool.pytest" in pyproject.read_text(encoding="utf-8", errors="replace"):
        pytest_configured = True
    if pytest_configured:
        result = subprocess.run(
            [sys.executable, "-m", "pytest", "--version"],
            cwd=repo_root,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        if result.returncode != 0:
            raise ISRASError("pytest is configured but unavailable")
        run([sys.executable, "-m", "pytest"], cwd=repo_root)
        print_result("Python pytest suite passes", True)


def check_powershell(repo_root: Path) -> None:
    files = paths(repo_root, ".ps1") + paths(repo_root, ".psm1")
    if not files:
        return
    if not command_available("pwsh"):
        print("INFO: PowerShell files exist; syntax validation is unavailable on this host.")
        return
    for path in files:
        script = (
            "$errors=$null;"
            f"[System.Management.Automation.Language.Parser]::ParseFile('{str(path).replace(chr(39), chr(39)*2)}',[ref]$null,[ref]$errors)|Out-Null;"
            "if($errors.Count -gt 0){$errors|ForEach-Object{Write-Error $_};exit 1}"
        )
        run(["pwsh", "-NoProfile", "-NonInteractive", "-Command", script], cwd=repo_root)
    print_result(f"PowerShell syntax valid for {len(files)} file(s)", True)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    args = parser.parse_args()
    repo_root = repository_root(args.repo_root)

    run(["git", "diff", "--check"], cwd=repo_root)
    print_result("Git diff is whitespace-clean", True)

    check_shell(repo_root)
    check_go(repo_root)
    check_dotnet(repo_root)
    check_python(repo_root)
    check_powershell(repo_root)

    print("\nProject portable checks PASSED.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ISRASError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
