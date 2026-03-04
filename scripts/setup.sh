#!/usr/bin/env bash
# setup.sh — simple golden-path setup for gogcli-enhanced
# Purpose: fast bootstrap for common case (single user/account)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

BOLD="\033[1m"
GREEN="\033[32m"
YELLOW="\033[33m"
RED="\033[31m"
RESET="\033[0m"

log() { echo -e "${GREEN}==>${RESET} $*"; }
warn() { echo -e "${YELLOW}Warning:${RESET} $*"; }
err() { echo -e "${RED}Error:${RESET} $*"; }

ACCOUNT="${GOG_ACCOUNT:-}"
CLIENT="${GOG_CLIENT:-}"
CREDENTIALS_PATH=""
MANUAL=0
NO_INPUT=0
CREDENTIALS_STDIN=0

usage() {
  cat <<EOF
Usage: ./scripts/setup.sh [options]

Simple setup (golden path):
  1) build/install gog
  2) ensure OAuth client credentials
  3) authorize account
  4) set default account
  5) validate Drive access

Options:
  --account <email>            Account email to authorize/use
  --credentials <path>         Path to OAuth client JSON to import
  --credentials-stdin          Read full OAuth JSON from stdin (non-interactive friendly)
  --client <name>              Optional gog client profile
  --manual                     Force manual auth flow
  --no-input                   Non-interactive mode (fails if prompts needed)
  -h, --help                   Show this help

Advanced/repair workflow moved to:
  ./scripts/setup-doctor.sh
EOF
}

require_cmd() { command -v "$1" >/dev/null 2>&1 || { err "Missing command: $1"; exit 1; }; }

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --account) ACCOUNT="${2:-}"; shift 2 ;;
      --credentials) CREDENTIALS_PATH="${2:-}"; shift 2 ;;
      --client) CLIENT="${2:-}"; shift 2 ;;
      --credentials-stdin) CREDENTIALS_STDIN=1; shift ;;
      --manual) MANUAL=1; shift ;;
      --no-input) NO_INPUT=1; shift ;;
      -h|--help) usage; exit 0 ;;
      *) err "Unknown option: $1"; usage; exit 2 ;;
    esac
  done
}

gog_cmd() {
  if [[ -n "$CLIENT" ]]; then
    gog --client "$CLIENT" "$@"
  else
    gog "$@"
  fi
}

build_and_install() {
  log "Building/installing gog..."
  ./scripts/install.sh
  if [[ -x "$ROOT_DIR/bin/gog" ]]; then
    export PATH="$ROOT_DIR/bin:$HOME/.local/bin:$PATH"
  else
    export PATH="$HOME/.local/bin:$PATH"
  fi
  require_cmd gog
  log "Using: $(command -v gog)"
  gog --version || true
}

import_pasted_credentials() {
  local pasted_json tmp_creds
  pasted_json="$(cat)"
  [[ -n "$pasted_json" ]] || { err "No JSON provided."; exit 1; }

  if ! printf '%s' "$pasted_json" | jq -e . >/dev/null 2>&1; then
    err "Provided content is not valid JSON."
    exit 1
  fi

  tmp_creds="$(mktemp /tmp/gog-credentials-XXXXXX.json)"
  printf '%s' "$pasted_json" > "$tmp_creds"
  gog_cmd auth credentials "$tmp_creds"
  rm -f "$tmp_creds"
}

ensure_credentials() {
  if gog_cmd auth credentials list >/dev/null 2>&1; then
    local count
    count="$(gog_cmd auth credentials list 2>/dev/null | wc -l | tr -d ' ')"
    if [[ "$count" -gt 0 ]]; then
      log "OAuth client credentials already present."
      return 0
    fi
  fi

  if [[ "$CREDENTIALS_STDIN" -eq 1 ]]; then
    log "Reading OAuth credentials JSON from stdin..."
    import_pasted_credentials
    return 0
  fi

  if [[ -n "$CREDENTIALS_PATH" ]]; then
    [[ -f "$CREDENTIALS_PATH" ]] || { err "Credentials file not found: $CREDENTIALS_PATH"; exit 1; }
    log "Importing OAuth credentials from: $CREDENTIALS_PATH"
    gog_cmd auth credentials "$CREDENTIALS_PATH"
    return 0
  fi

  if [[ "$NO_INPUT" -eq 1 ]]; then
    err "No credentials configured. Provide --credentials <path> or --credentials-stdin in --no-input mode."
    exit 1
  fi

  echo
  echo -e "${BOLD}OAuth credentials required${RESET}"
  echo "Choose input mode:"
  echo "  1) Path to Desktop OAuth JSON file"
  echo "  2) Paste full raw OAuth JSON"
  read -r -p "Select [1/2] (default 1): " mode
  mode="${mode:-1}"

  case "$mode" in
    1)
      read -r -p "Credentials JSON path: " CREDENTIALS_PATH
      [[ -f "$CREDENTIALS_PATH" ]] || { err "Credentials file not found: $CREDENTIALS_PATH"; exit 1; }
      gog_cmd auth credentials "$CREDENTIALS_PATH"
      return 0
      ;;
    2)
      echo "Paste full OAuth client JSON, then press Ctrl-D:"
      import_pasted_credentials
      return 0
      ;;
    *)
      err "Invalid selection: $mode (choose 1 or 2)"
      exit 1
      ;;
  esac
}

