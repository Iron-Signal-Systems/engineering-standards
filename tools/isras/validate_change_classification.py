#!/usr/bin/env python3
"""Validate proportional ISRAS change classification and campaign depth."""
from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

from jsonschema import Draft202012Validator

ORDER = ["C0", "C1", "C2", "C3", "C4", "C5", "C6"]
IMPACT_CLASS = {
    "tooling": "C1",
    "implementation": "C2",
    "security_or_authority": "C3",
    "schema_or_migration": "C4",
    "acceptance_semantics": "C5",
    "release_or_recovery": "C6",
}
CAMPAIGNS = {
    "C0": {"policy", "documentation-sync", "links", "whitespace"},
    "C1": {"portable", "unit-regression", "tool-environment", "fresh-clone"},
    "C2": {"integration", "traceability", "phase-review", "exact-pushed-source"},
    "C3": {
        "threat-abuse", "authority-record", "hostile-testing",
        "findings-separation", "revocation-retry-race",
    },
    "C4": {
        "migration-integrity", "compatibility", "rollback-restore",
        "representative-data", "destructive-safeguards", "historical-migration",
    },
    "C5": {
        "esia", "predecessor-revalidation", "approval-independence",
        "evidence-relationship",
    },
    "C6": {
        "trusted-build", "artifact-accounting", "sbom-provenance",
        "signature-verification", "remote-convergence", "deployment-recovery",
        "checkpoint-registration",
    },
}
CONTROL_RE = re.compile(r"ISRAS-[A-Z]{2,4}-[0-9]{3}")


def git(root: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args],
        cwd=root,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )


def controls(root: Path) -> set[str]:
    result: set[str] = set()
    for path in sorted((root / "standards/repository-assurance").glob("v*/CONTROL-CATALOG.md")):
        result.update(CONTROL_RE.findall(path.read_text(encoding="utf-8")))
    return result


def path_impacts(paths: list[str]) -> set[str]:
    impacts: set[str] = set()
    for path in paths:
        normalized = path.replace("\\", "/")
        lower = normalized.lower()
        if normalized in {"VERSION", "tools/validation/checkpoints.json"} or any(
            token in lower
            for token in (
                "release-finalization",
                "release-completion",
                "validate_release_state.py",
                "deployment-recovery",
            )
        ):
            impacts.add("release_or_recovery")
        if normalized.startswith("schemas/") or "/migrations/" in lower or lower.endswith(".sql"):
            impacts.add("schema_or_migration")
        if (
            normalized.startswith("standards/")
            or normalized.startswith("docs/acceptance/")
            or normalized.startswith("tools/validation/phase-gates/")
            or (
                normalized.startswith("tools/isras/validate_")
                and normalized.endswith(".py")
            )
        ):
            impacts.add("acceptance_semantics")
        if any(
            token in lower
            for token in (
                "security",
                "authority",
                "authentication",
                "authorization",
                "evidence",
                "ruleset",
                "bootstrap",
                "wheelhouse",
            )
        ):
            impacts.add("security_or_authority")
        if (
            normalized.startswith("tools/")
            or normalized.startswith("tests/")
            or normalized.startswith(".github/workflows/")
        ):
            impacts.add("tooling")
        if (
            Path(normalized).suffix in {".go", ".rs", ".cs", ".java", ".js", ".ts"}
            and not normalized.startswith(("tools/", "tests/"))
        ):
            impacts.add("implementation")
    return impacts


def required_campaigns(selected: str, impacts: dict[str, bool]) -> set[str]:
    """Return cumulative common campaigns plus only applicable risk branches."""
    required = set(CAMPAIGNS["C0"])
    selected_index = ORDER.index(selected) if selected in ORDER else 0
    if selected_index >= ORDER.index("C1"):
        required.update(CAMPAIGNS["C1"])
    if selected_index >= ORDER.index("C2"):
        required.update(CAMPAIGNS["C2"])
    if impacts.get("security_or_authority"):
        required.update(CAMPAIGNS["C3"])
    if impacts.get("schema_or_migration"):
        required.update(CAMPAIGNS["C4"])
    if selected_index >= ORDER.index("C5"):
        required.update(CAMPAIGNS["C5"])
    if selected_index >= ORDER.index("C6"):
        required.update(CAMPAIGNS["C6"])
    return required


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--record", required=True)
    parser.add_argument(
        "--skip-diff-floor",
        action="store_true",
        help="Template-test only; prohibited for actual candidate classification.",
    )
    args = parser.parse_args()
    root = Path(args.repo_root).resolve()
    record_path = (root / args.record).resolve()
    try:
        record_path.relative_to(root)
    except ValueError as exc:
        raise ValueError("classification record must remain inside the repository") from exc
    record = json.loads(record_path.read_text(encoding="utf-8"))
    schema = json.loads(
        (root / "schemas/change-classification-v1.schema.json").read_text(encoding="utf-8")
    )

    errors = [
        f"schema: {'/'.join(map(str, error.absolute_path)) or '<root>'}: {error.message}"
        for error in Draft202012Validator(schema).iter_errors(record)
    ]
    impacts = record.get("impacts", {})
    required_class = "C0"
    for impact, change_class in IMPACT_CLASS.items():
        if impacts.get(impact) and ORDER.index(change_class) > ORDER.index(required_class):
            required_class = change_class
    if impacts.get("editorial_only") and any(impacts.get(name) for name in IMPACT_CLASS):
        errors.append("editorial_only cannot be true with a material impact")
    if not impacts.get("editorial_only") and required_class == "C0":
        errors.append("a non-editorial record must identify at least one material impact")
    selected = record.get("selected_class", "C0")
    if selected in ORDER and ORDER.index(selected) < ORDER.index(required_class):
        errors.append(f"selected class {selected} is below required {required_class}")

    missing_campaigns = sorted(required_campaigns(selected, impacts) - set(record.get("campaigns", [])))
    if missing_campaigns:
        errors.append(f"{selected} is missing applicable campaigns: {missing_campaigns}")

    unknown_controls = sorted(set(record.get("applicable_controls", [])) - controls(root))
    if unknown_controls:
        errors.append(f"classification references unknown controls: {unknown_controls}")

    if not args.skip_diff_floor:
        base = record.get("base_commit", "")
        resolved = git(root, "rev-parse", "--verify", f"{base}^{{commit}}")
        if resolved.returncode != 0:
            errors.append("base_commit does not resolve to a Git commit")
        else:
            changed = git(root, "diff", "--name-only", base, "--")
            if changed.returncode != 0:
                errors.append("unable to determine changed paths from base_commit")
            else:
                paths = [line for line in changed.stdout.splitlines() if line]
                inferred = path_impacts(paths)
                missing_impacts = sorted(name for name in inferred if not impacts.get(name))
                if missing_impacts:
                    errors.append(
                        "changed paths require impact flags that are not asserted: "
                        f"{missing_impacts}"
                    )
                if not paths and not impacts.get("editorial_only"):
                    errors.append("non-editorial classification has no changed paths from base_commit")

    if errors:
        for error in sorted(set(errors)):
            print(f"FAIL: {error}")
        print(f"Change classification validation FAILED with {len(set(errors))} error(s).")
        return 1
    print(
        f"Change classification validation PASSED: selected={selected}; "
        f"minimum={required_class}; campaigns={len(record['campaigns'])}."
    )
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
