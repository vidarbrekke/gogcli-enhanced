#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: find-drive-folder-files.sh [options]

Find a Google Drive folder by name and list its files.

Options:
  --folder-name TEXT     Folder-name search term (required; exact first, then contains fallback)
  --workspace-dir PATH   OpenClaw workspace path containing config/mcporter.json
  --cache-file PATH      Optional custom cache file for local folder ID lookups
  --max-age-days N       Cache TTL in days (default: from DRIVE_FOLDER_CACHE_MAX_AGE_DAYS, 30)
  --no-cache             Disable cache lookup/write for folder IDs
  --exact-only           Skip fallback contains search
  --max-results N        Max results per MCP page (default: 100)
  --json                 Emit JSON output
  -h, --help             Show this help

Examples:
  find-drive-folder-files.sh --folder-name "Appraisal home valuation" --workspace-dir /path/to/openclaw/workspace
USAGE
}

if ! command -v jq >/dev/null 2>&1; then
  echo "find-drive-folder-files.sh: jq is required" >&2
  exit 1
fi
if ! command -v gog-agentic-call >/dev/null 2>&1; then
  echo "find-drive-folder-files.sh: gog-agentic-call not found in PATH" >&2
  exit 1
fi

FOLDER_NAME=""
WORKSPACE_DIR_OVERRIDE=""
MAX_RESULTS=100
OUTPUT_JSON=0
USE_CACHE=1
EXACT_ONLY=0
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
CACHE_SCRIPT="${SCRIPT_DIR}/drive-folder-cache.sh"
CACHE_FILE="${XDG_CACHE_HOME:-$HOME/.cache}/gogcli/drive-folder-cache.json"
CACHE_MAX_AGE_DAYS="${DRIVE_FOLDER_CACHE_MAX_AGE_DAYS:-30}"

