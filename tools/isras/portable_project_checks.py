#!/usr/bin/env python3
from __future__ import annotations

import argparse
import ast
import shutil
import subprocess
import sys
from pathlib import Path

from common import ISRASError, load_json, print_result, repository_root, run

EXCLUDED_PARTS = {
    ".git", ".venv", "venv", ".isras-tools-venv", "node_modules", "bin", "obj"
}
EMPTY_TREE = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"


def paths(repo_root: Path, suffix: str) -> list[Path]:
    return [
        p for p in repo_root.rglob(f"*{suffix}")
        if p.is_file() and not any(part in EXCLUDED_PARTS for part in p.parts)
    ]


def command_available(name: str) -> bool:
    return shutil.which(name) is not None


def selected_profiles(repo_root: Path) -> set[str]:
    manifest = load_json(repo_root / "REPOSITORY-ASSURANCE.json")
    return {str(value).lower() for value in manifest.get("profiles", [])}


def profile_contains(profiles: set[str], token: str) -> bool:
    return any(token in profile for profile in profiles)


def check_committed_whitespace(repo_root: Path) -> None:
    run(["git", "diff", "--check", EMPTY_TREE, "HEAD", "--"], cwd=repo_root)
    run(["git", "diff", "--check"], cwd=repo_root)
    run(["git", "diff", "--cached", "--check"], cwd=repo_root)
    print_result("Committed, staged, and working-tree text is whitespace-clean", True)


def check_shell(repo_root: Path) -> None:
    files = paths(repo_root, ".sh")
    if not files:
        return
    if sys.platform == "win32":
        print(
            "INFO: Bash syntax validation is not applicable on native Windows; "
            "Linux and macOS matrix jobs validate repository Bash scripts."
        )
        return
    if not command_available("bash"):
        raise ISRASError("Bash scripts exist but bash is unavailable")
    for path in files:
        run(["bash", "-n", str(path)], cwd=repo_root)
    print_result(f"Bash syntax valid for {len(files)} script(s)", True)


def check_go(repo_root: Path, profiles: set[str]) -> None:
    modules = paths(repo_root, "go.mod")
    if not modules:
        return
    if not command_available("go"):
        raise ISRASError("Go module exists but go is unavailable")
    secure_profile = profile_contains(profiles, "go")
    for go_mod in modules:
        module_root = go_mod.parent
        go_files = paths(module_root, ".go")
        if go_files:
            result = run(
                ["gofmt", "-l", *[str(p) for p in go_files]],
                cwd=module_root,
                capture=True,
            )
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
        if secure_profile:
            if not command_available("govulncheck"):
                raise ISRASError("Go assurance profile requires govulncheck")
            run(["govulncheck", "./..."], cwd=module_root)
        print_result(f"Go portable checks pass: {module_root.relative_to(repo_root) or '.'}", True)


def check_dotnet(repo_root: Path, profiles: set[str]) -> None:
    projects = paths(repo_root, ".sln") + paths(repo_root, ".csproj")
    if not projects:
        return
    if not command_available("dotnet"):
        raise ISRASError(".NET project exists but dotnet is unavailable")
    target = next((p for p in projects if p.suffix == ".sln"), projects[0])
    locked = bool(list(repo_root.rglob("packages.lock.json")))
    if profile_contains(profiles, "dotnet") and not locked:
        raise ISRASError(".NET assurance profile requires packages.lock.json")
    restore = ["dotnet", "restore", str(target)]
    if locked:
        restore.append("--locked-mode")
    run(restore, cwd=repo_root)
    run(["dotnet", "build", str(target), "--no-restore"], cwd=repo_root)
    run(["dotnet", "test", str(target), "--no-build"], cwd=repo_root)
    vulnerable = run(
        ["dotnet", "list", str(target), "package", "--vulnerable", "--include-transitive"],
        cwd=repo_root,
        capture=True,
    )
    if "has the following vulnerable packages" in vulnerable.stdout.lower():
        raise ISRASError(f".NET vulnerable packages reported:\n{vulnerable.stdout}")
    print_result(".NET portable checks pass", True)


