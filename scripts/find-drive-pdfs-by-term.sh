#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: find-drive-pdfs-by-term.sh [options]

Discover folders by folder-name term, then list PDFs in each and print page counts.

Options:
  --term TEXT            Folder-name search term (required)
  --workspace-dir PATH   OpenClaw workspace path containing config/mcporter.json
  --max-results N        Max results per MCP page (default: 25)
  --json                 Emit newline-delimited JSON records
  -h, --help             Show this help

Examples:
  find-drive-pdfs-by-term.sh --term "tax"
  find-drive-pdfs-by-term.sh --term "tax" --max-results 50 --workspace-dir /path/to/workspace
USAGE
}

if ! command -v jq >/dev/null 2>&1; then
  echo "find-drive-pdfs-by-term.sh: jq is required" >&2
  exit 1
fi
if ! command -v gog-agentic-call >/dev/null 2>&1; then
  echo "find-drive-pdfs-by-term.sh: gog-agentic-call not found in PATH" >&2
  exit 1
fi

TERM=""
MAX_RESULTS=25
OUTPUT_JSON=0
WORKSPACE_DIR_OVERRIDE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --term)
      TERM="$2"
      shift 2
      ;;
    --workspace-dir)
      WORKSPACE_DIR_OVERRIDE="$2"
      shift 2
      ;;
    --max-results)
      MAX_RESULTS="$2"
      shift 2
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

if [[ -z "$TERM" ]]; then
  echo "Missing required --term" >&2
  usage
  exit 2
fi

if [[ -n "$WORKSPACE_DIR_OVERRIDE" ]]; then
  export WORKSPACE_DIR="$WORKSPACE_DIR_OVERRIDE"
fi

if [[ "$OUTPUT_JSON" -eq 1 ]]; then
  print_record() {
    local folder_id="$1"
    local folder_name="$2"
    local file_id="$3"
    local file_name="$4"
    local page_count="$5"
    cat <<EOF_JSON
{ "folderId": "$folder_id", "folderName": "$folder_name", "fileId": "$file_id", "fileName": "$file_name", "pageCount": "$page_count" }
EOF_JSON
  }
else
  print_header() {
    printf '%-40s %-28s %-55s %-28s %s\n' "FOLDER" "FOLDER_ID" "FILE" "FILE_ID" "PAGES"
    printf '%-40.40s %-28s %-55.55s %-28s %s\n' "------" "--------" "----" "-------" "-----"
  }
  print_record() {
    local folder_id="$1"
    local folder_name="$2"
    local file_id="$3"
    local file_name="$4"
    local page_count="$5"
    printf '%-40.40s %-28s %-55.55s %-28s %s\n' "$folder_name" "$folder_id" "$file_name" "$file_id" "$page_count"
  }
  print_header
fi

call_mcp() {
  local tool="$1"
  local args_json="$2"
  gog-agentic-call "$tool" "$args_json"
}

require_ok() {
  local tool="$1"
  local args_json="$2"
  local resp
  resp="$(call_mcp "$tool" "$args_json")"
  if [[ "$(jq -r '.ok // false' <<<"$resp")" != "true" ]]; then
    echo "${tool} failed: $(jq -r '.error.message // "unknown error"' <<<"$resp")" >&2
    exit 1
  fi
  printf '%s\n' "$resp"
}

escape_drive_literal() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

escaped_term="$(escape_drive_literal "$TERM")"
folder_query="mimeType = 'application/vnd.google-apps.folder' and name contains \"$escaped_term\""

page=""
while :; do
  if [[ -n "$page" ]]; then
    folder_args="$(jq -nc --arg query "$folder_query" --argjson maxResults "$MAX_RESULTS" --arg page "$page" '{query:$query, rawQuery:true, maxResults:$maxResults, page:$page}')"
  else
    folder_args="$(jq -nc --arg query "$folder_query" --argjson maxResults "$MAX_RESULTS" '{query:$query, rawQuery:true, maxResults:$maxResults}')"
  fi

  folder_resp="$(require_ok drive.searchFiles "$folder_args")"
  while IFS=$'\t' read -r folder_id folder_name; do
    if [[ -z "$folder_id" ]]; then
      continue
    fi

    file_page=""
    while :; do
      if [[ -n "$file_page" ]]; then
        file_args="$(jq -nc --arg parent "$folder_id" --arg query "mimeType = 'application/pdf'" --argjson maxResults "$MAX_RESULTS" --arg page "$file_page" '{parentId:$parent, query:$query, maxResults:$maxResults, page:$page}')"
      else
        file_args="$(jq -nc --arg parent "$folder_id" --arg query "mimeType = 'application/pdf'" --argjson maxResults "$MAX_RESULTS" '{parentId:$parent, query:$query, maxResults:$maxResults}')"
      fi

      file_resp="$(require_ok drive.listFiles "$file_args")"
      while IFS=$'\t' read -r file_id file_name; do
        if [[ -z "$file_id" ]]; then
          continue
        fi
        get_file_args="$(jq -nc --arg fileId "$file_id" '{fileId:$fileId, pageCount:true}')"
        file_resp_detail="$(require_ok drive.getFile "$get_file_args")"
        page_count="$(jq -r '.result.pageCount // .result.pdfMetadata.pages // "n/a"' <<<"$file_resp_detail")"
        print_record "$folder_id" "$folder_name" "$file_id" "$file_name" "$page_count"
      done < <(jq -r '.result.files[]? | [.id, (.name // "")] | @tsv' <<<"$file_resp")

      file_next_page="$(jq -r '.result.nextPageToken // empty' <<<"$file_resp")"
      if [[ -z "$file_next_page" ]]; then
        break
      fi
      file_page="$file_next_page"
    done
  done < <(jq -r '.result.files[]? | [.id, (.name // "")] | @tsv' <<<"$folder_resp")

  next_page="$(jq -r '.result.nextPageToken // empty' <<<"$folder_resp")"
  if [[ -z "$next_page" ]]; then
    break
  fi
  page="$next_page"
done
