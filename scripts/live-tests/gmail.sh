#!/usr/bin/env bash

set -euo pipefail

run_gmail_tests() {
  if skip "gmail"; then
    echo "==> gmail (skipped)"
    return 0
  fi

  run_required "gmail" "gmail labels list" gog gmail labels list --json >/dev/null
  run_required "gmail" "gmail labels get" gog gmail labels get INBOX --json >/dev/null

  if ! skip "gmail-settings"; then
    run_required "gmail" "gmail settings sendas list" gog gmail settings sendas list --json >/dev/null
    run_required "gmail" "gmail settings vacation get" gog gmail settings vacation get --json >/dev/null
    run_required "gmail" "gmail settings filters list" gog gmail settings filters list --json >/dev/null
    if is_consumer_account "$ACCOUNT"; then
      echo "==> gmail delegates (skipped; Workspace/SA only)"
    else
      run_optional "gmail-delegates" "gmail settings delegates list" gog gmail settings delegates list --json >/dev/null
    fi
    run_required "gmail" "gmail settings forwarding list" gog gmail settings forwarding list --json >/dev/null
    run_required "gmail" "gmail settings autoforward get" gog gmail settings autoforward get --json >/dev/null
  fi

  local draft_json draft_id sent_draft_json sent_draft_msg_id
  draft_json=$(gog gmail drafts create --to "$EMAIL_TEST" --subject "gogcli smoke draft $TS" --body "smoke draft" --json)
  draft_id=$(extract_field "$draft_json" draftId)
  [ -n "$draft_id" ] || { echo "Failed to parse draft id" >&2; exit 1; }
  run_required "gmail" "gmail drafts get" gog gmail drafts get "$draft_id" --json >/dev/null
  run_required "gmail" "gmail drafts update" gog gmail drafts update "$draft_id" --subject "gogcli smoke draft updated $TS" --body "updated" --json >/dev/null
  sent_draft_json=$(gog gmail drafts send "$draft_id" --json)
  sent_draft_msg_id=$(extract_field "$sent_draft_json" messageId)
  [ -n "$sent_draft_msg_id" ] || { echo "Failed to parse sent draft message id" >&2; exit 1; }

  local body_file send_json send_msg_id send_thread_id
  body_file="$LIVE_TMP/gmail-body-$TS.txt"
  printf "hello from gogcli %s\n" "$TS" >"$body_file"
  send_json=$(gog gmail send --to "$EMAIL_TEST" --subject "gogcli smoke send $TS" --body-file "$body_file" --json)
  send_msg_id=$(extract_field "$send_json" messageId)
  send_thread_id=$(extract_field "$send_json" threadId)
  [ -n "$send_msg_id" ] || { echo "Failed to parse send message id" >&2; exit 1; }

  run_required "gmail" "gmail get message" gog gmail get "$send_msg_id" --format metadata --json >/dev/null
  if [ -n "$send_thread_id" ]; then
    run_required "gmail" "gmail thread get" gog gmail thread get "$send_thread_id" --json >/dev/null
    run_required "gmail" "gmail thread modify add label" gog gmail thread modify "$send_thread_id" --add STARRED --json >/dev/null
    run_required "gmail" "gmail thread modify remove label" gog gmail thread modify "$send_thread_id" --remove STARRED --json >/dev/null
  fi

  run_required "gmail" "gmail search" gog gmail search "subject:gogcli smoke send $TS" --json >/dev/null
  run_required "gmail" "gmail batch modify add" gog gmail batch modify "$send_msg_id" --add STARRED --json >/dev/null
  run_required "gmail" "gmail batch modify remove" gog gmail batch modify "$send_msg_id" --remove STARRED --json >/dev/null

  if [ -z "${GOG_LIVE_GMAIL_BATCH_DELETE:-}" ] || skip "gmail-batch-delete"; then
    echo "==> gmail batch delete (skipped)"
  else
    echo "==> gmail batch delete"
    if gog gmail batch delete "$send_msg_id" "$sent_draft_msg_id" --json >/dev/null; then
      :
    else
      echo "gmail batch delete failed; falling back to trash" >&2
      gog gmail batch modify "$send_msg_id" "$sent_draft_msg_id" --add TRASH --json >/dev/null || true
      if [ "${STRICT:-false}" = true ]; then
        return 1
      fi
    fi
  fi

  if [ -n "${GOG_LIVE_TRACK:-}" ]; then
    run_optional "gmail-track" "gmail send --track" gog gmail send --to "$EMAIL_TEST" --subject "gogcli smoke track $TS" --body-html "<p>track $TS</p>" --track --json >/dev/null
  fi
}
