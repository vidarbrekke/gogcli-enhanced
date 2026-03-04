#!/usr/bin/env bash
# setup-doctor.sh — advanced/repair setup for gogcli-enhanced (Linux)
# Contains the original full-featured setup flow (diagnostics, resets, recovery).

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
Usage: ./scripts/setup-doctor.sh [options]

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

  if [[ -t 0 ]] && exec {__tty_fd}<>/dev/tty 2>/dev/null; then
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

  if [[ -t 0 ]] && exec {__tty_fd}<>/dev/tty 2>/dev/null; then
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

detect_shell_rc_file() {
  local sh_name
  sh_name="$(basename "${SHELL:-}")"
  case "$sh_name" in
    zsh) echo "$HOME/.zshrc" ;;
    bash) echo "$HOME/.bashrc" ;;
    *) echo "$HOME/.profile" ;;
  esac
}

# Use OpenClaw cloud config root only when available/writable; otherwise use user-local config.
if is_cloud_context && [[ -d "/root/openclaw-stock-home/.config" ]] && [[ -w "/root/openclaw-stock-home/.config" ]]; then
  CANON_XDG_CONFIG_HOME="/root/openclaw-stock-home/.config"
else
  CANON_XDG_CONFIG_HOME="$HOME/.config"
fi
export XDG_CONFIG_HOME="$CANON_XDG_CONFIG_HOME"
SHELL_RC_FILE="$(detect_shell_rc_file)"

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
  if ! ask_yes_no "Install missing dependencies using apt-get now?" n; then
    err "Cannot continue without required dependencies. Install them manually and rerun."
    exit 1
  fi
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
    grep -Fq "export PATH=\"$bindir:\$PATH\"" "$SHELL_RC_FILE" 2>/dev/null || \
      echo "export PATH=\"$bindir:\$PATH\"" >> "$SHELL_RC_FILE"
    log "Added $bindir to $SHELL_RC_FILE PATH"
  fi
}

