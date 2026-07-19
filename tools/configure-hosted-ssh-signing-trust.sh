#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'USAGE'
Usage:
  configure-hosted-ssh-signing-trust.sh \
    --target /path/to/consuming-repository \
    --runtime-root /path/to/private/runtime-directory

The trust source is fixed to trust/ssh in the exact Engineering Standards
checkout containing this script. The consuming repository cannot select or
replace the trust source.
USAGE
}

TARGET_REQUESTED=""
RUNTIME_REQUESTED=""
while (($# > 0)); do
  case "$1" in
    --target)
      (($# >= 2)) || { usage >&2; exit 2; }
      TARGET_REQUESTED="$2"
      shift 2
      ;;
    --runtime-root)
      (($# >= 2)) || { usage >&2; exit 2; }
      RUNTIME_REQUESTED="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      printf 'FAIL: unsupported argument: %s\n' "$1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

[[ -n "$TARGET_REQUESTED" && -n "$RUNTIME_REQUESTED" ]] || {
  usage >&2
  exit 2
}

for command in git python3 sha256sum ssh-keygen install readlink tee awk; do
  command -v "$command" >/dev/null || {
    printf 'FAIL: required command is unavailable: %s\n' "$command" >&2
    exit 1
  }
done

SCRIPT_PATH="$(readlink -f -- "${BASH_SOURCE[0]}")"
SCRIPT_DIRECTORY="$(dirname -- "$SCRIPT_PATH")"
STANDARD_ROOT="$(git -C "$SCRIPT_DIRECTORY" rev-parse --show-toplevel 2>/dev/null)" || {
  echo 'FAIL: the trust bootstrap is not running from an Engineering Standards Git checkout' >&2
  exit 1
}
STANDARD_ROOT="$(readlink -f -- "$STANDARD_ROOT")"
TARGET_ROOT="$(git -C "$TARGET_REQUESTED" rev-parse --show-toplevel 2>/dev/null)" || {
  echo 'FAIL: target is not a Git repository' >&2
  exit 1
}
TARGET_ROOT="$(readlink -f -- "$TARGET_ROOT")"
RUNTIME_ROOT="$(readlink -m -- "$RUNTIME_REQUESTED")"
TRUST_ROOT="$STANDARD_ROOT/trust/ssh"
ALLOWED_SIGNERS="$TRUST_ROOT/iron-signal-systems.allowed-signers"
CHECKSUM_FILE="$TRUST_ROOT/iron-signal-systems.allowed-signers.sha256"
MANIFEST_FILE="$TRUST_ROOT/manifest.json"
RUNTIME_ALLOWED_SIGNERS="$RUNTIME_ROOT/allowed-signers"
SUCCESS_EVIDENCE="$TARGET_ROOT/.local/isras/hosted-ssh-signer-trust.json"
FAILURE_LOG="$TARGET_ROOT/.local/validation/logs/hosted-ssh-signer-trust.log"
STAGE="startup"

mkdir -p -- "$(dirname -- "$FAILURE_LOG")"
chmod 0700 -- "$(dirname -- "$FAILURE_LOG")" 2>/dev/null || true
: >"$FAILURE_LOG"
chmod 0600 "$FAILURE_LOG" 2>/dev/null || true
exec > >(tee -a "$FAILURE_LOG") 2>&1

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  return 1
}

on_error() {
  local status=$?
  printf '\nHOSTED SSH SIGNER TRUST: FAIL\n'
  printf 'Stage: %s\n' "$STAGE"
  printf 'Exit status: %s\n' "$status"
  printf 'Target: %s\n' "$TARGET_ROOT"
  printf 'Standard source: %s\n' "$STANDARD_ROOT"
  printf 'Failure log: %s\n' "$FAILURE_LOG"
  exit "$status"
}
trap on_error ERR

STAGE="validate pinned trust source"
for path in "$ALLOWED_SIGNERS" "$CHECKSUM_FILE" "$MANIFEST_FILE"; do
  [[ -f "$path" && ! -L "$path" ]] || fail "trust source is missing, non-regular, or symbolic: $path"
done

case "$TRUST_ROOT/" in
  "$STANDARD_ROOT"/*) ;;
  *) fail 'trust source escaped the exact Engineering Standards checkout' ;;
esac

for relative in \
  trust/ssh/iron-signal-systems.allowed-signers \
  trust/ssh/iron-signal-systems.allowed-signers.sha256 \
  trust/ssh/manifest.json \
  tools/configure-hosted-ssh-signing-trust.sh
do
  git -C "$STANDARD_ROOT" ls-files --error-unmatch -- "$relative" >/dev/null ||
    fail "required trust source is not tracked: $relative"
  tracked_blob="$(git -C "$STANDARD_ROOT" rev-parse "HEAD:$relative")"
  working_blob="$(git hash-object "$STANDARD_ROOT/$relative")"
  [[ "$tracked_blob" == "$working_blob" ]] ||
    fail "tracked trust source differs from the exact standard commit: $relative"
done

(
  cd "$TRUST_ROOT"
  sha256sum -c "$(basename -- "$CHECKSUM_FILE")"
)

python3 - "$ALLOWED_SIGNERS" "$CHECKSUM_FILE" "$MANIFEST_FILE" <<'PY'
import hashlib
import json
import pathlib
import re
import subprocess
import sys
import tempfile

allowed_path, checksum_path, manifest_path = map(pathlib.Path, sys.argv[1:])
allowed = allowed_path.read_bytes()
checksum_line = checksum_path.read_text(encoding="utf-8").strip()
manifest = json.loads(manifest_path.read_text(encoding="utf-8"))

digest = hashlib.sha256(allowed).hexdigest()
parts = checksum_line.split()
if len(parts) != 2 or parts[0] != digest or parts[1] != allowed_path.name:
    raise SystemExit("allowed-signers checksum declaration is invalid")
if manifest.get("schema_version") != 1:
    raise SystemExit("trust manifest schema version is invalid")
if manifest.get("file") != allowed_path.name:
    raise SystemExit("trust manifest file identity is invalid")
if manifest.get("sha256") != digest:
    raise SystemExit("trust manifest digest does not match allowed-signers bytes")
if manifest.get("authority") != "Iron Signal Systems Engineering Standards":
    raise SystemExit("trust manifest authority is invalid")

lines = []
for raw in allowed.decode("utf-8").splitlines():
    line = raw.strip()
    if not line or line.startswith("#"):
        continue
    fields = line.split()
    if len(fields) != 3:
        raise SystemExit("allowed-signers entries must contain exactly principal, key type, and key")
    principal, key_type, key_body = fields
    if not re.fullmatch(r"[A-Za-z0-9._+@-]{3,254}", principal):
        raise SystemExit("allowed-signers principal is invalid")
    if key_type not in {"ssh-ed25519", "sk-ssh-ed25519@openssh.com"}:
        raise SystemExit("allowed-signers key type is outside the accepted Ed25519 boundary")
    with tempfile.NamedTemporaryFile("w", encoding="utf-8") as key_file:
        key_file.write(f"{key_type} {key_body}\n")
        key_file.flush()
        output = subprocess.check_output(
            ["ssh-keygen", "-lf", key_file.name, "-E", "sha256"],
            text=True,
        ).strip()
    fingerprint = output.split()[1]
    lines.append({"principal": principal, "fingerprint": fingerprint})

if not lines:
    raise SystemExit("allowed-signers contains no trusted signer")
expected = manifest.get("signers")
if expected != lines:
    raise SystemExit("trust manifest signer inventory does not match allowed-signers")
if len({(item["principal"], item["fingerprint"]) for item in lines}) != len(lines):
    raise SystemExit("trust source contains a duplicate signer")
PY

STAGE="prepare private runtime trust"
[[ ! -e "$RUNTIME_ROOT" ]] || fail "runtime trust directory already exists: $RUNTIME_ROOT"
install -d -m 0700 -- "$RUNTIME_ROOT"
install -m 0600 -- "$ALLOWED_SIGNERS" "$RUNTIME_ALLOWED_SIGNERS"
[[ "$(sha256sum "$RUNTIME_ALLOWED_SIGNERS" | awk '{print $1}')" == \
   "$(sha256sum "$ALLOWED_SIGNERS" | awk '{print $1}')" ]] ||
  fail 'runtime allowed-signers copy does not match the pinned source'

STAGE="configure target-local SSH verification"
git -C "$TARGET_ROOT" config --local gpg.format ssh
git -C "$TARGET_ROOT" config --local gpg.ssh.allowedSignersFile "$RUNTIME_ALLOWED_SIGNERS"
[[ "$(git -C "$TARGET_ROOT" config --local --get gpg.format)" == "ssh" ]] ||
  fail 'target-local SSH signature format was not established'
[[ "$(git -C "$TARGET_ROOT" config --local --get gpg.ssh.allowedSignersFile)" == "$RUNTIME_ALLOWED_SIGNERS" ]] ||
  fail 'target-local allowed-signers path was not established'

STAGE="verify exact target commit and principal"
VERIFY_OUTPUT="$(git -C "$TARGET_ROOT" verify-commit HEAD 2>&1)"
printf '%s\n' "$VERIFY_OUTPUT"
TARGET_COMMIT="$(git -C "$TARGET_ROOT" rev-parse HEAD)"
TARGET_COMMITTER_EMAIL="$(git -C "$TARGET_ROOT" show -s --format=%ce HEAD)"
STANDARD_COMMIT="$(git -C "$STANDARD_ROOT" rev-parse HEAD)"
TRUST_SHA256="$(sha256sum "$ALLOWED_SIGNERS" | awk '{print $1}')"

python3 - \
  "$VERIFY_OUTPUT" \
  "$TARGET_COMMITTER_EMAIL" \
  "$MANIFEST_FILE" \
  "$TARGET_COMMIT" \
  "$STANDARD_COMMIT" \
  "$TRUST_SHA256" \
  "$SUCCESS_EVIDENCE" <<'PY'
import datetime
import json
import pathlib
import re
import sys

(
    output,
    committer_email,
    manifest_path,
    target_commit,
    standard_commit,
    trust_sha256,
    evidence_path,
) = sys.argv[1:]
manifest = json.loads(pathlib.Path(manifest_path).read_text(encoding="utf-8"))
pattern = re.compile(
    r'Good "git" signature for (?P<principal>\S+) with [A-Za-z0-9_-]+ key (?P<fingerprint>SHA256:\S+)'
)
match = pattern.search(output)
if not match:
    raise SystemExit("Git did not report a bounded SSH signer principal and fingerprint")
principal = match.group("principal")
fingerprint = match.group("fingerprint")
if principal != committer_email:
    raise SystemExit("verified SSH principal does not match the exact commit committer email")
if {"principal": principal, "fingerprint": fingerprint} not in manifest.get("signers", []):
    raise SystemExit("verified SSH principal and fingerprint are not in the pinned trust manifest")

path = pathlib.Path(evidence_path)
path.parent.mkdir(parents=True, exist_ok=True)
payload = {
    "schema_version": 1,
    "status": "PASS",
    "verified_at": datetime.datetime.now(datetime.timezone.utc).isoformat().replace("+00:00", "Z"),
    "target_commit": target_commit,
    "target_committer_email": committer_email,
    "verified_principal": principal,
    "verified_fingerprint": fingerprint,
    "standard_source_commit": standard_commit,
    "trust_sha256": trust_sha256,
    "trust_authority": manifest["authority"],
}
path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
path.chmod(0o600)
PY

STAGE="final report"
printf '\nHOSTED SSH SIGNER TRUST: PASS\n'
printf 'Target commit: %s\n' "$TARGET_COMMIT"
printf 'Standard source commit: %s\n' "$STANDARD_COMMIT"
printf 'Trust SHA-256: %s\n' "$TRUST_SHA256"
printf 'Evidence: %s\n' "$SUCCESS_EVIDENCE"

rm -f -- "$FAILURE_LOG"
trap - ERR
