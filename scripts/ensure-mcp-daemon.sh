#!/usr/bin/env bash
# After git pull / deploy: ensure mcporter daemon is running so the OpenClaw gateway sees gog-agentic.
# Run from the repo (e.g. on Linode after deploy). Safe to run repeatedly.
#
# Usage:
#   ./scripts/ensure-mcp-daemon.sh
#   WORKSPACE_DIR=/path/to/workspace ./scripts/ensure-mcp-daemon.sh
# Optional: RESTART_GATEWAY=1 to also restart openclaw-gateway (systemd user on Linux).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
OS_NAME="$(uname -s)"

source "$ROOT_DIR/scripts/lib/gog-agentic-config.sh"

# Workspace: where config/mcporter.json lives (OpenClaw workspace, not repo root).
if [[ -n "${WORKSPACE_DIR:-}" ]]; then
  WORKSPACE_DIR="$(cd "$WORKSPACE_DIR" && pwd)"
elif [[ "$ROOT_DIR" == *"/repositories/"* ]]; then
  WORKSPACE_DIR="${ROOT_DIR%/repositories/*}"
else
  WORKSPACE_DIR="${OPENCLAW_WORKSPACE:-$ROOT_DIR}"
fi
MCPORTER_CONFIG="$WORKSPACE_DIR/config/mcporter.json"

log() { echo "[ensure-mcp-daemon] $*"; }
warn() { echo "[ensure-mcp-daemon] WARN: $*" >&2; }

if [[ ! -f "$MCPORTER_CONFIG" ]]; then
  warn "Config not found: $MCPORTER_CONFIG (run setup.sh first?)"
  exit 1
fi

GOG_CMD="$(command -v gog 2>/dev/null || true)"
if [[ -z "$GOG_CMD" || "$GOG_CMD" != /* ]]; then
  if [[ -x "$HOME/.local/bin/gog" ]]; then
    GOG_CMD="$HOME/.local/bin/gog"
  else
    GOG_CMD="$ROOT_DIR/bin/gog"
  fi
fi

if [[ ! -x "$GOG_CMD" ]]; then
  warn "gog binary not found for mcporter config refresh: $GOG_CMD"
  exit 1
fi

gog_agentic_upsert_mcporter_config "$MCPORTER_CONFIG" "$GOG_CMD"
log "gog-agentic config refreshed (backend=$(gog_agentic_backend))."

if [[ -x "$ROOT_DIR/scripts/mcp-diagnose-gog.sh" ]]; then
  if "$ROOT_DIR/scripts/mcp-diagnose-gog.sh" "$MCPORTER_CONFIG" >/tmp/gog-mcp-diagnose.out 2>/tmp/gog-mcp-diagnose.err; then
    log "gog MCP diagnostic passed."
  else
    warn "gog MCP diagnostic failed; see /tmp/gog-mcp-diagnose.err"
  fi
fi

if ! command -v mcporter &>/dev/null; then
  warn "mcporter not in PATH; cannot start daemon."
  exit 1
fi

mcporter --config "$MCPORTER_CONFIG" daemon restart
log "mcporter daemon restarted (config: $MCPORTER_CONFIG)."
mcporter --config "$MCPORTER_CONFIG" daemon status || true

if [[ "${RESTART_GATEWAY:-0}" == "1" ]]; then
  if [[ "$OS_NAME" == "Darwin" ]]; then
    warn "Gateway restart is not automated on macOS. Restart your OpenClaw gateway manually using your normal launch method."
  elif systemctl --user restart openclaw-gateway 2>/dev/null; then
    log "OpenClaw gateway restarted."
  else
    warn "Could not restart openclaw-gateway (not running as user with that service?)."
  fi
fi
