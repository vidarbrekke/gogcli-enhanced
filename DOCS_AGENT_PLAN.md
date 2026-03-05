# Docs Agent Plan & MCP Parity

Single plan for (1) Docs/Sheets/Slides agent-friendly operations and (2) MCP agentic parity (Gmail, Calendar, Contacts, and small gaps). See `handover.md` for current status and key docs.

---

## Agent-friendly principles

- **Intent over primitives:** High-level ops (e.g. replace all X) with optional low-level control.
- **Safety:** `--dry-run`, `--validate-only`, `--execute-from-file`, request hashing.
- **Structured I/O:** JSON with `error_code`, `request_hash`, `resource_id`; parseable by agents.
- **Semantic ops:** Domain concepts (heading, style); composable commands.

---

## Current state (Docs vs Sheets vs Slides)

| Feature | Docs | Sheets | Slides |
|---------|------|--------|--------|
| Focused operations | batch + replace/insert/delete/append/insert-table/insert-image | values/append/clear/batch/delete-range | batch, replace-text, create-slide |
| Mail-merge | merge-data | merge-data | merge-data |
| Data extraction | ❌ | native | ❌ |

Docs has the most template/collab potential; merge-data and extract-data are high impact.

---

## Docs proposed operations (prioritized)

**Tier 1 (transformative):** `merge-data` ✅ done, `apply-template`, `extract-data` (outline/tables/links).

**Tier 2 (quick wins):** `replace-text` (alias for replace), `insert-table` from CSV/JSON ✅ insert-table exists, `apply-style`, `insert-toc`.

**Tier 3 (workflow):** `resolve-comments`, `accept-suggestions`, `watermark`, `update-footer`.

**Tier 4 (structure):** `split-sections`, `merge-docs`, `reorder-sections`.

**Tier 5 (advanced):** `inline-images`, `apply-conditional-format`, `extract-comments`.

**Explicitly deferred:** Rich formatting UI, interactive mode, server/config/version changes without approval.

---

## Implementation phases (Docs/Sheets/Slides)

- **Phase 1 (done):** Quick wins — replace-text, insert-table, apply-style, insert-toc; full agentic safety.
- **Phase 2:** Transformative — merge-data ✅, apply-template, extract-data.
- **Phase 3:** Workflow — resolve-comments, accept-suggestions, watermark, update-footer.
- **Phase 4:** Structure — split-sections, merge-docs, reorder-sections.
- **Phase 5:** Advanced — inline-images, conditional-format, extract-comments.

One command per PR; tests first; no server/config/version changes without approval. Success per command: JSON + dry-run/validate-only + structured errors + tests + `make test`/`make lint`.

---

## MCP agentic parity (action items)

Bring CLI parity into MCP so agents use tools first instead of exec. Pattern: add `ToolSpec` + provider method that builds `gog` args and calls `p.runCLI` (see existing `internal/mcp/providers/google/*.go`).

[x] **Sheets gaps:** Add `sheets_clear` and `sheets_metadata` in `sheets_tools.go` + provider methods in `tools.go`; wire in `sheetsSpecs(p)`.

[x] **Docs export:** Add `docs_export` in `docs_tools.go` (docId, format, optional out); provider writes to temp if out omitted, returns path in result; wire in `docsSpecs(p)`.

[x] **Gmail:** Create `gmail_tools.go` with `gmailSpecs(p)`: `gmail_search` (query, max, page, account), `gmail_send` (to, subject, body, account, optional cc/bcc). Add `gmailSearch`/`gmailSend` in `tools.go`; append `gmailSpecs(p)` in `Register()`.

[x] **Calendar:** Create `calendar_tools.go` with `calendarSpecs(p)`: `calendar_events` (calendarId, from, to, max, page, account). Add `calendarEvents` in `tools.go`; append `calendarSpecs(p)` in `Register()`.

[x] **Contacts:** Create `contacts_tools.go` with `contactsSpecs(p)`: `contacts_list` (max, page, account). Add `contactsList` in `tools.go`; append `contactsSpecs(p)` in `Register()`.

[x] **Docs:** Update `docs/TOOLS-gog-agentic-section.md` — add Gmail, Calendar, Contacts sections and example `mcporter call` commands; add sheets_clear, sheets_metadata, docs_export to Sheets/Docs lists; state “use MCP tools first” for these services.

[x] **Tests:** In `internal/mcp/google_tools_test.go` add tests for gmail_search, gmail_send, calendar_events, contacts_list, sheets_clear, sheets_metadata, docs_export (minimal args, envelope shape or error).

[x] **CI:** Run `make ci`; pnpm-gate fixed (use `pnpm run --if-present lint/build/test` when root package.json has no scripts). Go: `make test` passes; `make fmt-check` requires clean tree; `make lint` has 58 known issues (dupl, err113, goconst, gosec, prealloc, wsl). After deploy, confirm new tool names (e.g. `mcporter list gog-agentic`).

[x] **Runbook:** Optionally in `docs/openclaw-linode-runbook.md` add one-line smoke note for Gmail/Calendar/Contacts MCP tools.

---

## Recommended next sprint

- **Docs:** Tier 2 (apply-style; insert-toc not in Docs API) or Tier 1 (apply-template, extract-data). Append/Batch flow documented in `docs/editing.md`.
- **MCP:** Action items above completed (sheets/docs gaps, Gmail, Calendar, Contacts, docs + tests + runbook). Confirm tool list after deploy.

---

## Success criteria (per command / tool)

- JSON with deterministic fields; dry-run/validate-only where appropriate; structured errors (`error_code`); unit tests; `make test` and `make lint` pass. For MCP: tool appears in list, handler returns envelope consistent with existing tools.
