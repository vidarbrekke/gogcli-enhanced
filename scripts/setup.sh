#!/usr/bin/env bash
# setup.sh — Interactive Linux setup/reinstall wizard for gogcli-enhanced
# - Linux only
# - Fresh install + reinstall modes
# - Dependency checks (optional apt install with permission)
# - Optional auth/keyring bootstrap
# - Optional minimal MCP config template generation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "This setup wizard currently supports Linux only." >&2
  exit 1
fi

BOLD="\033[1m"
DIM="\033[2m"
GREEN="\033[32m"
YELLOW="\033[33m"
RED="\033[31m"
RESET="\033[0m"

log() { echo -e "${GREEN}==>${RESET} $*"; }
warn() { echo -e "${YELLOW}Warning:${RESET} $*"; }
err() { echo -e "${RED}Error:${RESET} $*"; }

ask_yes_no() {
  local prompt="$1"
  local default="${2:-y}" # y or n
  local suffix="[y/N]"
  if [[ "$default" == "y" ]]; then
    suffix="[Y/n]"
  fi
  while true; do
    read -r -p "$prompt $suffix " ans
    ans="${ans:-$default}"
    case "${ans,,}" in
      y|yes) return 0 ;;
      n|no) return 1 ;;
      *) echo "Please answer y or n." ;;
    esac
  done
}

choose_install_target() {
  echo
  echo -e "${BOLD}Install target${RESET}"
  echo "1) ~/.local/bin (recommended; no sudo)"
  echo "2) /usr/local/bin (system-wide; may require sudo)"
  while true; do
    read -r -p "Choose install target [1/2] (default 1): " choice
    choice="${choice:-1}"
    case "$choice" in
      1) INSTALL_TARGET="$HOME/.local/bin/gog"; return 0 ;;
      2) INSTALL_TARGET="/usr/local/bin/gog"; return 0 ;;
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

CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/gogcli"
BACKUP_BASE="${XDG_CONFIG_HOME:-$HOME/.config}/gogcli-backups"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"

CURRENT_GOG="$(command -v gog 2>/dev/null || true)"
BIN_IN_REPO="$ROOT_DIR/bin/gog"

has_cmd() { command -v "$1" >/dev/null 2>&1; }

print_state() {
  echo
  echo -e "${BOLD}Detected state${RESET}"
  echo "- Repo root: $ROOT_DIR"
  echo "- Existing gog on PATH: ${CURRENT_GOG:-<none>}"
  echo "- Repo binary: $( [[ -x "$BIN_IN_REPO" ]] && echo "$BIN_IN_REPO" || echo "<missing>" )"
  echo "- Config dir: $CONFIG_DIR $( [[ -d "$CONFIG_DIR" ]] && echo "(exists)" || echo "(missing)" )"
}

require_repo_layout() {
  if [[ ! -f "$ROOT_DIR/go.mod" || ! -x "$ROOT_DIR/scripts/install.sh" ]]; then
    err "Could not find expected repo layout (go.mod + scripts/install.sh)."
    err "Run this script from the gogcli-enhanced repository."
    exit 1
  fi
}

