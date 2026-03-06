#!/usr/bin/env bash
# setup-doctor.sh — advanced/repair setup for gogcli-enhanced (Ubuntu/macOS)
# Contains the original full-featured setup flow (diagnostics, resets, recovery).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/scripts/lib/gws-auth-bridge.sh"
source "$ROOT_DIR/scripts/lib/gog-agentic-config.sh"

OS_NAME="$(uname -s)"
if [[ "$OS_NAME" != "Linux" && "$OS_NAME" != "Darwin" ]]; then
  echo "This setup script currently supports Ubuntu/Linux and macOS only." >&2
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

install_system_packages() {
  local packages=("$@")
  case "$OS_NAME" in
    Linux)
      has_cmd apt-get || { err "apt-get not available. Install missing deps manually and rerun."; exit 1; }
      if ! ask_yes_no "Install missing dependencies using apt-get now?" n; then
        err "Cannot continue without required dependencies. Install them manually and rerun."
        exit 1
      fi
      log "Installing missing dependencies automatically with apt-get..."
      if has_cmd sudo; then
        sudo apt-get update
        sudo apt-get install -y "${packages[@]}"
      else
        apt-get update
        apt-get install -y "${packages[@]}"
      fi
      ;;
    Darwin)
      has_cmd brew || { err "Homebrew is required on macOS. Install Homebrew and rerun."; exit 1; }
      log "Installing missing dependencies automatically with Homebrew..."
      brew install "${packages[@]}"
      ;;
  esac
}

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
GWS_EXPORT_JSON=""

cleanup() {
  if [[ -n "$GWS_EXPORT_JSON" && -f "$GWS_EXPORT_JSON" ]]; then
    rm -f "$GWS_EXPORT_JSON"
  fi
}

trap cleanup EXIT

ensure_gws_export() {
  if [[ -n "$GWS_EXPORT_JSON" && -f "$GWS_EXPORT_JSON" ]]; then
    return 0
  fi

  GWS_EXPORT_JSON="$(mktemp /tmp/gws-export-XXXXXX.json)"
  if ! gws_bootstrap_export_file "$GWS_EXPORT_JSON" 0; then
    rm -f "$GWS_EXPORT_JSON"
    GWS_EXPORT_JSON=""
    return 1
  fi

  return 0
}

bootstrap_doctor_credentials_from_gws() {
  local creds_path="$1"
  if ! ensure_gws_export; then
    return 1
  fi

  if ! gws_write_gog_credentials_from_export "$GWS_EXPORT_JSON" "$creds_path"; then
    return 1
  fi

  AUTH_CREDENTIALS_OK=1
  log "Imported OAuth client credentials from official gws auth."
}

bootstrap_doctor_account_from_gws() {
  local gog_cmd="$1"
  local email="$2"
  local import_json detected_email target_email

  if ! ensure_gws_export; then
    return 1
  fi

  detected_email="$(gws_guess_email_from_export "$GWS_EXPORT_JSON" 2>/dev/null || true)"
  detected_email="$(printf '%s' "$detected_email" | tr -d '\r' | tr -d '\n' | xargs 2>/dev/null || true)"
  target_email="${email:-$detected_email}"
  if [[ -z "$target_email" ]]; then
    return 1
  fi

  import_json="$(mktemp /tmp/gog-token-import-XXXXXX.json)"
  if ! gws_write_gog_token_import_from_export "$GWS_EXPORT_JSON" "$target_email" "$import_json" ""; then
    rm -f "$import_json"
    return 1
  fi

  if ! "$gog_cmd" auth tokens import "$import_json" >/dev/null; then
    rm -f "$import_json"
    return 1
  fi
  rm -f "$import_json"

  "$gog_cmd" auth alias set default "$target_email" >/dev/null 2>&1 || true
  AUTH_ACCOUNT_OK=1
  log "Imported refresh token from official gws auth for $target_email."
}

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
  echo "3) Require the official gws auth flow, then import it into gog"
  echo
  echo "Detected context: $(is_cloud_context && echo 'cloud/headless' || echo 'local')"
  echo "Install target: $INSTALL_TARGET"
  echo
  if [[ "$ADVANCED" -eq 1 ]]; then
    echo "Advanced mode is ON."
  fi
}

