#!/usr/bin/env python3
from __future__ import annotations

import copy
import hashlib
import json
import os
import platform
import subprocess
import sys
import sysconfig
import tempfile
import unittest
from pathlib import Path

from jsonschema import Draft202012Validator, FormatChecker

ROOT = Path(__file__).resolve().parents[1]
PYTHON = os.environ.get("ISRAS_PYTHON", sys.executable)


def sha512(path: Path) -> str:
    h = hashlib.sha512()
    h.update(path.read_bytes())
    return h.hexdigest()


def run_tool(relative: str, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [PYTHON, str(ROOT / relative), "--repo-root", str(ROOT), *args],
        cwd=ROOT,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )


def git(*args: str, cwd: Path = ROOT) -> str:
    result = subprocess.run(
        ["git", *args], cwd=cwd, text=True,
        stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True,
    )
    return result.stdout.strip()


class ISRASV3HardeningTests(unittest.TestCase):
    def test_schemas_and_templates_are_valid(self):
        pairs = {
            "templates/engineering-standards/evidence-binding.json": "schemas/evidence-binding-v1.schema.json",
            "templates/engineering-standards/change-classification.json": "schemas/change-classification-v1.schema.json",
            "templates/engineering-standards/github-control-evidence.json": "schemas/github-control-evidence-v1.schema.json",
            "templates/engineering-standards/tool-bootstrap-lock.json": "schemas/tool-bootstrap-lock-v1.schema.json",
            "docs/acceptance/isras-v3.0.0-change-classification.json": "schemas/change-classification-v1.schema.json",
            "docs/engineering/external-standards-crosswalk.json": "schemas/external-standards-crosswalk-v1.schema.json",
            "REPOSITORY-ASSURANCE.json": "schemas/repository-assurance-v1.schema.json",
        }
        for instance_name, schema_name in pairs.items():
            with self.subTest(instance=instance_name):
                schema = json.loads((ROOT / schema_name).read_text(encoding="utf-8"))
                Draft202012Validator.check_schema(schema)
                instance = json.loads((ROOT / instance_name).read_text(encoding="utf-8"))
                errors = list(Draft202012Validator(schema, format_checker=FormatChecker()).iter_errors(instance))
                self.assertFalse(errors, errors[0].message if errors else "")

    def test_release_assured_schema_rejects_self_commit(self):
        schema = json.loads((ROOT / "schemas/repository-assurance-v1.schema.json").read_text())
        record = json.loads((ROOT / "REPOSITORY-ASSURANCE.json").read_text())
        record["standard"]["commit"] = "SELF"
        errors = list(Draft202012Validator(schema).iter_errors(record))
        self.assertTrue(errors)

    def test_repository_self_assurance_is_consistent(self):
        data = json.loads((ROOT / "REPOSITORY-ASSURANCE.json").read_text())
        standard = data["standard"]
        self_record = data["self_assurance"]
        self.assertEqual(data["adoption_level"], "RELEASE_ASSURED")
        self.assertEqual(self_record["governing_release"], standard["tag"])
        self.assertEqual(self_record["governing_commit"], standard["commit"])
        self.assertEqual(
            self_record["governing_source_manifest_sha256"],
            standard["source_manifest_sha256"],
        )

    def test_bootstrap_has_no_implicit_upgrade_and_release_is_isolated(self):
        files = [
            ROOT / "tools/environment/bootstrap_tools.sh",
            ROOT / "tools/environment/Bootstrap-Tools.ps1",
            ROOT / "tools/environment/bootstrap_tools_release.sh",
            ROOT / "tools/environment/Bootstrap-Tools-Release.ps1",
        ]
        combined = "\n".join(path.read_text(encoding="utf-8") for path in files)
        self.assertNotIn("pip install --upgrade pip", combined)
        shell = files[2].read_text(encoding="utf-8")
        for marker in (
            "release tool environment path already exists", "-I", "--isolated",
            "--no-index", "--no-cache-dir", "--only-binary=:all:",
            "--require-hashes", "--force-reinstall", "clean_tool_venv.py",
        ):
            self.assertIn(marker, shell)

    def test_wheelhouse_preparer_excludes_reports_and_retains_provenance(self):
        text = (ROOT / "tools/environment/prepare_tool_wheelhouse.py").read_text()
        self.assertIn("wheelhouse output directory must be empty", text)
        self.assertNotIn("resolution-report.json", text)
        self.assertIn("source_hashes", text)
        self.assertIn("source_url", text)
        self.assertIn("TemporaryDirectory", text)

    def test_prebootstrap_verifier_accepts_exact_set_and_rejects_extra_report(self):
        with tempfile.TemporaryDirectory() as temporary:
            wheelhouse = Path(temporary)
            wheels = wheelhouse / "wheels"
            wheels.mkdir()
            pip_wheel = wheels / "pip-25.1-py3-none-any.whl"
            dep_wheel = wheels / "example_pkg-1.0-py3-none-any.whl"
            pip_wheel.write_bytes(b"pip-wheel")
            dep_wheel.write_bytes(b"dependency-wheel")
            requirements = wheelhouse / "requirements.lock"
            requirements.write_text(
                f"example-pkg==1.0 --hash=sha512:{sha512(dep_wheel)}\n",
                encoding="utf-8",
            )
            bootstrap_pip = wheelhouse / "bootstrap-pip.lock"
            bootstrap_pip.write_text(
                f"pip==25.1 --hash=sha512:{sha512(pip_wheel)}\n",
                encoding="utf-8",
            )
            req_source = ROOT / "tools/requirements.txt"
            def provenance(path: Path) -> dict:
                return {
                    "source_url": f"https://example.invalid/{path.name}",
                    "source_hashes": {"sha256": hashlib.sha256(path.read_bytes()).hexdigest()},
                }
            lock = {
                "schema_version": "ISRAS-TOOL-BOOTSTRAP-LOCK-V1",
                "created_at": "2026-07-16T00:00:00Z",
                "platform": {
                    "system": platform.system(),
                    "machine": platform.machine(),
                    "python_implementation": platform.python_implementation(),
                    "python_abi": sysconfig.get_config_var("SOABI") or "unknown",
                },
                "python": platform.python_version(),
                "python_executable_sha512": sha512(Path(sys.executable).resolve()),
                "requirements_source_sha512": sha512(req_source),
                "resolver_pip_version": "25.1",
                "pip_wheel": {
                    "name": "pip", "version": "25.1", "filename": pip_wheel.name,
                    "sha512": sha512(pip_wheel), **provenance(pip_wheel),
                },
                "requirements_lock_sha512": sha512(requirements),
                "artifacts": [{
                    "name": "example-pkg", "version": "1.0", "filename": dep_wheel.name,
                    "sha512": sha512(dep_wheel), **provenance(dep_wheel),
                }],
            }
            lock_path = wheelhouse / "bootstrap-lock.json"
            lock_path.write_text(json.dumps(lock, indent=2) + "\n")
            files = [lock_path, bootstrap_pip, requirements, pip_wheel, dep_wheel]
            (wheelhouse / "SHA512SUMS").write_text(
                "".join(f"{sha512(path)}  {path.relative_to(wheelhouse).as_posix()}\n" for path in sorted(files)),
                encoding="utf-8",
            )
            result = run_tool("tools/environment/verify_wheelhouse.py", "--wheelhouse", str(wheelhouse))
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            (wheelhouse / "resolution-report.json").write_text("{}\n")
            result = run_tool("tools/environment/verify_wheelhouse.py", "--wheelhouse", str(wheelhouse))
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("extra", result.stdout)

    def test_actual_change_classification_passes(self):
        result = run_tool(
            "tools/isras/validate_change_classification.py",
            "--record", "docs/acceptance/isras-v3.0.0-change-classification.json",
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_c4_schema_campaign_does_not_inherit_c3_security_campaigns(self):
        record = json.loads((ROOT / "templates/engineering-standards/change-classification.json").read_text())
        record.update({"selected_class": "C4", "classification_rationale": "Schema-only candidate used to verify parallel campaign branches."})
        record["impacts"] = {
            "editorial_only": False, "tooling": False, "implementation": False,
            "security_or_authority": False, "schema_or_migration": True,
            "acceptance_semantics": False, "release_or_recovery": False,
        }
        record["campaigns"] = [
            "policy", "documentation-sync", "links", "whitespace",
            "portable", "unit-regression", "tool-environment", "fresh-clone",
            "integration", "traceability", "phase-review", "exact-pushed-source",
            "migration-integrity", "compatibility", "rollback-restore",
            "representative-data", "destructive-safeguards", "historical-migration",
        ]
        record["applicable_controls"] = ["ISRAS-DAT-001"]
        with tempfile.NamedTemporaryFile("w", suffix=".json", dir=ROOT, delete=False) as handle:
            json.dump(record, handle)
            path = Path(handle.name)
        try:
            result = run_tool(
                "tools/isras/validate_change_classification.py",
                "--record", path.relative_to(ROOT).as_posix(), "--skip-diff-floor",
            )
        finally:
            path.unlink(missing_ok=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertNotIn("threat-abuse", record["campaigns"])

    def test_underclassified_acceptance_change_fails(self):
        record = json.loads((ROOT / "templates/engineering-standards/change-classification.json").read_text())
        record["selected_class"] = "C1"
        record["impacts"]["editorial_only"] = False
        record["impacts"]["acceptance_semantics"] = True
        with tempfile.NamedTemporaryFile("w", suffix=".json", dir=ROOT, delete=False) as handle:
            json.dump(record, handle)
            path = Path(handle.name)
        try:
            result = run_tool(
                "tools/isras/validate_change_classification.py",
                "--record", path.relative_to(ROOT).as_posix(), "--skip-diff-floor",
            )
        finally:
            path.unlink(missing_ok=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("below required C5", result.stdout)

    def test_control_level_crosswalk_passes_candidate_and_blocks_formal_unpinned(self):
        data = json.loads((ROOT / "docs/engineering/external-standards-crosswalk.json").read_text())
        self.assertTrue(data["mappings"])
        self.assertNotIn("COVERED", {item["state"] for item in data["mappings"]})
        result = run_tool(
            "tools/isras/validate_external_standards_crosswalk.py",
            "--record", "docs/engineering/external-standards-crosswalk.json",
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        formal = run_tool(
            "tools/isras/validate_external_standards_crosswalk.py",
            "--record", "docs/engineering/external-standards-crosswalk.json",
            "--require-all-pinned",
        )
        self.assertNotEqual(formal.returncode, 0)
        self.assertIn("unpinned baselines", formal.stdout)

    def _github_record(self) -> dict:
        required_checks = ["policy", "portable", "integration-tools", "native-os-matrix"]
        rulesets = [
            {
                "id": 1, "name": "dev", "target": "branch", "enforcement": "active",
                "conditions": {"ref_name": {"include": ["refs/heads/dev"], "exclude": []}},
                "rules": [
                    {"type": "deletion"}, {"type": "non_fast_forward"},
                    {"type": "pull_request"},
                    {"type": "required_status_checks", "parameters": {"required_status_checks": [{"context": value, "integration_id": 0} for value in required_checks]}},
                ],
                "bypass_actors": [],
            },
            {
                "id": 2, "name": "main", "target": "branch", "enforcement": "active",
                "conditions": {"ref_name": {"include": ["refs/heads/main"], "exclude": []}},
                "rules": [{"type": "deletion"}, {"type": "non_fast_forward"}],
                "bypass_actors": [],
            },
            {
                "id": 3, "name": "tags", "target": "tag", "enforcement": "active",
                "conditions": {"ref_name": {"include": ["refs/tags/isras-*"], "exclude": []}},
                "rules": [{"type": "creation"}, {"type": "update"}, {"type": "deletion"}],
                "bypass_actors": [],
            },
        ]
        head = git("rev-parse", "HEAD")
        tree = git("rev-parse", "HEAD^{tree}")
        raw = {
            "repository": "Iron-Signal-Systems/engineering-standards",
            "source_commit": head, "commit_tree_sha": tree,
            "default_branch": "dev", "rulesets": rulesets,
            "branch_protection": {"dev": {"status": "NOT_CONFIGURED"}, "main": {"status": "NOT_CONFIGURED"}},
        }
        return {
            "schema_version": "ISRAS-GITHUB-CONTROL-EVIDENCE-V1",
            **raw,
            "collected_at": "2026-07-16T00:00:00Z",
            "collector": {"tool": "test", "version": "1", "actor": "tester"},
            "raw_configuration_sha512": hashlib.sha512(json.dumps(raw, sort_keys=True, separators=(",", ":")).encode()).hexdigest(),
        }

    def _validate_github_record(self, record: dict) -> subprocess.CompletedProcess[str]:
        with tempfile.NamedTemporaryFile("w", suffix=".json", dir=ROOT, delete=False) as handle:
            json.dump(record, handle)
            path = Path(handle.name)
        try:
            args = [
                "--record", path.relative_to(ROOT).as_posix(),
                "--expected-commit", record["source_commit"],
                "--expected-repository", record["repository"],
            ]
            for check in ("policy", "portable", "integration-tools", "native-os-matrix"):
                args.extend(["--required-dev-check", check])
            return run_tool("tools/isras/validate_github_control_evidence.py", *args)
        finally:
            path.unlink(missing_ok=True)

    def test_github_ruleset_target_include_exclude_and_named_checks(self):
        record = self._github_record()
        result = self._validate_github_record(record)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

        excluded = copy.deepcopy(record)
        excluded["rulesets"][0]["conditions"]["ref_name"]["exclude"] = ["refs/heads/dev"]
        raw = {key: excluded[key] for key in ("repository", "source_commit", "commit_tree_sha", "default_branch", "rulesets", "branch_protection")}
        excluded["raw_configuration_sha512"] = hashlib.sha512(json.dumps(raw, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
        result = self._validate_github_record(excluded)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("lacks effective protection", result.stdout)

    def test_evidence_binding_reads_internal_identity_and_real_outcome(self):
        head = git("rev-parse", "HEAD")
        assurance = json.loads((ROOT / "REPOSITORY-ASSURANCE.json").read_text())
        repository = assurance["repository"]
        validator_relative = "tools/isras/validate_policy.py"
        validator = ROOT / validator_relative
        with tempfile.TemporaryDirectory(dir=ROOT) as temporary:
            directory = Path(temporary)
            environment = directory / "environment.json"
            environment.write_text('{"environment":"test"}\n')
            environment_digest = sha512(environment)
            artifact = directory / "result.log"
            campaign = "test-campaign"
            artifact.write_text(
                f"candidate_commit: {head}\n"
                f"campaign_id: {campaign}\n"
                f"environment_sha512: {environment_digest}\n"
                "PASS checks: 2\nFAIL checks: 0\nvalidation PASSED.\n"
            )
            binding = {
                "schema_version": "ISRAS-EVIDENCE-BINDING-V1",
                "repository": repository, "campaign_id": campaign,
                "candidate_commit": head,
                "environment_artifact": {"path": environment.relative_to(ROOT).as_posix(), "sha512": environment_digest},
                "declared_test_ids": ["TEST-1"],
                "validators": [{
                    "validator_id": "policy", "version": "test", "source_commit": head,
                    "executable_path": validator_relative, "executable_sha512": sha512(validator),
                }],
                "artifacts": [{
                    "artifact_id": "result", "path": artifact.relative_to(ROOT).as_posix(),
                    "sha512": sha512(artifact), "validator_id": "policy", "test_ids": ["TEST-1"],
                    "identity_probe": {
                        "format": "TEXT",
                        "candidate_commit_marker": f"candidate_commit: {head}",
                        "campaign_id_marker": f"campaign_id: {campaign}",
                        "environment_sha512_marker": f"environment_sha512: {environment_digest}",
                    },
                    "outcome_probe": {"format": "TEXT_ISRAS_SUMMARY", "pass_marker": "validation PASSED."},
                }],
                "claims": [{
                    "claim_id": "claim", "control_id": "ISRAS-EVD-001",
                    "required_test_ids": ["TEST-1"], "artifact_ids": ["result"],
                    "expected_outcome": "PASS",
                }],
            }
            binding_path = directory / "binding.json"
            binding_path.write_text(json.dumps(binding))
            result = run_tool(
                "tools/isras/validate_evidence_relationships.py",
                "--binding", binding_path.relative_to(ROOT).as_posix(),
                "--expected-commit", head, "--expected-repository", repository,
                "--allow-untracked-fixtures",
            )
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            binding["artifacts"][0]["identity_probe"]["campaign_id_marker"] = "campaign_id: wrong"
            binding_path.write_text(json.dumps(binding))
            result = run_tool(
                "tools/isras/validate_evidence_relationships.py",
                "--binding", binding_path.relative_to(ROOT).as_posix(),
                "--expected-commit", head, "--expected-repository", repository,
                "--allow-untracked-fixtures",
            )
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("exact bound identity", result.stdout)

    def test_sha512_manifest_uses_index_not_working_tree(self):
        generator = ROOT / "tools/isras/generate_sha512_manifest.py"
        verifier = ROOT / "tools/isras/verify_sha512_manifest.py"
        with tempfile.TemporaryDirectory() as temporary:
            repo = Path(temporary)
            subprocess.run(["git", "init", "-q", repo], check=True)
            subprocess.run(["git", "config", "user.email", "test@example.invalid"], cwd=repo, check=True)
            subprocess.run(["git", "config", "user.name", "Test"], cwd=repo, check=True)
            (repo / "sample.txt").write_text("staged\n")
            subprocess.run(["git", "add", "sample.txt"], cwd=repo, check=True)
            (repo / "sample.txt").write_text("working-tree-only\n")
            result = subprocess.run([PYTHON, str(generator), "--repo-root", str(repo)], text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            subprocess.run(["git", "add", "SOURCE-SHA512SUMS.txt"], cwd=repo, check=True)
            result = subprocess.run([PYTHON, str(verifier), "--repo-root", str(repo)], text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            subprocess.run(["git", "add", "sample.txt"], cwd=repo, check=True)
            result = subprocess.run([PYTHON, str(verifier), "--repo-root", str(repo)], text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            self.assertNotEqual(result.returncode, 0)


if __name__ == "__main__":
    unittest.main()
