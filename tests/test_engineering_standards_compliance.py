from __future__ import annotations

import importlib.util
import tempfile
import tarfile
import io
import json
import subprocess
import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PYTHON = sys.executable


def load_module(name: str, path: Path):
    spec = importlib.util.spec_from_file_location(name, path)
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


VALIDATOR = load_module(
    "isras_v2_compliance_validator",
    ROOT / "tools/isras/validate_engineering_standards_compliance.py",
)


def load(path: str):
    return json.loads((ROOT / path).read_text(encoding="utf-8"))


def validate_records(phase, impact=None, authority=None, require_type=None, require_pass=False):
    results = VALIDATOR.Results()
    phase_schema = load("schemas/phase-standards-compliance-v1.schema.json")
    VALIDATOR.schema_validate(phase, phase_schema, "phase", results)
    if not results.failures:
        VALIDATOR.validate_phase(phase, results, require_type, require_pass)

    if impact is not None:
        impact_schema = load("schemas/engineering-standards-impact-assessment-v1.schema.json")
        before = len(results.failures)
        VALIDATOR.schema_validate(impact, impact_schema, "impact", results)
        if len(results.failures) == before:
            VALIDATOR.validate_impact(impact, results, False)

    if phase.get("newer_release_review", {}).get("impact_assessment_required"):
        results.check(
            impact is not None,
            "required Engineering Standards Impact Assessment was supplied",
        )

    if authority is not None:
        authority_schema = load("schemas/authority-boundary-record-v1.schema.json")
        before = len(results.failures)
        VALIDATOR.schema_validate(authority, authority_schema, "authority", results)
        if len(results.failures) == before:
            VALIDATOR.validate_authority(
                authority,
                results,
                phase.get("review_type") == "EXIT"
                and phase.get("hostile_validation", {}).get("required", False),
            )
        results.check(
            bool(phase.get("hostile_validation", {}).get("authority_boundary_records")),
            "all linked authority boundary records were supplied",
        )
    return results


def accepted_exit():
    phase = load("templates/engineering-standards/phase-exit-review.json")
    phase["review_status"] = "PASS"
    phase["source_commit"] = "1" * 40
    phase["adopted_isras"]["commit"] = "2" * 40
    phase["adopted_isras"]["source_manifest_sha256"] = "3" * 64
    phase["controls"][0].update(
        {
            "required_maturity": "VALIDATED",
            "actual_maturity": "VALIDATED",
            "compliance_result": "PASS",
            "evidence_paths": ["evidence/control.json"],
            "planned_actions": [],
        }
    )
    phase["synchronization"] = {
        key: "SYNCHRONIZED" for key in phase["synchronization"]
    }
    phase["hostile_validation"].update(
        {
            "required": True,
            "status": "PASS",
            "authority_boundary_records": ["evidence/authority.json"],
            "evidence_paths": ["evidence/hostile.json"],
        }
    )
    phase["historical_predecessor_status"] = "PASS"
    phase["exact_pushed_candidate_evaluated"] = True
    phase["approval"] = {
        "decision": "APPROVED",
        "approver": "reviewer",
        "date": "2026-01-02",
    }
    phase["decision"] = "PHASE_ACCEPTED"
    phase["reviewer"] = "reviewer"
    return phase


def validated_authority():
    record = load("templates/engineering-standards/authority-boundary-record.json")
    record["maturity"] = "VALIDATED"
    record["review"] = {
        "status": "PASS",
        "reviewer": "reviewer",
        "date": "2026-01-02",
    }
    record["hostile_tests"] = [
        {
            "test_class": "forged work envelope",
            "applicability": "APPLICABLE",
            "status": "PASS",
            "evidence_path": "evidence/forged.json",
        }
    ]
    record["evidence_paths"] = ["evidence/boundary.json"]
    return record