check_dependencies_auto() {
  local missing_pkgs=()
  for c in bash git make tar python3 jq; do
    has_cmd "$c" || missing_pkgs+=("$c")
  done
  if ! has_cmd curl && ! has_cmd wget; then
    missing_pkgs+=("curl")
  fi
  if ! has_cmd npm; then
    if [[ "$OS_NAME" == "Linux" ]]; then
      missing_pkgs+=("npm")
    else
      missing_pkgs+=("node")
    fi
  fi
  if ! has_cmd node; then
    if [[ "$OS_NAME" == "Linux" ]]; then
      missing_pkgs+=("nodejs")
    else
      missing_pkgs+=("node")
    fi
  fi

  if [[ ${#missing_pkgs[@]} -gt 0 ]]; then
    warn "Missing dependencies: ${missing_pkgs[*]}"
    install_system_packages "${missing_pkgs[@]}"
  fi

  if ! gws_bridge_available; then
    log "Installing official Google Workspace CLI (gws)..."
    npm install -g @googleworkspace/cli
  fi

  if ! gws_bridge_available; then
    err "gws is still unavailable after installation attempt."
    exit 1
  fi

  if ! has_cmd mcporter; then
    log "Installing mcporter..."
    npm install -g mcporter
  fi

  if ! has_cmd mcporter; then
    err "mcporter is still unavailable after installation attempt."
    exit 1
  fi

  log "Dependencies look good."
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
      python3 - "$SHELL_RC_FILE" "$pass" <<'PY'
import pathlib
import re
import sys

path = pathlib.Path(sys.argv[1])
pw = sys.argv[2].replace("'", "'\"'\"'")
text = path.read_text(encoding="utf-8")
text = re.sub(r"^export GOG_KEYRING_PASSWORD=.*$", f"export GOG_KEYRING_PASSWORD='{pw}'", text, flags=re.M)
path.write_text(text, encoding="utf-8")
PY
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
    if bootstrap_doctor_credentials_from_gws "$creds"; then
      use_existing="gws"
    fi
  fi

  if [[ "$use_existing" != "y" && "$use_existing" != "gws" ]]; then
    err "Official Google Workspace CLI auth is required for onboarding."
    err "Run 'gws auth setup' or 'gws auth login', confirm 'gws auth export --unmasked' works, then rerun setup."
    return 1
  fi

  configure_keyring_file "$gog_cmd"

  local existing_accounts=""
  existing_accounts="$($gog_cmd auth list 2>/dev/null | grep -Eo '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+' | sort -u | tr '\n' ' ' || true)"
  if [[ -n "${existing_accounts// }" ]]; then
    log "Reusing existing authorized account(s): $existing_accounts"
    AUTH_ACCOUNT_OK=1
    return 0
  fi

  if bootstrap_doctor_account_from_gws "$gog_cmd" ""; then
    return 0
  fi

  err "Could not import account auth from official gws onboarding."
  err "Run 'gws auth setup' or 'gws auth login', confirm 'gws auth export --unmasked' works, then rerun setup."
  return 1
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
  gog_agentic_upsert_mcporter_config "$mcporter_config" "$gog_cmd"

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

**Do not say tools are unavailable or that the gateway may not be loading config. Do not read TOOLS.md or config files to "diagnose".** For any Google Drive or Docs request, **call the exec tool immediately** with one of these commands. Tool names use underscores (e.g. drive_listFiles, docs_create). You can use: `mcporter call gog-agentic.drive_listFiles --args '{}'` or `mcporter call --server gog-agentic --tool drive_listFiles --args '{}'`.

- **List Drive root (files and folders):** `mcporter call gog-agentic.drive_listFiles --args '{}' --output json`
- **List all accessible Drive files (global):** `mcporter call gog-agentic.drive_listFiles --args '{"global":true,"maxResults":20}' --output json` (cannot combine `global:true` with `parentId`)
- **List only folders (use when user asks for "folders" or "all folders"):** `mcporter call gog-agentic.drive_searchFiles --args '{\"query\":\"mimeType = \\\"application/vnd.google-apps.folder\\\"\",\"rawQuery\":true}' --output json` — returns one page + nextPageToken. **For "how many folders in root" use one call with fetchAllPages:** `--args '{\"query\":\"mimeType = \\\"application/vnd.google-apps.folder\\\" and \\\"root\\\" in parents\",\"rawQuery\":true,\"fetchAllPages\":true}'` — response includes totalCount. Tool names use **underscores** (drive_searchFiles not drive.searchFiles). When user asks for "first N", add \"maxResults\": N. To get all folders page-by-page, call again with \"page\": \"<nextPageToken>\" until no nextPageToken.
- **Create folder:** `mcporter call gog-agentic.drive_ensureFolder --args '{"path":"FolderName"}' --output json`
- **Create doc:** `mcporter call gog-agentic.docs_create --args '{"title":"Doc Title"}' --output json` (add `"parentId":"<folderId>"` to place doc in a folder)
- **Search files:** `mcporter call gog-agentic.drive_searchFiles --args '{"query":"name or text"}' --output json`
- **Get spreadsheet values:** `mcporter call gog-agentic.sheets_valuesGet --args '{\"spreadsheetId\":\"<id>\",\"range\":\"Sheet1!A1:D10\"}' --output json`

OAuth is already set up. If the exec call fails (e.g. command not found or error output), report the error and then suggest the user ask the workspace admin to run the diagnostic and restart the daemon and gateway (runbook §8.0). Never reveal the keyring password or credentials.

Never invent or assume folder or file names. Only report what the API returned. If you got only N items, say so and offer to fetch more with page/pageToken; do not make up names.

For "create folder then doc": run drive_ensureFolder first, then docs_create with the returned folderId as parentId.
"""
existing = ""
if os.path.isfile(tools_path):
    with open(tools_path, "r", encoding="utf-8") as f:
        existing = f.read()
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
    gog_agentic_upsert_mcporter_config "$fallback_config" "$gog_cmd"
    log "Also registered 'gog-agentic' in fallback config: $fallback_config"
    # Ensure directive in this workspace bootstrap too
    tools_md_fb="$fallback_dir/TOOLS.md"
    python3 - "$tools_md_fb" <<'PY'
import sys, os, re
tools_path = sys.argv[1]
section = """
## Google Drive and Docs (gog-agentic MCP)

**Do not say tools are unavailable or that the gateway may not be loading config. Do not read TOOLS.md or config files to "diagnose".** For any Google Drive or Docs request, **call the exec tool immediately** with one of these commands. Tool names use underscores (e.g. drive_listFiles, docs_create). You can use: `mcporter call gog-agentic.drive_listFiles --args '{}'` or `mcporter call --server gog-agentic --tool drive_listFiles --args '{}'`.

- **List Drive root (files and folders):** `mcporter call gog-agentic.drive_listFiles --args '{}' --output json`
- **List all accessible Drive files (global):** `mcporter call gog-agentic.drive_listFiles --args '{"global":true,"maxResults":20}' --output json` (cannot combine `global:true` with `parentId`)
- **List only folders (use when user asks for "folders" or "all folders"):** `mcporter call gog-agentic.drive_searchFiles --args '{\"query\":\"mimeType = \\\"application/vnd.google-apps.folder\\\"\",\"rawQuery\":true}' --output json` — returns one page + nextPageToken. **For "how many folders in root" use one call with fetchAllPages:** `--args '{\"query\":\"mimeType = \\\"application/vnd.google-apps.folder\\\" and \\\"root\\\" in parents\",\"rawQuery\":true,\"fetchAllPages\":true}'` — response includes totalCount. Tool names use **underscores** (drive_searchFiles not drive.searchFiles). When user asks for "first N", add \"maxResults\": N. To get all folders page-by-page, call again with \"page\": \"<nextPageToken>\" until no nextPageToken.
- **Create folder:** `mcporter call gog-agentic.drive_ensureFolder --args '{"path":"FolderName"}' --output json`
- **Create doc:** `mcporter call gog-agentic.docs_create --args '{"title":"Doc Title"}' --output json` (add `"parentId":"<folderId>"` to place doc in a folder)
- **Search files:** `mcporter call gog-agentic.drive_searchFiles --args '{"query":"name or text"}' --output json`
- **Get spreadsheet values:** `mcporter call gog-agentic.sheets_valuesGet --args '{\"spreadsheetId\":\"<id>\",\"range\":\"Sheet1!A1:D10\"}' --output json`

OAuth is already set up. If the exec call fails (e.g. command not found or error output), report the error and then suggest the user ask the workspace admin to run the diagnostic and restart the daemon and gateway (runbook §8.0). Never reveal the keyring password or credentials.

Never invent or assume folder or file names. Only report what the API returned. If you got only N items, say so and offer to fetch more with page/pageToken; do not make up names.

For "create folder then doc": run drive_ensureFolder first, then docs_create with the returned folderId as parentId.
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
    gog_agentic_upsert_mcporter_config "$default_mcp" "$gog_cmd"
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

  warn "No valid Google token found for MCP runtime. Requiring official gws onboarding..."
  if ! ensure_gws_export; then
    err "Official gws onboarding did not complete successfully."
    return 1
  fi

  if ! bootstrap_doctor_account_from_gws "$gog_cmd" ""; then
    err "Could not import official gws auth into gog for MCP runtime."
    return 1
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
    echo "Run: gws auth setup"
    echo "Then rerun: ./scripts/setup-doctor.sh"
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
