#!/usr/bin/env bash
# Capture a real gws Gmail labels-list 403 (no Gmail scope) and promote the golden.
# Interactive: opens a browser for drive-only OAuth, then restores full auth login.
#
# Usage (from repo root):
#   scripts/capture-403-golden.sh
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"
GOLDEN_DIR="$ROOT_DIR/docs/merge/goldens/gmail-labels-403-forbidden/gws"
BACKUP_DIR="${TMPDIR:-/tmp}/gws-403-capture-$$"
GWS_BIN="${GOG_GWS_PATH:-gws}"

if ! command -v "$GWS_BIN" >/dev/null 2>&1 && [ ! -x "$GWS_BIN" ]; then
  echo "gws not found (set GOG_GWS_PATH or install gws)" >&2
  exit 1
fi

mkdir -p "$BACKUP_DIR" "$GOLDEN_DIR"
trap 'rm -rf "$BACKUP_DIR"' EXIT

echo "==> Backing up current gws credentials export to $BACKUP_DIR/credentials-full.json"
"$GWS_BIN" auth export --unmasked >"$BACKUP_DIR/credentials-full.json"

echo "==> Drive-only OAuth login (browser). Grant Drive readonly only — no Gmail."
echo "    When the browser opens, complete consent, then return here."
"$GWS_BIN" auth login -s drive --readonly

echo "==> Exporting drive-only credentials"
"$GWS_BIN" auth export --unmasked >"$HOME/.config/gws/credentials-no-gmail.json"

echo "==> Capturing gmail labels list (expect HTTP 403)"
set +e
GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE="$HOME/.config/gws/credentials-no-gmail.json" \
  "$GWS_BIN" gmail users labels list --params '{"userId":"me"}' \
  >"$BACKUP_DIR/stdout.json" 2>"$BACKUP_DIR/stderr.txt"
exit_code=$?
set -e

if ! python3 -c 'import json,sys; o=json.load(open(sys.argv[1])); e=o.get("error") or {};
raise SystemExit(0 if int(e.get("code") or 0)==403 else 1)' "$BACKUP_DIR/stdout.json"; then
  echo "capture did not return error.code=403; stdout follows:" >&2
  cat "$BACKUP_DIR/stdout.json" >&2
  echo "stderr:" >&2
  cat "$BACKUP_DIR/stderr.txt" >&2
  echo "Restoring full credentials via auth login..." >&2
  "$GWS_BIN" auth login || true
  exit 1
fi

cp "$BACKUP_DIR/stdout.json" "$GOLDEN_DIR/stdout.json"
printf '%s\n' '{}' >"$GOLDEN_DIR/stderr.json"
printf '%s\n' "$exit_code" >"$GOLDEN_DIR/exit_code.txt"
rm -f "$GOLDEN_DIR/PLACEHOLDER.txt"

{
  echo "gws version: $($GWS_BIN --version 2>/dev/null | head -1)"
  echo "command argv: gws gmail users labels list --params '{\"userId\":\"me\"}'"
  echo "OAuth scopes: https://www.googleapis.com/auth/drive.readonly"
  echo "profile/creds: $HOME/.config/gws/credentials-no-gmail.json"
} >"$GOLDEN_DIR/capture-info.txt"

echo "==> Wrote golden under $GOLDEN_DIR (PLACEHOLDER removed)"
echo "==> Re-authenticate with full scopes to restore normal gws use:"
echo "    $GWS_BIN auth login"
"$GWS_BIN" auth login

echo "==> Done. Run: make parity"
echo "    Then commit the golden + capture-info + any hard-gate docs updates."
