#!/usr/bin/env bash
# deploy.sh — single entrypoint for first-time bootstrap and ongoing updates.
# Run from repo root (e.g. on Linode). It can pull latest code, install missing
# dependencies, bootstrap first-time setup, build gog, and restart the MCP daemon.
#
# Usage:
#   ./scripts/deploy.sh
#   WORKSPACE_DIR=/path/to/workspace ./scripts/deploy.sh   # mcporter config path (default: parent of repositories/ or repo root)
#   ./scripts/deploy.sh --no-pull                          # Skip git pull (already pulled)
#   ./scripts/deploy.sh --yes                              # Non-interactive (skip confirmation prompts where possible)
#   RESTART_GATEWAY=1 ./scripts/deploy.sh                  # Also restart openclaw-gateway (systemd user)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/scripts/lib/gws-auth-bridge.sh"

OS_NAME="$(uname -s)"
if [[ "$OS_NAME" != "Linux" && "$OS_NAME" != "Darwin" ]]; then
  echo "[deploy] ERROR: deploy.sh supports Ubuntu/Linux and macOS only." >&2
  exit 1
fi

DO_PULL=1
NON_INTERACTIVE=0
for arg in "$@"; do
  case "$arg" in
    --no-pull) DO_PULL=0 ;;
    --yes|--non-interactive) NON_INTERACTIVE=1 ;;
    -h|--help)
      echo "Usage: $0 [--no-pull] [--yes]"
      echo "  --no-pull            Skip git pull (e.g. already pulled)."
      echo "  --yes, --non-interactive  Non-interactive; skip confirmation prompts (for automation)."
      echo "                         If the working tree has local changes, stashes them before pull and restores after."
      echo "  WORKSPACE_DIR=/path  Set mcporter config directory (default: auto-detect from repo path)."
      echo "  RESTART_GATEWAY=1    Also restart openclaw-gateway after daemon restart."
      exit 0
      ;;
  esac
done

log() { echo "[deploy] $*"; }
warn() { echo "[deploy] WARN: $*" >&2; }
err() { echo "[deploy] ERROR: $*" >&2; }
has_cmd() { command -v "$1" >/dev/null 2>&1; }

apt_install() {
  case "$OS_NAME" in
    Linux)
      if ! has_cmd apt-get; then
        err "apt-get not available. Install missing dependencies manually and rerun."
        exit 1
      fi
      if has_cmd sudo; then
        sudo apt-get update
        sudo apt-get install -y "$@"
      else
        apt-get update
        apt-get install -y "$@"
      fi
      ;;
    Darwin)
      if ! has_cmd brew; then
        err "Homebrew is required on macOS. Install Homebrew and rerun."
        exit 1
      fi
      brew install "$@"
      ;;
  esac
}

