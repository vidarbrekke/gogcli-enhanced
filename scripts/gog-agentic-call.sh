#!/usr/bin/env bash
set -euo pipefail

resolve_script_path() {
  local source_path="${BASH_SOURCE[0]:-$0}"
  while [[ -L "$source_path" ]]; do
    local source_dir
    source_dir="$(cd "$(dirname "$source_path")" && pwd -P)"
    source_path="$(readlink "$source_path")"
    [[ "$source_path" != /* ]] && source_path="$source_dir/$source_path"
  done
  printf '%s\n' "$(cd "$(dirname "$source_path")" && pwd -P)/$(basename "$source_path")"
}

SCRIPT_PATH="$(resolve_script_path)"
SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd -P)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd -P)"

usage() {
  cat <<EOF
Usage: gog-agentic-call TOOL_NAME [ARGS_JSON] [OUTPUT]

Examples:
  gog-agentic-call drive.listFiles '{}'
  gog-agentic-call drive_listFiles '{"page":"token"}'
  gog-agentic-call gog-agentic.drive_searchFiles '{"query":"test"}'

Accepts dotted or underscored tool names and resolves the workspace mcporter config automatically.
EOF
}

resolve_workspace_dir() {
  local candidates=()
  local checked=()
  local candidate

  if [[ -n "${WORKSPACE_DIR:-}" ]]; then
    candidates+=("$WORKSPACE_DIR")
  fi
  if [[ -n "${OPENCLAW_WORKSPACE_DIR:-}" ]]; then
    candidates+=("$OPENCLAW_WORKSPACE_DIR")
  fi
  if [[ -n "${OPENCLAW_WORKSPACE:-}" ]]; then
    candidates+=("$OPENCLAW_WORKSPACE")
  fi

  candidates+=("$(pwd -P)")

  if [[ "$ROOT_DIR" == *"/repositories/"* ]]; then
    candidates+=("${ROOT_DIR%/repositories/*}")
  fi
  candidates+=("$ROOT_DIR")

  if [[ -n "${HOME:-}" ]]; then
    candidates+=("$HOME/openclaw-stock-home/.openclaw/workspace")
    candidates+=("$HOME/.openclaw/workspace")
  fi

  for candidate in "${candidates[@]}"; do
    [[ -z "$candidate" ]] && continue
    if ! candidate="$(cd "$candidate" 2>/dev/null && pwd -P)"; then
      continue
    fi
    checked+=("$candidate")
    if [[ -f "$candidate/config/mcporter.json" ]]; then
      printf '%s\n' "$candidate"
      return 0
    fi
  done

  {
    echo "gog-agentic-call: config not found in any known workspace candidate."
    echo "Checked:"
    printf '%s\n' "${checked[@]}" | awk '{print "  - " $0}'
  } >&2
  return 1
}

normalize_tool_name() {
  local raw="$1"
  raw="${raw#gog-agentic.}"
  raw="${raw#gog-agentic/}"
  raw="${raw#gog-agentic:}"
  raw="${raw//./_}"
  printf '%s\n' "$raw"
}

normalize_drive_search_args() {
  local tool="$1"
  local args="$2"

  if [[ "$tool" != "drive_searchFiles" && "$tool" != "drive_search" ]]; then
    printf '%s\n' "$args"
    return 0
  fi

  if ! command -v python3 >/dev/null 2>&1; then
    printf '%s\n' "$args"
    return 0
  fi

  python3 - "$args" <<'PY'
import json
import re
import sys

raw = sys.argv[1]
try:
  data = json.loads(raw)
except Exception:
  print(raw)
  raise SystemExit(0)

if isinstance(data, dict):
  query = data.get("query")
  if isinstance(query, str):
    data["query"] = re.sub(r"\btitle\b\s*=", "name =", query)
  if "title" in data and "name" not in data:
    data["name"] = data.pop("title")

print(json.dumps(data, separators=(",", ":")))
PY
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

TOOL_NAME="${1:-}"
ARGS_JSON="${2:-\{\}}"
OUTPUT_MODE="${3:-json}"

if [[ -z "$TOOL_NAME" ]]; then
  usage >&2
  exit 2
fi

if ! command -v mcporter >/dev/null 2>&1; then
  echo "gog-agentic-call: mcporter not found in PATH" >&2
  exit 1
fi

if [[ -n "${MCPORTER_CONFIG:-}" ]]; then
  MCPORTER_CONFIG="$MCPORTER_CONFIG"
else
  WORKSPACE_DIR_RESOLVED="$(resolve_workspace_dir)" || exit 1
  MCPORTER_CONFIG="$WORKSPACE_DIR_RESOLVED/config/mcporter.json"
fi
if [[ ! -f "$MCPORTER_CONFIG" ]]; then
  echo "gog-agentic-call: config not found at $MCPORTER_CONFIG" >&2
  exit 1
fi

NORMALIZED_TOOL="$(normalize_tool_name "$TOOL_NAME")"
ARGS_JSON="$(normalize_drive_search_args "$NORMALIZED_TOOL" "$ARGS_JSON")"

exec mcporter --config "$MCPORTER_CONFIG" call "gog-agentic.${NORMALIZED_TOOL}" --args "$ARGS_JSON" --output "$OUTPUT_MODE"
