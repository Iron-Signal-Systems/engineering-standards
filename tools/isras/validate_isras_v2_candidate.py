#!/usr/bin/env python3
"""Validate completeness and release boundaries of the ISRAS v2 candidate."""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import subprocess
import sys
from pathlib import Path

from jsonschema import Draft202012Validator

ACCEPTED_V1_COMMIT = "c379417720faa595fa5cb89a1dfdb2259d6cb95e"
REQUIRED_NEW_CONTROLS = {
    "ISRAS-GOV-004","ISRAS-GOV-005","ISRAS-GOV-006","ISRAS-GOV-007",
    "ISRAS-PHS-001","ISRAS-PHS-002","ISRAS-PHS-003",
    "ISRAS-AUT-001","ISRAS-AUT-002","ISRAS-AUT-003","ISRAS-AUT-004",
    "ISRAS-AUT-005","ISRAS-AUT-006","ISRAS-AUT-007","ISRAS-TST-003",
    "ISRAS-EVD-003","ISRAS-EVD-004",
}
REQUIRED_FILES = {
    "standards/repository-assurance/v2/INDEX.md",
    "standards/repository-assurance/v2/STANDARD.md",
    "standards/repository-assurance/v2/CONTROL-CATALOG.md",
    "standards/repository-assurance/v2/MANDATORY-GOVERNANCE-AND-INHERITANCE.md",
    "standards/repository-assurance/v2/BOUNDED-AUTHORITY-AND-PRIVILEGE-NON-PROPAGATION.md",
    "standards/repository-assurance/v2/ENGINEERING-STANDARDS-IMPACT-ASSESSMENT.md",
    "standards/repository-assurance/v2/PHASE-ENTRY-AND-EXIT-COMPLIANCE.md",
    "standards/repository-assurance/v2/HOSTILE-AUTHORITY-VALIDATION.md",
    "standards/repository-assurance/v2/EVIDENCE-MODEL.md",
    "standards/repository-assurance/v2/VALIDATION-MODEL.md",
    "standards/repository-assurance/v2/RELEASE-VERSIONING-SUPPORT-AND-DEPRECATION.md",
    "standards/repository-assurance/v2/MIGRATION-GUIDE.md",
    "schemas/engineering-standards-impact-assessment-v1.schema.json",
    "schemas/phase-standards-compliance-v1.schema.json",
    "schemas/authority-boundary-record-v1.schema.json",
    "templates/engineering-standards/phase-entry-review.json",
    "templates/engineering-standards/phase-exit-review.json",
    "templates/engineering-standards/impact-assessment.json",
    "templates/engineering-standards/authority-boundary-record.json",
    "tools/isras/validate_engineering_standards_compliance.py",
    "tools/isras/validate_isras_v2_candidate.py",
    "tests/test_engineering_standards_compliance.py",
    "docs/acceptance/isras-v2.0.0-plan.md",
}

class Results:
    def __init__(self): self.ok=[]; self.fail=[]
    def check(self, condition, message): (self.ok if condition else self.fail).append(message)
    def report(self):
        for x in self.ok: print(f"PASS: {x}")
        for x in self.fail: print(f"FAIL: {x}")
        print(f"PASS checks: {len(self.ok)}")
        print(f"FAIL checks: {len(self.fail)}")
        if self.fail:
            print("ISRAS v2 candidate validation FAILED.")
            return 1
        print("ISRAS v2 candidate validation PASSED.")
        return 0

def load(path):
    with path.open(encoding="utf-8") as f: return json.load(f)

def git(root, *args):
    return subprocess.run(["git", *args], cwd=root, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False)

def main():
    p=argparse.ArgumentParser(); p.add_argument("--repo-root",type=Path,default=Path.cwd()); p.add_argument("--skip-v1-git-diff",action="store_true"); a=p.parse_args()
    root=a.repo_root.resolve(); r=Results()
    for rel in sorted(REQUIRED_FILES): r.check((root/rel).is_file(), f"required candidate file exists: {rel}")
    version=(root/"VERSION").read_text(encoding="utf-8").strip() if (root/"VERSION").exists() else ""
    r.check(version == "1.0.1", "VERSION remains 1.0.1 during v2 candidate development")
    standard=(root/"standards/repository-assurance/v2/STANDARD.md").read_text(encoding="utf-8")
    authority=(root/"standards/repository-assurance/v2/BOUNDED-AUTHORITY-AND-PRIVILEGE-NON-PROPAGATION.md").read_text(encoding="utf-8")
    r.check("unrestricted execution context" in standard.lower(), "normative unrestricted execution context term is present")
    r.check("God Access / God Mode" in standard, "explanatory God Access / God Mode phrase accompanies the normative term")
    r.check(bool(re.search(r"shall\s+create an \*\*unrestricted execution context\*\*", authority, re.IGNORECASE)), "bounded-authority invariant is explicit")
    catalog=(root/"standards/repository-assurance/v2/CONTROL-CATALOG.md").read_text(encoding="utf-8")
    ids=re.findall(r"ISRAS-[A-Z]{3}-[0-9]{3}", catalog)
    r.check(REQUIRED_NEW_CONTROLS.issubset(set(ids)), "all required v2 controls are cataloged")
    r.check(len(ids)==len(set(ids)), "control catalog identifiers are unique")
    schemas={
      "templates/engineering-standards/impact-assessment.json":"schemas/engineering-standards-impact-assessment-v1.schema.json",
      "templates/engineering-standards/phase-entry-review.json":"schemas/phase-standards-compliance-v1.schema.json",
      "templates/engineering-standards/phase-exit-review.json":"schemas/phase-standards-compliance-v1.schema.json",
      "templates/engineering-standards/authority-boundary-record.json":"schemas/authority-boundary-record-v1.schema.json",
    }
    for template,schema_path in schemas.items():
        schema=load(root/schema_path)
        try: Draft202012Validator.check_schema(schema); valid_schema=True
        except Exception: valid_schema=False
        r.check(valid_schema, f"schema is valid Draft 2020-12: {schema_path}")
        errors=list(Draft202012Validator(schema).iter_errors(load(root/template))) if valid_schema else [1]
        r.check(not errors, f"template conforms structurally: {template}")
    checkpoints=load(root/"tools/validation/checkpoints.json") if (root/"tools/validation/checkpoints.json").exists() else {}
    cp=checkpoints.get("checkpoints",{}).get("isras-v1.0.1",{})
    r.check(cp.get("commit")==ACCEPTED_V1_COMMIT and cp.get("status")=="accepted" and cp.get("tag")=="isras-v1.0.1", "accepted v1.0.1 checkpoint remains exact")
    v1=root/"standards/repository-assurance/v1"
    r.check(v1.is_dir() and (v1/"STANDARD.md").is_file() and (v1/"CONTROL-CATALOG.md").is_file(), "accepted v1 normative tree remains present")
    if not a.skip_v1_git_diff and (root/".git").exists():
        diff=git(root,"diff","--quiet",ACCEPTED_V1_COMMIT,"--","standards/repository-assurance/v1")
        r.check(diff.returncode==0, "accepted v1 normative tree is unchanged from v1.0.1")
    return r.report()

if __name__=="__main__": sys.exit(main())
