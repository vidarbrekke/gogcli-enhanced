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
  local tty_available=0

  if exec {__tty_fd}<>/dev/tty 2>/dev/null; then
    tty_available=1
  fi

  if [[ "$tty_available" -eq 1 ]]; then
    printf "%s" "$prompt" >&$__tty_fd
    if ! IFS= read -r -u "$__tty_fd" val; then
      val="$default"
    fi
    exec {__tty_fd}>&-
  else
    if ! read -r -p "$prompt" val; then
      val="$default"
    fi
  fi

  # Silent defaulting: Enter accepts default without extra warning noise.
  [[ -z "$val" ]] && val="$default"
  printf -v "$__var" '%s' "$val"
}

prompt_secret() {
  local __var="$1" prompt="$2" val=""
  local tty_available=0

  if exec {__tty_fd}<>/dev/tty 2>/dev/null; then
    tty_available=1
  fi

  if [[ "$tty_available" -eq 1 ]]; then
    # Hide input when TTY supports it; fall back gracefully if stty fails.
    printf "%s" "$prompt" >&$__tty_fd
    if stty -echo <&$__tty_fd 2>/dev/null; then
      if ! IFS= read -r -u "$__tty_fd" val; then
        stty echo <&$__tty_fd 2>/dev/null || true
        printf "\n" >&$__tty_fd
        exec {__tty_fd}>&-
        return 1
      fi
      stty echo <&$__tty_fd 2>/dev/null || true
      printf "\n" >&$__tty_fd
    else
      if ! IFS= read -r -u "$__tty_fd" val; then
        printf "\n" >&$__tty_fd
        exec {__tty_fd}>&-
        return 1
      fi
      printf "\n" >&$__tty_fd
    fi
    exec {__tty_fd}>&-
  else
    if ! read -r -s -p "$prompt" val; then
      echo
      return 1
    fi
    echo
  fi

  if [[ -z "$val" ]]; then
    warn "No input received (empty)."
  fi
  printf -v "$__var" '%s' "$val"
  return 0
}

ask_yes_no() {
  local prompt="$1" default="${2:-y}" suffix="[y/N]" default_label="N"
  if [[ "$default" == "y" ]]; then
    suffix="[Y/n]"
    default_label="Y"
  fi
  while true; do
    local ans
    prompt_line ans "$prompt $suffix (press Enter for default: $default_label): " "$default"
    case "${ans,,}" in
      y|yes) return 0 ;;
      n|no) return 1 ;;
      *) echo "Please answer y or n, then press Enter." ;;
    esac
  done
}

has_cmd() { command -v "$1" >/dev/null 2>&1; }

