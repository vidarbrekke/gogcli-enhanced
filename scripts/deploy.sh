#!/usr/bin/env bash
# deploy.sh — After git pull: build gog, copy binary to PATH location, restart mcporter daemon.
# Run from repo root (e.g. on Linode). Use when the repo is already set up and you only need to update the binary and daemon.
#
# Usage:
#   ./scripts/deploy.sh
#   WORKSPACE_DIR=/path/to/workspace ./scripts/deploy.sh   # mcporter config path (default: parent of repositories/ or repo root)
#   ./scripts/deploy.sh --no-pull                          # Skip git pull (already pulled)
#   ./scripts/deploy.sh --yes                              # Non-interactive (skip any confirmation prompts; for CI/automation)
#   RESTART_GATEWAY=1 ./scripts/deploy.sh                  # Also restart openclaw-gateway (systemd user)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

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

# Ensure scripts we call are executable (e.g. if clone/copy didn't preserve execute bits).
chmod +x "$ROOT_DIR/scripts/install.sh" "$ROOT_DIR/scripts/ensure-mcp-daemon.sh" 2>/dev/null || true

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

log "Building gog..."
./scripts/install.sh

# Copy binary so it's updated even when ~/.local/bin/gog is a real file (install.sh only symlinks when target is missing or already a symlink).
# Skip cp when target is already a symlink to our binary (avoids "same file" / cp error).
INSTALL_TARGET="${HOME}/.local/bin/gog"
BINARY="$ROOT_DIR/bin/gog"
if [[ -x "$BINARY" ]]; then
  mkdir -p "$(dirname "$INSTALL_TARGET")"
  do_copy=1
  if [[ -L "$INSTALL_TARGET" ]]; then
    # Resolve both to canonical paths and compare (readlink -f on Linux, realpath on macOS).
    canon_target=$(readlink -f "$INSTALL_TARGET" 2>/dev/null || realpath "$INSTALL_TARGET" 2>/dev/null || true)
    canon_binary=$(readlink -f "$BINARY" 2>/dev/null || realpath "$BINARY" 2>/dev/null || true)
    if [[ -n "$canon_target" && -n "$canon_binary" && "$canon_target" == "$canon_binary" ]]; then
      do_copy=0
      log "Binary already linked at $INSTALL_TARGET (skipping copy)"
    fi
  fi
  if [[ "$do_copy" -eq 1 ]]; then
    cp -f "$BINARY" "$INSTALL_TARGET"
    log "Copied binary to $INSTALL_TARGET"
  fi
fi

log "Restarting mcporter daemon..."
WORKSPACE_DIR="${WORKSPACE_DIR:-}" ./scripts/ensure-mcp-daemon.sh

if [[ "${STASH_POP:-0}" -eq 1 ]]; then
  log "Restoring stashed local changes..."
  git stash pop || log "WARNING: stash pop had conflicts; resolve manually (git stash list)"
fi

log "Deploy done."
