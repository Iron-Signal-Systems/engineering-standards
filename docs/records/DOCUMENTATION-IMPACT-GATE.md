# Documentation-impact gate implementation record

**Status:** A6 COMPLETE WORKING-TREE CANDIDATE — NOT RELEASED OR ADOPTABLE

## Implemented boundary

This step adds:

- a versioned language-neutral documentation-impact policy;
- a strict fail-closed policy parser;
- deterministic changed-path evaluation;
- typed rule and requirement evidence;
- a JSON Schema and governed example;
- hostile parser and evaluator tests;
- the governing documentation-impact standard.

## Enforcement model

Rules are triggered by exact paths or bounded prefixes with optional suffixes.
Every triggered rule evaluates each requirement independently. The report records
the complete sorted changed-path set, trigger paths, matched requirement paths,
per-requirement status, per-rule status, and overall pass/fail status.

Documentation-only changes do not trigger an implementation rule. Overlapping
rules are all evaluated; satisfying one rule does not waive another.

## Security properties

The parser and evaluator reject path traversal, absolute paths, backslashes,
control characters, `.git/`, `.local/`, symlinked policy paths, duplicate
identifiers, ambiguous pattern forms, unknown fields, trailing JSON, and
oversized policy files.

## Completed A6 enforcement

Exact Git comparison, validator CLI execution, durable evidence, and repository
self-validation workflow enforcement are implemented by Step 15B.

## Git, CLI, and hosted enforcement

Step 15B completes A6 by adding:

- exact commit-ID validation;
- bounded Git discovery and merge-base comparison;
- sorted changed-path collection with rename detection disabled;
- deterministic JSON and text evidence;
- failure evidence for invalid policy, comparison, or unsatisfied requirements;
- the `documentation-impact --base COMMIT --head COMMIT` validator command;
- self-validation workflow enforcement on Ubuntu, Arch Linux, and Fedora;
- always-retained documentation-impact artifacts;
- a temporary complete-candidate commit campaign proving the accumulated
  A1-A6 change set satisfies the gate.

The authoritative evidence directory is
`.local/validation/documentation-impact/`. No evidence file is written into the
tracked source tree.

## Step 15B-R1 correction

The complete-candidate campaign exposed two invalid trigger prefixes in the
governed policy:

- `internal/release`
- `cmd/isras-release`

The strict parser correctly rejected them because directory prefixes must end in
`/`. The earlier Step 15A unit campaign used a synthetic valid policy and did not
load the actual governed policy file.

R1 replaces the fragments with exact prefixes for every current release
implementation directory, synchronizes the governed example, and adds a
repository-level test that loads the real policy and verifies complete release
directory coverage. No A6 runtime, CLI, evidence, or workflow behavior is
weakened.