ensure_system_dependencies() {
  local missing=()
  for c in bash git make tar python3 jq; do
    has_cmd "$c" || missing+=("$c")
  done
  if ! has_cmd curl && ! has_cmd wget; then
    missing+=("curl")
  fi
  if ! has_cmd npm; then
    if [[ "$OS_NAME" == "Linux" ]]; then
      missing+=("npm")
    else
      missing+=("node")
    fi
  fi
  if ! has_cmd node; then
    if [[ "$OS_NAME" == "Linux" ]]; then
      missing+=("nodejs")
    else
      missing+=("node")
    fi
  fi
  if ! has_cmd pdfinfo; then
    if [[ "$OS_NAME" == "Linux" ]]; then
      missing+=("poppler-utils")
    else
      missing+=("poppler")
    fi
  fi
  if [[ ${#missing[@]} -gt 0 ]]; then
    log "Installing missing system dependencies: ${missing[*]}"
    apt_install "${missing[@]}"
  fi
}

ensure_npm_tool() {
  local cmd_name="$1"
  local pkg_name="$2"
  if has_cmd "$cmd_name"; then
    return 0
  fi
  log "Installing missing npm tool: $pkg_name"
  npm install -g "$pkg_name"
  if ! has_cmd "$cmd_name"; then
    err "$cmd_name is still unavailable after installing $pkg_name"
    exit 1
  fi
}

resolve_workspace_dir() {
  if [[ -n "${WORKSPACE_DIR:-}" ]]; then
    cd "$WORKSPACE_DIR" >/dev/null 2>&1 && pwd
  elif [[ "$ROOT_DIR" == *"/repositories/"* ]]; then
    printf '%s\n' "${ROOT_DIR%/repositories/*}"
  else
    printf '%s\n' "${OPENCLAW_WORKSPACE:-$ROOT_DIR}"
  fi
}

copy_built_binary() {
  local install_target="${HOME}/.local/bin/gog"
  local binary="$ROOT_DIR/bin/gog"
  if [[ ! -x "$binary" ]]; then
    err "Expected built binary at $binary"
    exit 1
  fi
  mkdir -p "$(dirname "$install_target")"
  local do_copy=1
  if [[ -L "$install_target" ]]; then
    local canon_target canon_binary
    canon_target=$(readlink -f "$install_target" 2>/dev/null || realpath "$install_target" 2>/dev/null || true)
    canon_binary=$(readlink -f "$binary" 2>/dev/null || realpath "$binary" 2>/dev/null || true)
    if [[ -n "$canon_target" && -n "$canon_binary" && "$canon_target" == "$canon_binary" ]]; then
      do_copy=0
      log "Binary already linked at $install_target (skipping copy)"
    fi
  fi
  if [[ "$do_copy" -eq 1 ]]; then
    cp -f "$binary" "$install_target"
    log "Copied binary to $install_target"
  fi
}

bootstrap_needed() {
  local workspace_dir="$1"
  local gog_config_dir="${XDG_CONFIG_HOME:-$HOME/.config}/gogcli"
  local mcporter_config="$workspace_dir/config/mcporter.json"

  [[ -f "$gog_config_dir/credentials.json" ]] || return 0
  [[ -f "$mcporter_config" ]] || return 0
  return 1
}

# Ensure scripts we call are executable (e.g. if clone/copy didn't preserve execute bits).
chmod +x "$ROOT_DIR/scripts/install.sh" "$ROOT_DIR/scripts/ensure-mcp-daemon.sh" "$ROOT_DIR/scripts/setup-doctor.sh" 2>/dev/null || true

STASH_POP=0
if [[ "$DO_PULL" -eq 1 ]]; then
  status_line=$(git status -s 2>/dev/null || true)
  if [[ -n "$status_line" ]]; then
    log "Pre-pull git status: $status_line"
    if [[ "$NON_INTERACTIVE" -eq 1 ]]; then
      log "Stashing local changes so pull can proceed..."
      git stash push -m "deploy.sh pre-pull $(date +%Y%m%d-%H%M%S)"
      STASH_POP=1
    else
      log "ERROR: Working tree has local changes; git pull would fail or overwrite them."
      echo "  Commit, stash, or discard changes, then re-run. Or run with --no-pull after stashing:"
      echo "    git stash && ./scripts/deploy.sh --no-pull"
      echo "  To stash automatically in non-interactive mode, use: ./scripts/deploy.sh --yes"
      exit 1
    fi
  else
    log "Pre-pull git status: working tree clean"
  fi
  log "Running git pull..."
  git pull
fi

ensure_system_dependencies
ensure_npm_tool gws @googleworkspace/cli
ensure_npm_tool mcporter mcporter

WORKSPACE_DIR_RESOLVED="$(resolve_workspace_dir)"
if bootstrap_needed "$WORKSPACE_DIR_RESOLVED"; then
  log "First-time bootstrap detected; delegating to setup-doctor.sh"
  ./scripts/setup-doctor.sh --no-clear
else
  log "Building gog..."
  ./scripts/install.sh
  copy_built_binary

  log "Restarting mcporter daemon..."
  WORKSPACE_DIR="$WORKSPACE_DIR_RESOLVED" ./scripts/ensure-mcp-daemon.sh
fi

if [[ "${STASH_POP:-0}" -eq 1 ]]; then
  log "Restoring stashed local changes..."
  git stash pop || log "WARNING: stash pop had conflicts; resolve manually (git stash list)"
fi

if [[ "$OS_NAME" == "Darwin" ]]; then
  log "macOS note: if your OpenClaw gateway needs a restart, do it manually using your normal launch method."
fi

log "Deploy done."