check_dependencies() {
  local missing=()
  local required=(bash git make tar)
  local net_any=false

  for c in "${required[@]}"; do
    has_cmd "$c" || missing+=("$c")
  done

  if has_cmd curl || has_cmd wget; then
    net_any=true
  else
    missing+=("curl-or-wget")
  fi

  if [[ ${#missing[@]} -eq 0 ]]; then
    log "Dependencies look good."
    return 0
  fi

  warn "Missing dependencies: ${missing[*]}"

  if ! has_cmd apt-get; then
    warn "apt-get not found; auto-install unavailable."
    echo "Please install missing dependencies manually and re-run setup."
    exit 1
  fi

  if ask_yes_no "Install missing dependencies with apt-get now?" y; then
    local apt_pkgs=()
    local need_update=false

    for m in "${missing[@]}"; do
      case "$m" in
        bash|git|make|tar) apt_pkgs+=("$m") ;;
        curl-or-wget) apt_pkgs+=("curl") ;;
      esac
    done

    # dedupe apt package list
    mapfile -t apt_pkgs < <(printf "%s\n" "${apt_pkgs[@]}" | awk '!seen[$0]++')

    if [[ ${#apt_pkgs[@]} -gt 0 ]]; then
      log "Running: sudo apt-get update"
      sudo apt-get update
      log "Running: sudo apt-get install -y ${apt_pkgs[*]}"
      sudo apt-get install -y "${apt_pkgs[@]}"
    fi
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

  if [[ -n "$CURRENT_GOG" ]]; then
    candidate_paths+=("$CURRENT_GOG")
  fi

  mapfile -t candidate_paths < <(printf "%s\n" "${candidate_paths[@]}" | awk '!seen[$0]++')

  for p in "${candidate_paths[@]}"; do
    if [[ -f "$p" || -L "$p" ]]; then
      if [[ "$p" == "/usr/local/bin/gog" ]]; then
        log "Removing $p"
        sudo rm -f "$p"
      else
        log "Removing $p"
        rm -f "$p"
      fi
      removed=true
    fi
  done

  if [[ "$removed" == false ]]; then
    log "No installed gog binary found to remove."
  fi
}

reset_config() {
  if [[ -d "$CONFIG_DIR" ]]; then
    log "Removing config directory: $CONFIG_DIR"
    rm -rf "$CONFIG_DIR"
  else
    log "No existing config directory to remove."
  fi
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

run_build() {
  log "Building gog via scripts/install.sh"
  "$ROOT_DIR/scripts/install.sh"
  if [[ ! -x "$BIN_IN_REPO" ]]; then
    err "Build completed but $BIN_IN_REPO not found/executable."
    exit 1
  fi
}

setup_auth_optional() {
  local gog_cmd="$INSTALL_TARGET"
  [[ -x "$gog_cmd" ]] || gog_cmd="$(command -v gog 2>/dev/null || true)"
  [[ -x "$gog_cmd" ]] || gog_cmd="$BIN_IN_REPO"

  if ! ask_yes_no "Do you want to configure auth now (credentials/account/keyring)?" n; then
    return 0
  fi

  echo
  echo "Auth setup tips:"
  echo "- Preferred: paste Client ID + Client Secret (this wizard will generate JSON for you)."
  echo "- Alternative: provide path to an existing OAuth client JSON file."
  echo "- Existing credentials can be replaced any time with: gog auth credentials <path>"
  echo "- For cloud/headless environments, use manual or remote auth flow (not local callback)."

  if ask_yes_no "Add or replace OAuth credentials now?" n; then
    echo
    echo "Credential input method:"
    echo "1) Paste Client ID + Client Secret (recommended)"
    echo "2) Use existing OAuth client JSON file path"
    read -r -p "Choose [1/2] (default 1): " cred_mode
    cred_mode="${cred_mode:-1}"

    case "$cred_mode" in
      1)
        read -r -p "OAuth Client ID: " oauth_client_id
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
          log "Stored OAuth credentials (generated from pasted ID/secret)."
        else
          warn "Skipped: client ID/secret missing."
        fi
        ;;
      2)
        read -r -p "Path to OAuth client JSON: " cred_path
        if [[ -n "${cred_path:-}" && -f "$cred_path" ]]; then
          "$gog_cmd" auth credentials "$cred_path"
          log "Stored OAuth credentials."
        else
          warn "Skipped: file not found."
        fi
        ;;
      *)
        warn "Invalid choice; skipped credential setup."
        ;;
    esac
  fi

  if ask_yes_no "Add/authorize a Google account now (gog auth add ...)?" n; then
    read -r -p "Account email: " account_email
    if [[ -n "${account_email:-}" ]]; then
      echo
      echo "Auth flow mode:"
      echo "1) Manual (recommended for servers/headless)"
      echo "2) Remote split (step 1 + step 2)"
      echo "3) Local browser callback"
      read -r -p "Choose auth flow [1/2/3] (default 1): " auth_flow
      auth_flow="${auth_flow:-1}"

      case "$auth_flow" in
        1)
          "$gog_cmd" auth add "$account_email" --services user --manual
          ;;
        2)
          log "Starting remote auth step 1 (URL generation)."
          "$gog_cmd" auth add "$account_email" --services user --remote --step 1
          echo
          echo "Open the URL above in your browser, approve access, then paste full redirect URL below."
          read -r -p "Full redirect URL: " auth_url
          if [[ -n "${auth_url:-}" ]]; then
            "$gog_cmd" auth add "$account_email" --services user --remote --step 2 --auth-url "$auth_url"
          else
            warn "No auth URL provided; skipped remote step 2."
          fi
          ;;
        3)
          "$gog_cmd" auth add "$account_email"
          ;;
        *)
          warn "Invalid auth flow choice; defaulting to manual flow."
          "$gog_cmd" auth add "$account_email" --services user --manual
          ;;
      esac
    fi
  fi

  if ask_yes_no "Configure keyring backend now?" n; then
    echo "Choose backend: auto | file | keychain"
    read -r -p "Backend [auto]: " backend
    backend="${backend:-auto}"
    case "$backend" in
      auto|file|keychain)
        "$gog_cmd" auth keyring "$backend"
        ;;
      *)
        warn "Invalid backend; skipping keyring config."
        ;;
    esac
  fi
}

write_mcp_template_optional() {
  local gog_cmd="$INSTALL_TARGET"
  [[ -x "$gog_cmd" ]] || gog_cmd="gog"

  if ! ask_yes_no "Create a minimal MCP client config template now?" y; then
    return 0
  fi

  local default_template="$CONFIG_DIR/mcp-client-template.json"
  read -r -p "Template path [$default_template]: " template_path
  template_path="${template_path:-$default_template}"
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
  echo "Use this as a starting point for your MCP client config."
}

verify_install() {
  echo
  log "Verification"
  if [[ -x "$INSTALL_TARGET" ]]; then
    "$INSTALL_TARGET" --version || true
  elif has_cmd gog; then
    gog --version || true
  else
    warn "gog not found on PATH yet. If installed to ~/.local/bin, open a new shell."
  fi
}

main_menu() {
  echo
  echo -e "${BOLD}Select setup mode${RESET}"
  echo "1) Fresh install"
  echo "2) Reinstall (preserve existing config)"
  echo "3) Reinstall (backup config + reset config)"
  echo "4) Reinstall (clean reset: remove config + installed binary)"
  while true; do
    read -r -p "Choice [1-4]: " MODE
    case "$MODE" in
      1|2|3|4) return 0 ;;
      *) echo "Please choose 1, 2, 3, or 4." ;;
    esac
  done
}

run_mode_actions() {
  case "$MODE" in
    1)
      log "Mode: Fresh install"
      ;;
    2)
      log "Mode: Reinstall (preserve config)"
      ;;
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

echo
echo -e "${GREEN}Setup complete.${RESET}"
echo "Next steps:"
echo "- Run: gog --help"
echo "- If auth not configured: rerun setup and paste OAuth Client ID + Secret (or use gog auth credentials <path>)"
echo "- Then authorize account: gog auth add <you@gmail.com>"
