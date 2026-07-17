# Clean-Clone Release Validation

## Purpose

A release candidate is not accepted merely because validation passes in the
developer's existing working directory. The exact pushed commit shall also pass
the repository's committed validation from a newly created clone of the
canonical origin.

This is a release-level control. It does not claim independent review,
certification, or third-party assurance.

## Required boundary

Clean-clone validation shall:

1. require a completely clean source repository;
2. require a named local branch;
3. verify the current commit's cryptographic signature;
4. prove that the selected remote branch points to the exact local commit;
5. create a new clone from the configured canonical `origin`;
6. check out the exact commit in detached state;
7. build the validator from the committed clone source;
8. install the exact declared `govulncheck` version;
9. run the complete validator in `release` mode;
10. fail if the clone acquires tracked or unignored source changes;
11. retain a local log, summary, and clone for review.

The campaign shall not create a Git tag, release, commit, or push.

## Command

Build the repository-owned command:

```bash
./tools/build-release-validator.sh
```

After the exact signed commit has been pushed to its remote branch:

```bash
./.local/bin/isras-release-validate
```

The current branch is used by default. A different existing remote branch may
be selected explicitly:

```bash
./.local/bin/isras-release-validate --ref dev
```

## Local evidence

Each run creates a private directory under:

```text
.local/validation/releases/
```

The directory contains:

- `release-validation.log`;
- `release-summary.txt`;
- the retained clean clone under `repository/`.

This evidence is local and ignored by Git. The committed validator source and
its tests remain the durable reviewable test implementation for the exact
commit.

## Failure behavior

A failure shall identify the failed stage, retain available evidence, and print
safe commands for reviewing the log and rerunning the campaign. A failed
campaign shall not delete the retained clone automatically.

## Network behavior

The campaign requires network access to:

- read and clone the canonical Git origin;
- install the exact declared `govulncheck` version;
- query the Go vulnerability database during release-mode validation.

An unavailable network or vulnerability database is a failure. The validator
shall not report a successful vulnerability check when current vulnerability
data could not be obtained.
