#!/usr/bin/env python3
"""Validate the ISRAS control-level external standards crosswalk."""
from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

from jsonschema import Draft202012Validator, FormatChecker

CONTROL_RE = re.compile(r"ISRAS-[A-Z]{2,4}-[0-9]{3}")


def catalog_controls(root: Path) -> set[str]:
    result: set[str] = set()
    for path in sorted((root / "standards/repository-assurance").glob("v*/CONTROL-CATALOG.md")):
        result.update(CONTROL_RE.findall(path.read_text(encoding="utf-8")))
    return result


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=".")
    parser.add_argument(
        "--record",
        default="docs/engineering/external-standards-crosswalk.json",
    )
    parser.add_argument(
        "--require-all-pinned",
        action="store_true",
        help="Formal phase-entry/acceptance mode.",
    )
    args = parser.parse_args()

    root = Path(args.repo_root).resolve()
    record_path = (root / args.record).resolve()
    try:
        record_path.relative_to(root)
    except ValueError as exc:
        raise ValueError("crosswalk record must remain inside the repository") from exc
    schema = json.loads(
        (root / "schemas/external-standards-crosswalk-v1.schema.json").read_text(
            encoding="utf-8"
        )
    )
    record = json.loads(record_path.read_text(encoding="utf-8"))
    errors = [
        f"schema: {'/'.join(map(str, error.absolute_path)) or '<root>'}: {error.message}"
        for error in Draft202012Validator(
            schema, format_checker=FormatChecker()
        ).iter_errors(record)
    ]

    baselines = {item["baseline_id"]: item for item in record.get("baselines", [])}
    if len(baselines) != len(record.get("baselines", [])):
        errors.append("baseline identifiers are not unique")
    mappings = {item["control_id"]: item for item in record.get("mappings", [])}
    if len(mappings) != len(record.get("mappings", [])):
        errors.append("control mappings are not unique")

    controls = catalog_controls(root)
    missing = sorted(controls - set(mappings))
    extra = sorted(set(mappings) - controls)
    if missing:
        errors.append(f"crosswalk omits ISRAS controls: {missing}")
    if extra:
        errors.append(f"crosswalk references unknown ISRAS controls: {extra}")

    referenced_baselines: set[str] = set()
    for control_id, mapping in mappings.items():
        for reference in mapping.get("external_references", []):
            baseline_id = reference.get("baseline_id")
            referenced_baselines.add(str(baseline_id))
            if baseline_id not in baselines:
                errors.append(
                    f"{control_id}: unknown external baseline {baseline_id!r}"
                )
        if mapping.get("state") == "COVERED":
            errors.append(
                f"{control_id}: broad COVERED state is prohibited until requirement-level "
                "human review records full correspondence"
            )

    unused = sorted(set(baselines) - referenced_baselines)
    if unused:
        errors.append(f"crosswalk contains unused baselines: {unused}")

    unpinned = sorted(
        baseline_id
        for baseline_id, item in baselines.items()
        if item.get("pin_status") != "PINNED"
    )
    if args.require_all_pinned and unpinned:
        errors.append(f"formal crosswalk validation has unpinned baselines: {unpinned}")

    if errors:
        for error in sorted(set(errors)):
            print(f"FAIL: {error}")
        print(f"External standards crosswalk validation FAILED with {len(set(errors))} error(s).")
        return 1

    print(
        "External standards crosswalk validation PASSED: "
        f"{len(mappings)} controls, {len(baselines)} baselines."
    )
    if unpinned:
        print(
            "INFO: candidate crosswalk still requires immutable pins before formal phase entry: "
            + ", ".join(unpinned)
        )
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
