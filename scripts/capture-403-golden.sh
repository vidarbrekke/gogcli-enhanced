#!/usr/bin/env bash
# Capture a real gws Gmail labels-list 403 (no Gmail scope) and promote the golden.
# Tries ~/.config/gws/credentials-no-gmail.json first; otherwise drive-only OAuth (browser).
#
# Usage (from repo root):
#   scripts/capture-403-golden.sh
#   scripts/capture-403-golden.sh --check          # preflight only
#   scripts/capture-403-golden.sh --force-browser  # skip existing-creds probe
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"
GOLDEN_DIR="$ROOT_DIR/docs/merge/goldens/gmail-labels-403-forbidden/gws"
BACKUP_DIR="${TMPDIR:-/tmp}/gws-403-capture-$$"
GWS_BIN="${GOG_GWS_PATH:-gws}"
NO_GMAIL_CREDS="${GOG_GWS_NO_GMAIL_CREDS:-$HOME/.config/gws/credentials-no-gmail.json}"

CHECK_ONLY=false
FORCE_BROWSER=false
while [ $# -gt 0 ]; do
  case "$1" in
    --check) CHECK_ONLY=true ;;
    --force-browser) FORCE_BROWSER=true ;;
    -h|--help)
      sed -n '2,8p' "$0"
      echo "  --check          Preflight: test existing no-gmail creds; exit 0 if 403 ready"
      echo "  --force-browser  Always run drive-only OAuth (ignore existing creds file)"
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      exit 2
      ;;
  esac
  shift
done

if ! command -v "$GWS_BIN" >/dev/null 2>&1 && [ ! -x "$GWS_BIN" ]; then
  echo "gws not found (set GOG_GWS_PATH or install gws)" >&2
  exit 1
fi

is_403_stdout() {
  python3 -c 'import json,sys; o=json.load(open(sys.argv[1])); e=o.get("error") or {};
raise SystemExit(0 if int(e.get("code") or 0)==403 else 1)' "$1"
}

error_code_stdout() {
  python3 -c 'import json,sys
try:
  o=json.load(open(sys.argv[1])); e=o.get("error") or {}
  print(int(e.get("code") or 0))
except Exception:
  print(0)' "$1"
}

try_capture_with_creds() {
  local creds_file="$1"
  local out="$2"
  local err="$3"
  set +e
  GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE="$creds_file" \
    "$GWS_BIN" gmail users labels list --params '{"userId":"me"}' >"$out" 2>"$err"
  local code=$?
  set -e
  printf '%s\n' "$code"
}

probe_no_gmail_creds() {
  local creds_file="$1"
  if [ ! -f "$creds_file" ]; then
    echo "missing"
    return
  fi
  local out err code
  out="$(mktemp)"
  err="$(mktemp)"
  code="$(try_capture_with_creds "$creds_file" "$out" "$err")"
  if is_403_stdout "$out"; then
    echo "ready:403:$out:$err:$code"
  else
    echo "fail:$(error_code_stdout "$out"):$out"
  fi
  rm -f "$out" "$err"
}

if [ "$CHECK_ONLY" = true ]; then
  echo "==> Preflight: $NO_GMAIL_CREDS"
  result="$(probe_no_gmail_creds "$NO_GMAIL_CREDS")"
  case "$result" in
    missing)
      echo "no credentials file; run script without --check for drive-only OAuth"
      exit 1
      ;;
    ready:403:*)
      echo "ok: existing creds return HTTP 403; run script without --check to write golden"
      exit 0
      ;;
    fail:*)
      code="${result#fail:}"
      code="${code%%:*}"
      echo "stale or wrong creds (got error.code=$code, need 403); re-run without --check for fresh drive-only login" >&2
      exit 1
      ;;
  esac
fi

mkdir -p "$BACKUP_DIR" "$GOLDEN_DIR"
trap 'rm -rf "$BACKUP_DIR"' EXIT

if [ "$FORCE_BROWSER" = false ]; then
  result="$(probe_no_gmail_creds "$NO_GMAIL_CREDS")"
  case "$result" in
    ready:403:*)
      IFS=':' read -r _ _ stdout_path stderr_path exit_code <<<"$result"
      cp "$stdout_path" "$BACKUP_DIR/stdout.json"
      cp "$stderr_path" "$BACKUP_DIR/stderr.txt"
      printf '%s\n' "$exit_code" >"$BACKUP_DIR/exit_code.txt"
      echo "==> Using existing no-gmail creds (HTTP 403); skipping browser OAuth"
      ;;
  esac
fi

if [ ! -f "$BACKUP_DIR/stdout.json" ]; then
  echo "==> Backing up current gws credentials export to $BACKUP_DIR/credentials-full.json"
  "$GWS_BIN" auth export --unmasked >"$BACKUP_DIR/credentials-full.json"

  echo "==> Drive-only OAuth login (browser). Grant Drive readonly only — no Gmail."
  echo "    When the browser opens, complete consent, then return here."
  "$GWS_BIN" auth login -s drive --readonly

  echo "==> Exporting drive-only credentials to $NO_GMAIL_CREDS"
  mkdir -p "$(dirname "$NO_GMAIL_CREDS")"
  "$GWS_BIN" auth export --unmasked >"$NO_GMAIL_CREDS"

  echo "==> Capturing gmail labels list (expect HTTP 403)"
  exit_code="$(try_capture_with_creds "$NO_GMAIL_CREDS" "$BACKUP_DIR/stdout.json" "$BACKUP_DIR/stderr.txt")"
  printf '%s\n' "$exit_code" >"$BACKUP_DIR/exit_code.txt"
fi

if ! is_403_stdout "$BACKUP_DIR/stdout.json"; then
  got="$(error_code_stdout "$BACKUP_DIR/stdout.json")"
  echo "capture did not return error.code=403 (got $got); stdout follows:" >&2
  cat "$BACKUP_DIR/stdout.json" >&2
  echo "stderr:" >&2
  cat "$BACKUP_DIR/stderr.txt" >&2
  if [ "$FORCE_BROWSER" = false ] && [ -f "$NO_GMAIL_CREDS" ]; then
    echo "Hint: stale credentials-no-gmail.json often returns 401; run with --force-browser" >&2
  fi
  if [ -f "$BACKUP_DIR/credentials-full.json" ]; then
    echo "Restoring full credentials via auth login..." >&2
    "$GWS_BIN" auth login || true
  fi
  exit 1
fi

cp "$BACKUP_DIR/stdout.json" "$GOLDEN_DIR/stdout.json"
printf '{}\n' >"$GOLDEN_DIR/stderr.json"
cp "$BACKUP_DIR/exit_code.txt" "$GOLDEN_DIR/exit_code.txt"
rm -f "$GOLDEN_DIR/PLACEHOLDER.txt"

{
  echo "gws version: $($GWS_BIN --version 2>/dev/null | head -1)"
  echo "command argv: gws gmail users labels list --params '{\"userId\":\"me\"}'"
  echo "OAuth scopes: https://www.googleapis.com/auth/drive.readonly"
  echo "profile/creds: $NO_GMAIL_CREDS"
} >"$GOLDEN_DIR/capture-info.txt"

echo "==> Wrote golden under $GOLDEN_DIR (PLACEHOLDER removed)"

if [ -f "$BACKUP_DIR/credentials-full.json" ]; then
  echo "==> Re-authenticate with full scopes to restore normal gws use:"
  echo "    $GWS_BIN auth login"
  "$GWS_BIN" auth login
fi

echo "==> Done. Run: make parity"
echo "    Then commit the golden under docs/merge/goldens/gmail-labels-403-forbidden/gws/"
