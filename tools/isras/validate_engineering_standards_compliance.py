#!/usr/bin/env python3
"""Validate ISRAS v2 phase, impact, and authority-boundary records."""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

try:
    from jsonschema import Draft202012Validator
except ImportError as exc:  # pragma: no cover
    raise SystemExit("jsonschema is required; install tools/requirements.txt") from exc

MATURITY = {"DOCUMENTED": 1, "IMPLEMENTED": 2, "VALIDATED": 3, "ACCEPTED": 4}
ZERO40 = "0" * 40
ZERO64 = "0" * 64


class Results:
    def __init__(self) -> None:
        self.passes: list[str] = []
        self.failures: list[str] = []

    def check(self, condition: bool, message: str) -> None:
        (self.passes if condition else self.failures).append(message)

    def report(self) -> int:
        for item in self.passes:
            print(f"PASS: {item}")
        for item in self.failures:
            print(f"FAIL: {item}")
        print(f"PASS checks: {len(self.passes)}")
        print(f"FAIL checks: {len(self.failures)}")
        if self.failures:
            print("Engineering standards compliance validation FAILED.")
            return 1
        print("Engineering standards compliance validation PASSED.")
        return 0


def load_json(path: Path) -> Any:
    with path.open(encoding="utf-8") as handle:
        return json.load(handle)


def schema_validate(instance: Any, schema: Any, label: str, results: Results) -> None:
    errors = sorted(Draft202012Validator(schema).iter_errors(instance), key=lambda e: list(e.path))
    if not errors:
        results.check(True, f"{label} conforms to its JSON schema")
        return
    for error in errors:
        location = "/".join(str(x) for x in error.path) or "<root>"
        results.check(False, f"{label} schema error at {location}: {error.message}")


def unique_control_ids(controls: list[dict[str, Any]], label: str, results: Results) -> None:
    ids = [item["control_id"] for item in controls]
    results.check(len(ids) == len(set(ids)), f"{label} control identifiers are unique")


def validate_impact(record: dict[str, Any], results: Results, require_complete: bool) -> None:
    unique_control_ids(record["controls"], "impact assessment", results)
    current = record["current_baseline"]
    candidate = record["candidate_release"]
    results.check(current["commit"] != ZERO40, "current ISRAS baseline commit is non-placeholder")
    results.check(current["source_manifest_sha256"] != ZERO64, "current ISRAS manifest digest is non-placeholder")
    if record["trigger"] != "CANDIDATE_PLANNING":
        results.check(candidate["accepted"] is True, "assessed release is accepted")
    for control in record["controls"]:
        if control["classification"] == "NOT_APPLICABLE":
            results.check(len(control["justification"].strip()) >= 20, f"{control['control_id']} non-applicability is justified")
        if control["classification"] == "REQUIRES_FUTURE_WORK":
            results.check(control["exception_or_deferment"] is not None, f"{control['control_id']} future work has a governance record")
    if require_complete:
        results.check(record["status"] in {"COMPLETE", "ACCEPTED"}, "impact assessment is complete")
        results.check(record["approval"]["decision"] == "APPROVED", "impact assessment is approved")
        results.check(record["completed_at"] is not None, "impact assessment completion time is recorded")


