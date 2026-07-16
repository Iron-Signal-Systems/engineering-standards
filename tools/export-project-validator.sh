#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'USAGE'
Usage:
  ./tools/export-project-validator.sh /path/to/project

Copies the repository-owned Go validator into another Go project so the adopting
project retains the exact validation and unit-test source in its own history.

The target must:
  - be a clean Git repository;
  - contain a go.mod file;
  - not already contain cmd/isras-validate or internal/isras;
  - not already contain validation/secret-allowlist.json.

The export does not commit or push the target project.
USAGE
}

[[ $# -eq 1 ]] || { usage >&2; exit 2; }
source_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[[ -n "$source_root" ]] || { echo "ERROR: run from the engineering-standards repository" >&2; exit 1; }

target="$(realpath -e "$1")"
[[ -d "$target/.git" ]] || { echo "ERROR: target is not a Git repository: $target" >&2; exit 1; }
[[ -f "$target/go.mod" ]] || { echo "ERROR: target has no go.mod: $target" >&2; exit 1; }

if [[ -n "$(git -C "$target" status --porcelain=v1 --untracked-files=all)" ]]; then
  echo "ERROR: target repository must be clean before validator export" >&2
  git -C "$target" status --short --branch >&2
  exit 1
fi

for path in cmd/isras-validate internal/isras validation/secret-allowlist.json validation/tool-versions.json tools/isras/build-validator.sh; do
  [[ ! -e "$target/$path" ]] || { echo "ERROR: target path already exists: $path" >&2; exit 1; }
done

target_module="$(cd "$target" && go list -m -f '{{.Path}}')"
[[ -n "$target_module" ]] || { echo "ERROR: unable to resolve target module path" >&2; exit 1; }

mkdir -p \
  "$target/cmd" \
  "$target/internal/isras" \
  "$target/validation" \
  "$target/tools/isras"

cp -a "$source_root/cmd/isras-validate" "$target/cmd/"
for package in dashboard executil failurelog model redact repository secrets validation; do
  cp -a "$source_root/internal/$package" "$target/internal/isras/$package"
done
cp "$source_root/validation/secret-allowlist.json" "$target/validation/secret-allowlist.json"
cp "$source_root/validation/tool-versions.json" "$target/validation/tool-versions.json"
cp "$source_root/tools/build-validator.sh" "$target/tools/isras/build-validator.sh"
chmod 0755 "$target/tools/isras/build-validator.sh"

find "$target/cmd/isras-validate" "$target/internal/isras" -type f -name '*.go' -print0 |
  while IFS= read -r -d '' file; do
    sed -i \
      "s#github.com/Iron-Signal-Systems/engineering-standards/internal/#$target_module/internal/isras/#g" \
      "$file"
  done

if [[ -f "$target/.gitignore" ]]; then
  if ! grep -qxF '.local/' "$target/.gitignore"; then
    printf '\n# Repository-local ISRAS validation output\n.local/\n' >> "$target/.gitignore"
  fi
else
  printf '# Repository-local ISRAS validation output\n.local/\n' > "$target/.gitignore"
fi

cd "$target"
gofmt -w cmd/isras-validate internal/isras
go test -count=1 ./...
go vet ./...
go build ./...
go mod tidy -diff
go mod verify

git add \
  .gitignore \
  cmd/isras-validate \
  internal/isras \
  validation/secret-allowlist.json \
  validation/tool-versions.json \
  tools/isras/build-validator.sh

cat <<RESULT

PROJECT VALIDATOR EXPORTED
────────────────────────────────────────────────────────────────────
Target:        $target
Go module:     $target_module
Source owned:  yes — copied into the target repository
Tests copied:  yes
Committed:     no
Pushed:        no

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
