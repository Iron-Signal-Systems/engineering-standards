# Go module inventory container ownership correction

**Status:** HOSTED-CI CORRECTION CANDIDATE — NOT MERGED, RELEASED, OR ADOPTABLE

## Hosted failure

Signed commit `7bab89e2ef346c7ed471ab297f5c3e37f23e4d9c` passed the
standard ISRAS validation workflow and the native Ubuntu 22.04 job. GitHub Actions
Linux platform run `29867575084` failed in all three bind-mounted container jobs:

- Arch Linux;
- Fedora 43;
- Fedora 44.

Each failure occurred in
`TestRepositoryModuleInventoryExcludesLocalRuntimeEvidence` while the module
inventory attempted `git ls-files`. Documentation-impact validation passed before
the failure, and the retained package-test log identified the same module
inventory error in the container jobs.

## Root cause

The workflow correctly established the mounted workspace as a Git safe directory
using global Git configuration. The bounded module-inventory command then changed
`HOME` to the repository root. That intentionally isolated caller configuration,
but it also made the workflow's global `safe.directory` declaration unavailable.

Native Ubuntu used matching workspace ownership and therefore passed. The
container jobs observed the host-mounted workspace under a different ownership
identity, so Git rejected the repository before `ls-files` could enumerate
governed module paths.

## Correction

The module inventory now:

- disables global Git configuration with `GIT_CONFIG_GLOBAL`;
- disables system Git configuration with `GIT_CONFIG_NOSYSTEM`;
- supplies one command-scoped `safe.directory` value;
- binds that value to the exact cleaned repository root;
- retains the bounded executable, PATH, locale, NUL output, and repository-owned
  path filtering controls.

The correction does not use `safe.directory=*`, does not trust a parent directory,
does not inherit caller Git configuration, and does not weaken `.local`
exclusion.

## Regression evidence

A repository-level test executes the exact production Git command with Git's
different-owner test condition enabled. It requires successful NUL-delimited
`go.mod` enumeration, the exact repository-root `safe.directory` argument, and
global/system configuration isolation.

## Claim boundary

This correction requires a new signed commit, a non-forced push, and successful
hosted validation on the exact new commit before IFI consumer validation may
begin. It does not change the previously published ISRAS `0.1.4` release.
