# Workstream A A1-A6 Complete Local Acceptance Candidate

**Status:** COMPLETE LOCAL ACCEPTANCE CAMPAIGN CANDIDATE — NOT COMMITTED, PUSHED, MERGED, TAGGED, RELEASED, OR ADOPTABLE

## Authority boundary

This record defines the complete local acceptance campaign for the accumulated
ISRAS `0.1.5` Workstream A candidate. It does not itself establish an accepted
release, move a protected branch, update a consuming project, or authorize Iron
File Intelligence or Iron Atlas adoption.

The authoritative release remains the exact signed tag and published release
accepted through the governed release process.

## Candidate identity

The campaign constructs one disposable commit from:

- PR #35 base commit `c9345d6d731600df7bd4ba4a133c07265db55e5a`;
- the pushed correction head `e9199367b6669e89a09356b1fb2a89a6f5112346`;
- every accepted uncommitted A1-A6 working-tree change;
- this acceptance-campaign record.

The disposable commit hash is generated at runtime and retained in the campaign
evidence archive. The actual working branch remains unstaged and uncommitted.

## Accepted scope under test

The candidate includes:

1. selected Go toolchain preservation and minimum-version enforcement;
2. bounded Go child-process environments and evidence schema v2;
3. deterministic governed Go module inventory;
4. exact pinned govulncheck identity and streaming protocol validation;
5. every-module scanner execution and typed evidence;
6. exact governed vulnerability exceptions and fail-closed reconciliation;
7. repository identity, public-use boundaries, and language-neutral additive
   profiles;
8. documentation-impact policy, exact Git comparison, CLI evidence, and hosted
   Ubuntu, Arch Linux, and Fedora enforcement.

## Complete campaign

The committed disposable candidate must pass:

- patch whitespace and committed-tree cleanliness;
- all Go tests;
- all Go race tests;
- Go vet and complete build;
- validator `system`, `repo`, `go`, and `secrets` commands;
- every tracked JSON document parse;
- every tracked shell script syntax check;
- documentation, schema/example, workflow, and repository-identity contracts;
- exact governed documentation-impact evaluation from base to candidate head;
- live public `Execute` govulncheck validation using the already-acquired exact
  pinned scanner;
- scanner identity and governed configuration verification;
- a negative committed implementation-only change that must fail the
  documentation-impact gate while retaining JSON and text failure evidence.

## Acceptance conditions

Local Workstream A acceptance is valid only when every positive gate passes, the
negative gate fails for the intended unsatisfied documentation requirements, the
candidate source tree remains clean, and complete evidence is archived with a
SHA-256 digest.

## Remaining release work

After this local campaign passes:

1. review the final diff and create the required signed repository commit;
2. push the branch and pass hosted validation on the exact new commit;
3. run Iron File Intelligence as an external consuming-project regression;
4. complete PR review and merge to `dev`;
5. revalidate the merged commit;
6. create and publish the signed immutable `isras-v0.1.5` release;
7. explicitly update IFI to the exact accepted release.

No step may be skipped merely because the disposable local candidate passes.
