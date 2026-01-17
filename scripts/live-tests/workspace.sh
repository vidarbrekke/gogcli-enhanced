#!/usr/bin/env bash

set -euo pipefail

run_workspace_tests() {
  if is_consumer_account "$ACCOUNT"; then
    echo "==> groups (skipped; Workspace only)"
  else
    run_optional "groups" "groups list" gog groups list --json --max 5 >/dev/null
    if [ -n "${GOG_LIVE_GROUP_EMAIL:-}" ]; then
      run_optional "groups" "groups members" gog groups members "$GOG_LIVE_GROUP_EMAIL" --json --max 5 >/dev/null
    fi
  fi

  if skip "keep"; then
    echo "==> keep (skipped)"
    return 0
  fi

  if is_consumer_account "$ACCOUNT"; then
    echo "==> keep (skipped; Workspace only)"
    return 0
  fi

  if [ -z "${GOG_KEEP_SERVICE_ACCOUNT:-}" ] || [ -z "${GOG_KEEP_IMPERSONATE:-}" ]; then
    if [ "${STRICT:-false}" = true ]; then
      echo "Missing GOG_KEEP_SERVICE_ACCOUNT/GOG_KEEP_IMPERSONATE for keep tests." >&2
      return 1
    fi
    echo "==> keep (optional; set GOG_KEEP_SERVICE_ACCOUNT and GOG_KEEP_IMPERSONATE)"
    return 0
  fi

  run_optional "keep" "keep list" gog keep list --service-account "$GOG_KEEP_SERVICE_ACCOUNT" --impersonate "$GOG_KEEP_IMPERSONATE" --json --max 5 >/dev/null
  run_optional "keep" "keep search" gog keep search "gogcli" --service-account "$GOG_KEEP_SERVICE_ACCOUNT" --impersonate "$GOG_KEEP_IMPERSONATE" --json >/dev/null
}
