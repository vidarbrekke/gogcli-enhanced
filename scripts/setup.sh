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
clear_screen() { clear 2>/dev/null || printf '\033c'; }

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

ask_yes_no() {
  local prompt="$1"
  local default="${2:-y}" # y or n
  local suffix="[y/N]"
  [[ "$default" == "y" ]] && suffix="[Y/n]"

  while true; do
    clear_screen
    read -r -p "$prompt $suffix " ans
    ans="${ans:-$default}"
    case "${ans,,}" in
      y|yes) return 0 ;;
      n|no) return 1 ;;
      *) echo "Please answer y or n." ;;
    esac
  done
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
  echo "- Repo binary: $( [[ -x "$BIN_IN_REPO" ]] && echo "$BIN_IN_REPO" || echo "<missing>" )"
  echo "- Config dir: $CONFIG_DIR $( [[ -d "$CONFIG_DIR" ]] && echo "(exists)" || echo "(missing)" )"
  echo
}

main_menu() {
  while true; do
    clear_screen
    echo -e "${BOLD}Select setup mode${RESET}"
    echo "1) Fresh install"
    echo "2) Reinstall (preserve existing config)"
    echo "3) Reinstall (backup config + reset config)"
    echo "4) Reinstall (clean reset: remove config + installed binary)"
    read -r -p "Choice [1-4]: " MODE
    case "$MODE" in
      1|2|3|4) return 0 ;;
      *) echo "Please choose 1, 2, 3, or 4." ;;
    esac
  done
}

choose_install_target() {
  while true; do
    clear_screen
    echo -e "${BOLD}Install target${RESET}"
    echo "1) ~/.local/bin/gog (recommended, no sudo)"
    echo "2) /usr/local/bin/gog (system-wide, may require sudo)"
    read -r -p "Choose install target [1/2] (default 1): " choice
    choice="${choice:-1}"
    case "$choice" in
      1)
        INSTALL_TARGET="$HOME/.local/bin/gog"
        INSTALL_COMMAND_HINT="$HOME/.local/bin/gog"
        return 0
        ;;
      2)
        INSTALL_TARGET="/usr/local/bin/gog"
        INSTALL_COMMAND_HINT="gog"
        return 0
        ;;
      *) echo "Please choose 1 or 2." ;;
    esac
  done
}

ensure_path_hint() {
  local bindir
  bindir="$(dirname "$INSTALL_TARGET")"
  if [[ ":$PATH:" != *":$bindir:"* ]]; then
    warn "$bindir is not in PATH for this shell."
    if ask_yes_no "Append PATH export to ~/.bashrc?" y; then
      grep -Fq "export PATH=\"$bindir:\$PATH\"" "$HOME/.bashrc" 2>/dev/null || \
        echo "export PATH=\"$bindir:\$PATH\"" >> "$HOME/.bashrc"
      log "Updated ~/.bashrc"
      echo "Open a new shell (or run: source ~/.bashrc)."
    fi
  fi
}

check_dependencies() {
  local missing=()
  local required=(bash git make tar)

  for c in "${required[@]}"; do
    has_cmd "$c" || missing+=("$c")
  done
  if ! has_cmd curl && ! has_cmd wget; then
    missing+=("curl-or-wget")
  fi

  if [[ ${#missing[@]} -eq 0 ]]; then
    log "Dependencies look good."
    return 0
  fi

  warn "Missing dependencies: ${missing[*]}"

  if ! has_cmd apt-get; then
    err "apt-get not found; auto-install unavailable in this script."
    echo "Install missing packages manually, then rerun setup."
    exit 1
  fi

  if ask_yes_no "Install missing dependencies with apt-get now?" y; then
    local apt_pkgs=()
    for m in "${missing[@]}"; do
      case "$m" in
        bash|git|make|tar) apt_pkgs+=("$m") ;;
        curl-or-wget) apt_pkgs+=("curl") ;;
      esac
    done
    mapfile -t apt_pkgs < <(printf "%s\n" "${apt_pkgs[@]}" | awk '!seen[$0]++')

    log "Running: sudo apt-get update"
    sudo apt-get update
    log "Running: sudo apt-get install -y ${apt_pkgs[*]}"
    sudo apt-get install -y "${apt_pkgs[@]}"
  else
    echo "Install dependencies manually, then rerun ./scripts/setup.sh"
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

  [[ "$removed" == false ]] && log "No installed gog binary found to remove."
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
    3)
      log "Mode: Reinstall (backup + reset config)"
      backup_config_if_exists
      reset_config
      ;;
    4)
      log "Mode: Reinstall (clean reset)"
      backup_config_if_exists
      reset_config
      remove_installed_binary
      ;;
  esac
}

run_build() {
  log "Building gog via scripts/install.sh"
  "$ROOT_DIR/scripts/install.sh"
  [[ -x "$BIN_IN_REPO" ]] || { err "Build succeeded but $BIN_IN_REPO missing or not executable."; exit 1; }
}