is_cloud_context() {
  [[ "$ROOT_DIR" == /root/openclaw-stock-home/.openclaw/workspace/repositories/* ]] && return 0
  [[ -z "${DISPLAY:-}" && -z "${WAYLAND_DISPLAY:-}" ]] && return 0
  return 1
}

# Canonical config root for ALL gog subprocesses to avoid split-brain paths.
CANON_XDG_CONFIG_HOME="/root/openclaw-stock-home/.config"
export XDG_CONFIG_HOME="$CANON_XDG_CONFIG_HOME"

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
  echo -e "${BOLD}gogcli-enhanced easy setup${RESET}"
  echo "This setup will automatically:"
  echo "1) Install/update gogcli-enhanced"
  echo "2) Activate and verify the OpenClaw MCP server (gog-agentic)"
  echo "3) Reuse your Google auth if available, or guide you step-by-step"
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
    prompt_line opt "Select [1/2/3] then press Enter (default 1): " "1"
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
    # gog keyring backend uses a path that may be a directory (not a single file).
    if [[ -e "$p" ]]; then found="$p"; break; fi
  done

  if [[ -n "$found" ]]; then
    clear_screen
    echo "Found existing keyring: $found"
    echo "Enter EXISTING keyring password to unlock stored Google tokens."
    echo "(Characters are hidden; type password and press Enter.)"
    local oldp
    prompt_secret oldp "Existing keyring password + Enter: " || return 0
    export GOG_KEYRING_BACKEND=file
    export GOG_KEYRING_PASSWORD="$oldp"
    persist_keyring_env_auto "$oldp"
    return 0
  fi

  clear_screen
  echo "No existing secure token lock was found."
  echo "Create a new password now to protect your Google connection."
  echo "- This encrypts your Google sign-in tokens on disk ($CONFIG_DIR/keyring)"
  echo "- OpenClaw uses this to access Google on future sessions"
  echo
  echo "Input note: after each prompt, type your password and press Enter."
  echo "(Characters are hidden while typing.)"

  local p1 p2
  prompt_secret p1 "Type new keyring password + Enter: " || return 0
  prompt_secret p2 "Confirm keyring password + Enter: " || return 0
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

  log "Setting up Google connection..."

  # Credentials reuse first, with explicit replacement when needed.
  local creds="$CONFIG_DIR/credentials.json"
  local use_existing="n"

  if [[ -f "$creds" ]]; then
    local cid csec placeholder=0
    cid="$(python3 - <<PY
import json
p=${creds@Q}
j=json.load(open(p))
obj=j.get('installed') or j.get('web') or j
print((obj.get('client_id') or '').strip())
PY
)"
    csec="$(python3 - <<PY
import json
p=${creds@Q}
j=json.load(open(p))
obj=j.get('installed') or j.get('web') or j
print((obj.get('client_secret') or '').strip())
PY
)"

    [[ -z "$cid" || -z "$csec" ]] && placeholder=1
    [[ "$cid" == "id.apps.googleusercontent.com" || "$csec" == "secret" ]] && placeholder=1
    [[ "$cid" == *"YOUR_CLIENT_ID"* || "$csec" == *"YOUR_CLIENT_SECRET"* ]] && placeholder=1

    if [[ "$placeholder" -eq 0 ]]; then
      # Novice default: silently reuse valid credentials to reduce friction.
      use_existing="y"
      if [[ "$ADVANCED" -eq 1 ]]; then
        clear_screen
        echo "Found existing OAuth credentials at: $creds"
        if ask_yes_no "Use existing OAuth credentials?" y; then
          use_existing="y"
        else
          use_existing="n"
        fi
      else
        log "Reusing existing OAuth credentials."
      fi
    else
      warn "Existing OAuth credentials look invalid/placeholder."
    fi
  fi

  if [[ "$use_existing" == "y" ]]; then
    # Normalize reused credentials into flat schema expected by runtime.
    python3 - <<PY
import json, os
p=${creds@Q}
j=json.load(open(p))
obj=j.get('installed') or j.get('web') or j
flat={'client_id': obj.get('client_id',''), 'client_secret': obj.get('client_secret','')}
os.makedirs(os.path.dirname(p), exist_ok=True)
with open(p,'w',encoding='utf-8') as f:
    json.dump(flat,f,indent=2)
    f.write('\n')
PY
    AUTH_CREDENTIALS_OK=1
    log "Reusing existing OAuth credentials."
  else
    clear_screen
    echo "Enter your Google OAuth app credentials."
    echo "(Desktop app client from Google Cloud Console)"
    local cid csec
    prompt_line cid "Paste OAuth Client ID then press Enter: "
    prompt_secret csec "Paste OAuth Client Secret (hidden) then Enter: " || csec=""
    if [[ -n "$cid" && -n "$csec" ]]; then
      mkdir -p "$CONFIG_DIR"
      python3 - <<PY
import json, os
p=${creds@Q}
flat={'client_id': ${cid@Q}, 'client_secret': ${csec@Q}}
os.makedirs(os.path.dirname(p), exist_ok=True)
with open(p,'w',encoding='utf-8') as f:
    json.dump(flat,f,indent=2)
    f.write('\n')
PY
      AUTH_CREDENTIALS_OK=1
      log "Stored OAuth credentials."
    else
      err "OAuth credentials were not provided. Cannot continue auth setup."
      return 1
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
  echo "Final step: connect your Google account now."
  echo "Type your Google email at the prompt below, then press Enter."
  local email
  prompt_line email "Google account email to authorize, then press Enter: "
  if [[ -z "$email" ]]; then
    err "Email is required for novice setup. Re-run and provide account email."
    return 1
  fi

  if is_cloud_context; then
    echo "A browser URL will be shown next. Open it, approve, and paste redirect URL back in this terminal."
    if "$gog_cmd" auth add "$email" --services user --manual; then
      AUTH_ACCOUNT_OK=1
    else
      err "Authorization did not complete. Retry now with: $gog_cmd auth add $email --services user --manual"
      return 1
    fi
  else
    if "$gog_cmd" auth add "$email"; then
      AUTH_ACCOUNT_OK=1
    else
      err "Authorization did not complete. Retry now with: $gog_cmd auth add $email"
      return 1
    fi
  fi

  # Hard post-check: token must exist.
  if ! $gog_cmd auth list --check >/tmp/gog-auth-postcheck.out 2>/tmp/gog-auth-postcheck.err; then
    err "Post-auth check failed; no valid token found."
    err "See /tmp/gog-auth-postcheck.err and /tmp/gog-auth-postcheck.out"
    return 1
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
    echo "Status: ✅ Ready — MCP is active and your Google account is connected"
  else
    echo "Status: ⚠️ Almost ready — MCP is active, but your Google account is not connected yet"
    if is_cloud_context; then
      echo "Run: ${INSTALL_CMD_HINT:-$INSTALL_TARGET} auth add <you@gmail.com> --services user --manual"
    else
      echo "Run: ${INSTALL_CMD_HINT:-$INSTALL_TARGET} auth add <you@gmail.com>"
    fi
  fi

  echo
  echo "OpenClaw-ready summary:"
  echo "gogcli-enhanced is a Google Workspace MCP server, and is ready for use."
  echo "Use plain language in OpenClaw. Example:"
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