def validate_authority(record: dict[str, Any], results: Results, require_validated: bool) -> None:
    ident = record["identity_model"]
    auth = record["authorization"]
    propagation = record["privilege_propagation"]
    database = record["database_authority"]
    accumulation = record["role_accumulation"]
    break_glass = record["elevation_and_break_glass"]
    revocation = record["revocation"]
    audit = record["audit"]

    results.check(ident["service_identity_separate_from_user_identity"], f"{record['boundary_id']} separates service and user identities")
    results.check(ident["administrative_identity_separate_from_ordinary_identity"], f"{record['boundary_id']} separates administrative and ordinary identities")
    results.check(not ident["credentials_reused_across_unrelated_boundaries"], f"{record['boundary_id']} prohibits unrelated credential reuse")
    results.check(auth["deny_by_default"], f"{record['boundary_id']} is deny-by-default")
    results.check(auth["independent_at_boundary"], f"{record['boundary_id']} independently authorizes at the boundary")
    results.check(auth["delegation_scoped_to_operation"], f"{record['boundary_id']} scopes delegation to the operation")
    results.check(propagation["automatic_propagation_prohibited"], f"{record['boundary_id']} prohibits automatic privilege propagation")
    results.check(propagation["unrestricted_execution_context_prohibited"], f"{record['boundary_id']} prohibits unrestricted execution contexts")
    results.check(propagation["retry_replay_recovery_cannot_increase_authority"], f"{record['boundary_id']} prevents authority increase during retry, replay, or recovery")
    results.check(not database["runtime_uses_owner_or_superuser"], f"{record['boundary_id']} runtime does not use database owner or superuser")
    if database["applicable"]:
        results.check(database["migration_authority_separate_from_runtime"], f"{record['boundary_id']} separates migration and runtime authority")
        results.check(database["operation_appropriate_roles"], f"{record['boundary_id']} uses operation-appropriate database roles")
    results.check(accumulation["evaluated"] and accumulation["unrestricted_authority_prevented"], f"{record['boundary_id']} prevents unrestricted accumulated authority")
    if break_glass["applicable"]:
        results.check(all(break_glass[k] for k in ["explicit","time_bounded","attributable","audited","reviewed_after_use"]), f"{record['boundary_id']} bounds and audits break-glass access")
    results.check(revocation["supported"] and revocation["session_invalidation_supported"], f"{record['boundary_id']} supports revocation and session invalidation")
    results.check(revocation["queued_or_retried_work_reauthorized"], f"{record['boundary_id']} reauthorizes queued or retried work")
    results.check(all(audit.values()), f"{record['boundary_id']} records minimum audit fields")

    applicable = [t for t in record["hostile_tests"] if t["applicability"] == "APPLICABLE"]
    if require_validated or MATURITY[record["maturity"]] >= MATURITY["VALIDATED"]:
        results.check(bool(applicable), f"{record['boundary_id']} has applicable hostile tests")
        results.check(all(t["status"] == "PASS" and t["evidence_path"] for t in applicable), f"{record['boundary_id']} applicable hostile tests passed with evidence")
        results.check(bool(record["evidence_paths"]), f"{record['boundary_id']} retains boundary evidence")


