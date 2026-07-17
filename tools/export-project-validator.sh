#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'USAGE'
Usage:
  ./tools/export-project-validator.sh [--dry-run] /path/to/project

Builds and validates the repository-owned Go validator in a detached scratch
clone of the target's exact commit. After every check passes, the proven patch
is applied to the real target and staged for review.

The target must:
  - be a clean, non-bare Git working tree;
  - contain a go.mod file;
  - not already contain cmd/isras-validate or internal/isras;
  - not already contain validation/secret-allowlist.json.

Normal mode may update go.mod and go.sum through go mod tidy when the copied
validator changes the target import graph. Existing module requirements may be
promoted from indirect to direct, but they may not be removed or change version.

The export does not commit or push the target project.
USAGE
}

mode="apply"
case "${1:-}" in
  --dry-run)
    mode="dry-run"
    shift
    ;;
  --help|-h)
    usage
    exit 0
    ;;
esac

[[ $# -eq 1 ]] || { usage >&2; exit 2; }

go_timeout_seconds="${ISRAS_EXPORT_GO_TIMEOUT_SECONDS:-900}"
[[ "$go_timeout_seconds" =~ ^[1-9][0-9]*$ ]] || {
  echo "ERROR: ISRAS_EXPORT_GO_TIMEOUT_SECONDS must be a positive integer" >&2
  exit 2
}
command -v timeout >/dev/null 2>&1 || {
  echo "ERROR: required timeout command is unavailable" >&2
  exit 1
}

run_go() {
  timeout --foreground --signal=TERM --kill-after=30s \
    "${go_timeout_seconds}s" go "$@"
}

source_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[[ -n "$source_root" ]] || {
  echo "ERROR: run from the engineering-standards repository" >&2
  exit 1
}

if [[ -n "$(git -C "$source_root" status --porcelain=v1 --untracked-files=all)" ]]; then
  echo "ERROR: engineering-standards must be clean before validator export" >&2
  git -C "$source_root" status --short --branch >&2
  exit 1
fi

target_input="$(realpath -e "$1")"
git -C "$target_input" rev-parse --is-inside-work-tree >/dev/null 2>&1 || {
  echo "ERROR: target is not a Git working tree: $target_input" >&2
  exit 1
}

[[ "$(git -C "$target_input" rev-parse --is-bare-repository)" == "false" ]] || {
  echo "ERROR: target must not be a bare Git repository: $target_input" >&2
  exit 1
}

target="$(git -C "$target_input" rev-parse --show-toplevel)"
target="$(realpath -e "$target")"
[[ -f "$target/go.mod" ]] || {
  echo "ERROR: target has no go.mod: $target" >&2
  exit 1
}

if [[ -n "$(git -C "$target" status --porcelain=v1 --untracked-files=all)" ]]; then
  echo "ERROR: target repository must be clean before validator export" >&2
  git -C "$target" status --short --branch >&2
  exit 1
fi

readonly -a export_paths=(
  ".gitignore"
  "cmd/isras-validate"
  "internal/isras"
  "validation/secret-allowlist.json"
  "validation/tool-versions.json"
  "tools/isras/build-validator.sh"
)

for path in \
  cmd/isras-validate \
  internal/isras \
  validation/secret-allowlist.json \
  validation/tool-versions.json \
  tools/isras/build-validator.sh; do
  [[ ! -e "$target/$path" ]] || {
    echo "ERROR: target path already exists: $path" >&2
    exit 1
  }
done

target_head="$(git -C "$target" rev-parse --verify HEAD)"
target_module="$(cd "$target" && run_go list -m -f '{{.Path}}')"
[[ -n "$target_module" ]] || {
  echo "ERROR: unable to resolve target module path" >&2
  exit 1
}

work_root="$(mktemp -d "${TMPDIR:-/tmp}/isras-validator-export.XXXXXXXX")"
scratch="$work_root/target"
patch_file="$work_root/export.patch"
before_requirements="$work_root/requires.before"
after_requirements="$work_root/requires.after"
transaction_started=0
journal="$(git -C "$target" rev-parse --git-path isras-validator-export.transaction)"

rollback_target() {
  if (( transaction_started == 0 )); then
    return
  fi

  printf '\nExport failed; restoring target to %s.\n' "$target_head" >&2
  git -C "$target" reset --hard "$target_head" >/dev/null 2>&1 || true
  git -C "$target" clean -fd -- \
    .gitignore \
    cmd/isras-validate \
    internal/isras \
    validation/secret-allowlist.json \
    validation/tool-versions.json \
    tools/isras/build-validator.sh \
    go.sum >/dev/null 2>&1 || true
  rm -f -- "$journal"
}

cleanup() {
  status=$?
  if (( status != 0 )); then
    rollback_target
  fi
  rm -rf -- "$work_root"
  exit "$status"
}
trap cleanup EXIT INT TERM HUP

extract_requirements() {
  run_go mod edit -json | awk '
    /"Require"[[:space:]]*:/ { in_require=1; next }
    in_require && /^[[:space:]]*\]/ { exit }
    in_require && /"Path"[[:space:]]*:/ {
      line=$0
      sub(/^.*"Path"[[:space:]]*:[[:space:]]*"/, "", line)
      sub(/".*$/, "", line)
      path=line
    }
    in_require && /"Version"[[:space:]]*:/ {
      line=$0
      sub(/^.*"Version"[[:space:]]*:[[:space:]]*"/, "", line)
      sub(/".*$/, "", line)
      if (path != "") {
        print path "\t" line
        path=""
      }
    }
  ' | LC_ALL=C sort
}

verify_requirement_preservation() {
  local module version observed
  while IFS=$'\t' read -r module version; do
    [[ -n "$module" ]] || continue
    observed="$(awk -F '\t' -v module="$module" '$1 == module { print $2; exit }' "$after_requirements")"
    if [[ -z "$observed" ]]; then
      printf 'ERROR: go mod tidy removed existing requirement: %s %s\n' "$module" "$version" >&2
      return 1
    fi
    if [[ "$observed" != "$version" ]]; then
      printf 'ERROR: go mod tidy changed existing requirement: %s %s -> %s\n' \
        "$module" "$version" "$observed" >&2
      return 1
    fi
  done < "$before_requirements"
}

is_allowed_export_path() {
  case "$1" in
    .gitignore|go.mod|go.sum|\
    cmd/isras-validate|cmd/isras-validate/*|\
    internal/isras|internal/isras/*|\
    validation/secret-allowlist.json|\
    validation/tool-versions.json|\
    tools/isras/build-validator.sh)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

echo "Creating detached scratch clone of target commit $target_head..."
git clone --quiet --no-hardlinks --no-checkout "$target" "$scratch"
git -C "$scratch" checkout --quiet --detach "$target_head"

mkdir -p \
  "$scratch/cmd" \
  "$scratch/internal/isras" \
  "$scratch/validation" \
  "$scratch/tools/isras"

cp -a "$source_root/cmd/isras-validate" "$scratch/cmd/"
for package in dashboard executil failurelog model redact repository secrets validation; do
  cp -a "$source_root/internal/$package" "$scratch/internal/isras/$package"
done
cp "$source_root/validation/secret-allowlist.json" \
  "$scratch/validation/secret-allowlist.json"
cp "$source_root/validation/tool-versions.json" \
  "$scratch/validation/tool-versions.json"
cp "$source_root/tools/build-validator.sh" \
  "$scratch/tools/isras/build-validator.sh"
chmod 0755 "$scratch/tools/isras/build-validator.sh"

find "$scratch/cmd/isras-validate" "$scratch/internal/isras" \
  -type f -name '*.go' -print0 |
  while IFS= read -r -d '' file; do
    sed -i \
      "s#github.com/Iron-Signal-Systems/engineering-standards/internal/#$target_module/internal/isras/#g" \
      "$file"
  done

if [[ -f "$scratch/.gitignore" ]]; then
  if ! grep -qxF '.local/' "$scratch/.gitignore"; then
    printf '\n# Repository-local ISRAS validation output\n.local/\n' >> "$scratch/.gitignore"
  fi
else
  printf '# Repository-local ISRAS validation output\n.local/\n' > "$scratch/.gitignore"
fi

cd "$scratch"
extract_requirements > "$before_requirements"

echo "Formatting exported Go source..."
gofmt -w cmd/isras-validate internal/isras

echo "Applying deterministic module update in scratch clone..."
run_go mod tidy
extract_requirements > "$after_requirements"
verify_requirement_preservation

if [[ -f "$target/go.sum" && ! -f "$scratch/go.sum" ]]; then
  echo "ERROR: go mod tidy removed the target go.sum file" >&2
  exit 1
fi

if ! tidy_diff="$(run_go mod tidy -diff)"; then
  echo "ERROR: second go mod tidy -diff invocation failed" >&2
  exit 1
fi
if [[ -n "$tidy_diff" ]]; then
  echo "ERROR: module graph is not stable after go mod tidy" >&2
  printf '%s\n' "$tidy_diff" >&2
  exit 1
fi

echo "Running complete target tests in scratch clone..."
run_go test -count=1 ./...
run_go vet ./...
run_go build ./...
run_go mod verify

git add -- \
  .gitignore \
  go.mod \
  cmd/isras-validate \
  internal/isras \
  validation/secret-allowlist.json \
  validation/tool-versions.json \
  tools/isras/build-validator.sh
if [[ -e go.sum || -n "$(git status --porcelain=v1 -- go.sum)" ]]; then
  git add -A -- go.sum
fi

mapfile -t scratch_paths < <(git diff --cached --name-only)
((${#scratch_paths[@]} > 0)) || {
  echo "ERROR: scratch export produced no staged changes" >&2
  exit 1
}
for path in "${scratch_paths[@]}"; do
  is_allowed_export_path "$path" || {
    echo "ERROR: scratch export changed unexpected path: $path" >&2
    exit 1
  }
done

echo
echo "Module-file changes produced by export:"
if git diff --cached --quiet -- go.mod go.sum; then
  echo "  none"
else
  git diff --cached -- go.mod go.sum
fi

git diff --cached --binary --no-ext-diff > "$patch_file"
git -C "$target" apply --check --index "$patch_file"

if [[ "$mode" == "dry-run" ]]; then
  cat <<RESULT

PROJECT VALIDATOR EXPORT DRY RUN PASSED
────────────────────────────────────────────────────────────────────
Target:        $target
Target commit: $target_head
Go module:     $target_module
Modified:      no
Committed:     no
Pushed:        no

Proposed staged paths:
$(printf '  %s\n' "${scratch_paths[@]}")
RESULT
  exit 0
fi

cat > "$journal" <<JOURNAL
head=$target_head
target=$target
started=$(date -u +%Y-%m-%dT%H:%M:%SZ)
JOURNAL
transaction_started=1

cd "$target"
echo "Applying validated export patch to target..."
git apply --index "$patch_file"

mapfile -t target_paths < <(git diff --cached --name-only)
if [[ "$(printf '%s\n' "${scratch_paths[@]}")" != "$(printf '%s\n' "${target_paths[@]}")" ]]; then
  echo "ERROR: target staged paths do not match validated scratch export" >&2
  exit 1
fi

scratch_patch_sha="$(sha256sum "$patch_file" | awk '{print $1}')"
target_patch="$work_root/target.patch"
git diff --cached --binary --no-ext-diff > "$target_patch"
target_patch_sha="$(sha256sum "$target_patch" | awk '{print $1}')"
[[ "$scratch_patch_sha" == "$target_patch_sha" ]] || {
  echo "ERROR: target staged patch does not match validated scratch patch" >&2
  exit 1
}

echo "Revalidating applied target tree..."
if ! tidy_diff="$(run_go mod tidy -diff)"; then
  echo "ERROR: target go mod tidy -diff invocation failed" >&2
  exit 1
fi
[[ -z "$tidy_diff" ]] || {
  echo "ERROR: applied target module graph is not stable" >&2
  printf '%s\n' "$tidy_diff" >&2
  exit 1
}
run_go test -count=1 ./...
run_go vet ./...
run_go build ./...
run_go mod verify

git diff --cached --check
rm -f -- "$journal"
transaction_started=0

cat <<RESULT

PROJECT VALIDATOR EXPORTED TRANSACTIONALLY
────────────────────────────────────────────────────────────────────
Target:        $target
Target commit: $target_head
Go module:     $target_module
Source owned:  yes — copied into the target repository
Scratch tested: yes
Target tested:  yes
Staged:         yes
Committed:      no
Pushed:         no

Review the staged export:
  cd '$target'
  git status --short --branch
  git diff --cached --stat
  git diff --cached

Build the target-owned validator:
  ./tools/isras/build-validator.sh

Run project validation:
  ./.local/bin/isras-validate all
RESULT
