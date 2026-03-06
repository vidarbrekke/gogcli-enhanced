# Developer handover: next phase

**Quick start:** See **`README-NEXT-DEV.md`** in this folder for a one-page intro and next steps.

**Canonical handover:** repo-root `handover.md`. This file is a **concise brief** for the developer taking over further development, testing, and debugging.

**Date:** 2026-03-06

---

## 1. Purpose and problem

**App:** `gogcli-enhanced` is a Go CLI and MCP server (gog-agentic) for Google Workspace. It is the **agent-safe control plane**: deterministic lifecycle, stable JSON contracts (`error_code`), safety semantics. Agents call gog-agentic MCP tools; live requests are served by the **gog** binary (Go) and, optionally for some Tier A reads, by the **gws** (Google Workspace Rust CLI) invoked by gog and normalized.

**When gog binary+MCP is used vs gws:**
- **Default (all live requests):** gog binary runs `gog mcp serve` (gog-agentic). Every MCP tool call (Drive, Gmail, Docs, Sheets, etc.) is handled by gog and native Google APIs. gws is not in the path.
- **Optional Tier A backend:** When `GOG_BACKEND=gws` is set, the **gog** binary still serves the request but for specific Tier A commands (currently `gmail labels list`, `gmail labels get`) it invokes the **gws** CLI, captures stdout/stderr/exit, normalizes with `internal/parity/normalize`, and returns the result. So the agent still talks only to gog-agentic; gws is a backend called by gog when the env is set.
- **Parity/CI:** gws is used only as a **fixture provider** (no live API calls). Runner compares gws fixtures to native goldens and schemas.

**Problem for next phase:** Extend Tier A routing to more commands (per `docs/merge/command-migration-matrix.md`), add integration tests and manual smoke for `GOG_BACKEND=gws`, and hard-gate 403 after a real golden is captured.

---

## 2. Completed work and outcomes

- **Parity runner** (`cmd/gog-parity`, `internal/parity/*`): fixture load from `docs/merge/goldens/<case>/{native,gws}/`, classification (ERROR if exit != 0 or top-level `error` in stdout/stderr), gws error normalization (HTTP → `error_code`; `google_reason` drift-only), schema validation, breaking vs drift diff, `parity-report.json`. CI runs it; 401/404 hard-gated; 403 soft. `make parity` to run.
- **Live gws routing (optional):** `GOG_BACKEND=gws`; `internal/backend/gws` (RunLabelsList, RunLabelsGet, Path, HasTopLevelError); `internal/cmd/backend.go` (Backend()), `backend_error.go` (BackendError + stableExitCode mapping); `internal/cmd/gmail_labels.go` branches on Backend() for list/get, invokes gws, normalizes errors with parity logic. Default remains native.
- **TOOLS doc:** `docs/TOOLS-gog-agentic-section.md` — auth status → exec (`gog auth status --json`), no auth.* MCP tools; Drive list vs search (folders vs mixed); tool names underscores.
- **Linode/deploy:** gog deployed; `scripts/deploy.sh` (pull, build, copy binary, restart mcporter); docs: `docs/LINODE-TEST-QUERIES.md`, `docs/LINODE-AUTH-RESTORE.md`, `docs/openclaw-linode-runbook.md`.

---

## 3. Failures, open issues, lessons learned

- **OAuth client deleted (Linode):** New client JSON deployed; re-auth via `gog auth add … --manual` and paste redirect URL over SSH. See `docs/LINODE-AUTH-RESTORE.md`.
- **No MCP tool for Gmail labels or auth status:** Agents use exec: `gog gmail labels list -a ACCOUNT --json`, `gog auth status --json`. Tool names use underscores; `--server` is a flag of `mcporter call`. Documented in `docs/TOOLS-gog-agentic-section.md`.
- **gws errors on stdout:** Classification and live gws invocation must check both stdout and stderr for top-level `error`.
- **403 golden:** Placeholder only; 403 soft-gated. Real capture per `docs/merge/CAPTURE-403-RUNBOOK.md` (maintainer).

---

## 4. Files changed, insights, gotchas

- **Backend/routing:** `internal/cmd/backend.go`, `backend_error.go`, `gmail_labels.go`; `internal/backend/gws/gws.go`. `exit_codes.go`: BackendError mapped to exit codes. Do not parse `message` for semantics; `google_reason` is drift-only.
- **Parity:** `cmd/gog-parity/main.go` (hard-gated cases, schemaFileForCase, invocationCtxForCase); `internal/parity/io` (DiscoverCases, LoadFixture, IsPlaceholder); `internal/parity/classify`, `normalize`, `schema`, `diff`. Gmail labels compared set-by-id; sort keys/diff entries for deterministic reports. Discovery must not silently skip unreadable case dirs.
- **Gotchas:** gws binary from `GOG_GWS_PATH` or `"gws"`. When adding Tier A commands, reuse parity normalizer; add fixtures for both providers; run `make parity` before and after.

---

## 5. Key files and directories

| Path | Purpose |
|------|---------|
| `handover.md` (repo root) | Single source of truth; gog vs gws usage; full execution plan. |
| `AGENTS.md` | Build, test, lint, PR workflow. |
| `gogcli-developer-handover/HANDOVER.md` | This file; next-phase brief. |
| `docs/merge/` | Parity goldens, schemas, policy, runbooks. |
| `docs/merge/goldens/` | Fixtures per case/provider (stdout.json, stderr.json, exit_code.txt; PLACEHOLDER.txt). |
| `docs/merge/schemas/` | JSON schemas (e.g. gmail-labels-list.json). |
| `docs/merge/GWS-VS-GOG-ROUTING.md` | When gws vs gog; how to implement/test routing. |
| `docs/merge/command-migration-matrix.md` | Tier A/B/C; per-command migration. |
| `docs/merge/CAPTURE-403-RUNBOOK.md` | One-time 403 golden capture. |
| `cmd/gog-parity/` | Parity runner CLI. |
| `internal/parity/` | io, classify, normalize, schema, diff. |
| `internal/backend/gws/` | gws CLI invocation (RunLabelsList, RunLabelsGet). |
| `internal/cmd/backend.go`, `backend_error.go` | Backend switch (GOG_BACKEND); normalized gws errors. |
| `docs/TOOLS-gog-agentic-section.md` | MCP tool names; auth/labels via exec; Drive list vs search. |
| `docs/LINODE-TEST-QUERIES.md` | CLI and OpenClaw verification. |
| `docs/openclaw-linode-runbook.md` | OpenClaw + gog on Linode. |
| `scripts/deploy.sh` | Pull, build, copy binary, restart mcporter. |

**Next steps:** Add more Tier A commands to routing per matrix; integration tests for `GOG_BACKEND=gws`; manual smoke with gws on PATH; capture 403 golden and promote to hard gate.