class EngineeringStandardsComplianceTests(unittest.TestCase):
    def test_release_version_is_v2_0_0(self):
        self.assertEqual(
            (ROOT / "VERSION").read_text(encoding="utf-8").strip(),
            "2.0.0",
        )

    def test_candidate_validator_passes(self):
        result = subprocess.run(
            [
                PYTHON,
                ROOT / "tools/isras/validate_isras_v2_candidate.py",
                "--repo-root",
                ROOT,
                "--skip-v1-git-diff",
                "--release-finalization",
            ],
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_release_source_validator_passes_at_accepted_commit(self):
        release_commit = (
            "781246e69f8a9a382c25040f94b62dfe3b25ba89"
        )

        archive = subprocess.run(
            ["git", "archive", release_commit],
            cwd=ROOT,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
        self.assertEqual(
            archive.returncode,
            0,
            archive.stderr.decode("utf-8", errors="replace"),
        )

        with tempfile.TemporaryDirectory(
            prefix="isras-v2-release-source-"
        ) as temporary:
            checkout = Path(temporary)

            with tarfile.open(
                fileobj=io.BytesIO(archive.stdout),
                mode="r:",
            ) as handle:
                handle.extractall(checkout)

            result = subprocess.run(
                [
                    PYTHON,
                    checkout / "tools/isras/validate_isras_v2_release.py",
                    "--repo-root",
                    checkout,
                    "--skip-v1-git-diff",
                ],
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                check=False,
            )

        self.assertEqual(
            result.returncode,
            0,
            result.stdout + result.stderr,
        )

    def test_validated_exit_and_authority_pass(self):
        results = validate_records(
            accepted_exit(),
            authority=validated_authority(),
            require_type="EXIT",
            require_pass=True,
        )
        self.assertFalse(results.failures, results.failures)

    def test_unrestricted_execution_context_is_rejected(self):
        authority = validated_authority()
        authority["privilege_propagation"][
            "unrestricted_execution_context_prohibited"
        ] = False
        results = validate_records(
            accepted_exit(), authority=authority, require_type="EXIT", require_pass=True
        )
        self.assertTrue(results.failures)
        self.assertTrue(
            any("unrestricted execution contexts" in item for item in results.failures)
        )

    def test_automatic_privilege_propagation_is_rejected(self):
        authority = validated_authority()
        authority["privilege_propagation"][
            "automatic_propagation_prohibited"
        ] = False
        results = validate_records(
            accepted_exit(), authority=authority, require_type="EXIT", require_pass=True
        )
        self.assertTrue(results.failures)

    def test_database_owner_runtime_is_rejected(self):
        authority = validated_authority()
        authority["database_authority"]["applicable"] = True
        authority["database_authority"]["runtime_uses_owner_or_superuser"] = True
        results = validate_records(
            accepted_exit(), authority=authority, require_type="EXIT", require_pass=True
        )
        self.assertTrue(results.failures)

    def test_maturity_overclaim_is_rejected(self):
        phase = accepted_exit()
        phase["controls"][0]["actual_maturity"] = "DOCUMENTED"
        results = validate_records(
            phase,
            authority=validated_authority(),
            require_type="EXIT",
            require_pass=True,
        )
        self.assertTrue(
            any("meets required exit maturity" in item for item in results.failures)
        )

    def test_missing_hostile_evidence_is_rejected(self):
        authority = validated_authority()
        authority["hostile_tests"][0]["status"] = "NOT_RUN"
        authority["hostile_tests"][0]["evidence_path"] = None
        results = validate_records(
            accepted_exit(), authority=authority, require_type="EXIT", require_pass=True
        )
        self.assertTrue(results.failures)

    def test_required_impact_assessment_must_be_supplied(self):
        phase = accepted_exit()
        phase["newer_release_review"] = {
            "newer_accepted_release_exists": True,
            "impact_assessment_required": True,
            "impact_assessment_path": "docs/engineering/esia.json",
        }
        results = validate_records(
            phase,
            authority=validated_authority(),
            require_type="EXIT",
            require_pass=True,
        )
        self.assertTrue(
            any(
                "required Engineering Standards Impact Assessment was supplied"
                in item
                for item in results.failures
            )
        )

    def test_placeholder_exit_commit_is_rejected(self):
        phase = accepted_exit()
        phase["source_commit"] = "0" * 40
        results = validate_records(
            phase,
            authority=validated_authority(),
            require_type="EXIT",
            require_pass=True,
        )
        self.assertTrue(
            any("source commit is non-placeholder" in item for item in results.failures)
        )


    def test_identity_separation_collapse_is_rejected(self):
        authority = validated_authority()
        authority["identity_model"][
            "administrative_identity_separate_from_ordinary_identity"
        ] = False
        results = validate_records(
            accepted_exit(), authority=authority, require_type="EXIT", require_pass=True
        )
        self.assertTrue(results.failures)

    def test_role_accumulation_to_unrestricted_authority_is_rejected(self):
        authority = validated_authority()
        authority["role_accumulation"]["unrestricted_authority_prevented"] = False
        results = validate_records(
            accepted_exit(), authority=authority, require_type="EXIT", require_pass=True
        )
        self.assertTrue(results.failures)

    def test_out_of_sync_phase_exit_is_rejected(self):
        phase = accepted_exit()
        phase["synchronization"]["architecture"] = "OUT_OF_SYNC"
        results = validate_records(
            phase,
            authority=validated_authority(),
            require_type="EXIT",
            require_pass=True,
        )
        self.assertTrue(
            any("phase exit artifacts are synchronized" in item for item in results.failures)
        )

    def test_open_deviation_blocks_phase_exit(self):
        phase = accepted_exit()
        phase["deviations"] = [
            {
                "record_id": "DEV-001",
                "status": "OPEN",
                "description": "An unresolved mandatory control deviation remains open.",
            }
        ]
        results = validate_records(
            phase,
            authority=validated_authority(),
            require_type="EXIT",
            require_pass=True,
        )
        self.assertTrue(
            any("no open or expired deviation" in item for item in results.failures)
        )


if __name__ == "__main__":
    unittest.main()
