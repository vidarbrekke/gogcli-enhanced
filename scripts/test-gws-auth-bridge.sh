#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")/.." && pwd)"
cd "$ROOT_DIR"

source scripts/lib/gws-auth-bridge.sh

pass() { echo "[PASS] $*"; }
fail() { echo "[FAIL] $*"; exit 1; }

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

EXPORT_JSON="$TMP_DIR/gws-export.json"
cat >"$EXPORT_JSON" <<'EOF'
{
  "client_id": "cid",
  "client_secret": "csecret",
  "refresh_token": "refresh",
  "type": "authorized_user"
}
EOF

if gws_export_has_required_fields "$EXPORT_JSON"; then
  pass "valid gws export is accepted"
else
  fail "valid gws export should be accepted"
fi

CREDS_JSON="$TMP_DIR/credentials.json"
gws_write_gog_credentials_from_export "$EXPORT_JSON" "$CREDS_JSON"
python3 - "$CREDS_JSON" <<'PY'
import json
import sys
with open(sys.argv[1], "r", encoding="utf-8") as f:
    data = json.load(f)
assert data == {"client_id": "cid", "client_secret": "csecret"}
PY
pass "credentials bridge writes gog flat credentials"

TOKEN_JSON="$TMP_DIR/token.json"
gws_write_gog_token_import_from_export "$EXPORT_JSON" "user@example.com" "$TOKEN_JSON" "work"
python3 - "$TOKEN_JSON" <<'PY'
import json
import sys
with open(sys.argv[1], "r", encoding="utf-8") as f:
    data = json.load(f)
assert data["email"] == "user@example.com"
assert data["refresh_token"] == "refresh"
assert data["client"] == "work"
PY
pass "token bridge writes gog token import payload"

INVALID_JSON="$TMP_DIR/invalid-export.json"
cat >"$INVALID_JSON" <<'EOF'
{
  "client_id": "cid",
  "client_secret": "csecret",
  "type": "authorized_user"
}
EOF

if gws_export_has_required_fields "$INVALID_JSON"; then
  fail "invalid gws export should be rejected"
else
  pass "invalid gws export is rejected"
fi

echo "All gws auth bridge tests passed."
