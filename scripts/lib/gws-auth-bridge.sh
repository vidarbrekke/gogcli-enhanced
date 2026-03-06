#!/usr/bin/env bash

gws_bridge_log() {
  if declare -F log >/dev/null 2>&1; then
    log "$@"
  else
    echo "$@"
  fi
}

gws_bridge_warn() {
  if declare -F warn >/dev/null 2>&1; then
    warn "$@"
  else
    echo "Warning: $*" >&2
  fi
}

gws_bridge_err() {
  if declare -F err >/dev/null 2>&1; then
    err "$@"
  else
    echo "Error: $*" >&2
  fi
}

gws_bridge_bin() {
  printf '%s\n' "${GWS_BIN:-gws}"
}

gws_bridge_available() {
  command -v "$(gws_bridge_bin)" >/dev/null 2>&1
}

gws_capture_export_json() {
  local out_path="$1"
  "$(gws_bridge_bin)" auth export --unmasked >"$out_path"
}

gws_capture_status_json() {
  local out_path="$1"
  "$(gws_bridge_bin)" auth status >"$out_path"
}

gws_status_field() {
  local status_path="$1"
  local field="$2"
  python3 - "$status_path" "$field" <<'PY'
import json
import sys

status_path, field = sys.argv[1:3]
with open(status_path, "r", encoding="utf-8") as f:
    data = json.load(f)
value = data.get(field)
if isinstance(value, bool):
    print("true" if value else "false")
elif value is None:
    print("")
else:
    print(str(value))
PY
}

gws_export_has_required_fields() {
  local export_path="$1"
  python3 - "$export_path" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as f:
    data = json.load(f)

required = ["client_id", "client_secret", "refresh_token"]
for key in required:
    if not str(data.get(key, "")).strip():
        raise SystemExit(1)

token_type = str(data.get("type", "")).strip()
if token_type and token_type != "authorized_user":
    raise SystemExit(1)
PY
}

gws_write_gog_credentials_from_export() {
  local export_path="$1"
  local credentials_path="$2"
  python3 - "$export_path" "$credentials_path" <<'PY'
import json
import os
import sys

export_path, credentials_path = sys.argv[1:3]
with open(export_path, "r", encoding="utf-8") as f:
    data = json.load(f)

client_id = str(data.get("client_id", "")).strip()
client_secret = str(data.get("client_secret", "")).strip()
if not client_id or not client_secret:
    raise SystemExit(1)

os.makedirs(os.path.dirname(credentials_path), exist_ok=True)
with open(credentials_path, "w", encoding="utf-8") as f:
    json.dump({"client_id": client_id, "client_secret": client_secret}, f, indent=2)
    f.write("\n")
PY
}

gws_write_gog_token_import_from_export() {
  local export_path="$1"
  local email="$2"
  local out_path="$3"
  local client_name="${4:-}"
  python3 - "$export_path" "$email" "$out_path" "$client_name" <<'PY'
import json
import os
import sys

export_path, email, out_path, client_name = sys.argv[1:5]
email = email.strip()
if not email:
    raise SystemExit(1)

with open(export_path, "r", encoding="utf-8") as f:
    data = json.load(f)

refresh_token = str(data.get("refresh_token", "")).strip()
if not refresh_token:
    raise SystemExit(1)

payload = {
    "email": email,
    "refresh_token": refresh_token,
}
if client_name.strip():
    payload["client"] = client_name.strip()

os.makedirs(os.path.dirname(out_path), exist_ok=True)
with open(out_path, "w", encoding="utf-8") as f:
    json.dump(payload, f, indent=2)
    f.write("\n")
PY
}

gws_guess_email_from_export() {
  local export_path="$1"
  python3 - "$export_path" <<'PY'
import json
import sys
import urllib.parse
import urllib.request

export_path = sys.argv[1]
with open(export_path, "r", encoding="utf-8") as f:
    data = json.load(f)

client_id = str(data.get("client_id", "")).strip()
client_secret = str(data.get("client_secret", "")).strip()
refresh_token = str(data.get("refresh_token", "")).strip()
if not client_id or not client_secret or not refresh_token:
    raise SystemExit(1)

token_body = urllib.parse.urlencode(
    {
        "client_id": client_id,
        "client_secret": client_secret,
        "refresh_token": refresh_token,
        "grant_type": "refresh_token",
    }
).encode("utf-8")

token_req = urllib.request.Request(
    "https://oauth2.googleapis.com/token",
    data=token_body,
    headers={"Content-Type": "application/x-www-form-urlencoded"},
)
with urllib.request.urlopen(token_req, timeout=10) as resp:
    token_data = json.loads(resp.read().decode("utf-8"))

access_token = str(token_data.get("access_token", "")).strip()
if not access_token:
    raise SystemExit(1)

userinfo_req = urllib.request.Request(
    "https://www.googleapis.com/oauth2/v2/userinfo",
    headers={"Authorization": f"Bearer {access_token}"},
)
with urllib.request.urlopen(userinfo_req, timeout=10) as resp:
    userinfo = json.loads(resp.read().decode("utf-8"))

email = str(userinfo.get("email", "")).strip()
if not email:
    raise SystemExit(1)

print(email)
PY
}

gws_bootstrap_export_file() {
  local out_path="$1"
  local no_input="${2:-0}"
  local status_path

  if ! gws_bridge_available; then
    return 1
  fi

  if gws_capture_export_json "$out_path" >/dev/null 2>&1 && gws_export_has_required_fields "$out_path"; then
    return 0
  fi

  if [[ "$no_input" -eq 1 ]]; then
    return 1
  fi

  status_path="$(mktemp /tmp/gws-status-XXXXXX.json)"
  if ! gws_capture_status_json "$status_path" >/dev/null 2>&1; then
    rm -f "$status_path"
    return 1
  fi

  if [[ "$(gws_status_field "$status_path" "client_config_exists")" == "true" ]]; then
    # Official interactive re-auth path: `gws auth login`
    gws_bridge_log "Launching official Google Workspace CLI login..."
    "$(gws_bridge_bin)" auth login
  else
    # Official first-time bootstrap path: `gws auth setup`
    gws_bridge_log "Launching official Google Workspace CLI setup..."
    "$(gws_bridge_bin)" auth setup
  fi
  rm -f "$status_path"

  if ! gws_capture_export_json "$out_path"; then
    return 1
  fi

  gws_export_has_required_fields "$out_path"
}
