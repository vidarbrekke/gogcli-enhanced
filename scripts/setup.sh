#!/usr/bin/env bash
# setup.sh — Interactive Linux setup/reinstall wizard for gogcli-enhanced

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "This setup wizard currently supports Linux only." >&2
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
for arg in "$@"; do
  case "$arg" in
    --no-clear) CLEAR_PROMPTS=0 ;;
    --help|-h)
      echo "Usage: ./scripts/setup.sh [--no-clear]"
      echo "  --no-clear   Keep terminal scrollback; do not clear between prompts"
      exit 0
      ;;
  esac
done

# Self-check: validate script syntax before doing any work.
if ! bash -n "$0" 2>/tmp/gog-setup-syntax.err; then
  echo "Setup script failed syntax self-check."
  echo "Please update to latest version and retry."
  echo "Details:"
  tail -n 5 /tmp/gog-setup-syntax.err || true
  exit 2
fi

clear_screen() {
  if [[ "$CLEAR_PROMPTS" -eq 1 ]]; then
    clear 2>/dev/null || printf '\033c'
  fi
}

prompt_line() {
  local __var_name="$1"
  local prompt="$2"
  local default="${3:-}"
  local val=""

  if ! read -r -p "$prompt" val; then
    warn "No input received (EOF). Using default: ${default:-<empty>}"
    val="$default"
  fi
  [[ -z "$val" ]] && val="$default"
  printf -v "$__var_name" '%s' "$val"
}

prompt_secret() {
  local __var_name="$1"
  local prompt="$2"
  local val=""

  if ! read -r -s -p "$prompt" val; then
    echo
    warn "No secret input received (EOF)."
    return 1
  fi
  echo
  printf -v "$__var_name" '%s' "$val"
  return 0
}

ask_yes_no() {
  local prompt="$1"
  local default="${2:-y}"
  local suffix="[y/N]"
  [[ "$default" == "y" ]] && suffix="[Y/n]"
  while true; do
    local ans
    clear_screen
    prompt_line ans "$prompt $suffix " "$default"
    case "${ans,,}" in
      y|yes) return 0 ;;
      n|no) return 1 ;;
      *) echo "Please answer y or n." ;;
    esac
  done
}

CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/gogcli"
BACKUP_BASE="${XDG_CONFIG_HOME:-$HOME/.config}/gogcli-backups"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"

CURRENT_GOG="$(command -v gog 2>/dev/null || true)"
BIN_IN_REPO="$ROOT_DIR/bin/gog"
INSTALL_TARGET=""
INSTALL_COMMAND_HINT=""

DID_CONFIGURE_AUTH=0
DID_STORE_CREDENTIALS=0
DID_AUTHORIZE_ACCOUNT=0

has_cmd() { command -v "$1" >/dev/null 2>&1; }

