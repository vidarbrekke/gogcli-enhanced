#!/usr/bin/env bash
set -euo pipefail

# Collect a reproducible auth diagnostics report for gog/gogcli-enhanced.
# Safe-by-default: does not print token values; only config/status and file schema summaries.

GOG_BIN="${GOG_BIN:-/root/.local/bin/gog}"
WORKSPACE_ROOT="${WORKSPACE_ROOT:-/root/openclaw-stock-home/.openclaw/workspace}"
REPORT_DIR="${REPORT_DIR:-$WORKSPACE_ROOT/tmp}"
TS="$(date -u +%Y%m%dT%H%M%SZ)"
REPORT="$REPORT_DIR/gog-auth-diagnostic-$TS.txt"

mkdir -p "$REPORT_DIR"

run() {
  local title="$1"
  shift
  {
    echo
    echo "===== $title ====="
    echo "$ $*"
    "$@"
  } >>"$REPORT" 2>&1 || true
}

json_shape() {
  local p="$1"
  if [[ ! -f "$p" ]]; then
    echo "MISSING: $p" >>"$REPORT"
    return
  fi
  python3 - <<PY >>"$REPORT" 2>&1
import json
p=${p@Q}
try:
    j=json.load(open(p))
except Exception as e:
    print(f"INVALID_JSON: {p}: {e}")
    raise SystemExit(0)
keys=sorted(list(j.keys())) if isinstance(j,dict) else []
print(f"FILE: {p}")
print(f"TOP_LEVEL_KEYS: {keys}")
if isinstance(j,dict):
    if 'client_id' in j or 'client_secret' in j:
        print("SCHEMA: flat_client_credentials")
    elif 'installed' in j or 'web' in j:
        print("SCHEMA: oauth_client_nested")
    else:
        print("SCHEMA: other")
PY
}

{
  echo "gog auth diagnostic report"
  echo "generated_utc: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
  echo "host: $(hostname)"
  echo "pwd: $(pwd)"
  echo "gog_bin: $GOG_BIN"
} >"$REPORT"

run "binary_version" "$GOG_BIN" version
run "gog_help_header" bash -lc "$GOG_BIN --help | sed -n '1,30p'"
run "auth_status" "$GOG_BIN" auth status
run "auth_list_check" "$GOG_BIN" auth list --check
run "auth_credentials_list" "$GOG_BIN" auth credentials list

{
  echo
  echo "===== credentials_file_schema_checks ====="
  json_shape "/root/.config/gogcli/credentials.json"
  json_shape "/root/openclaw-stock-home/.config/gogcli/credentials.json"
} >>"$REPORT"

run "keyring_paths" bash -lc "ls -la /root/.config/gogcli /root/openclaw-stock-home/.config/gogcli 2>/dev/null || true"
run "mcporter_config" bash -lc "cat /root/openclaw-stock-home/.openclaw/workspace/config/mcporter.json 2>/dev/null || true"
run "mcporter_list" bash -lc "mcporter --config /root/openclaw-stock-home/.openclaw/workspace/config/mcporter.json list"

echo "REPORT_PATH=$REPORT"
echo "Done. Share this file with a developer for exact reproduction."