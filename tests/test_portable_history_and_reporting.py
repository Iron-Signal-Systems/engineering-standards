from __future__ import annotations

import contextlib
import importlib.util
import io
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path, PureWindowsPath
from unittest import mock

sys.dont_write_bytecode = True

ROOT = Path(__file__).resolve().parents[1]


def load_module(name: str, relative: str):
    spec = importlib.util.spec_from_file_location(name, ROOT / relative)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"unable to load {relative}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[name] = module
    try:
        spec.loader.exec_module(module)
    except Exception:
        sys.modules.pop(name, None)
        raise
    return module


class PortableHistoryAndReportingTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.history = load_module(
            "prepare_portable_history",
            "tools/isras/prepare_portable_history.py",
        )
        cls.runner = load_module(
            "run_portable_validation",
            "tools/isras/run_portable_validation.py",
        )

    def test_required_history_includes_all_accepted_checkpoints(self) -> None:
        requirements = {
            item.commit: item
            for item in self.history.discover_requirements(ROOT)
        }
        expected = {
            "f9655ddbbf04430fc468aab405f2ed880df3e97d",
            "c379417720faa595fa5cb89a1dfdb2259d6cb95e",
            "781246e69f8a9a382c25040f94b62dfe3b25ba89",
            "d34fad82781a4e8485f8907fbfd34f236fa79ad2",
        }
        self.assertTrue(expected.issubset(requirements))

    def test_required_history_includes_v3_classification_base(self) -> None:
        requirements = {
            item.commit: item
            for item in self.history.discover_requirements(ROOT)
        }
        base = "08a0a514ec308f76dbf80ffdcb8caa70ce6e345f"
        self.assertIn(base, requirements)
        self.assertTrue(
            any("classification base" in value for value in requirements[base].purposes)
        )

    def test_checkpoint_requirements_use_tag_fetch_refs(self) -> None:
        requirements = {
            item.commit: item
            for item in self.history.discover_requirements(ROOT)
        }
        release = requirements["d34fad82781a4e8485f8907fbfd34f236fa79ad2"]
        self.assertIn("refs/tags/isras-v2.0.1", release.fetch_refs)

    def test_runner_has_specific_stage_failure_codes(self) -> None:
        names = [stage.name for stage in self.runner.STAGES]
        self.assertEqual(
            names,
            [
                "history-preflight",
                "environment-profile",
                "policy",
                "release-state",
                "project-checks",
            ],
        )
        for stage in self.runner.STAGES:
            self.assertRegex(stage.failure_code, r"^ISRAS-PORTABLE-[A-Z-]+-001$")

    def test_runner_uses_isolated_repository_tool_bootstrap(self) -> None:
        command = self.runner.build_command(self.runner.STAGES[1], ROOT)
        self.assertEqual(command[1], "-I")
        self.assertEqual(
            Path(command[2]).resolve(),
            (ROOT / "tools/isras/invoke_repo_tool.py").resolve(),
        )
        self.assertEqual(command[5:7], ["--tool", "tools/isras/doctor.py"])

    def test_runner_build_command_uses_windows_native_paths(self) -> None:
        root = PureWindowsPath("D:/a/engineering-standards/engineering-standards")
        command = self.runner.build_command(self.runner.STAGES[1], root)
        self.assertEqual(
            PureWindowsPath(command[2]),
            root / "tools/isras/invoke_repo_tool.py",
        )
        self.assertEqual(PureWindowsPath(command[4]), root)
        self.assertEqual(command[5:7], ["--tool", "tools/isras/doctor.py"])

    def test_isolated_bootstrap_resolves_sibling_common_module(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            isras = root / "tools/isras"
            isras.mkdir(parents=True)
            shutil.copyfile(
                ROOT / "tools/isras/invoke_repo_tool.py",
                isras / "invoke_repo_tool.py",
            )
            (isras / "common.py").write_text(
                'BOOTSTRAP_TOKEN = "COMMON_IMPORT_OK"\n', encoding="utf-8"
            )
            (isras / "synthetic_tool.py").write_text(
                "from common import BOOTSTRAP_TOKEN\n"
                "print(BOOTSTRAP_TOKEN)\n",
                encoding="utf-8",
            )
            result = subprocess.run(
                [
                    sys.executable,
                    "-I",
                    str(isras / "invoke_repo_tool.py"),
                    "--repo-root",
                    str(root),
                    "--tool",
                    "tools/isras/synthetic_tool.py",
                ],
                cwd=root,
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                check=False,
            )
            self.assertEqual(result.returncode, 0, result.stdout)
            self.assertIn("COMMON_IMPORT_OK", result.stdout)

    def test_project_unittest_execution_is_verbose(self) -> None:
        source = (ROOT / "tools/isras/portable_project_checks.py").read_text(
            encoding="utf-8"
        )
        self.assertIn(
            '[sys.executable, "-m", "unittest", "discover", "-v", "-s", "tests", "-p", "test_*.py"]',
            source,
        )

    def test_workflows_prefetch_history_with_scoped_authentication(self) -> None:
        for relative in (
            ".github/workflows/reusable-portable-validation.yml",
            ".github/workflows/native-os-matrix.yml",
        ):
            source = (ROOT / relative).read_text(encoding="utf-8")
            self.assertIn("prepare_portable_history.py", source)
            self.assertIn("ISRAS_GIT_HTTP_EXTRAHEADER", source)
            self.assertIn("required historical", source.lower())

    def test_runner_closes_streamed_subprocess_output(self) -> None:
        class FakeProcess:
            def __init__(self) -> None:
                self.stdout = io.StringIO("synthetic output\n")
                self.exited = False

            def __enter__(self):
                return self

            def __exit__(self, exc_type, exc, traceback):
                self.exited = True
                self.stdout.close()
                return False

            def wait(self) -> int:
                return 0

        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            isras = root / "tools/isras"
            isras.mkdir(parents=True)
            (isras / "invoke_repo_tool.py").write_text("# bootstrap\n", encoding="utf-8")
            (isras / "synthetic-validator.py").write_text("# validator\n", encoding="utf-8")
            stage = self.runner.Stage(
                "synthetic-stage",
                "ISRAS-PORTABLE-SYNTHETIC-001",
                "tools/isras/synthetic-validator.py",
            )
            process = FakeProcess()
            stdout = io.StringIO()
            with mock.patch.dict(os.environ, {"GITHUB_ACTIONS": "false"}, clear=False):
                with mock.patch.object(
                    self.runner, "git_head", return_value="UNRESOLVED"
                ):
                    with mock.patch.object(
                        self.runner.subprocess, "Popen", return_value=process
                    ):
                        with contextlib.redirect_stdout(stdout):
                            result = self.runner.run_stage(stage, root)
            self.assertEqual(result, 0)
            self.assertTrue(process.exited)
            self.assertTrue(process.stdout.closed)

    def test_runner_emits_exact_failure_context(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            isras = root / "tools/isras"
            isras.mkdir(parents=True)
            shutil.copyfile(
                ROOT / "tools/isras/invoke_repo_tool.py",
                isras / "invoke_repo_tool.py",
            )
            validator = isras / "synthetic-validator.py"
            validator.write_text("raise SystemExit(7)\n", encoding="utf-8")
            stage = self.runner.Stage(
                "synthetic-stage",
                "ISRAS-PORTABLE-SYNTHETIC-001",
                "tools/isras/synthetic-validator.py",
            )
            stdout = io.StringIO()
            stderr = io.StringIO()
            synthetic_context = {
                "GITHUB_ACTIONS": "true",
                "GITHUB_WORKFLOW": "Synthetic portable workflow",
                "GITHUB_JOB": "synthetic-portable",
                "RUNNER_OS": "SyntheticOS",
            }
            with mock.patch.dict(os.environ, synthetic_context, clear=False):
                with contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr):
                    result = self.runner.run_stage(stage, root)
            self.assertEqual(result, 7)
            failure = stderr.getvalue()
            for marker in (
                "FAIL: portable validation stage failed",
                "failure_code=ISRAS-PORTABLE-SYNTHETIC-001",
                "stage=synthetic-stage",
                "validator=tools/isras/synthetic-validator.py",
                "bootstrap=tools/isras/invoke_repo_tool.py",
                "tested_commit=UNRESOLVED",
                "workflow=Synthetic portable workflow",
                "job=synthetic-portable",
                "runner_os=SyntheticOS",
                "command=",
                "exit_code=7",
                "::error title=ISRAS portable validation failed::",
            ):
                self.assertIn(marker, failure)

    def test_history_failure_emits_required_commit_and_fetch_result(self) -> None:
        requirement = self.history.Requirement(
            commit="08a0a514ec308f76dbf80ffdcb8caa70ce6e345f",
            purposes={"classification base test"},
            fetch_refs={"08a0a514ec308f76dbf80ffdcb8caa70ce6e345f"},
        )
        details = {
            "tested_commit": "UNRESOLVED",
            "remote_url": "origin",
            "workflow": "LOCAL",
            "job": "LOCAL",
            "runner_os": "test",
        }
        process = subprocess.CompletedProcess(
            args=["git", "fetch"],
            returncode=128,
            stdout="",
            stderr="fatal: missing object",
        )
        stderr = io.StringIO()
        with contextlib.redirect_stderr(stderr):
            self.history.print_failure(
                requirement,
                details,
                next(iter(requirement.fetch_refs)),
                process,
            )
        failure = stderr.getvalue()
        for marker in (
            "FAIL: required historical commit unavailable",
            "failure_code=ISRAS-CI-HISTORY-001",
            "required_commit=08a0a514ec308f76dbf80ffdcb8caa70ce6e345f",
            "purpose=classification base test",
            "observed=required commit object is unavailable",
            "fetch_attempted=true",
            "fetch_exit_code=128",
            "fetch_stderr=fatal: missing object",
        ):
            self.assertIn(marker, failure)


if __name__ == "__main__":
    unittest.main()
