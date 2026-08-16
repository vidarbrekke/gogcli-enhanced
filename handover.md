# Developer Handover — gogcli-enhanced

This file is the canonical developer handover. Supporting handover files must only point here.

## 1. Purpose and current problem

`gogcli-enhanced` is a Go CLI and MCP control plane for Google Workspace. It exposes native Google API operations through `gog`, adds deterministic JSON/error contracts and agent-safe edit workflows, and registers the `gog-agentic` MCP server for OpenClaw. The current problem is maintaining those contracts and safety controls while selectively adopting useful capabilities from the official `gws` CLI and the upstream `openclaw/gogcli` project.

Live agent requests use `gog-agentic` and native APIs by default. `GOG_BACKEND=gws` is an optional backend switch inside `gog`; it is not a second MCP server. Writes and safety-critical workflows remain native. Upstream changes must be ported surgically because this fork contains custom MCP, gws routing, parity, OpenClaw deployment, sedmat, and agentic edit code.

## 2. Completed work and outcomes

The parity foundation is complete: `cmd/gog-parity` and `internal/parity/` load fixtures, classify errors from stdout/stderr/exit status, normalize provider errors, validate schemas, and distinguish breaking changes from drift. `google_reason`, free-form messages, and unordered list output are non-contractual unless explicitly documented. The 401 and 404 cases are hard-gated; the 403 case remains soft until a real fixture replaces its placeholder.

Optional live gws routing is implemented for Gmail labels list/get and Drive list/get/search. Drive gws routing is intentionally limited: list/search are single-page paths, `drive ls --global`, `--all`, search `--all`, and get `--page-count` remain native. The routing implementation is in `internal/backend/gws/`, `internal/cmd/backend.go`, `internal/cmd/backend_error.go`, `internal/cmd/gmail_labels.go`, and `internal/cmd/drive.go`.

OpenClaw setup is operational: setup/deploy scripts install `gog`, import official `gws` authentication, register `gog-agentic`, sync the tracked skill, inject concise tool guidance, and manage the mcporter daemon. Agents should prefer MCP via `gog-agentic-call`; raw `gog` is only for commands without an MCP tool, such as auth status.

The August 2026 upstream assessment compared this fork with `openclaw/gogcli` at `eb85a993` (merge-base `0ed89978`). A full merge was rejected because upstream was approximately 1,179 commits ahead and had major overlapping rewrites. HTTP hardening from upstream issues #995/#998 was adapted and landed: outbound tracking requests have bounded response-header waits, and Gmail watch/OAuth callback servers have bounded read, idle, and header limits. `make ci` passed for that port.

## 3. Failures, open issues, and lessons

- Do not merge `openclaw/main` wholesale. Port one bounded feature or fix per PR and preserve local contracts.
- Do not cherry-pick revoked-token recovery commit `229da128`. It depends on upstream `AuthDependencies`, `persistingTokenSource`, and `app.Runtime`, which this fork does not have. A future implementation must be adapted to the current client/transport stack. MCP and gws execution must remain non-interactive and return a clear reauthorization error instead of opening a browser.
- Capture a real Gmail 403 fixture using `docs/merge/CAPTURE-403-RUNBOOK.md`, then promote it from soft to hard parity gating.
- Live gws routing has unit/contract coverage but still needs authenticated smoke coverage for every routed command and native rollback path.
- gws may emit structured errors on stdout. Classification must inspect exit status, stderr, and stdout. Never parse free-form message text for behavior.
- Google Drive has no PDF page-count metadata field. Current page-count resolution is out-of-band and may download/parse content; preserve explicit status/source/confidence output.
- `drive ls --all` means pagination in this fork; `--global` means all accessible files. Do not adopt upstream’s conflicting `--all` semantics.
- `make ci` includes `git diff --exit-code` in `fmt-check`; run it after committing or use `make fmt`, `make lint`, and `make test` while changes are uncommitted.

## 4. Files changed, insights, and gotchas

Recent upstream-sync changes are documented in `docs/merge/OPENCLAW-SYNC-2026-08.md`. HTTP client changes are localized to `internal/googleapi/client.go`, `internal/cmd/http_client.go`, Gmail tracking/watch code, and `internal/googleauth/oauth_flow.go`. Preserve the transport-level `ResponseHeaderTimeout`; do not add a global `http.Client.Timeout` that could terminate large downloads after response headers arrive.

The native/gws switch is command-specific, not automatic capability discovery. New routed commands require argument mapping, output normalization, error mapping, tests proving native services are not called, authenticated smoke tests, and a tested `GOG_BACKEND=native` rollback. Do not route Tier B/C writes through gws without explicit parity and safety approval.

`docs/TOOLS-gog-agentic-section.md`, `openclaw/skills/gog/SKILL.md`, and the injected section in `scripts/setup-doctor.sh` describe the same agent behavior. Update all three when tool-routing instructions change. Keep credential values out of docs, logs, chat, fixtures, and commits.

## 5. Key files and directories

- `AGENTS.md` — repository conventions, testing, commit, and PR rules.
- `cmd/gog/` — main CLI entrypoint.
- `internal/cmd/` — commands, backend selection, output contracts, and validation.
- `internal/mcp/` — `gog-agentic` MCP server and Google tool providers.
- `internal/backend/gws/` — external `gws` adapter.
- `cmd/gog-parity/`, `internal/parity/` — fixture-only parity runner and normalization.
- `docs/merge/` — routing policy, schemas, fixtures, upstream assessments, and runbooks.
- `docs/TOOLS-gog-agentic-section.md` — canonical OpenClaw tool instructions.
- `openclaw/skills/gog/SKILL.md` — tracked OpenClaw skill.
- `scripts/setup.sh`, `scripts/setup-doctor.sh`, `scripts/deploy.sh` — onboarding and deployment.
- `docs/openclaw-linode-runbook.md` — headless OpenClaw operations and troubleshooting.

## Next phase

1. Add authenticated smoke tests for Gmail labels and Drive list/get/search under both `GOG_BACKEND=gws` and `GOG_BACKEND=native`.
2. Capture and hard-gate the real Gmail 403 parity fixture.
3. Continue upstream ports as separate PRs. Prioritize Gmail reply/draft/attachment workflows, Drive conditional replacement/sync, and Calendar date-window/timezone fixes. Keep Docs/Slides rewrites, module rename, Go 1.26 migration, and revoked-token recovery as separate design decisions.
4. For every PR, run `make parity` when provider contracts change and `make ci` before merge.