def requirements_are_pinned(path: Path) -> bool:
    meaningful = []
    for line in path.read_text(encoding="utf-8").splitlines():
        value = line.strip()
        if not value or value.startswith("#") or value.startswith("-"):
            continue
        meaningful.append(value)
    return bool(meaningful) and all("==" in value for value in meaningful)


def check_python(repo_root: Path, profiles: set[str]) -> None:
    files = paths(repo_root, ".py")
    if not files:
        return
    for path in files:
        try:
            ast.parse(path.read_text(encoding="utf-8"), filename=str(path))
        except (SyntaxError, UnicodeDecodeError) as exc:
            raise ISRASError(f"Python syntax failed for {path}: {exc}") from exc
    print_result(f"Python syntax valid for {len(files)} file(s)", True)

    if profile_contains(profiles, "python"):
        lock_candidates = [
            repo_root / "requirements.lock",
            repo_root / "poetry.lock",
            repo_root / "uv.lock",
            repo_root / "Pipfile.lock",
        ]
        pinned_requirements = any(
            path.is_file() and requirements_are_pinned(path)
            for path in [repo_root / "requirements.txt", repo_root / "tools/requirements.txt"]
        )
        if not any(path.is_file() for path in lock_candidates) and not pinned_requirements:
            raise ISRASError("Python assurance profile requires a lock file or exact pinned requirements")

    test_files = list((repo_root / "tests").glob("test_*.py")) if (repo_root / "tests").is_dir() else []
    if test_files:
        run([sys.executable, "-m", "unittest", "discover", "-s", "tests", "-p", "test_*.py"], cwd=repo_root)
        print_result("Python unittest suite passes", True)

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


def check_powershell(repo_root: Path, profiles: set[str]) -> None:
    files = paths(repo_root, ".ps1") + paths(repo_root, ".psm1")
    if not files:
        return
    if not command_available("pwsh"):
        print("INFO: PowerShell files exist; validation is unavailable on this host.")
        return
    for path in files:
        escaped = str(path).replace("'", "''")
        script = (
            "$errors=$null;"
            f"[System.Management.Automation.Language.Parser]::ParseFile('{escaped}',[ref]$null,[ref]$errors)|Out-Null;"
            "if($errors.Count -gt 0){$errors|ForEach-Object{Write-Error $_};exit 1}"
        )
        run(["pwsh", "-NoProfile", "-NonInteractive", "-Command", script], cwd=repo_root)
    if profile_contains(profiles, "powershell"):
        analyzer = run(
            ["pwsh", "-NoProfile", "-NonInteractive", "-Command", "Get-Module -ListAvailable PSScriptAnalyzer | Select-Object -First 1"],
            cwd=repo_root,
            capture=True,
        )
        if not analyzer.stdout.strip():
            raise ISRASError("PowerShell assurance profile requires PSScriptAnalyzer")
        run([
            "pwsh", "-NoProfile", "-NonInteractive", "-Command",
            "Invoke-ScriptAnalyzer -Path . -Recurse -Severity Error; if($Error.Count){exit 1}",
        ], cwd=repo_root)
        pester = run(
            ["pwsh", "-NoProfile", "-NonInteractive", "-Command", "Get-Module -ListAvailable Pester | Select-Object -First 1"],
            cwd=repo_root,
            capture=True,
        )
        if not pester.stdout.strip():
            raise ISRASError("PowerShell assurance profile requires Pester")
        if (repo_root / "tests").exists():
            run([
                "pwsh", "-NoProfile", "-NonInteractive", "-Command",
                "Invoke-Pester -Path tests -CI; if($LASTEXITCODE){exit $LASTEXITCODE}",
            ], cwd=repo_root)
    print_result(f"PowerShell checks pass for {len(files)} file(s)", True)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    args = parser.parse_args()
    repo_root = repository_root(args.repo_root)
    profiles = selected_profiles(repo_root)

    check_committed_whitespace(repo_root)
    check_shell(repo_root)
    check_go(repo_root, profiles)
    check_dotnet(repo_root, profiles)
    check_python(repo_root, profiles)
    check_powershell(repo_root, profiles)

    print("\nProject portable checks PASSED.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ISRASError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
