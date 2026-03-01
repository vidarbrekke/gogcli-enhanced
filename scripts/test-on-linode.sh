#!/usr/bin/env bash
# Run on Linode (or any Linux server) to test gogcli-enhanced: build, test, optional sed/MCP smoke.
# Usage:
#   REPO_DIR=~/openclaw-stock-home/.openclaw/workspace/repositories/gogcli-enhanced ./scripts/test-on-linode.sh
#   ./scripts/test-on-linode.sh   # uses REPO_DIR default below
# Optional env:
#   BRANCH=feature/upstream-sedmat-import   # test sedmat branch (default: main)
#   SKIP_SED_SMOKE=1   # skip "gog docs sed --help" and MCP list
set -euo pipefail

REPO_DIR="${REPO_DIR:-$HOME/openclaw-stock-home/.openclaw/workspace/repositories/gogcli-enhanced}"
BRANCH="${BRANCH:-main}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"

log() { echo "[test-on-linode] $*"; }
err() { echo "[test-on-linode] ERROR: $*" >&2; }

if [[ ! -d "$REPO_DIR" ]]; then
  err "Repo dir not found: $REPO_DIR"
  exit 1
fi

cd "$REPO_DIR"
log "Repo: $REPO_DIR (branch: $BRANCH)"

# Ensure we have Go and make
if ! command -v go &>/dev/null; then
  err "Go not found. Install Go and re-run."
  exit 1
fi
if ! command -v make &>/dev/null; then
  err "make not found. Install build-essential and re-run."
  exit 1
fi

# Fetch and checkout
git fetch origin
git checkout "$BRANCH"
git pull --ff-only origin "$BRANCH" || true
log "Build..."
make build
log "Tests..."
make test

# Optional smoke: sed help (only on branches that have it)
if [[ "${SKIP_SED_SMOKE:-0}" != "1" ]]; then
  if ./bin/gog docs sed --help &>/dev/null; then
    log "gog docs sed --help OK"
  else
    log "gog docs sed not present or failed (expected on main if sedmat only on feature branch)"
  fi
  # Quick MCP tools list (no auth needed)
  if ./bin/gog mcp tools list --json 2>/dev/null | head -1 >/dev/null; then
    log "gog mcp tools list OK"
  else
    log "gog mcp tools list skipped or failed (non-fatal)"
  fi
fi

log "Done. Build and tests passed."
