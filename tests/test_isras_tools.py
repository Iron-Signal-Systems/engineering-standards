from __future__ import annotations

import json
import os
import shutil
import stat
import subprocess
import tempfile
import unittest
from pathlib import Path


STANDARDS_ROOT = Path(__file__).resolve().parents[1]
PYTHON = shutil.which("python3") or shutil.which("python")


def run(args, cwd=None, check=True):
    result = subprocess.run(
        [str(x) for x in args],
        cwd=str(cwd) if cwd else None,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if check and result.returncode != 0:
        raise AssertionError(
            f"command failed: {' '.join(map(str,args))}\n"
            f"stdout:\n{result.stdout}\nstderr:\n{result.stderr}"
        )
    return result


class ISRToolsTests(unittest.TestCase):
    def make_repo(self, base: Path, name: str = "sample") -> Path:
        repo = base / name
        run(["git", "init", "-b", "dev", repo])
        run(["git", "config", "user.name", "ISRAS Test"], cwd=repo)
        run(["git", "config", "user.email", "isras-test@example.invalid"], cwd=repo)
        (repo / "README.md").write_text("# Sample\n", encoding="utf-8")
        run(["git", "add", "."], cwd=repo)
        run(["git", "commit", "-m", "initial"], cwd=repo)
        return repo

    def adopt(self, repo: Path, origin: str) -> None:
        run([
            PYTHON,
            STANDARDS_ROOT / "tools/isras/adopt.py",
            "--target", repo,
            "--repository", "Iron-Signal-Systems/sample",
            "--canonical-origin", origin,
            "--development-branch", "dev",
            "--release-branch", "main",
            "--profile", "general",
        ], cwd=STANDARDS_ROOT)

    def test_adopter_and_policy_validation(self):
        with tempfile.TemporaryDirectory() as temp:
            base = Path(temp)
            repo = self.make_repo(base)
            origin = str(base / "remote.git")
            self.adopt(repo, origin)
            manifest = json.loads((repo / "REPOSITORY-ASSURANCE.json").read_text())
            self.assertEqual(manifest["repository"], "Iron-Signal-Systems/sample")
            self.assertEqual(manifest["canonical_origin"], origin)
            result = run([
                PYTHON,
                repo / "tools/isras/validate_policy.py",
                "--repo-root", repo,
            ], cwd=repo)
            self.assertIn("ISRAS policy validation PASSED", result.stdout)

    def test_adopter_refuses_overwrite(self):
        with tempfile.TemporaryDirectory() as temp:
            base = Path(temp)
            repo = self.make_repo(base)
            origin = str(base / "remote.git")
            self.adopt(repo, origin)
            result = run([
                PYTHON,
                STANDARDS_ROOT / "tools/isras/adopt.py",
                "--target", repo,
                "--repository", "Iron-Signal-Systems/sample",
                "--canonical-origin", origin,
                "--development-branch", "dev",
                "--release-branch", "main",
                "--profile", "general",
            ], cwd=STANDARDS_ROOT, check=False)
            self.assertEqual(result.returncode, 2)
            self.assertIn("Refusing to overwrite", result.stdout)


    def test_skip_existing_writes_only_missing_files(self):
        with tempfile.TemporaryDirectory() as temp:
            base = Path(temp)
            repo = self.make_repo(base)
            origin = str(base / "remote.git")
            security = repo / "SECURITY.md"
            security.write_text("# Existing security policy\n", encoding="utf-8")
            result = run([
                PYTHON,
                STANDARDS_ROOT / "tools/isras/adopt.py",
                "--target", repo,
                "--repository", "Iron-Signal-Systems/sample",
                "--canonical-origin", origin,
                "--development-branch", "dev",
                "--release-branch", "main",
                "--profile", "general",
                "--skip-existing",
            ], cwd=STANDARDS_ROOT)
            self.assertIn("SKIP: SECURITY.md", result.stdout)
            self.assertEqual(security.read_text(), "# Existing security policy\n")
            self.assertTrue((repo / "REPOSITORY-ASSURANCE.json").exists())

    def test_fresh_clone_and_historical_checkpoint(self):
        with tempfile.TemporaryDirectory() as temp:
            base = Path(temp)
            remote = base / "remote.git"
            run(["git", "init", "--bare", remote])
            repo = self.make_repo(base)
            self.adopt(repo, str(remote))
            run(["git", "remote", "add", "origin", remote], cwd=repo)

            gate = repo / "tools/validation/phase-gates/test_checkpoint.sh"
            gate.parent.mkdir(parents=True, exist_ok=True)
            gate.write_text(
                "#!/usr/bin/env bash\n"
                "set -Eeuo pipefail\n"
                "test \"$(git branch --show-current)\" = dev\n"
                "test -f REPOSITORY-ASSURANCE.json\n",
                encoding="utf-8",
            )
            gate.chmod(gate.stat().st_mode | stat.S_IXUSR)
            run(["git", "add", "."], cwd=repo)
            run(["git", "commit", "-m", "adopt assurance"], cwd=repo)
            checkpoint_commit = run(["git", "rev-parse", "HEAD"], cwd=repo).stdout.strip()

            registry_path = repo / "tools/validation/checkpoints.json"
            registry = json.loads(registry_path.read_text())
            registry["checkpoints"]["test-checkpoint"] = {
                "status": "accepted",
                "commit": checkpoint_commit,
                "tag": None,
                "gate": "tools/validation/phase-gates/test_checkpoint.sh",
                "environment_profile": "portable",
                "required_branch_name": "dev",
                "expected_result": {"fail": 0},
            }
            registry_path.write_text(json.dumps(registry, indent=2) + "\n")
            run(["git", "add", registry_path], cwd=repo)
            run(["git", "commit", "-m", "record accepted checkpoint"], cwd=repo)
            run(["git", "push", "-u", "origin", "dev"], cwd=repo)

            fresh = run([
                PYTHON,
                repo / "tools/isras/validate_fresh_clone.py",
                "--repo-root", repo,
            ], cwd=repo)
            self.assertIn("Fresh-clone and remote-completeness validation PASSED", fresh.stdout)

            historical = run([
                PYTHON,
                repo / "tools/isras/validate_checkpoint.py",
                "--repo-root", repo,
                "--checkpoint", "test-checkpoint",
            ], cwd=repo)
            self.assertIn("Historical checkpoint validation PASSED", historical.stdout)


if __name__ == "__main__":
    unittest.main()
