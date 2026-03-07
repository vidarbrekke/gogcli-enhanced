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
  if [[ -n "${WORKSPACE_DIR:-}" ]]; then
    cd "$WORKSPACE_DIR" >/dev/null 2>&1 && pwd
  elif [[ "$ROOT_DIR" == *"/repositories/"* ]]; then
    printf '%s\n' "${ROOT_DIR%/repositories/*}"
  else
    printf '%s\n' "${OPENCLAW_WORKSPACE:-$ROOT_DIR}"
  fi
}

normalize_tool_name() {
  local raw="$1"
  raw="${raw#gog-agentic.}"
  raw="${raw#gog-agentic/}"
  raw="${raw#gog-agentic:}"
  raw="${raw//./_}"
  printf '%s\n' "$raw"
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

WORKSPACE_DIR_RESOLVED="$(resolve_workspace_dir)"
MCPORTER_CONFIG="$WORKSPACE_DIR_RESOLVED/config/mcporter.json"
if [[ ! -f "$MCPORTER_CONFIG" ]]; then
  echo "gog-agentic-call: config not found at $MCPORTER_CONFIG" >&2
  exit 1
fi

NORMALIZED_TOOL="$(normalize_tool_name "$TOOL_NAME")"

exec mcporter --config "$MCPORTER_CONFIG" call "gog-agentic.${NORMALIZED_TOOL}" --args "$ARGS_JSON" --output "$OUTPUT_MODE"