copy_binary_to_target() {
  local target_dir
  target_dir="$(dirname "$INSTALL_TARGET")"

  if [[ "$target_dir" == "/usr/local/bin" ]]; then
    sudo mkdir -p "$target_dir"
    sudo cp "$BIN_IN_REPO" "$INSTALL_TARGET"
    sudo chmod +x "$INSTALL_TARGET"
  else
    mkdir -p "$target_dir"
    cp "$BIN_IN_REPO" "$INSTALL_TARGET"
    chmod +x "$INSTALL_TARGET"
  fi

  log "Installed binary to $INSTALL_TARGET"
}

persist_keyring_env_optional() {
  local pass="$1"
  if ask_yes_no "Persist keyring env vars to ~/.bashrc for future shells?" n; then
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

  if [[ -f "$keyring_file" ]]; then
    while true; do
      clear_screen
      warn "Detected existing encrypted keyring at $keyring_file"
      echo "1) Reuse existing keyring password"
      echo "2) Rotate/reset keyring (backup + remove keyring file)"
      echo "3) Skip password setup for now"
      read -r -p "Choose [1/2/3] (default 1): " kr_mode
      kr_mode="${kr_mode:-1}"
      case "$kr_mode" in
        1)
          if ask_yes_no "Enter existing keyring password now?" y; then
            clear_screen
            read -r -s -p "Existing keyring password: " p1
            echo
            export GOG_KEYRING_BACKEND=file
            export GOG_KEYRING_PASSWORD="$p1"
            log "Keyring password set for this setup session."
            persist_keyring_env_optional "$p1"
          fi
          return 0
          ;;
        2)
          local backup_dir="$BACKUP_BASE/$TIMESTAMP-keyring"
          mkdir -p "$backup_dir"
          cp -a "$keyring_file" "$backup_dir/" || true
          rm -f "$keyring_file"
          log "Backed up old keyring to $backup_dir and removed current keyring file."
          break
          ;;
        3)
          warn "Skipped keyring password setup. You may be prompted later."
          return 0
          ;;
        *) echo "Please choose 1, 2, or 3." ;;
      esac
    done
  fi

  if ask_yes_no "Create new keyring password now?" y; then
    clear_screen
    read -r -s -p "New keyring password: " p1
    echo
    clear_screen
    read -r -s -p "Confirm keyring password: " p2
    echo

    if [[ -z "${p1:-}" ]]; then
      warn "Empty password; skipped env setup."
    elif [[ "$p1" != "$p2" ]]; then
      warn "Passwords did not match; skipped env setup."
    else
      export GOG_KEYRING_BACKEND=file
      export GOG_KEYRING_PASSWORD="$p1"
      log "Keyring password set for this setup session."
      persist_keyring_env_optional "$p1"
    fi
  fi
}

