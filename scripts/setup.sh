#!/usr/bin/env bash
# Super-simple setup for gogcli-enhanced (Linux)
# Goal: novice-friendly, minimal decisions, safe defaults.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "This setup script currently supports Linux only." >&2
  exit 1
fi

BOLD="\033[1m"
GREEN="\033[32m"
YELLOW="\033[33m"
RED="\033[31m"
RESET="\033[0m"

log() { echo -e "${GREEN}==>${RESET} $*"; }
warn() { echo -e "${YELLOW}Warning:${RESET} $*"; }
err() { echo -e "${RED}Error:${RESET} $*"; }

CLEAR_PROMPTS=1
ADVANCED=0
RESET_CONFIG=0
CLEAN_RESET=0

for arg in "$@"; do
  case "$arg" in
    --no-clear) CLEAR_PROMPTS=0 ;;
    --advanced) ADVANCED=1 ;;
    --reset-config) RESET_CONFIG=1 ;;
    --clean-reset) CLEAN_RESET=1 ;;
    --help|-h)
      cat <<EOF
Usage: ./scripts/setup.sh [options]

Default mode is novice-friendly automatic setup.

Options:
  --no-clear      Keep terminal scrollback (no clear between prompts)
  --advanced      Show advanced prompts/options
  --reset-config  Backup + reset gog config before install
  --clean-reset   Backup + reset config + remove installed gog binary
EOF
      exit 0
      ;;
  esac
done

# Self-check
if ! bash -n "$0" 2>/tmp/gog-setup-syntax.err; then
  echo "Setup script failed syntax self-check."
  tail -n 5 /tmp/gog-setup-syntax.err || true
  exit 2
fi

clear_screen() {
  if [[ "$CLEAR_PROMPTS" -eq 1 ]]; then
    clear 2>/dev/null || printf '\033c'
  fi
}

prompt_line() {
  local __var="$1" prompt="$2" default="${3:-}" val=""
  if ! read -r -p "$prompt" val; then
    val="$default"
  fi
  [[ -z "$val" ]] && val="$default"
  printf -v "$__var" '%s' "$val"
}

prompt_secret() {
  local __var="$1" prompt="$2" val=""
  if ! read -r -s -p "$prompt" val; then
    echo
    return 1
  fi
  echo
  printf -v "$__var" '%s' "$val"
  return 0
}

ask_yes_no() {
  local prompt="$1" default="${2:-y}" suffix="[y/N]"
  [[ "$default" == "y" ]] && suffix="[Y/n]"
  while true; do
    local ans
    prompt_line ans "$prompt $suffix " "$default"
    case "${ans,,}" in
      y|yes) return 0 ;;
      n|no) return 1 ;;
      *) echo "Please answer y or n." ;;
    esac
  done
}

has_cmd() { command -v "$1" >/dev/null 2>&1; }

