# ISRAS Adopter Quick Start

## 1. Obtain the selected release

Clone the canonical standards repository and check out the selected exact
release tag:

```bash
git clone       git@github.com:Iron-Signal-Systems/engineering-standards.git       engineering-standards

cd engineering-standards

release="$(tr -d '\r\n' < VERSION)"
tag="isras-v${release}"

git fetch origin       "refs/tags/${tag}:refs/tags/${tag}"

git tag -v "$tag"

accepted_commit="$(git rev-parse "${tag}^{commit}")"

git fetch origin main
test "$(git rev-parse refs/remotes/origin/main)" = "$accepted_commit"

printf 'ISRAS version: %s\n' "$release"
printf 'ISRAS tag: %s\n' "$tag"
printf 'ISRAS commit: %s\n' "$accepted_commit"
```

Do not adopt from a floating `dev` branch or from an unverified tag.

## 2. Record the exact standard identity

The adopting repository must record:

- ISRAS version;
- signed tag;
- exact 40-character commit;
- source-manifest digest;
- adoption date;
- applicable compatibility or exception records.

## 3. Preview adoption

```bash
export TARGET_REPOSITORY=/absolute/path/to/target-repository
export TARGET_SLUG=Iron-Signal-Systems/target-repository
export TARGET_ORIGIN=git@github.com:Iron-Signal-Systems/target-repository.git

python3 tools/isras/adopt.py       --target "$TARGET_REPOSITORY"       --repository "$TARGET_SLUG"       --canonical-origin "$TARGET_ORIGIN"       --development-branch dev       --release-branch main       --profile general       --dry-run
```

Review every proposed file before applying the baseline.

## 4. Validate the adopting repository

After applying and customizing the baseline:

```bash
chmod +x tools/environment/bootstrap_tools.sh
chmod +x tools/validation/validate_portable.sh
chmod +x tools/validation/validate_fresh_clone.sh

./tools/environment/bootstrap_tools.sh
export ISRAS_PYTHON="$PWD/.isras-tools-venv/bin/python"

./tools/validation/validate_portable.sh
./tools/validation/validate_fresh_clone.sh
```

Product-specific canonical, specialized, security, recovery, performance, and
operational campaigns remain the responsibility of the adopting repository.