is_cloud_context() {
  if [[ "$ROOT_DIR" == /root/openclaw-stock-home/.openclaw/workspace/repositories/* ]]; then
    return 0
  fi
  if [[ -z "${DISPLAY:-}" && -z "${WAYLAND_DISPLAY:-}" ]]; then
    return 0
  fi
  return 1
}

require_repo_layout() {
  if [[ ! -f "$ROOT_DIR/go.mod" || ! -x "$ROOT_DIR/scripts/install.sh" ]]; then
    err "Could not find expected repo layout (go.mod + scripts/install.sh)."
    err "Run this script from the gogcli-enhanced repository root."
    exit 1
  fi
}

print_state() {
  clear_screen
  echo -e "${BOLD}Detected state${RESET}"
  echo "- Repo root: $ROOT_DIR"
  echo "- Existing gog on PATH: ${CURRENT_GOG:-<none>}"
  echo "- Config dir: $CONFIG_DIR $( [[ -d "$CONFIG_DIR" ]] && echo "(exists)" || echo "(missing)" )"
  if is_cloud_context; then
    echo "- Context: cloud/headless"
  else
    echo "- Context: local"
  fi
  echo
}

main_menu() {
  while true; do
    clear_screen
    echo -e "${BOLD}Setup mode${RESET}"
    echo "Choose how you want to run installation:"
    echo "1) Fresh install (auto defaults, least questions)"
    echo "2) Reinstall (keep existing config)"
    echo "3) Reinstall (backup then reset config)"
    echo "4) Reinstall (clean reset: remove config + binary)"
    prompt_line MODE "Select an option [1-4]: "
    case "$MODE" in
      1|2|3|4) return 0 ;;
      *) echo "Please choose 1, 2, 3, or 4." ;;
    esac
  done
}

decide_install_target() {
  # Mode 1: infer automatically, avoid asking.
  if [[ "$MODE" == "1" ]]; then
    if [[ -n "$CURRENT_GOG" ]]; then
      INSTALL_TARGET="$CURRENT_GOG"
      if [[ "$CURRENT_GOG" == "/usr/local/bin/gog" ]]; then
        INSTALL_COMMAND_HINT="gog"
      else
        INSTALL_COMMAND_HINT="$CURRENT_GOG"
      fi
    else
      INSTALL_TARGET="$HOME/.local/bin/gog"
      INSTALL_COMMAND_HINT="$HOME/.local/bin/gog"
    fi
    log "Mode 1: inferred install target: $INSTALL_TARGET"
    return 0
  fi

  while true; do
    clear_screen
    echo -e "${BOLD}Install location${RESET}"
    echo "Where should the gog binary be installed?"
    echo "1) ~/.local/bin/gog (recommended, no sudo)"
    echo "2) /usr/local/bin/gog (system-wide, may require sudo)"
    prompt_line choice "Select install location [1/2] (default 1): " "1"
    choice="${choice:-1}"
    case "$choice" in
      1) INSTALL_TARGET="$HOME/.local/bin/gog"; INSTALL_COMMAND_HINT="$HOME/.local/bin/gog"; return 0 ;;
      2) INSTALL_TARGET="/usr/local/bin/gog"; INSTALL_COMMAND_HINT="gog"; return 0 ;;
      *) echo "Please choose 1 or 2." ;;
    esac
  done
}

ensure_path_hint() {
  local bindir
  bindir="$(dirname "$INSTALL_TARGET")"
  if [[ ":$PATH:" != *":$bindir:"* ]]; then
    warn "$bindir is not in PATH for this shell. Adding it to ~/.bashrc automatically."
    grep -Fq "export PATH=\"$bindir:\$PATH\"" "$HOME/.bashrc" 2>/dev/null || \
      echo "export PATH=\"$bindir:\$PATH\"" >> "$HOME/.bashrc"
    log "Updated ~/.bashrc"
    echo "Open a new shell (or run: source ~/.bashrc)."
  fi
}

check_dependencies() {
  local missing=()
  local required=(bash git make tar)
  for c in "${required[@]}"; do
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

  if has_cmd apt-get; then
    log "Auto-installing missing dependencies with apt-get (safe mode)."
    sudo apt-get update
    sudo apt-get install -y "${missing[@]}"
  else
    err "apt-get not found; cannot auto-install dependencies on this host."
    echo "Please install: ${missing[*]} and rerun setup."
    exit 1
  fi
}

backup_config_if_exists() {
  if [[ -d "$CONFIG_DIR" ]]; then
    local backup_dir="$BACKUP_BASE/$TIMESTAMP"
    mkdir -p "$backup_dir"
    cp -a "$CONFIG_DIR" "$backup_dir/"
    log "Backed up config to $backup_dir"
  fi
}

remove_installed_binary() {
  local removed=false
  local candidate_paths=("$HOME/.local/bin/gog" "/usr/local/bin/gog")
  [[ -n "$CURRENT_GOG" ]] && candidate_paths+=("$CURRENT_GOG")
  mapfile -t candidate_paths < <(printf "%s\n" "${candidate_paths[@]}" | awk '!seen[$0]++')

  for p in "${candidate_paths[@]}"; do
    if [[ -f "$p" || -L "$p" ]]; then
      if [[ "$p" == "/usr/local/bin/gog" ]]; then
        sudo rm -f "$p"
      else
        rm -f "$p"
      fi
      log "Removed $p"
      removed=true
    fi
  done

  if [[ "$removed" == false ]]; then
    log "No installed gog binary found to remove."
  fi
  return 0
}

reset_config() {
  if [[ -d "$CONFIG_DIR" ]]; then
    rm -rf "$CONFIG_DIR"
    log "Removed config directory: $CONFIG_DIR"
  else
    log "No existing config directory to remove."
  fi
}

run_mode_actions() {
  case "$MODE" in
    1) log "Mode: Fresh install" ;;
    2) log "Mode: Reinstall (preserve config)" ;;
    3) log "Mode: Reinstall (backup + reset config)"; backup_config_if_exists; reset_config ;;
    4) log "Mode: Reinstall (clean reset)"; backup_config_if_exists; reset_config; remove_installed_binary ;;
  esac
}

run_build() {
  log "Building gog via scripts/install.sh"
  "$ROOT_DIR/scripts/install.sh"
  [[ -x "$BIN_IN_REPO" ]] || { err "Build succeeded but $BIN_IN_REPO missing or not executable."; exit 1; }
}

copy_binary_to_target() {
  local target_dir tmp_target
  target_dir="$(dirname "$INSTALL_TARGET")"
  tmp_target="$target_dir/.gog.tmp.$$"

  if [[ "$target_dir" == "/usr/local/bin" ]]; then
    sudo mkdir -p "$target_dir"
    sudo cp "$BIN_IN_REPO" "$tmp_target"
    sudo chmod +x "$tmp_target"
    sudo mv -f "$tmp_target" "$INSTALL_TARGET"
  else
    mkdir -p "$target_dir"
    cp "$BIN_IN_REPO" "$tmp_target"
    chmod +x "$tmp_target"
    mv -f "$tmp_target" "$INSTALL_TARGET"
  fi

  log "Installed binary to $INSTALL_TARGET"
}

persist_keyring_env_optional() {
  local pass="$1"
  if ask_yes_no "Save keyring password in ~/.bashrc for auto-unlock in future shells? (less secure: stored as plaintext)" n; then
    grep -Fq 'export GOG_KEYRING_BACKEND=file' "$HOME/.bashrc" 2>/dev/null || \
      echo 'export GOG_KEYRING_BACKEND=file' >> "$HOME/.bashrc"

    if grep -Fq 'export GOG_KEYRING_PASSWORD=' "$HOME/.bashrc" 2>/dev/null; then
      sed -i "s|^export GOG_KEYRING_PASSWORD=.*$|export GOG_KEYRING_PASSWORD='${pass//\'/\'\"\'\"\'}'|" "$HOME/.bashrc"
    else
      echo "export GOG_KEYRING_PASSWORD='${pass//\'/\'\"\'\"\'}'" >> "$HOME/.bashrc"
    fi
    log "Saved keyring env vars to ~/.bashrc"
  fi
}

setup_file_keyring_password() {
  local keyring_file="$CONFIG_DIR/keyring"
  local keyring_found=""
  local keyring_candidates=()

  keyring_candidates+=("$keyring_file")
  keyring_candidates+=("/root/.config/gogcli/keyring")
  keyring_candidates+=("/root/openclaw-stock-home/.config/gogcli/keyring")
  mapfile -t keyring_candidates < <(printf "%s\n" "${keyring_candidates[@]}" | awk '!seen[$0]++')

  for kp in "${keyring_candidates[@]}"; do
    if [[ -f "$kp" ]]; then
      keyring_found="$kp"
      break
    fi
  done

  if [[ -n "$keyring_found" ]]; then
    # If keyring exists in an alternate config root, copy it into active config dir.
    if [[ "$keyring_found" != "$keyring_file" && ! -f "$keyring_file" ]]; then
      mkdir -p "$CONFIG_DIR"
      cp -a "$keyring_found" "$keyring_file"
      keyring_found="$keyring_file"
      log "Imported existing keyring into active config: $keyring_file"
    fi

    clear_screen
    warn "Detected existing encrypted keyring at $keyring_found"
    echo "Default is to reuse existing keyring password."
    if ask_yes_no "Enter existing keyring password now?" y; then
      clear_screen
      prompt_secret p1 "Existing keyring password: " || return 0
      echo
      export GOG_KEYRING_BACKEND=file
      export GOG_KEYRING_PASSWORD="$p1"
      log "Keyring password set for this setup session."
      persist_keyring_env_optional "$p1"
      return 0
    fi

    if ! ask_yes_no "Create/reset to a NEW keyring password instead?" n; then
      warn "Skipped keyring password entry. You can retry later if unlock fails."
      return 0
    fi
  fi

  clear_screen
  echo "No existing keyring password found."
  echo "Create a new keyring password now."
  echo "What this password is for:"
  echo "- It encrypts your stored Google refresh tokens at $CONFIG_DIR/keyring"
  echo "- gog/OpenClaw must unlock this keyring before using Google APIs"
  echo "How it is used later:"
  echo "- If saved to ~/.bashrc as GOG_KEYRING_PASSWORD, unlock is automatic"
  echo "- If not saved, you must provide it in each new shell/session"
  clear_screen
  prompt_secret p1 "New keyring password: " || return 0
  echo
  clear_screen
  prompt_secret p2 "Confirm keyring password: " || return 0
  echo
  if [[ -z "${p1:-}" ]]; then
    warn "Empty password; skipped keyring env setup."
  elif [[ "$p1" != "$p2" ]]; then
    warn "Passwords did not match; skipped keyring env setup."
  else
    export GOG_KEYRING_BACKEND=file
    export GOG_KEYRING_PASSWORD="$p1"
    log "Keyring password set for this setup session."
    persist_keyring_env_optional "$p1"
  fi
}

setup_auth_optional() {
  local gog_cmd="$INSTALL_TARGET"
  [[ -x "$gog_cmd" ]] || gog_cmd="$(command -v gog 2>/dev/null || true)"
  [[ -x "$gog_cmd" ]] || gog_cmd="$BIN_IN_REPO"

  DID_CONFIGURE_AUTH=1
  log "Configuring Google auth now (automatic)."

  clear_screen
  echo "Auth setup plan:"
  echo "1) OAuth credentials"
  echo "2) Secure token storage"
  echo "3) Account authorization"
  echo

  # 1) credentials: auto-detect and auto-reuse existing credentials.
  local creds_reuse=0
  local creds_candidates=()
  local creds_path=""
  creds_candidates+=("${XDG_CONFIG_HOME:-$HOME/.config}/gogcli/credentials.json")
  creds_candidates+=("/root/.config/gogcli/credentials.json")
  creds_candidates+=("/root/openclaw-stock-home/.config/gogcli/credentials.json")
  mapfile -t creds_candidates < <(printf "%s\n" "${creds_candidates[@]}" | awk '!seen[$0]++')

  for p in "${creds_candidates[@]}"; do
    if [[ -f "$p" ]]; then
      creds_path="$p"
      break
    fi
  done

  if [[ -n "$creds_path" ]]; then
    clear_screen
    echo "Found existing OAuth app credentials at: $creds_path"
    DID_STORE_CREDENTIALS=1
    creds_reuse=1
    log "Reusing existing OAuth credentials (automatic)."
  fi

  if [[ "$creds_reuse" -eq 0 ]]; then
    clear_screen
    prompt_line oauth_client_id "Paste OAuth Client ID (ends with apps.googleusercontent.com): "
    clear_screen
    prompt_secret oauth_client_secret "Paste OAuth Client Secret (input hidden): " || oauth_client_secret=""
    echo
    if [[ -n "${oauth_client_id:-}" && -n "${oauth_client_secret:-}" ]]; then
      local generated_json
      generated_json="$(mktemp)"
      cat > "$generated_json" <<EOF
{
  "installed": {
    "client_id": "$oauth_client_id",
    "client_secret": "$oauth_client_secret",
    "auth_uri": "https://accounts.google.com/o/oauth2/auth",
    "token_uri": "https://oauth2.googleapis.com/token"
  }
}
EOF
      "$gog_cmd" auth credentials "$generated_json"
      rm -f "$generated_json"
      DID_STORE_CREDENTIALS=1
      log "Stored OAuth credentials."
    else
      warn "Client ID/secret missing; credentials not stored."
    fi
  fi

  # 2) token storage: infer and apply file backend by default for reliability.
  log "Configuring secure token storage (file backend)."
  "$gog_cmd" auth keyring file
  setup_file_keyring_password

  # 3) account authorization with smart flow by context.
  local existing_accounts list_check_ok
  existing_accounts="$($gog_cmd auth list 2>/dev/null | grep -Eo '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+' | sort -u | tr '\n' ' ' || true)"
  list_check_ok=0
  if $gog_cmd auth list --check >/tmp/gog-auth-check.out 2>/tmp/gog-auth-check.err; then
    list_check_ok=1
  fi

  if [[ -n "${existing_accounts// }" ]]; then
    clear_screen
    echo "Found existing authorized account(s): $existing_accounts"
    if [[ "$list_check_ok" -eq 1 ]]; then
      echo "Credential check: existing account tokens look valid."
    else
      echo "Credential check: could not verify existing tokens (may still work)."
    fi
    if ask_yes_no "Reuse existing authorized account(s) and skip adding a new one?" y; then
      log "Reusing existing account setup."
      DID_AUTHORIZE_ACCOUNT=1
      return 0
    fi
    echo "Okay, we will add another account."
  fi

  # 3) authorize account: proceed directly; fallback is leaving email empty.
  clear_screen
  echo "Google account authorization"
  echo "- Required if you want OpenClaw to actually access Gmail/Drive/Docs/etc now"
  echo "- If skipped, MCP is installed/active but Google requests will fail until you authorize later"
  prompt_line account_email "Google account email to authorize (leave empty to skip for now): "
  if [[ -n "${account_email:-}" ]]; then
    if is_cloud_context; then
      log "Cloud/headless detected: using manual auth flow."
      echo "When prompted with 'Visit this URL to authorize', open it in your browser."
      echo "If your terminal supports links, the URL is clickable."
      echo "Fallback: copy/paste the URL into browser manually."
      if "$gog_cmd" auth add "$account_email" --services user --manual; then
        DID_AUTHORIZE_ACCOUNT=1
      else
        warn "Authorization did not complete in this run."
        warn "You can retry later with: $gog_cmd auth add $account_email --services user --manual"
      fi
    else
      log "Local environment detected: using local browser callback flow."
      if "$gog_cmd" auth add "$account_email"; then
        DID_AUTHORIZE_ACCOUNT=1
      else
        warn "Authorization did not complete in this run."
        warn "You can retry later with: $gog_cmd auth add $account_email"
      fi
    fi
  fi
}

configure_openclaw_mcp_auto() {
  local gog_cmd="$INSTALL_TARGET"
  [[ -x "$gog_cmd" ]] || gog_cmd="$BIN_IN_REPO"

  local workspace_dir=""
  if [[ -n "${OPENCLAW_WORKSPACE:-}" ]]; then
    workspace_dir="$OPENCLAW_WORKSPACE"
  elif [[ "$ROOT_DIR" == *"/repositories/"* ]]; then
    workspace_dir="${ROOT_DIR%/repositories/*}"
  else
    workspace_dir="$ROOT_DIR"
  fi

  local mcp_config_path="$workspace_dir/config/mcporter.json"
  local server_name="gog-agentic"

  mkdir -p "$(dirname "$mcp_config_path")"

  python3 - <<PY
import json, os
p = os.path.abspath(${mcp_config_path@Q})
os.makedirs(os.path.dirname(p), exist_ok=True)
if os.path.exists(p):
    with open(p, 'r', encoding='utf-8') as f:
        data = json.load(f)
else:
    data = {}
if not isinstance(data, dict):
    data = {}
mcp = data.get('mcpServers')
if not isinstance(mcp, dict):
    mcp = {}
mcp[${server_name@Q}] = {
    'command': ${gog_cmd@Q},
    'args': ['mcp', 'serve']
}
data['mcpServers'] = mcp
if 'imports' not in data or not isinstance(data.get('imports'), list):
    data['imports'] = []
with open(p, 'w', encoding='utf-8') as f:
    json.dump(data, f, indent=2)
    f.write('\n')
print(p)
PY

  log "Activated MCP server entry '${server_name}' in: $mcp_config_path"

  echo "Running verification check. Stay tuned."
  local verify_ok=1
  python3 - <<PY || verify_ok=0
import json
p = ${mcp_config_path@Q}
with open(p, 'r', encoding='utf-8') as f:
    d = json.load(f)
entry = d.get('mcpServers', {}).get(${server_name@Q}, {})
assert entry.get('command') == ${gog_cmd@Q}
assert entry.get('args') == ['mcp', 'serve']
print('config_ok')
PY

  if [[ "$verify_ok" -eq 1 ]]; then
    log "Verification passed: MCP config entry is active and correct."
  else
    err "Verification failed: MCP config entry is missing or invalid."
    exit 1
  fi

  echo "Running MCP discoverability check (mcporter list). Stay tuned."
  if has_cmd mcporter; then
    if mcporter --config "$mcp_config_path" list >/tmp/gog-agentic-mcporter-list.out 2>/tmp/gog-agentic-mcporter-list.err; then
      if grep -q "gog-agentic" /tmp/gog-agentic-mcporter-list.out; then
        log "Discoverability passed: mcporter lists gog-agentic."
      else
        warn "mcporter ran, but gog-agentic was not listed."
        warn "Check output: /tmp/gog-agentic-mcporter-list.out"
      fi
    else
      warn "mcporter list failed."
      warn "stderr: $(tail -n 3 /tmp/gog-agentic-mcporter-list.err 2>/dev/null || true)"
    fi
  else
    warn "mcporter CLI not found; skipping discoverability check."
  fi
}

verify_install() {
  clear_screen
  log "Verification"
  if [[ -x "$INSTALL_TARGET" ]]; then
    "$INSTALL_TARGET" --version || true
  elif has_cmd gog; then
    gog --version || true
  else
    warn "gog not found on PATH yet."
  fi
}

print_completion_summary() {
  echo
  echo -e "${GREEN}Setup complete.${RESET}"
  echo "Next steps:"
  echo "- CLI help: ${INSTALL_COMMAND_HINT:-$INSTALL_TARGET} --help"

  if [[ "$DID_CONFIGURE_AUTH" -eq 0 ]]; then
    echo "- Auth not configured in this run. Rerun setup and complete: credentials -> token storage -> account auth"
    if is_cloud_context; then
      echo "  Retry command: ${INSTALL_COMMAND_HINT:-$INSTALL_TARGET} auth add <you@gmail.com> --services user --manual"
    else
      echo "  Retry command: ${INSTALL_COMMAND_HINT:-$INSTALL_TARGET} auth add <you@gmail.com>"
    fi
  else
    [[ "$DID_STORE_CREDENTIALS" -eq 0 ]] && \
      echo "- OAuth credentials not added in this run. Add with: ${INSTALL_COMMAND_HINT:-$INSTALL_TARGET} auth credentials <path-to-json>"
    if [[ "$DID_AUTHORIZE_ACCOUNT" -eq 0 ]]; then
      echo "- Account authorization not completed in this run."
      if is_cloud_context; then
        echo "  Retry command: ${INSTALL_COMMAND_HINT:-$INSTALL_TARGET} auth add <you@gmail.com> --services user --manual"
      else
        echo "  Retry command: ${INSTALL_COMMAND_HINT:-$INSTALL_TARGET} auth add <you@gmail.com>"
      fi
    fi
  fi

  echo
  echo "OpenClaw-ready summary:"
  echo "gogcli-enhanced is a Google Workspace MCP server, and is ready for use."
  echo ""
  echo "How to use with OpenClaw:"
  echo "- Ask naturally in chat (example: Create a new Google Doc called Test1 in a new Drive folder called testing123)."
  echo "- OpenClaw can route these requests through the gog-agentic MCP server automatically."
  echo "- If auth/account setup was skipped above, complete it first so OpenClaw can access your Google data."
}

require_repo_layout
print_state
main_menu
decide_install_target
check_dependencies
run_mode_actions
run_build
copy_binary_to_target
ensure_path_hint
setup_auth_optional
configure_openclaw_mcp_auto
verify_install
print_completion_summary
