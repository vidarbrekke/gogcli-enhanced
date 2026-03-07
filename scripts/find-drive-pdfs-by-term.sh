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

print_record_json() {
  local folder_id="$1"
  local folder_name="$2"
  local file_id="$3"
  local file_name="$4"
  local page_count_json="$5"
  local page_count_display="$6"
  local file_mime_type="$7"
  local pdf_metadata="$8"
  local pdf_metadata_envelope="$9"
  local file_lookup_ok="${10}"
  local file_error_code="${11}"
  local file_error_message="${12}"

  jq -nc \
    --arg folderId "$folder_id" \
    --arg folderName "$folder_name" \
    --arg fileId "$file_id" \
    --arg fileName "$file_name" \
    --argjson pageCount "$page_count_json" \
    --arg pageCountDisplay "$page_count_display" \
    --arg fileMimeType "$file_mime_type" \
    --argjson pdfMetadata "$pdf_metadata" \
    --argjson pdfMetadataEnvelope "$pdf_metadata_envelope" \
    --argjson fileLookupOk "$file_lookup_ok" \
    --arg errorCode "$file_error_code" \
    --arg errorMessage "$file_error_message" \
    '{folderId:$folderId, folderName:$folderName, fileId:$fileId, fileName:$fileName, fileMimeType:$fileMimeType, pageCount:$pageCount, pageCountDisplay:$pageCountDisplay, pdfMetadata:$pdfMetadata, pdfMetadataEnvelope:$pdfMetadataEnvelope, fileLookup:{ok:$fileLookupOk, error:(if $errorCode == "" then null else {code:$errorCode, message:$errorMessage} end)}}'
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
  while IFS= read -r folder_json; do
    folder_id="$(jq -r '.id // ""' <<<"$folder_json")"
    folder_name="$(jq -r '.name // ""' <<<"$folder_json")"
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
      while IFS= read -r file_obj; do
        file_id="$(jq -r '.id // ""' <<<"$file_obj")"
        file_name="$(jq -r '.name // ""' <<<"$file_obj")"
        file_mime_type="$(jq -r '.mimeType // ""' <<<"$file_obj")"
        if [[ -z "$file_id" ]]; then
          continue
        fi
        get_file_args="$(jq -nc --arg fileId "$file_id" '{fileId:$fileId, pageCount:true}')"
        file_resp_detail="$(call_mcp drive.getFile "$get_file_args")" || true
        file_lookup_ok="false"
        file_error_code=""
        file_error_message=""
        pdf_metadata="{}"
        pdf_metadata_envelope="{}"
        if response_ok "$file_resp_detail"; then
          file_lookup_ok="true"
          page_count="$(jq -r '.result.pageCount // .result.pdfMetadata.pages // empty' <<<"$file_resp_detail" 2>/dev/null || true)"
          pdf_metadata="$(jq -c '.result.pdfMetadata // {}' <<<"$file_resp_detail" 2>/dev/null || echo '{}')"
          pdf_metadata_envelope="$(jq -c '.result.pdfMetadataEnvelope // {}' <<<"$file_resp_detail" 2>/dev/null || echo '{}')"
        else
          file_error_code="$(jq -r '.error.code // "command_failed"' <<<"$file_resp_detail" 2>/dev/null || true)"
          file_error_message="$(jq -r '.error.message // "unknown_error"' <<<"$file_resp_detail" 2>/dev/null || true)"
          page_count=""
        fi

        if [[ -n "$page_count" ]]; then
          page_count_json="$page_count"
          page_count_display="$page_count"
        else
          page_count_json="null"
          page_count_display="n/a"
        fi

        if [[ "$OUTPUT_JSON" -eq 1 ]]; then
          print_record_json "$folder_id" "$folder_name" "$file_id" "$file_name" "$page_count_json" "$page_count_display" "$file_mime_type" "$pdf_metadata" "$pdf_metadata_envelope" "$file_lookup_ok" "$file_error_code" "$file_error_message"
        else
          print_record "$folder_id" "$folder_name" "$file_id" "$file_name" "$page_count_display"
        fi
      done < <(jq -c '.result.files[]?' <<<"$file_resp")

      file_next_page="$(jq -r '.result.nextPageToken // empty' <<<"$file_resp")"
      if [[ -z "$file_next_page" ]]; then
        break
      fi
      file_page="$file_next_page"
    done
  done < <(jq -c '.result.files[]?' <<<"$folder_resp")

  next_page="$(jq -r '.result.nextPageToken // empty' <<<"$folder_resp")"
  if [[ -z "$next_page" ]]; then
    break
  fi
  page="$next_page"
done
