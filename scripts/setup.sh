#!/usr/bin/env bash
# setup.sh — simple golden-path setup for gogcli-enhanced
# Purpose: fast bootstrap for common case (single user/account)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/scripts/lib/gws-auth-bridge.sh"

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
NO_INPUT=0
CLI_ONLY=0
GWS_EXPORT_JSON=""

usage() {
  cat <<EOF
Usage: ./scripts/setup.sh [options]

Simple setup (golden path):
  1) build/install gog
  2) require official gws auth setup/login
  3) import that auth into gog automatically
  4) set default account
  5) validate Drive access

Options:
  --account <email>            Account email to authorize/use
  --client <name>              Optional gog client profile
  --no-input                   Non-interactive mode (fails if prompts needed)
  --cli-only                   CLI/cron-only mode (no MCP registration; use for backup scripts)
  -h, --help                   Show this help

Advanced/repair workflow moved to:
  ./scripts/setup-doctor.sh
EOF
}

cleanup() {
  if [[ -n "$GWS_EXPORT_JSON" && -f "$GWS_EXPORT_JSON" ]]; then
    rm -f "$GWS_EXPORT_JSON"
  fi
}

trap cleanup EXIT

require_cmd() { command -v "$1" >/dev/null 2>&1 || { err "Missing command: $1"; exit 1; }; }

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --account) ACCOUNT="${2:-}"; shift 2 ;;
      --client) CLIENT="${2:-}"; shift 2 ;;
      --no-input) NO_INPUT=1; shift ;;
      --cli-only) CLI_ONLY=1; shift ;;
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

ensure_gws_export() {
  if [[ -n "$GWS_EXPORT_JSON" && -f "$GWS_EXPORT_JSON" ]]; then
    return 0
  fi

  GWS_EXPORT_JSON="$(mktemp /tmp/gws-export-XXXXXX.json)"
  if ! gws_bootstrap_export_file "$GWS_EXPORT_JSON" "$NO_INPUT"; then
    rm -f "$GWS_EXPORT_JSON"
    GWS_EXPORT_JSON=""
    return 1
  fi

  return 0
}

detect_account_from_gws() {
  local detected=""
  if ! ensure_gws_export; then
    return 1
  fi

  detected="$(gws_guess_email_from_export "$GWS_EXPORT_JSON" 2>/dev/null || true)"
  detected="$(printf '%s' "$detected" | tr -d '\r' | tr -d '\n' | xargs 2>/dev/null || true)"
  [[ -n "$detected" ]] || return 1
  ACCOUNT="$detected"
  log "Detected Google account from official gws auth: $ACCOUNT"
}

bootstrap_gog_credentials_from_gws() {
  local creds_path
  creds_path="$(python3 - <<'PY'
from pathlib import Path
import os
base = Path(os.path.expanduser(os.environ.get("XDG_CONFIG_HOME", "~/.config"))).expanduser()
print(base / "gogcli" / "credentials.json")
PY
)"

  if ! ensure_gws_export; then
    return 1
  fi

  if ! gws_write_gog_credentials_from_export "$GWS_EXPORT_JSON" "$creds_path"; then
    return 1
  fi

  log "Imported OAuth client credentials from official gws auth."
}

bootstrap_gog_account_from_gws() {
  local import_json detected_email target_email
  if ! ensure_gws_export; then
    return 1
  fi

  detected_email="$(gws_guess_email_from_export "$GWS_EXPORT_JSON" 2>/dev/null || true)"
  detected_email="$(printf '%s' "$detected_email" | tr -d '\r' | tr -d '\n' | xargs 2>/dev/null || true)"
  target_email="${ACCOUNT:-$detected_email}"
  if [[ -z "$target_email" ]]; then
    return 1
  fi

  import_json="$(mktemp /tmp/gog-token-import-XXXXXX.json)"
  if ! gws_write_gog_token_import_from_export "$GWS_EXPORT_JSON" "$target_email" "$import_json" "$CLIENT"; then
    rm -f "$import_json"
    return 1
  fi

  if ! gog_cmd auth tokens import "$import_json" >/dev/null; then
    rm -f "$import_json"
    return 1
  fi
  rm -f "$import_json"

  ACCOUNT="$target_email"
  gog_cmd auth alias set default "$ACCOUNT" >/dev/null 2>&1 || true
  log "Imported refresh token from official gws auth for $ACCOUNT."
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

ensure_credentials() {
  if gog_cmd auth credentials list >/dev/null 2>&1; then
    local count
    count="$(gog_cmd auth credentials list 2>/dev/null | wc -l | tr -d ' ')"
    if [[ "$count" -gt 0 ]]; then
      log "OAuth client credentials already present."
      return 0
    fi
  fi

  if bootstrap_gog_credentials_from_gws; then
    return 0
  fi

  err "Official Google Workspace CLI auth is required for onboarding."
  err "Run 'gws auth setup' or 'gws auth login', confirm 'gws auth export --unmasked' works, then rerun setup."
  [[ "$NO_INPUT" -eq 1 ]] && err "Non-interactive mode requires an existing successful gws auth export."
  exit 1
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

  if detect_account_from_gws; then
    return 0
  fi

  err "Could not determine the Google account from official gws auth."
  err "Run 'gws auth status' and 'gws auth export --unmasked', or pass --account explicitly, then rerun setup."
  exit 1
}

authorize_account() {
  # If already present, skip add
  if gog_cmd auth list 2>/dev/null | awk '{print $1}' | grep -Fxq "$ACCOUNT"; then
    log "Account already authorized: $ACCOUNT"
  elif bootstrap_gog_account_from_gws; then
    return 0
  else
    err "Could not import account auth from official gws onboarding for $ACCOUNT."
    err "Run 'gws auth setup' or 'gws auth login', confirm 'gws auth export --unmasked' works, then rerun setup."
    exit 1
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
  echo "  gws auth status"
  echo
  [[ "$CLI_ONLY" -eq 1 ]] && echo "For cron/scripts: set GOG_KEYRING_PASSWORD or GOG_KEYRING_PASSWORD_FILE in the environment."
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