setup_auth_optional() {
  local gog_cmd="$INSTALL_TARGET"
  [[ -x "$gog_cmd" ]] || gog_cmd="$(command -v gog 2>/dev/null || true)"
  [[ -x "$gog_cmd" ]] || gog_cmd="$BIN_IN_REPO"

  if ! ask_yes_no "Do you want to configure auth now (credentials, token storage, account)?" n; then
    return 0
  fi
  DID_CONFIGURE_AUTH=1

  clear_screen
  echo "Auth setup quick guide:"
  echo "1) Add OAuth client credentials"
  echo "2) Configure token storage (recommended before account auth)"
  echo "3) Authorize account"
  echo

  if ask_yes_no "Add or replace OAuth credentials now?" n; then
    while true; do
      clear_screen
      echo "Credential input method:"
      echo "1) Paste OAuth Client ID + Client Secret (recommended)"
      echo "2) Use existing OAuth client JSON file path"
      read -r -p "Choose [1/2] (default 1): " cred_mode
      cred_mode="${cred_mode:-1}"

      case "$cred_mode" in
        1)
          clear_screen
          read -r -p "OAuth Client ID: " oauth_client_id
          clear_screen
          read -r -s -p "OAuth Client Secret (input hidden): " oauth_client_secret
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
          break
          ;;
        2)
          clear_screen
          read -r -p "Path to OAuth client JSON: " cred_path
          if [[ -n "${cred_path:-}" && -f "$cred_path" ]]; then
            "$gog_cmd" auth credentials "$cred_path"
            DID_STORE_CREDENTIALS=1
            log "Stored OAuth credentials."
          else
            warn "File not found; credentials not stored."
          fi
          break
          ;;
        *) echo "Please choose 1 or 2." ;;
      esac
    done
  fi

  if ask_yes_no "Configure secure token storage now (recommended)?" y; then
    while true; do
      clear_screen
      echo "Where should gog store encrypted tokens?"
      echo "1) file (recommended on servers): encrypted file at $CONFIG_DIR/keyring"
      echo "2) keychain (OS keyring, if supported)"
      read -r -p "Choose storage [1/2] (default 1): " storage
      storage="${storage:-1}"
      case "$storage" in
        1)
          "$gog_cmd" auth keyring file
          setup_file_keyring_password
          break
          ;;
        2)
          "$gog_cmd" auth keyring keychain
          break
          ;;
        *) echo "Please choose 1 or 2." ;;
      esac
    done
  fi

  if ask_yes_no "Add/authorize a Google account now?" n; then
    clear_screen
    read -r -p "Account email: " account_email
    if [[ -n "${account_email:-}" ]]; then
      while true; do
        clear_screen
        echo "Auth flow mode:"
        echo "1) Manual (recommended for servers/headless)"
        echo "2) Remote split (step 1 + step 2)"
        echo "3) Local browser callback"
        read -r -p "Choose auth flow [1/2/3] (default 1): " auth_flow
        auth_flow="${auth_flow:-1}"

        case "$auth_flow" in
          1)
            "$gog_cmd" auth add "$account_email" --services user --manual
            DID_AUTHORIZE_ACCOUNT=1
            break
            ;;
          2)
            "$gog_cmd" auth add "$account_email" --services user --remote --step 1
            clear_screen
            echo "Open the URL printed above in your browser, approve access, then paste full redirect URL below."
            read -r -p "Full redirect URL: " auth_url
            if [[ -n "${auth_url:-}" ]]; then
              "$gog_cmd" auth add "$account_email" --services user --remote --step 2 --auth-url "$auth_url"
              DID_AUTHORIZE_ACCOUNT=1
            else
              warn "No redirect URL provided; remote step 2 skipped."
            fi
            break
            ;;
          3)
            "$gog_cmd" auth add "$account_email"
            DID_AUTHORIZE_ACCOUNT=1
            break
            ;;
          *) echo "Please choose 1, 2, or 3." ;;
        esac
      done
    fi
  fi
}

write_mcp_template_optional() {
  local gog_cmd="$INSTALL_TARGET"
  [[ -x "$gog_cmd" ]] || gog_cmd="gog"

  if ! ask_yes_no "Create a minimal MCP client config template now?" y; then
    return 0
  fi

  local default_template="$CONFIG_DIR/mcp-client-template.json"

  while true; do
    clear_screen
    echo "This creates a starter JSON file your MCP client can import/copy from."
    echo "- Press Enter to use default path"
    echo "- Or enter a full custom path ending in .json"
    echo "- Existing file at that path will be overwritten"
    read -r -p "Template file path [$default_template]: " template_path
    template_path="${template_path:-$default_template}"

    if [[ "$template_path" =~ ^[0-9]+$ ]]; then
      warn "That looks like a menu choice, not a file path. Try again."
      continue
    fi
    if [[ "$template_path" != *.json ]]; then
      warn "Template path should end with .json"
      continue
    fi
    break
  done

  mkdir -p "$(dirname "$template_path")"
  cat > "$template_path" <<EOF
{
  "mcpServers": {
    "gog": {
      "command": "$gog_cmd",
      "args": ["mcp", "serve"]
    }
  }
}
EOF

  log "Wrote MCP template: $template_path"
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
    echo "- Auth not configured in this run. Rerun setup and follow: credentials -> token storage -> account auth"
  else
    [[ "$DID_STORE_CREDENTIALS" -eq 0 ]] && \
      echo "- OAuth credentials not added in this run. Add with: ${INSTALL_COMMAND_HINT:-$INSTALL_TARGET} auth credentials <path-to-json>"
    [[ "$DID_AUTHORIZE_ACCOUNT" -eq 0 ]] && \
      echo "- Account authorization not completed in this run. Recommended on server: ${INSTALL_COMMAND_HINT:-$INSTALL_TARGET} auth add <you@gmail.com> --services user --manual"
  fi

  echo
  echo "What this tool does:"
  echo "gog is a Google Workspace CLI + optional MCP server."
  echo "It lets you script Gmail, Calendar, Drive, Docs, Sheets, Slides, Tasks, Contacts, and more from terminal or agents."
  echo
  echo "How to use it:"
  echo "- CLI: run commands like '${INSTALL_COMMAND_HINT:-$INSTALL_TARGET} gmail labels list' or '${INSTALL_COMMAND_HINT:-$INSTALL_TARGET} drive ls'"
  echo "- MCP: run '${INSTALL_COMMAND_HINT:-$INSTALL_TARGET} mcp serve' and point your MCP client to that command"
}

require_repo_layout
print_state
main_menu
choose_install_target
check_dependencies
run_mode_actions
run_build
copy_binary_to_target
ensure_path_hint
setup_auth_optional
write_mcp_template_optional
verify_install
print_completion_summary
