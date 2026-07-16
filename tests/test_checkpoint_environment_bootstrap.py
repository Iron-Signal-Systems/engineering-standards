from __future__ import annotations

import importlib.util
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path, PureWindowsPath
from unittest import mock

sys.dont_write_bytecode = True
ROOT = Path(__file__).resolve().parents[1]
ISRAS = ROOT / "tools/isras"
if str(ISRAS) not in sys.path:
    sys.path.insert(0, str(ISRAS))


def load_checkpoint_module():
    path = ISRAS / "validate_checkpoint.py"
    spec = importlib.util.spec_from_file_location("validate_checkpoint_bootstrap_test", path)
    if spec is None or spec.loader is None:
        raise RuntimeError("unable to load validate_checkpoint.py")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


checkpoint = load_checkpoint_module()


class HistoricalCheckpointBootstrapTests(unittest.TestCase):
    def test_posix_spec_uses_accepted_tree_shell_bootstrap(self) -> None:
        clone = Path("/tmp/accepted-checkpoint")
        spec = checkpoint.historical_environment_spec(clone, "posix")
        self.assertEqual(spec.bootstrap, clone / "tools/environment/bootstrap_tools.sh")
        self.assertEqual(spec.venv, clone / ".isras-tools-venv")
        self.assertEqual(spec.python, spec.venv / "bin/python")
        self.assertEqual(spec.command, ("bash", str(spec.bootstrap)))

    def test_windows_spec_uses_accepted_tree_powershell_bootstrap(self) -> None:
        clone = PureWindowsPath("D:/accepted-checkpoint")
        spec = checkpoint.historical_environment_spec(clone, "nt")
        self.assertEqual(
            PureWindowsPath(spec.bootstrap),
            clone / "tools/environment/Bootstrap-Tools.ps1",
        )
        self.assertEqual(PureWindowsPath(spec.python), clone / ".isras-tools-venv/Scripts/python.exe")
        self.assertEqual(spec.command[0:3], ("pwsh", "-NoProfile", "-File"))
        self.assertEqual(PureWindowsPath(spec.command[3]), PureWindowsPath(spec.bootstrap))
        self.assertEqual(spec.command[4], "-VenvPath")
        self.assertEqual(PureWindowsPath(spec.command[5]), PureWindowsPath(spec.venv))

    def test_bootstrap_invokes_historical_script_and_returns_created_python(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            clone = Path(temp)
            bootstrap = clone / "tools/environment/bootstrap_tools.sh"
            bootstrap.parent.mkdir(parents=True)
            bootstrap.write_text("#!/usr/bin/env bash\n", encoding="utf-8")
            observed: dict[str, object] = {}

            def fake_run(args, *, cwd=None, env=None, **_kwargs):
                observed["args"] = tuple(args)
                observed["cwd"] = cwd
                observed["venv"] = env.get("ISRAS_TOOLS_VENV")
                python = clone / ".isras-tools-venv/bin/python"
                python.parent.mkdir(parents=True)
                python.write_text("synthetic python\n", encoding="utf-8")
                return subprocess.CompletedProcess(args, 0)

            with mock.patch.object(checkpoint, "run", side_effect=fake_run):
                spec = checkpoint.create_historical_environment(clone, "posix")

            self.assertEqual(observed["args"], ("bash", str(bootstrap)))
            self.assertEqual(observed["cwd"], clone)
            self.assertEqual(observed["venv"], str(clone / ".isras-tools-venv"))
            self.assertEqual(spec.python, clone / ".isras-tools-venv/bin/python")

    def test_bootstrap_rejects_missing_historical_script(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            clone = Path(temp)
            with self.assertRaisesRegex(
                checkpoint.ISRASError,
                "historical tool bootstrap is missing",
            ):
                checkpoint.create_historical_environment(clone, "posix")

    def test_bootstrap_rejects_missing_declared_python(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            clone = Path(temp)
            bootstrap = clone / "tools/environment/bootstrap_tools.sh"
            bootstrap.parent.mkdir(parents=True)
            bootstrap.write_text("#!/usr/bin/env bash\n", encoding="utf-8")
            with mock.patch.object(
                checkpoint,
                "run",
                return_value=subprocess.CompletedProcess([], 0),
            ):
                with self.assertRaisesRegex(
                    checkpoint.ISRASError,
                    "did not create its declared Python",
                ):
                    checkpoint.create_historical_environment(clone, "posix")

    def test_gate_environment_preserves_context_and_pins_historical_python(self) -> None:
        specification = checkpoint.HistoricalToolEnvironment(
            bootstrap=Path("/accepted/tools/environment/bootstrap_tools.sh"),
            venv=Path("/accepted/.isras-tools-venv"),
            python=Path("/accepted/.isras-tools-venv/bin/python"),
            command=("bash", "bootstrap"),
        )
        with mock.patch.dict(os.environ, {"SYNTHETIC_CONTEXT": "retained"}, clear=False):
            environment = checkpoint.checkpoint_gate_environment(specification)
        self.assertEqual(environment["SYNTHETIC_CONTEXT"], "retained")
        self.assertEqual(environment["ISRAS_TOOLS_VENV"], str(specification.venv))
        self.assertEqual(environment["ISRAS_PYTHON"], str(specification.python))

    def test_main_bootstraps_before_executing_historical_gate(self) -> None:
        source = (ISRAS / "validate_checkpoint.py").read_text(encoding="utf-8")
        bootstrap = source.index("tool_environment = create_historical_environment(clone)")
        shell_gate = source.index('["bash", str(gate_path)]')
        powershell_gate = source.index('["pwsh", "-NoProfile", "-File", str(gate_path)]')
        self.assertLess(bootstrap, shell_gate)
        self.assertLess(bootstrap, powershell_gate)
        self.assertIn("env=gate_environment", source)


if __name__ == "__main__":
    unittest.main()
