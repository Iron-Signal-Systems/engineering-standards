# Documentation Impact

## Purpose

Implementation, validation, schema, workflow, release, adoption, and governance
changes are incomplete when the applicable documentation and acceptance records
do not change with them. The documentation-impact gate verifies that required
documentation paths are present in the same reviewed change set.

## Language-neutral boundary

The gate classifies repository paths rather than programming languages. Go source
is covered by the first active policy, but the model supports additional
languages, platforms, schemas, workflows, and project-specific boundaries without
changing the evaluator.

## Versioned policy

The governing policy is `validation/documentation-impact-policy.json`. Its
versioned schema defines deterministic trigger patterns and requirement groups.

A trigger pattern selects changed paths by exact repository path or by bounded
prefix and optional suffix. A requirement is either:

- `all`: every declared pattern must match at least one changed path; or
- `any`: at least one declared pattern must match a changed path.

A requirement may not declare both forms.

## Initial governed rules

The first policy covers:

- implementation and validator source;
- the documentation-impact policy and evaluator themselves;
- JSON schemas and governed examples;
- hosted validation workflows;
- release, project-pin, provenance, publication, and adoption implementation.

Applicable changes require the unreleased changelog, a governing standard, and an
implementation or acceptance record. Schema changes additionally require a
governed example. Workflow changes require the testing or project-command
execution standard. Release and adoption changes require an applicable lifecycle
standard.

## Fail-closed behavior

The policy parser rejects unknown fields, unsupported versions, duplicate rule or
requirement identifiers, unsafe paths, ambiguous patterns, symlinks, nonregular
files, oversized input, and multiple JSON values.

The evaluator rejects unsafe changed paths, sorts all evidence deterministically,
evaluates every triggered rule, and fails when any requirement is unsatisfied.

## Enforced boundary

The policy model, exact Git comparison, validator command, durable evidence,
and repository self-validation workflow enforcement are implemented together.

## Git comparison and enforced evidence

The validator command is:

```text
isras-validate documentation-impact --base COMMIT --head COMMIT
```

Both values must be exact lowercase 40-character commit IDs. Symbolic ref names,
abbreviated IDs, and caller-controlled Git options are rejected.

The collector verifies both commits, computes their merge base, and evaluates the
changed paths from the merge base to the exact head. Rename detection is disabled
so that old and new paths remain independently visible to policy evaluation.

The command records deterministic JSON and text evidence under:

```text
.local/validation/documentation-impact/
```

Evidence contains the policy path and SHA-256, requested and resolved commits,
merge base, sorted changed paths, every triggered rule, every requirement,
matched paths, and the final status. Unsatisfied requirements and policy,
comparison, or filesystem failures produce retained failure evidence before the
command exits nonzero.

## Hosted enforcement

The repository validation workflow and every Linux platform-validation job invoke
the command after building the exact candidate validator.

Pull requests use the event's exact base and head commits. Pushes use the event's
before and head commits; a new branch uses the head parent. Scheduled and manual
runs compare the current head with its parent. Full history checkout is required.

The reusable consuming-project workflow does not infer a documentation policy for
an arbitrary target repository. It already retains target validation evidence;
consumer documentation-impact enforcement requires that project's explicit
adoption of a governed policy.

## Bounded release-directory policy

Release and publication implementation rules use exact directory prefixes.
Prefix fragments such as `internal/release` or `cmd/isras-release` are invalid
because they are not directory boundaries and could match unintended names.

The governed policy enumerates every current `internal/release*` and
`cmd/isras-release*` directory with a trailing slash. A repository-level test
loads the actual governed policy, discovers the current release directories, and
fails when any directory is not represented by an exact bounded prefix.