persist_keyring_env_auto() {
  local pass="$1"
  local do_persist=0
  clear_screen
  echo "Auto-unlock option"
  echo "- YES: smoother setup; password stored in plaintext in $SHELL_RC_FILE"
  echo "- NO: more secure; you'll need to provide password each new session"
  ask_yes_no "Enable auto-unlock persistence?" n && do_persist=1 || do_persist=0

  if [[ "$do_persist" -eq 1 ]]; then
    grep -Fq 'export GOG_KEYRING_BACKEND=file' "$SHELL_RC_FILE" 2>/dev/null || \
      echo 'export GOG_KEYRING_BACKEND=file' >> "$SHELL_RC_FILE"
    if grep -Fq 'export GOG_KEYRING_PASSWORD=' "$SHELL_RC_FILE" 2>/dev/null; then
      sed -i "s|^export GOG_KEYRING_PASSWORD=.*$|export GOG_KEYRING_PASSWORD='${pass//\'/\'\"\'\"\'}'|" "$SHELL_RC_FILE"
    else
      echo "export GOG_KEYRING_PASSWORD='${pass//\'/\'\"\'\"\'}'" >> "$SHELL_RC_FILE"
    fi
    log "Saved keyring auto-unlock env vars to $SHELL_RC_FILE"
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
    if [[ -t 0 ]]; then
      echo "(Characters are hidden; type password and press Enter.)"
    else
      echo "(No interactive TTY detected; input visibility depends on your shell.)"
    fi
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
  if [[ -t 0 ]]; then
    echo "(Characters are hidden while typing.)"
  else
    echo "(No interactive TTY detected; input visibility depends on your shell.)"
  fi

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
    echo ""
    echo -e "${BOLD}ACTION REQUIRED ON YOUR LOCAL MACHINE${RESET}"
    echo "A browser URL will be shown next. Open it, approve, then paste the redirect URL back here."
    step1_out="$("$gog_cmd" auth add "$email" --services user --remote --step 1 2>/dev/null)" || true
    auth_url=""
    if [[ -n "$step1_out" ]]; then
      auth_url="$(echo "$step1_out" | awk -F'\t' '$1=="auth_url"{print $2; exit}')"
    fi
    if [[ -z "$auth_url" ]]; then
      err "Could not get auth URL. Run manually: $gog_cmd auth add $email --services user --remote --step 1"
      return 1
    fi
    echo ""
    echo "Open this URL in your browser to authorize:"
    echo "  $auth_url"
    echo ""
    echo "After you click Allow, paste the **entire** URL from your browser's address bar (it may start with https://localhost or similar)."
    echo ""
    local redirect_url
    prompt_line redirect_url "Paste the full redirect URL here, then press Enter: "
    if [[ -z "$redirect_url" ]]; then
      err "Redirect URL is required. Re-run and paste the URL after authorizing."
      return 1
    fi
    if "$gog_cmd" auth add "$email" --services user --remote --step 2 --auth-url "$redirect_url"; then
      AUTH_ACCOUNT_OK=1
    else
      err "Authorization did not complete. Retry: $gog_cmd auth add $email --services user --remote --step 2 --auth-url <url>"
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
  # Use absolute path so mcporter/OpenClaw can start gog regardless of working directory
  if [[ "$gog_cmd" != /* ]]; then
    if command -v realpath >/dev/null 2>&1; then
      gog_cmd="$(realpath "$gog_cmd" 2>/dev/null)" || true
    elif command -v readlink >/dev/null 2>&1; then
      local gog_dir gog_base
      gog_dir="$(cd "$(dirname "$gog_cmd")" && pwd)"
      gog_base="$(basename "$gog_cmd")"
      gog_cmd="$gog_dir/$gog_base"
    fi
  fi
  [[ -z "$gog_cmd" || ! -x "$gog_cmd" ]] && gog_cmd="$INSTALL_TARGET"

  local workspace_dir
  if [[ -n "${OPENCLAW_WORKSPACE:-}" ]]; then
    workspace_dir="$OPENCLAW_WORKSPACE"
  elif [[ "$ROOT_DIR" == *"/repositories/"* ]]; then
    workspace_dir="${ROOT_DIR%/repositories/*}"
  else
    workspace_dir="$ROOT_DIR"
  fi

  # Optional env so the spawned gog process uses file keyring (password must be set by the process that runs OpenClaw)
  local env_json="{}"
  # Ensure MCP child process reads same config/keyring root used during setup.
  if [[ -n "${XDG_CONFIG_HOME:-}" ]]; then
    env_json="{\"XDG_CONFIG_HOME\": \"${XDG_CONFIG_HOME}\"}"
  fi
  if [[ -n "${GOG_KEYRING_BACKEND:-}" ]]; then
    if [[ "$env_json" == "{}" ]]; then
      env_json="{\"GOG_KEYRING_BACKEND\": \"${GOG_KEYRING_BACKEND}\"}"
    else
      env_json="{\"XDG_CONFIG_HOME\": \"${XDG_CONFIG_HOME}\", \"GOG_KEYRING_BACKEND\": \"${GOG_KEYRING_BACKEND}\"}"
    fi
  fi

  # On headless, MCP child only gets env from mcporter.json. Ensure keyring password is available via a file when using file keyring.
  local password_file_path=""
  if [[ -n "${GOG_KEYRING_BACKEND:-}" ]] && [[ "$GOG_KEYRING_BACKEND" == "file" ]]; then
    if [[ -n "${GOG_KEYRING_PASSWORD_FILE:-}" ]]; then
      password_file_path="$GOG_KEYRING_PASSWORD_FILE"
      log "Using existing keyring password file for MCP: $password_file_path"
    elif [[ -n "${GOG_KEYRING_PASSWORD:-}" ]]; then
      password_file_path="$CONFIG_DIR/keyring.password"
      mkdir -p "$CONFIG_DIR"
      printf '%s' "$GOG_KEYRING_PASSWORD" > "$password_file_path"
      chmod 600 "$password_file_path"
      log "Wrote keyring password to $password_file_path for MCP headless use"
    elif [[ -t 0 ]]; then
      if ask_yes_no "Save keyring password to a file so the MCP server can unlock without a TTY?" "n"; then
        local pw=""
        if prompt_secret pw "Keyring password: "; then
          password_file_path="$CONFIG_DIR/keyring.password"
          mkdir -p "$CONFIG_DIR"
          printf '%s' "$pw" > "$password_file_path"
          chmod 600 "$password_file_path"
          log "Wrote keyring password to $password_file_path for MCP headless use"
        fi
      fi
    fi
  fi
  export MCP_PASSWORD_FILE_PATH="$password_file_path"

  local mcporter_config="$workspace_dir/config/mcporter.json"
  mkdir -p "$(dirname "$mcporter_config")"

  python3 - <<PY
import json, os
p = ${mcporter_config@Q}
env_obj = json.loads(${env_json@Q})
if os.environ.get("MCP_PASSWORD_FILE_PATH"):
    env_obj["GOG_KEYRING_PASSWORD_FILE"] = os.environ.get("MCP_PASSWORD_FILE_PATH")
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
entry = {'command': ${gog_cmd@Q}, 'args': ['mcp', 'serve'], 'lifecycle': {'mode': 'keep-alive'}}
if env_obj:
    entry['env'] = env_obj
m['gog-agentic'] = entry
data['mcpServers'] = m
if 'imports' not in data or not isinstance(data.get('imports'), list):
    data['imports'] = []
with open(p, 'w', encoding='utf-8') as f:
    json.dump(data, f, indent=2)
    f.write('\n')
PY

  log "Activated MCP server entry 'gog-agentic' in $mcporter_config"
  if [[ -n "${GOG_KEYRING_BACKEND:-}" ]] && [[ "$GOG_KEYRING_BACKEND" == "file" ]]; then
    if [[ -n "$MCP_PASSWORD_FILE_PATH" ]]; then
      log "Keyring password file configured for MCP (headless unlock)."
    else
      log "On headless: add GOG_KEYRING_PASSWORD_FILE to gog-agentic env in mcporter.json so MCP can unlock the keyring (see docs/openclaw-linode-runbook.md)."
    fi
  fi
  MCP_CONFIG_PATH="$mcporter_config"

  # Inject directive into OpenClaw bootstrap so the agent always prefers gog-agentic for Drive/Docs (no manual instruction needed).
  local tools_md="$workspace_dir/TOOLS.md"
  python3 - "$tools_md" <<'PY'
import sys, os, re
tools_path = sys.argv[1]
section = """
## Google Drive and Docs (gog-agentic MCP)

**Do not say tools are unavailable or that the gateway may not be loading config. Do not read TOOLS.md or config files to "diagnose".** For any Google Drive or Docs request, **call the exec tool immediately** with one of these commands. Tool names contain dots so you MUST use `--server gog-agentic --tool <toolName>` — do NOT use the `gog-agentic.drive.listFiles` dot-selector (it splits on the first dot and fails with "tool not found").

- **List Drive root (files and folders):** `mcporter call --server gog-agentic --tool drive.listFiles --args '{}' --output json`
- **List all accessible Drive files (global):** `mcporter call --server gog-agentic --tool drive.listFiles --args '{"global":true,"maxResults":20}' --output json` (cannot combine `global:true` with `parentId`)
- **List only folders (use when user asks for "folders" or "all folders"):** `mcporter call --server gog-agentic --tool drive.searchFiles --args '{\"query\":\"mimeType = \\\"application/vnd.google-apps.folder\\\"\",\"rawQuery\":true}' --output json` — returns one page + nextPageToken. **When user asks for \"first N\" (e.g. first 15), add \"maxResults\": N to the args**, e.g. \"maxResults\":15. To get all folders, call again with \"page\": \"<nextPageToken>\" until the response has no nextPageToken.
- **Create folder:** `mcporter call --server gog-agentic --tool drive.ensureFolder --args '{"path":"FolderName"}' --output json`
- **Create doc:** `mcporter call --server gog-agentic --tool docs.create --args '{"title":"Doc Title"}' --output json` (add `"parentId":"<folderId>"` to place doc in a folder)
- **Search files:** `mcporter call --server gog-agentic --tool drive.searchFiles --args '{"query":"name or text"}' --output json`

OAuth is already set up. If the exec call fails (e.g. command not found or error output), report the error and then suggest the user ask the workspace admin to run the diagnostic and restart the daemon and gateway (runbook §8.0). Never reveal the keyring password or credentials.

Never invent or assume folder or file names. Only report what the API returned. If you got only N items, say so and offer to fetch more with page/pageToken; do not make up names.

For "create folder then doc": run ensureFolder first, then docs.create with the returned folderId as parentId.
"""
existing = ""
if os.path.isfile(tools_path):
    with open(tools_path, "r", encoding="utf-8") as f:
        existing = f.read()
heading = "## Google Drive and Docs (gog-agentic MCP)"
pattern = r'(?ms)^## Google Drive and Docs \(gog-agentic MCP\)\n.*?(?=^## |\Z)'
canonical = section.strip() + "\n"
if re.search(pattern, existing):
    # Replace all duplicate sections with one canonical block.
    cleaned = re.sub(pattern, "", existing).rstrip()
    new_content = (cleaned + "\n\n" + canonical) if cleaned else canonical
else:
    cleaned = existing.rstrip()
    new_content = (cleaned + "\n\n" + canonical) if cleaned else canonical
with open(tools_path, "w", encoding="utf-8") as f:
    f.write(new_content)
PY
  log "Ensured gog-agentic directive in $tools_md (OpenClaw bootstrap)"

  # Fallback: if a well-known OpenClaw workspace exists elsewhere, register there too so OpenClaw finds gog regardless of repo location
  local fallback_dir
  for fallback_dir in "$HOME/openclaw-stock-home/.openclaw/workspace" "$HOME/.openclaw/workspace"; do
    [[ -d "$fallback_dir" ]] || continue
    [[ "$(cd "$fallback_dir" && pwd)" == "$(cd "$workspace_dir" && pwd)" ]] && continue
    local fallback_config="$fallback_dir/config/mcporter.json"
    mkdir -p "$(dirname "$fallback_config")"
    python3 - <<PY
import json, os
p = ${fallback_config@Q}
env_obj = json.loads(${env_json@Q})
if os.environ.get("MCP_PASSWORD_FILE_PATH"):
    env_obj["GOG_KEYRING_PASSWORD_FILE"] = os.environ.get("MCP_PASSWORD_FILE_PATH")
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
entry = {'command': ${gog_cmd@Q}, 'args': ['mcp', 'serve'], 'lifecycle': {'mode': 'keep-alive'}}
if env_obj:
    entry['env'] = env_obj
m['gog-agentic'] = entry
data['mcpServers'] = m
if 'imports' not in data or not isinstance(data.get('imports'), list):
    data['imports'] = []
with open(p, 'w', encoding='utf-8') as f:
    json.dump(data, f, indent=2)
    f.write('\n')
PY
    log "Also registered 'gog-agentic' in fallback config: $fallback_config"
    # Ensure directive in this workspace bootstrap too
    tools_md_fb="$fallback_dir/TOOLS.md"
    python3 - "$tools_md_fb" <<'PY'
import sys, os, re
tools_path = sys.argv[1]
section = """
## Google Drive and Docs (gog-agentic MCP)

**Do not say tools are unavailable or that the gateway may not be loading config. Do not read TOOLS.md or config files to "diagnose".** For any Google Drive or Docs request, **call the exec tool immediately** with one of these commands. Tool names contain dots so you MUST use `--server gog-agentic --tool <toolName>` — do NOT use the `gog-agentic.drive.listFiles` dot-selector (it splits on the first dot and fails with "tool not found").

- **List Drive root (files and folders):** `mcporter call --server gog-agentic --tool drive.listFiles --args '{}' --output json`
- **List all accessible Drive files (global):** `mcporter call --server gog-agentic --tool drive.listFiles --args '{"global":true,"maxResults":20}' --output json` (cannot combine `global:true` with `parentId`)
- **List only folders (use when user asks for "folders" or "all folders"):** `mcporter call --server gog-agentic --tool drive.searchFiles --args '{\"query\":\"mimeType = \\\"application/vnd.google-apps.folder\\\"\",\"rawQuery\":true}' --output json` — returns one page + nextPageToken. **When user asks for \"first N\" (e.g. first 15), add \"maxResults\": N to the args**, e.g. \"maxResults\":15. To get all folders, call again with \"page\": \"<nextPageToken>\" until the response has no nextPageToken.
- **Create folder:** `mcporter call --server gog-agentic --tool drive.ensureFolder --args '{"path":"FolderName"}' --output json`
- **Create doc:** `mcporter call --server gog-agentic --tool docs.create --args '{"title":"Doc Title"}' --output json` (add `"parentId":"<folderId>"` to place doc in a folder)
- **Search files:** `mcporter call --server gog-agentic --tool drive.searchFiles --args '{"query":"name or text"}' --output json`

OAuth is already set up. If the exec call fails (e.g. command not found or error output), report the error and then suggest the user ask the workspace admin to run the diagnostic and restart the daemon and gateway (runbook §8.0). Never reveal the keyring password or credentials.

Never invent or assume folder or file names. Only report what the API returned. If you got only N items, say so and offer to fetch more with page/pageToken; do not make up names.

For "create folder then doc": run ensureFolder first, then docs.create with the returned folderId as parentId.
"""
existing = ""
if os.path.isfile(tools_path):
    with open(tools_path, "r", encoding="utf-8") as f:
        existing = f.read()
pattern = r'(?ms)^## Google Drive and Docs \(gog-agentic MCP\)\n.*?(?=^## |\Z)'
canonical = section.strip() + "\n"
if re.search(pattern, existing):
    cleaned = re.sub(pattern, "", existing).rstrip()
    new_content = (cleaned + "\n\n" + canonical) if cleaned else canonical
else:
    cleaned = existing.rstrip()
    new_content = (cleaned + "\n\n" + canonical) if cleaned else canonical
with open(tools_path, "w", encoding="utf-8") as f:
    f.write(new_content)
PY
    log "Ensured gog-agentic directive in $tools_md_fb"
  done

  # So the gateway finds gog-agentic even when it uses the default mcporter path (~/.mcporter/mcporter.json)
  local default_mcp="$HOME/.mcporter/mcporter.json"
  if [[ -f "$default_mcp" ]] || is_cloud_context; then
    mkdir -p "$(dirname "$default_mcp")"
    python3 - <<PY
import json, os
default_path = ${default_mcp@Q}
env_obj = json.loads(${env_json@Q})
if os.environ.get("MCP_PASSWORD_FILE_PATH"):
    env_obj["GOG_KEYRING_PASSWORD_FILE"] = os.environ.get("MCP_PASSWORD_FILE_PATH")
entry = {'command': ${gog_cmd@Q}, 'args': ['mcp', 'serve'], 'lifecycle': {'mode': 'keep-alive'}}
if env_obj:
    entry['env'] = env_obj
if os.path.exists(default_path):
    with open(default_path, 'r', encoding='utf-8') as f:
        data = json.load(f)
else:
    data = {}
if not isinstance(data, dict):
    data = {}
m = data.get('mcpServers')
if not isinstance(m, dict):
    m = {}
m['gog-agentic'] = entry
data['mcpServers'] = m
with open(default_path, 'w', encoding='utf-8') as f:
    json.dump(data, f, indent=2)
    f.write('\n')
PY
    log "Merged gog-agentic into default mcporter config: $default_mcp"
    if has_cmd mcporter; then
      mcporter --config "$default_mcp" daemon restart 2>/dev/null && log "Started mcporter daemon (default path so gateway finds gog-agentic)." || true
    fi
  fi

  # When gateway runs with HOME set to OpenClaw home (e.g. systemd override), it resolves ~/.mcporter to that HOME.
  # Ensure gog-agentic exists there so the agent gets tools even when MCPORTER_CONFIG is not used.
  if [[ "$workspace_dir" == *"/.openclaw/workspace" ]]; then
    local openclaw_home="${workspace_dir%/.openclaw/workspace}"
    local openclaw_mcp="$openclaw_home/.mcporter/mcporter.json"
    if [[ -n "$openclaw_home" ]] && [[ "$openclaw_home" != "$HOME" ]]; then
      mkdir -p "$(dirname "$openclaw_mcp")"
      python3 - <<PY
import json, os
target = ${openclaw_mcp@Q}
src = ${mcporter_config@Q}
with open(src, 'r', encoding='utf-8') as f:
    ws = json.load(f)
entry = (ws.get('mcpServers') or {}).get('gog-agentic')
if not entry:
    exit(0)
if os.path.exists(target):
    with open(target, 'r', encoding='utf-8') as f:
        data = json.load(f)
else:
    data = {}
if not isinstance(data.get('mcpServers'), dict):
    data['mcpServers'] = {}
data['mcpServers']['gog-agentic'] = entry
with open(target, 'w', encoding='utf-8') as f:
    json.dump(data, f, indent=2)
    f.write('\n')
PY
      log "Merged gog-agentic into OpenClaw HOME mcporter config: $openclaw_mcp"
    fi
  fi

  echo "Running verification check. Stay tuned."

  # Run diagnostic so we catch keyring/path issues at setup time (not when the agent tries to use tools).
  if [[ -x "$ROOT_DIR/scripts/mcp-diagnose-gog.sh" ]]; then
    if "$ROOT_DIR/scripts/mcp-diagnose-gog.sh" "$mcporter_config" >/tmp/gog-mcp-diagnose.out 2>/tmp/gog-mcp-diagnose.err; then
      log "gog MCP diagnostic passed (tools exposed)."
    else
      warn "gog MCP diagnostic failed; agent may not see gog-agentic tools until keyring/path are fixed."
      [[ -s /tmp/gog-mcp-diagnose.err ]] && warn "See: /tmp/gog-mcp-diagnose.err"
    fi
  fi

  # Start mcporter daemon so the gateway can see gog-agentic immediately (keep-alive mode requires daemon).
  if has_cmd mcporter; then
    if mcporter --config "$mcporter_config" daemon restart 2>/dev/null; then
      log "Started mcporter daemon (keep-alive for gog-agentic)."
    else
      warn "Could not start mcporter daemon; run manually: mcporter --config $mcporter_config daemon restart"
    fi
  fi

  # Restart OpenClaw gateway so it picks up the tool list (one-command setup; no separate step for the user).
  if systemctl --user restart openclaw-gateway 2>/dev/null; then
    log "Restarted OpenClaw gateway so the agent sees gog-agentic tools."
  else
    # Gateway may not be installed, or we may not be the user that runs it; don't fail setup.
    log "OpenClaw gateway was not restarted (not running under this user?). Restart it manually so the agent sees gog-agentic."
  fi

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

enforce_token_ready_for_mcp() {
  local gog_cmd="$INSTALL_TARGET"
  [[ -x "$gog_cmd" ]] || gog_cmd="$BIN_IN_REPO"

  # Gate success on the exact auth environment used by gog-agentic.
  if XDG_CONFIG_HOME="$XDG_CONFIG_HOME" \
    GOG_KEYRING_BACKEND="${GOG_KEYRING_BACKEND:-file}" \
    GOG_KEYRING_PASSWORD_FILE="${GOG_KEYRING_PASSWORD_FILE:-$CONFIG_DIR/keyring.password}" \
    "$gog_cmd" auth list --check >/tmp/gog-auth-gate.out 2>/tmp/gog-auth-gate.err; then
    log "Google token check passed for MCP runtime."
    return 0
  fi

  warn "No valid Google token found for MCP runtime. Completing auth now..."
  clear_screen
  echo "Final auth gate: connect your Google account now so MCP works immediately."
  local email
  prompt_line email "Google account email to authorize, then press Enter: "
  if [[ -z "$email" ]]; then
    err "Email is required to finish setup with a working MCP token."
    return 1
  fi

  if is_cloud_context; then
    local step1_out auth_url redirect_url
    echo ""
    echo -e "${BOLD}ACTION REQUIRED ON YOUR LOCAL MACHINE${RESET}"
    echo "Open the URL below in your browser, then paste the redirect URL back here."
    step1_out="$("$gog_cmd" auth add "$email" --services user --remote --step 1 2>/dev/null)" || true
    auth_url="$(echo "$step1_out" | awk -F'\t' '$1=="auth_url"{print $2; exit}')"
    if [[ -z "$auth_url" ]]; then
      err "Could not generate auth URL. Run manually: $gog_cmd auth add $email --services user --remote --step 1"
      return 1
    fi
    echo
    echo "Open this URL in your browser:"
    echo "  $auth_url"
    echo
    echo "After you click Allow, paste the **entire** URL from your browser's address bar."
    echo
    prompt_line redirect_url "Paste the full redirect URL here, then press Enter: "
    [[ -n "$redirect_url" ]] || { err "Redirect URL is required."; return 1; }
    "$gog_cmd" auth add "$email" --services user --remote --step 2 --auth-url "$redirect_url" >/tmp/gog-auth-gate-step2.out 2>/tmp/gog-auth-gate-step2.err || {
      err "Auth step 2 failed. See /tmp/gog-auth-gate-step2.err"
      return 1
    }
  else
    "$gog_cmd" auth add "$email" >/tmp/gog-auth-gate-local.out 2>/tmp/gog-auth-gate-local.err || {
      err "Local auth failed. See /tmp/gog-auth-gate-local.err"
      return 1
    }
  fi

  if ! XDG_CONFIG_HOME="$XDG_CONFIG_HOME" \
    GOG_KEYRING_BACKEND="${GOG_KEYRING_BACKEND:-file}" \
    GOG_KEYRING_PASSWORD_FILE="${GOG_KEYRING_PASSWORD_FILE:-$CONFIG_DIR/keyring.password}" \
    "$gog_cmd" auth list --check >/tmp/gog-auth-gate-final.out 2>/tmp/gog-auth-gate-final.err; then
    err "Setup cannot finish: token still unavailable to MCP runtime."
    err "Check /tmp/gog-auth-gate-final.err and rerun setup."
    return 1
  fi

  AUTH_ACCOUNT_OK=1
  log "Google token is now available for MCP runtime."
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
  if [[ -n "${MCP_CONFIG_PATH:-}" ]]; then
    echo "MCP config was written to: $MCP_CONFIG_PATH"
    echo "mcporter daemon and OpenClaw gateway were started/restarted so the agent sees gog-agentic tools."
  fi
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
enforce_token_ready_for_mcp
verify_binary
print_final
