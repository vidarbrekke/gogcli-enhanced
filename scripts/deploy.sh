#!/usr/bin/env bash
# deploy.sh — After git pull: build gog, copy binary to PATH location, restart mcporter daemon.
# Run from repo root (e.g. on Linode). Use when the repo is already set up and you only need to update the binary and daemon.
#
# Usage:
#   ./scripts/deploy.sh
#   WORKSPACE_DIR=/path/to/workspace ./scripts/deploy.sh   # mcporter config path (default: parent of repositories/ or repo root)
#   ./scripts/deploy.sh --no-pull                          # Skip git pull (already pulled)
#   RESTART_GATEWAY=1 ./scripts/deploy.sh                  # Also restart openclaw-gateway (systemd user)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

DO_PULL=1
for arg in "$@"; do
  case "$arg" in
    --no-pull) DO_PULL=0 ;;
    -h|--help)
      echo "Usage: $0 [--no-pull]"
      echo "  WORKSPACE_DIR=/path  Set mcporter config directory (default: auto-detect from repo path)."
      echo "  RESTART_GATEWAY=1    Also restart openclaw-gateway after daemon restart."
      exit 0
      ;;
  esac
done

log() { echo "[deploy] $*"; }

# Ensure scripts we call are executable (e.g. if clone/copy didn't preserve execute bits).
chmod +x "$ROOT_DIR/scripts/install.sh" "$ROOT_DIR/scripts/ensure-mcp-daemon.sh" 2>/dev/null || true

if [[ "$DO_PULL" -eq 1 ]]; then
  log "Running git pull..."
  git pull
fi

log "Building gog..."
./scripts/install.sh

# Copy binary so it's updated even when ~/.local/bin/gog is a real file (install.sh only symlinks when target is missing or already a symlink).
INSTALL_TARGET="${HOME}/.local/bin/gog"
if [[ -x "$ROOT_DIR/bin/gog" ]]; then
  mkdir -p "$(dirname "$INSTALL_TARGET")"
  cp -f "$ROOT_DIR/bin/gog" "$INSTALL_TARGET"
  log "Copied binary to $INSTALL_TARGET"
fi

log "Restarting mcporter daemon..."
WORKSPACE_DIR="${WORKSPACE_DIR:-}" ./scripts/ensure-mcp-daemon.sh

log "Deploy done."