def validate_phase(record: dict[str, Any], results: Results, require_type: str | None, require_pass: bool) -> None:
    unique_control_ids(record["controls"], "phase review", results)
    if require_type:
        results.check(record["review_type"] == require_type, f"review type is {require_type}")
    baseline = record["adopted_isras"]
    results.check(baseline["commit"] != ZERO40, "adopted ISRAS commit is non-placeholder")
    results.check(baseline["source_manifest_sha256"] != ZERO64, "adopted ISRAS manifest digest is non-placeholder")

    newer = record["newer_release_review"]
    results.check(not newer["impact_assessment_required"] or bool(newer["impact_assessment_path"]), "required impact assessment path is recorded")

    for control in record["controls"]:
        cid = control["control_id"]
        if control["applicability"] == "NOT_APPLICABLE":
            results.check(len(control["applicability_justification"].strip()) >= 20, f"{cid} non-applicability is justified")
            continue
        if record["review_type"] == "EXIT":
            results.check(MATURITY[control["actual_maturity"]] >= MATURITY[control["required_maturity"]], f"{cid} meets required exit maturity")
            results.check(control["compliance_result"] == "PASS", f"{cid} compliance result passes")
            if MATURITY[control["actual_maturity"]] >= MATURITY["VALIDATED"]:
                results.check(bool(control["evidence_paths"]), f"{cid} validated or accepted maturity has evidence")
        else:
            if MATURITY[control["actual_maturity"]] < MATURITY[control["required_maturity"]]:
                results.check(bool(control["planned_actions"]), f"{cid} entry maturity gap has planned work")

    if require_pass:
        results.check(record["review_status"] == "PASS", "phase review status is PASS")
        results.check(record["approval"]["decision"] == "APPROVED", "phase review is approved")
        expected = "PHASE_MAY_BEGIN" if record["review_type"] == "ENTRY" else "PHASE_ACCEPTED"
        results.check(record["decision"] == expected, f"phase review decision is {expected}")

    if record["review_type"] == "EXIT" and (require_pass or record["review_status"] == "PASS"):
        results.check(record["source_commit"] != ZERO40, "phase exit source commit is non-placeholder")
        results.check(record["exact_pushed_candidate_evaluated"], "exact pushed candidate was evaluated")
        allowed_sync = {"SYNCHRONIZED", "NOT_REQUIRED"}
        results.check(all(value in allowed_sync for value in record["synchronization"].values()), "phase exit artifacts are synchronized")
        results.check(record["historical_predecessor_status"] in {"PASS", "NOT_REQUIRED"}, "historical predecessor handling passes")
        results.check(not any(item["status"] in {"OPEN", "EXPIRED"} for item in record["deviations"]), "phase exit has no open or expired deviation")
        hostile = record["hostile_validation"]
        results.check((not hostile["required"] and hostile["status"] == "NOT_REQUIRED") or (hostile["required"] and hostile["status"] == "PASS"), "required hostile validation passes")
        if hostile["required"]:
            results.check(bool(hostile["authority_boundary_records"]), "hostile validation links authority boundary records")
            results.check(bool(hostile["evidence_paths"]), "hostile validation links evidence")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", type=Path, default=Path.cwd())
    parser.add_argument("--phase-review", type=Path, required=True)
    parser.add_argument("--impact-assessment", type=Path)
    parser.add_argument("--authority-boundary", type=Path, action="append", default=[])
    parser.add_argument("--require-review-type", choices=["ENTRY","EXIT"])
    parser.add_argument("--require-pass", action="store_true")
    parser.add_argument("--require-complete-impact", action="store_true")
    return parser.parse_args()


def resolve(root: Path, value: Path) -> Path:
    return value if value.is_absolute() else root / value


def main() -> int:
    args = parse_args()
    root = args.repo_root.resolve()
    results = Results()

    phase_path = resolve(root, args.phase_review)
    phase = load_json(phase_path)
    phase_schema = load_json(root / "schemas/phase-standards-compliance-v1.schema.json")
    schema_validate(phase, phase_schema, str(phase_path), results)
    if not results.failures:
        validate_phase(phase, results, args.require_review_type, args.require_pass)

    impact = None
    if args.impact_assessment:
        impact_path = resolve(root, args.impact_assessment)
        impact = load_json(impact_path)
        impact_schema = load_json(root / "schemas/engineering-standards-impact-assessment-v1.schema.json")
        before = len(results.failures)
        schema_validate(impact, impact_schema, str(impact_path), results)
        if len(results.failures) == before:
            validate_impact(impact, results, args.require_complete_impact)

    if phase.get("newer_release_review", {}).get("impact_assessment_required"):
        results.check(impact is not None, "required Engineering Standards Impact Assessment was supplied")

    authority_schema = load_json(root / "schemas/authority-boundary-record-v1.schema.json")
    supplied_authority_paths: set[str] = set()
    for item in args.authority_boundary:
        boundary_path = resolve(root, item)
        supplied_authority_paths.add(str(item))
        boundary = load_json(boundary_path)
        before = len(results.failures)
        schema_validate(boundary, authority_schema, str(boundary_path), results)
        if len(results.failures) == before:
            validate_authority(boundary, results, phase.get("review_type") == "EXIT" and phase.get("hostile_validation", {}).get("required", False))

    required_records = set(phase.get("hostile_validation", {}).get("authority_boundary_records", []))
    if required_records:
        results.check(required_records.issubset(supplied_authority_paths), "all linked authority boundary records were supplied")

    return results.report()


if __name__ == "__main__":
    sys.exit(main())
