# Remaining upstream work

Latest assessment: August 2026

Upstream: `openclaw/gogcli` (`openclaw/main` at `eb85a993`)

Assessment details: [`OPENCLAW-SYNC-2026-08.md`](OPENCLAW-SYNC-2026-08.md)

The fork and upstream diverged after merge-base `0ed89978`. Upstream was approximately 1,179 commits ahead at assessment time. Do not merge upstream wholesale: this fork has overlapping MCP, gws routing, parity, OpenClaw deployment, sedmat, and agentic edit implementations.

## Completed

- Earlier low-risk upstream sync and Docs paragraph reconciliation.
- Native/gws parity runner and optional live routing for Gmail labels and Drive list/get/search.
- August HTTP hardening port for outbound tracking, Gmail watch, and OAuth callback servers.
- Existing fork behavior retained for `drive ls --all` (pagination) and `--global` (all accessible files).

## Next ports

Handle each item in a separate PR with tests and `make ci`:

1. Gmail reply/draft/attachment workflows.
2. Drive conditional replacement and recursive sync.
3. Calendar date-window and display-timezone fixes.
4. MCP capability policies, adapted to this fork’s command allowlist and OpenClaw runtime.

## Deferred design decisions

- Revoked-token recovery commit `229da128`: requires a fork-shaped implementation; do not cherry-pick.
- Module rename from `github.com/steipete/gogcli` to `github.com/openclaw/gogcli`.
- Go 1.26/toolchain migration.
- Broad Docs/Slides/Sheets rewrites and new YouTube/Photos/backup surfaces.

## Provider follow-up

- Add authenticated smoke tests for every `GOG_BACKEND=gws` route and native rollback.
- Capture a real Gmail 403 fixture and promote it to hard parity gating.
- Preserve `google_reason` and free-form messages as drift-only data.
- Keep Tier B/C writes native until parity and safety gates are explicitly approved.