is_cloud_context() {
  [[ "$ROOT_DIR" == /root/openclaw-stock-home/.openclaw/workspace/repositories/* ]] && return 0
  [[ -z "${DISPLAY:-}" && -z "${WAYLAND_DISPLAY:-}" ]] && return 0
  return 1
}

CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/gogcli"
BACKUP_BASE="${XDG_CONFIG_HOME:-$HOME/.config}/gogcli-backups"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
BIN_IN_REPO="$ROOT_DIR/bin/gog"
CURRENT_GOG="$(command -v gog 2>/dev/null || true)"
INSTALL_TARGET=""
INSTALL_CMD_HINT=""

AUTH_CREDENTIALS_OK=0
AUTH_ACCOUNT_OK=0

require_repo_layout() {
  [[ -f "$ROOT_DIR/go.mod" && -x "$ROOT_DIR/scripts/install.sh" ]] || {
    err "Run this from the gogcli-enhanced repository root (missing go.mod/scripts/install.sh)."
    exit 1
  }
}

infer_install_target() {
  if [[ -n "$CURRENT_GOG" ]]; then
    INSTALL_TARGET="$CURRENT_GOG"
  else
    INSTALL_TARGET="$HOME/.local/bin/gog"
  fi
  if [[ "$INSTALL_TARGET" == "/usr/local/bin/gog" ]]; then
    INSTALL_CMD_HINT="gog"
  else
    INSTALL_CMD_HINT="$INSTALL_TARGET"
  fi
}

print_plan() {
  clear_screen
  echo -e "${BOLD}gogcli-enhanced quick setup${RESET}"
  echo "This will automatically:"
  echo "1) Build/install gog"
  echo "2) Configure/verify MCP server (gog-agentic)"
  echo "3) Reuse Google auth if found, otherwise guide you"
  echo
  echo "Detected context: $(is_cloud_context && echo 'cloud/headless' || echo 'local')"
  echo "Install target: $INSTALL_TARGET"
  echo
  if [[ "$ADVANCED" -eq 1 ]]; then
    echo "Advanced mode is ON."
  fi
}

check_dependencies_auto() {
  local missing=()
  for c in bash git make tar; do
    has_cmd "$c" || missing+=("$c")
  done
  if ! has_cmd curl && ! has_cmd wget; then
    missing+=("curl")
  fi

  if [[ ${#missing[@]} -eq 0 ]]; then
    log "Dependencies look good."
    return 0
  fi

  warn "Missing dependencies: ${missing[*]}"
  has_cmd apt-get || { err "apt-get not available. Install missing deps manually and rerun."; exit 1; }
  log "Installing missing dependencies automatically..."
  sudo apt-get update
  sudo apt-get install -y "${missing[@]}"
}

backup_and_reset_if_requested() {
  local do_backup=0
  [[ "$RESET_CONFIG" -eq 1 || "$CLEAN_RESET" -eq 1 ]] && do_backup=1

  if [[ "$ADVANCED" -eq 1 && "$RESET_CONFIG" -eq 0 && "$CLEAN_RESET" -eq 0 ]]; then
    clear_screen
    echo "Advanced reset options:"
    echo "1) Keep config (default)"
    echo "2) Backup + reset config"
    echo "3) Backup + reset config + remove installed binary"
    local opt
    prompt_line opt "Select [1/2/3] (default 1): " "1"
    case "$opt" in
      2) RESET_CONFIG=1; do_backup=1 ;;
      3) CLEAN_RESET=1; do_backup=1 ;;
    esac
  fi

  if [[ "$do_backup" -eq 1 && -d "$CONFIG_DIR" ]]; then
    local backup_dir="$BACKUP_BASE/$TIMESTAMP"
    mkdir -p "$backup_dir"
    cp -a "$CONFIG_DIR" "$backup_dir/"
    log "Backed up config to $backup_dir"
  fi

  if [[ "$RESET_CONFIG" -eq 1 || "$CLEAN_RESET" -eq 1 ]]; then
    if [[ -d "$CONFIG_DIR" ]]; then
      rm -rf "$CONFIG_DIR"
      log "Removed config directory: $CONFIG_DIR"
    fi
  fi

  if [[ "$CLEAN_RESET" -eq 1 ]]; then
    for p in "$HOME/.local/bin/gog" "/usr/local/bin/gog" "$CURRENT_GOG"; do
      [[ -n "$p" ]] || continue
      if [[ -f "$p" || -L "$p" ]]; then
        if [[ "$p" == "/usr/local/bin/gog" ]]; then sudo rm -f "$p"; else rm -f "$p"; fi
        log "Removed $p"
      fi
    done
  fi
}

build_and_install() {
  log "Building gog..."
  "$ROOT_DIR/scripts/install.sh"
  [[ -x "$BIN_IN_REPO" ]] || { err "Build finished but $BIN_IN_REPO missing."; exit 1; }

  local dir tmp
  dir="$(dirname "$INSTALL_TARGET")"
  tmp="$dir/.gog.tmp.$$"

  if [[ "$dir" == "/usr/local/bin" ]]; then
    sudo mkdir -p "$dir"
    sudo cp "$BIN_IN_REPO" "$tmp"
    sudo chmod +x "$tmp"
    sudo mv -f "$tmp" "$INSTALL_TARGET"
  else
    mkdir -p "$dir"
    cp "$BIN_IN_REPO" "$tmp"
    chmod +x "$tmp"
    mv -f "$tmp" "$INSTALL_TARGET"
  fi
  log "Installed binary to $INSTALL_TARGET"
}

ensure_path_auto() {
  local bindir
  bindir="$(dirname "$INSTALL_TARGET")"
  if [[ ":$PATH:" != *":$bindir:"* ]]; then
    grep -Fq "export PATH=\"$bindir:\$PATH\"" "$HOME/.bashrc" 2>/dev/null || \
      echo "export PATH=\"$bindir:\$PATH\"" >> "$HOME/.bashrc"
    log "Added $bindir to ~/.bashrc PATH"
  fi
}

persist_keyring_env_auto() {
  local pass="$1"
  # novice mode: default to persistence for reliability unless advanced user opts out.
  local do_persist=1
  if [[ "$ADVANCED" -eq 1 ]]; then
    clear_screen
    echo "Auto-unlock option"
    echo "- YES: smoother setup; password stored in ~/.bashrc (plaintext)"
    echo "- NO: more secure; you'll need to provide password each new session"
    ask_yes_no "Enable auto-unlock persistence?" y && do_persist=1 || do_persist=0
  fi

  if [[ "$do_persist" -eq 1 ]]; then
    grep -Fq 'export GOG_KEYRING_BACKEND=file' "$HOME/.bashrc" 2>/dev/null || \
      echo 'export GOG_KEYRING_BACKEND=file' >> "$HOME/.bashrc"
    if grep -Fq 'export GOG_KEYRING_PASSWORD=' "$HOME/.bashrc" 2>/dev/null; then
      sed -i "s|^export GOG_KEYRING_PASSWORD=.*$|export GOG_KEYRING_PASSWORD='${pass//\'/\'\"\'\"\'}'|" "$HOME/.bashrc"
    else
      echo "export GOG_KEYRING_PASSWORD='${pass//\'/\'\"\'\"\'}'" >> "$HOME/.bashrc"
    fi
    log "Saved keyring auto-unlock env vars to ~/.bashrc"
  else
    warn "Auto-unlock disabled; you'll need keyring password in future sessions."
  fi
}

configure_keyring_file() {
  local gog_cmd="$1"
  "$gog_cmd" auth keyring file >/dev/null || true

  local keyring_file="$CONFIG_DIR/keyring"
  local found=""
  for p in "$keyring_file" "/root/.config/gogcli/keyring" "/root/openclaw-stock-home/.config/gogcli/keyring"; do
    if [[ -f "$p" ]]; then found="$p"; break; fi
  done

  if [[ -n "$found" ]]; then
    clear_screen
    echo "Found existing keyring: $found"
    echo "Enter EXISTING keyring password to unlock stored Google tokens."
    local oldp
    prompt_secret oldp "Existing keyring password: " || return 0
    export GOG_KEYRING_BACKEND=file
    export GOG_KEYRING_PASSWORD="$oldp"
    persist_keyring_env_auto "$oldp"
    return 0
  fi

  clear_screen
  echo "No existing keyring file found."
  echo "Create a NEW keyring password now."
  echo "- This encrypts your Google tokens on disk ($CONFIG_DIR/keyring)"
  echo "- OpenClaw needs this password to unlock Google access in future sessions"

  local p1 p2
  prompt_secret p1 "New keyring password: " || return 0
  prompt_secret p2 "Confirm keyring password: " || return 0
  if [[ -z "$p1" || "$p1" != "$p2" ]]; then
    warn "Passwords were empty or mismatched; keyring unlock may fail later."
    return 0
  fi
  export GOG_KEYRING_BACKEND=file
  export GOG_KEYRING_PASSWORD="$p1"
  persist_keyring_env_auto "$p1"
}

configure_auth() {
  local gog_cmd="$INSTALL_TARGET"
  [[ -x "$gog_cmd" ]] || gog_cmd="$BIN_IN_REPO"

  log "Configuring Google auth..."

  # Credentials reuse first
  local creds=""
  for p in "${XDG_CONFIG_HOME:-$HOME/.config}/gogcli/credentials.json" "/root/.config/gogcli/credentials.json" "/root/openclaw-stock-home/.config/gogcli/credentials.json"; do
    if [[ -f "$p" ]]; then creds="$p"; break; fi
  done

  if [[ -n "$creds" ]]; then
    log "Reusing existing OAuth credentials: $creds"
    AUTH_CREDENTIALS_OK=1
  else
    clear_screen
    echo "Google OAuth app credentials are required once per machine."
    local cid csec
    prompt_line cid "Paste OAuth Client ID: "
    prompt_secret csec "Paste OAuth Client Secret (hidden): " || csec=""
    if [[ -n "$cid" && -n "$csec" ]]; then
      local tmp
      tmp="$(mktemp)"
      cat > "$tmp" <<EOF
{
  "installed": {
    "client_id": "$cid",
    "client_secret": "$csec",
    "auth_uri": "https://accounts.google.com/o/oauth2/auth",
    "token_uri": "https://oauth2.googleapis.com/token"
  }
}
EOF
      "$gog_cmd" auth credentials "$tmp" >/dev/null
      rm -f "$tmp"
      AUTH_CREDENTIALS_OK=1
      log "Stored OAuth credentials."
    else
      warn "Skipped OAuth credential entry."
    fi
  fi

  configure_keyring_file "$gog_cmd"

  local existing_accounts=""
  existing_accounts="$($gog_cmd auth list 2>/dev/null | grep -Eo '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+' | sort -u | tr '\n' ' ' || true)"
  if [[ -n "${existing_accounts// }" ]]; then
    log "Reusing existing authorized account(s): $existing_accounts"
    AUTH_ACCOUNT_OK=1
    return 0
  fi

  clear_screen
  echo "Google account authorization is still needed for live Gmail/Drive/Docs access."
  local email
  prompt_line email "Google account email to authorize now (leave empty to skip): "
  if [[ -z "$email" ]]; then
    warn "Skipped account authorization."
    return 0
  fi

  if is_cloud_context; then
    echo "A browser URL will be shown next. Open it, approve, copy redirect URL back here."
    if "$gog_cmd" auth add "$email" --services user --manual; then
      AUTH_ACCOUNT_OK=1
    else
      warn "Authorization incomplete. Retry later: $gog_cmd auth add $email --services user --manual"
    fi
  else
    if "$gog_cmd" auth add "$email"; then
      AUTH_ACCOUNT_OK=1
    else
      warn "Authorization incomplete. Retry later: $gog_cmd auth add $email"
    fi
  fi
}

configure_openclaw_mcp_auto() {
  local gog_cmd="$INSTALL_TARGET"
  [[ -x "$gog_cmd" ]] || gog_cmd="$BIN_IN_REPO"

  local workspace_dir
  if [[ -n "${OPENCLAW_WORKSPACE:-}" ]]; then
    workspace_dir="$OPENCLAW_WORKSPACE"
  elif [[ "$ROOT_DIR" == *"/repositories/"* ]]; then
    workspace_dir="${ROOT_DIR%/repositories/*}"
  else
    workspace_dir="$ROOT_DIR"
  fi

  local mcporter_config="$workspace_dir/config/mcporter.json"
  mkdir -p "$(dirname "$mcporter_config")"

  python3 - <<PY
import json, os
p = ${mcporter_config@Q}
if os.path.exists(p):
    with open(p, 'r', encoding='utf-8') as f:
        data = json.load(f)
else:
    data = {}
if not isinstance(data, dict):
    data = {}
m = data.get('mcpServers')
if not isinstance(m, dict):
    m = {}
m['gog-agentic'] = {'command': ${gog_cmd@Q}, 'args': ['mcp', 'serve']}
data['mcpServers'] = m
if 'imports' not in data or not isinstance(data.get('imports'), list):
    data['imports'] = []
with open(p, 'w', encoding='utf-8') as f:
    json.dump(data, f, indent=2)
    f.write('\n')
PY

  log "Activated MCP server entry 'gog-agentic' in $mcporter_config"
  echo "Running verification check. Stay tuned."

  python3 - <<PY
import json
p = ${mcporter_config@Q}
with open(p, 'r', encoding='utf-8') as f:
    d = json.load(f)
e = d.get('mcpServers', {}).get('gog-agentic', {})
assert e.get('command') == ${gog_cmd@Q}
assert e.get('args') == ['mcp', 'serve']
print('config_ok')
PY

  if has_cmd mcporter; then
    if mcporter --config "$mcporter_config" list >/tmp/gog-agentic-list.out 2>/tmp/gog-agentic-list.err; then
      if grep -q 'gog-agentic' /tmp/gog-agentic-list.out; then
        log "Discoverability passed: mcporter lists gog-agentic."
      else
        warn "mcporter ran but gog-agentic was not listed."
        warn "See: /tmp/gog-agentic-list.out"
      fi
    else
      warn "mcporter list failed."
      warn "See: /tmp/gog-agentic-list.err"
    fi
  else
    warn "mcporter CLI not found; skipped discoverability check."
  fi
}

verify_binary() {
  log "Verification"
  if [[ -x "$INSTALL_TARGET" ]]; then
    "$INSTALL_TARGET" --version || true
  else
    warn "Installed binary not found at expected target: $INSTALL_TARGET"
  fi
}

print_final() {
  echo
  echo -e "${GREEN}Setup complete.${RESET}"

  if [[ "$AUTH_ACCOUNT_OK" -eq 1 ]]; then
    echo "Status: ✅ MCP active + Google account authorized"
  else
    echo "Status: ⚠️ MCP active, but Google account authorization is still needed"
    if is_cloud_context; then
      echo "Run: ${INSTALL_CMD_HINT:-$INSTALL_TARGET} auth add <you@gmail.com> --services user --manual"
    else
      echo "Run: ${INSTALL_CMD_HINT:-$INSTALL_TARGET} auth add <you@gmail.com>"
    fi
  fi

  echo
  echo "OpenClaw-ready summary:"
  echo "gogcli-enhanced is a Google Workspace MCP server, and is ready for use."
  echo "Ask naturally in OpenClaw, for example:"
  echo "- Create a new Google Doc called Test1 in a new Drive folder called testing123"
}

require_repo_layout
infer_install_target
print_plan
check_dependencies_auto
backup_and_reset_if_requested
build_and_install
ensure_path_auto
configure_auth
configure_openclaw_mcp_auto
verify_binary
print_final