usage_error() {
  echo "$1" >&2
  usage
  exit 2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --folder-name)
      [[ $# -lt 2 ]] && usage_error "Missing value for --folder-name"
      FOLDER_NAME="$2"
      shift 2
      ;;
    --workspace-dir)
      [[ $# -lt 2 ]] && usage_error "Missing value for --workspace-dir"
      WORKSPACE_DIR_OVERRIDE="$2"
      shift 2
      ;;
    --cache-file)
      [[ $# -lt 2 ]] && usage_error "Missing value for --cache-file"
      CACHE_FILE="$2"
      shift 2
      ;;
    --max-age-days)
      [[ $# -lt 2 ]] && usage_error "Missing value for --max-age-days"
      CACHE_MAX_AGE_DAYS="$2"
      shift 2
      ;;
    --max-results)
      [[ $# -lt 2 ]] && usage_error "Missing value for --max-results"
      MAX_RESULTS="$2"
      shift 2
      ;;
    --no-cache)
      USE_CACHE=0
      shift
      ;;
    --exact-only)
      EXACT_ONLY=1
      shift
      ;;
    --json)
      OUTPUT_JSON=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 2
      ;;
  esac
done

if [[ -z "$FOLDER_NAME" ]]; then
  usage_error "Missing required --folder-name"
fi

if [[ -n "$WORKSPACE_DIR_OVERRIDE" ]]; then
  export WORKSPACE_DIR="$WORKSPACE_DIR_OVERRIDE"
fi

escape_drive_literal() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

call_mcp() {
  local tool="$1"
  local args_json="$2"
  local resp
  if ! resp="$(gog-agentic-call "$tool" "$args_json")"; then
    printf '%s\n' '{"ok":false,"error":{"code":"command_failed","message":"gog-agentic-call exited non-zero"}}'
    return 1
  fi
  printf '%s\n' "$resp"
}

response_ok() {
  local resp="$1"
  local ok
  ok="$(jq -r '.ok // false' <<<"$resp" 2>/dev/null || true)"
  [[ "$ok" == "true" ]]
}

require_ok() {
  local tool="$1"
  local args_json="$2"
  local resp
  resp="$(call_mcp "$tool" "$args_json")" || true
  if ! response_ok "$resp"; then
    echo "${tool} failed: $(jq -r '.error.message // "unknown error"' <<<"$resp")" >&2
    exit 1
  fi
  printf '%s\n' "$resp"
}

cache_lookup() {
  local lookup_output
  local folder_id
  local folder_name

  if [[ "$USE_CACHE" -ne 1 ]]; then
    return 1
  fi
  if [[ ! -x "$CACHE_SCRIPT" ]]; then
    return 1
  fi

  if ! lookup_output="$("$CACHE_SCRIPT" lookup --name "$FOLDER_NAME" --json --cache-file "$CACHE_FILE" --max-age-days "$CACHE_MAX_AGE_DAYS" 2>/dev/null)"; then
    return 1
  fi
  folder_id="$(jq -r '.[0].id // empty' <<<"$lookup_output" 2>/dev/null || true)"
  folder_name="$(jq -r '.[0].name // empty' <<<"$lookup_output" 2>/dev/null || true)"
  if [[ -z "$folder_id" ]]; then
    return 1
  fi
  echo "$folder_id"
  echo "$folder_name"
  return 0
}

cache_store() {
  local folder_id="$1"
  local folder_name="$2"
  if [[ "$USE_CACHE" -ne 1 ]]; then
    return 0
  fi
  if [[ ! -x "$CACHE_SCRIPT" ]]; then
    return 0
  fi
  "$CACHE_SCRIPT" set --name "$folder_name" --id "$folder_id" --cache-file "$CACHE_FILE" --max-age-days "$CACHE_MAX_AGE_DAYS" >/dev/null || true
}

find_folder_by_query() {
  local query="$1"
  local args
  args="$(jq -nc --arg query "$query" --argjson maxResults "$MAX_RESULTS" '{query:$query, rawQuery:true, maxResults:$maxResults}')"
  require_ok drive.searchFiles "$args"
}

resolve_folder() {
  local cache_id
  local cache_name
  local escaped
  local exact_query
  local contains_query
  local exact_resp
  local contains_resp
  local folder_obj

  if cache_id="$(cache_lookup)"; then
    FOLDER_ID="$(printf '%s\n' "$cache_id" | sed -n '1p')"
    FOLDER_NAME_RESOLVED="$(printf '%s\n' "$cache_id" | sed -n '2p')"
    if [[ -z "$FOLDER_NAME_RESOLVED" ]]; then
      FOLDER_NAME_RESOLVED="$FOLDER_NAME"
    fi
    SOURCE="cache"
    return 0
  fi

  escaped="$(escape_drive_literal "$FOLDER_NAME")"
  exact_query="mimeType = 'application/vnd.google-apps.folder' and name = \"$escaped\""
  exact_resp="$(find_folder_by_query "$exact_query")"
  folder_obj="$(jq -c '(.result.files // .files // []) | .[0] // empty' <<<"$exact_resp")"
  if [[ -n "$folder_obj" ]]; then
    FOLDER_ID="$(jq -r '.id // ""' <<<"$folder_obj")"
    FOLDER_NAME_RESOLVED="$(jq -r '.name // ""' <<<"$folder_obj")"
    if [[ -n "$FOLDER_ID" ]]; then
      SOURCE="exact-search"
      cache_store "$FOLDER_ID" "$FOLDER_NAME_RESOLVED"
      return 0
    fi
  fi

  if [[ "$EXACT_ONLY" -eq 1 ]]; then
    return 1
  fi

  contains_query="mimeType = 'application/vnd.google-apps.folder' and name contains \"$escaped\""
  contains_resp="$(find_folder_by_query "$contains_query")"
  folder_obj="$(jq -c '(.result.files // .files // []) | .[0] // empty' <<<"$contains_resp")"
  if [[ -n "$folder_obj" ]]; then
    FOLDER_ID="$(jq -r '.id // ""' <<<"$folder_obj")"
    FOLDER_NAME_RESOLVED="$(jq -r '.name // ""' <<<"$folder_obj")"
    if [[ -n "$FOLDER_ID" ]]; then
      SOURCE="contains-search"
      cache_store "$FOLDER_ID" "$FOLDER_NAME_RESOLVED"
      return 0
    fi
  fi

  return 1
}

list_folder_files() {
  local page=""
  local file_resp
  local file_args
  local file_page
  local files_tmp

  files_tmp="$(mktemp)"
  while :; do
    if [[ -n "$page" ]]; then
      file_args="$(jq -nc --arg parent "$FOLDER_ID" --argjson maxResults "$MAX_RESULTS" --arg page "$page" '{parentId:$parent, maxResults:$maxResults, page:$page}')"
    else
      file_args="$(jq -nc --arg parent "$FOLDER_ID" --argjson maxResults "$MAX_RESULTS" '{parentId:$parent, maxResults:$maxResults}')"
    fi

    file_resp="$(require_ok drive.listFiles "$file_args")"
    jq -c '.result.files // .files // [] | .[]? | {name:(.name // ""), id:(.id // ""), mimeType:(.mimeType // "")}' <<<"$file_resp" >> "$files_tmp"

    file_page="$(jq -r '.result.nextPageToken // .nextPageToken // empty' <<<"$file_resp")"
    if [[ -z "$file_page" ]]; then
      break
    fi
    page="$file_page"
  done
  printf '%s\n' "$files_tmp"
}

FOLDER_ID=""
FOLDER_NAME_RESOLVED=""
SOURCE=""
trap 'rm -f "${FILES_TMP:-}"' EXIT

if ! resolve_folder; then
  echo "No matching folder found for: $FOLDER_NAME" >&2
  exit 1
fi

FILES_TMP="$(list_folder_files)"
FILE_COUNT="$(jq -s 'length' "$FILES_TMP")"

if [[ "$OUTPUT_JSON" -eq 1 ]]; then
  jq -nc \
    --arg query "$FOLDER_NAME" \
    --arg folderName "$FOLDER_NAME_RESOLVED" \
    --arg folderId "$FOLDER_ID" \
    --arg source "$SOURCE" \
    --argjson files "$(jq -s '.' "$FILES_TMP")" \
    '{
      query: $query,
      cacheSource: $source,
      folder: {name: $folderName, id: $folderId},
      fileCount: ($files | length),
      files: $files
    }'
else
  echo "cache-source: $SOURCE"
  echo "folder-id: $FOLDER_ID"
  echo "folder-name: $FOLDER_NAME_RESOLVED"
  echo "file-count: $FILE_COUNT"
  if [[ "$FILE_COUNT" -eq 0 ]]; then
    echo "No files found."
    exit 0
  fi
  while IFS= read -r file_obj; do
    printf '%s\n' "$file_obj" | jq -r '"  \(.name) | \(.id) | \(.mimeType // "n/a")"'
  done < "$FILES_TMP"
fi

