#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  drive-folder-cache.sh lookup --name "<folder name>" [--contains] [--id-only] [--json] [--cache-file PATH]
  drive-folder-cache.sh set --name "<folder name>" --id "<folder id>" [--cache-file PATH]
  drive-folder-cache.sh --help

Description:
  Lightweight local cache for Google Drive folder IDs.
  Store and reuse known folder IDs to avoid repeated folder discovery.

Examples:
  drive-folder-cache.sh set --name "Appraisal home valuation" --id "10Ll..."
  drive-folder-cache.sh lookup --name "Appraisal home valuation" --id-only
  drive-folder-cache.sh lookup --name "Appraisal" --contains --json
EOF
}

if ! command -v jq >/dev/null 2>&1; then
  echo "drive-folder-cache.sh: jq is required" >&2
  exit 1
fi

command_name=""
name=""
folder_id=""
cache_file="${XDG_CACHE_HOME:-$HOME/.cache}/gogcli/drive-folder-cache.json"
contains=0
id_only=0
json_output=0

normalize_name() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]'
}

ensure_cache() {
  local dir
  dir="$(dirname "$cache_file")"
  mkdir -p "$dir"
  if [[ ! -f "$cache_file" ]]; then
    cat <<'JSON' > "$cache_file"
{"entries":[]}
JSON
  fi
}

lookup_cache() {
  local query_lower="$1"
  local filter_expr
  if [[ "$contains" -eq 1 ]]; then
    filter_expr='map(select((.nameLower | contains($query_lower) or .name | ascii_downcase | contains($query_lower)) ) )'
  else
    filter_expr='map(select(.nameLower == $query_lower))'
  fi
  jq -c --arg query_lower "$query_lower" \
    "(.entries | $filter_expr | sort_by(.updatedAt) | reverse)" \
    "$cache_file"
}

write_cache() {
  local name_arg="$1"
  local id_arg="$2"
  local normalized
  normalized="$(normalize_name "$name_arg")"
  local now
  now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  local tmp_file
  tmp_file="$(mktemp)"

  jq --arg name "$name_arg" \
     --arg name_lower "$normalized" \
     --arg id "$id_arg" \
     --arg updated "$now" \
     '(.entries | map(select(not (.nameLower == $name_lower and .id == $id))) ) as $filtered
      | .entries = ($filtered + [{ "name": $name, "nameLower": $name_lower, "id": $id, "updatedAt": $updated }])
      ' \
    "$cache_file" > "$tmp_file"

  mv "$tmp_file" "$cache_file"
  printf '%s\n' "$name_arg -> $id_arg"
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -lt 1 ]]; then
  usage >&2
  exit 2
fi

command_name="$1"
shift

while [[ $# -gt 0 ]]; do
  case "$1" in
    --name)
      name="$2"
      shift 2
      ;;
    --id)
      folder_id="$2"
      shift 2
      ;;
    --cache-file)
      cache_file="$2"
      shift 2
      ;;
    --contains)
      contains=1
      shift
      ;;
    --id-only)
      id_only=1
      shift
      ;;
    --json)
      json_output=1
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

if [[ -z "$name" ]]; then
  echo "--name is required" >&2
  exit 2
fi

ensure_cache

case "$command_name" in
  lookup)
    query="$(normalize_name "$name")"
    matches="$(lookup_cache "$query")"
    count="$(jq 'length' <<<"$matches")"
    if [[ "$count" -eq 0 ]]; then
      exit 1
    fi

    if [[ "$id_only" -eq 1 && "$contains" -eq 0 ]]; then
      jq -r '.[0].id // empty' <<<"$matches"
      exit 0
    fi

    if [[ "$json_output" -eq 1 ]]; then
      printf '%s\n' "$matches"
    else
      printf '%s\n' "$matches" | jq -r '.[] | "- \(.name) (\(.id)) [updated: \(.updatedAt)]"'
    fi
    ;;
  set)
    if [[ -z "$folder_id" ]]; then
      echo "--id is required for set" >&2
      exit 2
    fi
    write_cache "$name" "$folder_id"
    ;;
  *)
    echo "Unknown command: $command_name" >&2
    usage
    exit 2
    ;;
esac
