#!/usr/bin/env bash
# Authenticated smoke for Tier A GOG_BACKEND routing (native + gws).
# Opt-in: requires OAuth credentials. gws half also requires `gws` on PATH
# (or GOG_GWS_PATH) authenticated to the default imported account.
#
# Usage:
#   scripts/smoke-gws-routing.sh
#   scripts/smoke-gws-routing.sh --native-only
#   scripts/smoke-gws-routing.sh --gws-only
#   scripts/smoke-gws-routing.sh --account you@gmail.com   # native only; ignored for gws
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"

RUN_NATIVE=true
RUN_GWS=true
ACCOUNT=""

usage() {
  cat <<'USAGE'
Usage: scripts/smoke-gws-routing.sh [options]

Smoke Gmail labels list/get and Drive ls/get/search under GOG_BACKEND=native
and GOG_BACKEND=gws. Validates JSON shapes used by agents (--json).

Options:
  --native-only       Only run native backend checks
  --gws-only          Only run gws backend checks
  --account <email>   Account for native path (-a). Defaults to GOG_IT_ACCOUNT
                      or first auth list entry. Must not be used with gws.
  -h, --help          Show this help

Env:
  GOG_BIN             Path to gog binary (default: $ROOT/bin/gog, built if missing)
  GOG_GWS_PATH        Path to gws binary (default: gws on PATH)
  GOG_IT_ACCOUNT      Default native account
USAGE
}

while [ $# -gt 0 ]; do
  case "$1" in
    --native-only) RUN_GWS=false ;;
    --gws-only) RUN_NATIVE=false ;;
    --account)
      ACCOUNT="$2"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

PY="${PYTHON:-python3}"
if ! command -v "$PY" >/dev/null 2>&1; then
  PY="python"
fi

BIN="${GOG_BIN:-$ROOT_DIR/bin/gog}"
if [ ! -x "$BIN" ]; then
  make -C "$ROOT_DIR" build >/dev/null
fi

if [ -z "$ACCOUNT" ]; then
  ACCOUNT="${GOG_IT_ACCOUNT:-}"
fi
if [ -z "$ACCOUNT" ] && [ "$RUN_NATIVE" = true ]; then
  acct_json="$("$BIN" auth list --json 2>/dev/null || true)"
  if [ -n "$acct_json" ]; then
    ACCOUNT="$($PY -c 'import json,sys
try:
  obj=json.load(sys.stdin)
  accts=obj.get("accounts") or []
  print(accts[0].get("email","") if accts else "")
except Exception:
  print("")' <<<"$acct_json")"
  fi
fi

assert_json() {
  local label="$1"
  local payload="$2"
  local kind="$3"
  "$PY" -c '
import json, sys
label, kind = sys.argv[1], sys.argv[2]
raw = sys.stdin.read()
try:
    obj = json.loads(raw)
except Exception as e:
    raise SystemExit(f"{label}: invalid JSON: {e}")
if kind == "labels":
    if not isinstance(obj.get("labels"), list) or not obj["labels"]:
        raise SystemExit(f"{label}: expected non-empty labels[]")
elif kind == "label":
    lab = obj.get("label") if isinstance(obj.get("label"), dict) else obj
    if not isinstance(lab, dict) or not lab.get("id"):
        raise SystemExit(f"{label}: expected label with id")
elif kind == "files":
    if not isinstance(obj.get("files"), list):
        raise SystemExit(f"{label}: expected files[]")
elif kind == "file":
    f = obj.get("file") if isinstance(obj.get("file"), dict) else obj
    if not isinstance(f, dict) or not f.get("id"):
        raise SystemExit(f"{label}: expected file with id")
else:
    raise SystemExit(f"unknown kind {kind}")
print(f"ok {label}")
' "$label" "$kind" <<<"$payload"
}

first_file_id() {
  "$PY" -c '
import json,sys
obj=json.load(sys.stdin)
files=obj.get("files") or []
print(files[0].get("id","") if files else "")
' <<<"$1"
}

run_suite() {
  local backend="$1"
  local account="${2:-}"

  echo "==> backend=$backend${account:+ account=$account}"

  export GOG_BACKEND="$backend"
  # gws rejects explicit account; clear accidental env for that half
  if [ "$backend" = "gws" ]; then
    unset GOG_ACCOUNT || true
  fi

  local -a gog_cmd=("$BIN")
  if [ -n "$account" ]; then
    gog_cmd+=(--account "$account")
  fi

  local out file_id

  out="$("${gog_cmd[@]}" gmail labels list --json)"
  assert_json "$backend gmail labels list" "$out" "labels"

  out="$("${gog_cmd[@]}" gmail labels get INBOX --json)"
  assert_json "$backend gmail labels get" "$out" "label"

  out="$("${gog_cmd[@]}" drive ls --json --max 5)"
  assert_json "$backend drive ls" "$out" "files"
  file_id="$(first_file_id "$out")"
  if [ -z "$file_id" ]; then
    echo "$backend drive ls returned no files; skipping drive get (still ok for empty Drive)" >&2
  else
    out="$("${gog_cmd[@]}" drive get "$file_id" --json)"
    assert_json "$backend drive get" "$out" "file"
  fi

  out="$("${gog_cmd[@]}" drive search --raw 'trashed = false' --json --max 5)"
  assert_json "$backend drive search" "$out" "files"
}

if [ "$RUN_NATIVE" = true ]; then
  if [ -z "$ACCOUNT" ]; then
    echo "native smoke needs an account (pass --account or set GOG_IT_ACCOUNT)" >&2
    exit 1
  fi
  run_suite native "$ACCOUNT"
fi

if [ "$RUN_GWS" = true ]; then
  GWS_BIN="${GOG_GWS_PATH:-gws}"
  if ! command -v "$GWS_BIN" >/dev/null 2>&1 && [ ! -x "$GWS_BIN" ]; then
    echo "gws smoke needs gws on PATH or GOG_GWS_PATH (or use --native-only)" >&2
    exit 1
  fi
  run_suite gws
fi

echo "smoke-gws-routing: all requested backends passed"