pick_account_if_needed() {
  if [[ -n "$ACCOUNT" ]]; then
    return 0
  fi

  # Try existing account from auth list
  local first
  first="$(gog_cmd auth list 2>/dev/null | awk 'NR==1{print $1}')"
  if [[ -n "$first" ]]; then
    ACCOUNT="$first"
    log "Using existing account: $ACCOUNT"
    return 0
  fi

  if [[ "$NO_INPUT" -eq 1 ]]; then
    err "No account found. Provide --account <email> in --no-input mode."
    exit 1
  fi

  read -r -p "Google account email to authorize: " ACCOUNT
  [[ -n "$ACCOUNT" ]] || { err "Account email is required"; exit 1; }
}

authorize_account() {
  # If already present, skip add
  if gog_cmd auth list 2>/dev/null | awk '{print $1}' | grep -Fxq "$ACCOUNT"; then
    log "Account already authorized: $ACCOUNT"
  else
    log "Authorizing account: $ACCOUNT"
    if [[ "$MANUAL" -eq 1 ]]; then
      gog_cmd auth add "$ACCOUNT" --services user --manual
    else
      # Auto flow first; fallback to manual if it fails
      if ! gog_cmd auth add "$ACCOUNT" --services user; then
        warn "Auto OAuth flow failed; falling back to manual flow..."
        gog_cmd auth add "$ACCOUNT" --services user --manual
      fi
    fi
    
    # For manual auth, we need to handle the redirect URL
    if [[ "$MANUAL" -eq 1 ]]; then
      echo
      echo "${BOLD}Manual OAuth flow${RESET}"
      echo "1. Open this URL in your browser:"
      echo "   $(gog_cmd auth add "$ACCOUNT" --services user --manual --step 1 2>/dev/null | grep -o 'https://[^ ]*')"
      echo "2. After authenticating, copy the FULL redirect URL from your browser"
      echo "   (e.g., https://localhost:8080/oauth2/callback?code=...&state=...)"
      echo "3. Paste it here when ready:"
      read -r -p "Paste redirect URL: " REDIRECT_URL
      [[ -n "$REDIRECT_URL" ]] || { err "No redirect URL provided."; exit 1; }
      
      # Complete manual auth
      gog_cmd auth add "$ACCOUNT" --services user --manual --step 2 --auth-url "$REDIRECT_URL"
    fi
  fi

  # Set alias 'default' for easier non-interactive operations
  gog_cmd auth alias set default "$ACCOUNT" >/dev/null 2>&1 || true
}

validate_access() {
  log "Validating auth + Drive access..."
  gog_cmd --account "$ACCOUNT" --no-input auth status >/dev/null
  gog_cmd drive ls --account "$ACCOUNT" --json --results-only >/dev/null
  log "Validation successful for account: $ACCOUNT"
}

print_next_steps() {
  echo
  echo -e "${BOLD}Setup complete.${RESET}"
  echo "Account: $ACCOUNT"
  [[ -n "$CLIENT" ]] && echo "Client profile: $CLIENT"
  echo
  echo "Try commands:"
  echo "  gog drive ls --account $ACCOUNT"
  echo "  gog drive upload /path/to/file --parent <folderId> --account $ACCOUNT"
  echo
  echo "Advanced/repair workflows: ./scripts/setup-doctor.sh"
}

main() {
  parse_args "$@"
  require_cmd bash
  require_cmd make
  build_and_install
  ensure_credentials
  pick_account_if_needed
  authorize_account
  validate_access
  print_next_steps
}

main "$@"
