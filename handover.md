# Handover: Roadmap to Extend Agentic Edit Framework to Sheets and Slides

## Context

What is already done:

- Docs has a full edit suite:
  - `gog docs edit replace`
  - `gog docs edit append`
  - `gog docs edit insert`
  - `gog docs edit delete`
  - `gog docs edit batch`
- Docs edit has agent-friendly safety and pipeline flags:
  - `--dry-run` (works **without auth** for replace/insert/delete — agents can build plans offline)
  - `--require-revision`
  - `--validate-only`
  - `--pretty`
  - `--output-request-file` (all subcommands: replace, append, insert, delete, batch)
  - `--execute-from-file`
- JSON error handling with structured envelopes (`error_code`, `doc_id`, `request_index`).
- Docs code split into focused files:
  - `docs_cmd.go` — command wiring
  - `docs_edit_cmd.go` — edit subcommands
  - `docs_edit_helpers.go` — hash, normalize, dry-run, error types
  - `docs_read_cmd.go` — export, info, create, copy, cat

What is not done yet:

- Sheets has useful update commands already, but not the same "agentic edit framework".
- Slides does not yet have an inline edit framework.

---

## Goal

Bring Sheets and Slides up to the same "agent-safe, machine-friendly" standard as Docs edits:

1. consistent command shape,
2. safety rails by default,
3. deterministic JSON outputs,
4. structured errors that agents can branch on.

---

## Read This First (Junior Dev Checklist)

Before coding:

1. Read:
   - `AGENTS.md`
   - `docs/editing.md`
   - `docs/refactor/external-review-feedback.md` (prioritized improvements)
   - `internal/cmd/docs_cmd.go`, `docs_edit_cmd.go`, `docs_edit_helpers.go`, `docs_read_cmd.go`
   - `internal/cmd/docs_edit_test.go`
2. Run baseline checks locally:
   - `make test`
3. Learn from existing patterns:
   - docs edit safety flags and dry-run behavior
   - JSON stderr error envelope behavior in `internal/cmd/root.go`
4. Create a feature branch per phase (small PRs are easiest to review).

---

## High-Level Plan

We will do this in 5 phases.

- Phase 1: Shared foundation (small infra + standards)
- Phase 2: Sheets agentic extension
- Phase 3: Slides edit MVP
- Phase 4: Harden for agent workflows
- Phase 5: Docs + handoff cleanup

Each phase includes:
- what to build,
- tests to write,
- done criteria.

---

## Phase 1: Shared Foundation (1-2 days)

### Why

Avoid copying Docs logic into Sheets and Slides with tiny differences. Create reusable helpers for safety and JSON consistency.

### Build

1. Create shared helper types/functions in `internal/cmd` (or a focused helper file):
   - common safety flags:
     - `--dry-run`
     - `--require-revision` (where API supports it)
     - `--validate-only`
     - `--pretty`
     - `--output-request-file`
     - `--execute-from-file`
   - request normalization helper
   - request hash helper
   - structured error helper (`error_code`, `operation`, `resource_id`, optional `request_index`)
2. Ensure helper names are service-neutral (not `docs...` if reused by sheets/slides).

### Tests

- Unit tests for helpers:
  - request hash deterministic
  - normalize + write to file/stdout behavior
  - error object contains expected JSON fields

### Done criteria

- Shared helper tests pass.
- No behavior change to existing Docs commands.

---

## Phase 2: Sheets Agentic Extension (3-5 days)

### Why

Sheets already supports updates, so this is the fastest place to deliver value.

### Command design (proposed)

Add a new umbrella command:

- `gog sheets edit ...`

MVP subcommands:

1. `gog sheets edit values`
   - wraps existing values update behavior but with full agentic safety options
2. `gog sheets edit append`
   - wraps append behavior with same safety options
3. `gog sheets edit clear`
   - destructive; must require guard behavior like docs delete
4. `gog sheets edit batch`
   - uses Sheets `spreadsheets.batchUpdate` request body with agentic flags

### Build

1. Reuse existing sheets logic where possible (do not rewrite working behavior).
2. Add validate-only:
   - local request validation only, no API call.
3. Add dry-run:
   - emit computed request + hash, no mutation.
4. Add normalized output options:
   - `--pretty`
   - `--output-request-file`
   - `--execute-from-file`
5. Add structured JSON errors for sheets edit operations.

### Tests

Add/extend tests in `internal/cmd`:

- command success paths:
  - values update
  - append
  - clear
  - batch
- safety behavior:
  - dry-run does not call API
  - validate-only does not call API/auth
  - destructive clear requires explicit force intent in human mode
- JSON error envelope tests:
  - has `error_code`, `operation`, and target identifiers
- request hash tests:
  - hash is present and deterministic

### Done criteria

- All sheets edit commands pass tests.
- Existing sheets commands still pass all prior tests (`make test` green).

---

## Phase 3: Slides Edit MVP (4-6 days)

### Why

Slides has a robust batch update API; we should start with a safe MVP that maps directly to API requests.

### Command design (proposed)

Add:

- `gog slides edit batch`

Optional convenience commands (if time allows):

- `gog slides edit insert-text`
- `gog slides edit replace-text`

For junior dev pace, start with `batch` first.

### Build

1. Implement `slides edit batch` with the same framework as docs/sheets:
   - validate-only
   - dry-run
   - pretty/normalized request output
   - request hash
   - execute-from-file
2. Add local validation:
   - request list is non-empty
   - each request has exactly one operation (if applicable to request struct shape)
3. Add structured JSON error metadata.

### Tests

- happy path batch request sent correctly
- validate-only path (no API call)
- execute-from-file path
- invalid request schema path with proper error code and request index

### Done criteria

- Slides edit batch works end-to-end with agentic flags.
- Tests and lints are green.

---

## Phase 4: Agentic Hardening Across Services (2-3 days)

### Why

By now, Docs/Sheets/Slides should behave similarly; this phase closes inconsistencies.

### Build

1. Standardize JSON success shape fields where practical:
   - include operation name, resource IDs, request hash where relevant.
2. Standardize JSON error shape:
   - always include `error_code`
   - include service + operation metadata
3. Ensure no mixed stdout JSON objects in any workflow.
4. Confirm destructive commands have explicit behavior:
   - human mode confirmation/force
   - non-destructive preview path for agents

### Tests

- Add one "contract" test per service asserting required JSON fields for:
  - validate-only output
  - dry-run output
  - failure output envelope

### Done criteria

- Predictable machine contract for all three services.

---

## Phase 5: Docs + Handoff Cleanup (1-2 days)

### Build

1. Update docs:
   - `README.md` command examples for Sheets and Slides edit workflows
   - add section to `docs/editing.md` (or split into service-specific docs)
2. Update `AGENTS.md` with final recommended agent sequence for all three:
   - validate-only -> review hash -> execute-from-file -> require revision where supported
3. Update `CHANGELOG.md`.

### Tests

- Run full suite:
  - `make test`
  - `make lint`

### Done criteria

- New developer can follow docs and successfully run each edit flow.

---

## Recent Changes (External Review Quick Wins)

Completed 2025-02:

1. **Dry-run without auth** — `docs edit replace`, `insert`, `delete` now run `--dry-run` without requiring `requireAccount()`. Agents can generate exact `BatchUpdateDocumentRequest` payloads offline.
2. **`--output-request-file` and `--pretty` on all edit subcommands** — replace, append, insert, delete, and batch share these flags via `DocsEditSafetyFlags`.
3. **Docs code split** — `internal/cmd/docs.go` refactored into `docs_cmd.go`, `docs_edit_cmd.go`, `docs_edit_helpers.go`, `docs_read_cmd.go` for maintainability.

See `docs/refactor/external-review-feedback.md` for the full review and remaining prioritized items (marker-based insert/delete, standardized JSON envelope, `--timeout`/`--retries`, `docs positions` helper, etc.).

---

## Session Handover: MCP Drive Pagination + Cursor Rules (2026-03)

**For new developers:** This section documents what was done in a recent session (trials, errors, and final fixes) so you understand the current state and why things work the way they do.

### 1. Google Drive “only 4 folders” in OpenClaw (Linode)

**Symptom:** When the agent (OpenClaw) listed “all folders in Google Drive root,” it only ever saw 4 folders, even though the Drive had more.

**What we tried (trials and errors):**

- **Assumption 1:** The agent was calling `drive.listFiles` with `{}`, which returns files and folders mixed; after truncation only a few items were visible and only some were folders.  
  **Change:** We made `drive.listFiles` with empty/trashed=false query **redirect** to `drive.searchFiles` with a folders-only query (`mimeType = 'application/vnd.google-apps.folder'`), so the response contained only folders.  
  **Result:** Still only 4 folders visible.

- **Assumption 2:** Response was too large and the gateway truncated by size.  
  **Change:** We capped `--all` JSON output at 50 items (`driveListAllMaxOutputItems` in `internal/cmd/drive.go`), then lowered to 10, with `--compact` (id, name, mimeType only).  
  **Result:** Still only 4. So the limit was outside our code.

- **Root cause (5 Whys):**  
  1. User sees 4 → agent received only 4 items.  
  2. gog returns up to 10; something downstream truncates.  
  3. Our MCP server and transport do not truncate; truncation happens in the **gateway** or when the agent uses the **exec** tool to run `mcporter call ...` (tool result = exec stdout).  
  4. Many gateways cap exec/tool result length (e.g. ~1–2 KB).  
  5. **Conclusion:** The gateway (or exec tool) enforces a max length on tool result content; we cannot change that from this repo.

**Fix we implemented:** Keep each response small enough to fit the limit. From MCP we **no longer use `--all`** for `drive.listFiles` and `drive.searchFiles`. We request **one page of 4 items** (`mcpDrivePageSize = 4` in `internal/mcp/providers/google/tools.go`) with `--compact` and return **`nextPageToken`**. The agent can call again with `page: nextPageToken` until there is no token, and thus get all folders.

**Relevant code:**

- `internal/mcp/providers/google/tools.go`: `mcpDrivePageSize = 4`; when `page` is empty we pass `--max 4` and `--compact` (no `--all`). Tool descriptions say to loop with `nextPageToken` for “list all folders.”
- `internal/cmd/drive.go`: `driveListAllMaxOutputItems = 10` still applies to **CLI** `gog drive ls --all` / `gog drive search ... --all` only.
- `docs/openclaw-linode-runbook.md` §8.10: Documents the 5 Whys, root cause, and that the agent should paginate using `nextPageToken`.
- `docs/TOOLS-gog-agentic-section.md` and `scripts/setup.sh`: “List only folders” bullet updated to say “call again with `page: nextPageToken` until no token.”

**Deploy (Linode):** After any gog binary update: run **`./scripts/deploy.sh`** from the repo (set `WORKSPACE_DIR=/root/openclaw-stock-home/.openclaw/workspace` if the workspace is not auto-detected). That does git pull, build, copy binary to `~/.local/bin/gog`, and mcporter daemon restart. Alternatively: `git pull`, `./scripts/install.sh`, copy `bin/gog` to `/root/.local/bin/gog`, then restart the daemon. Otherwise the agent keeps using the old binary and behavior.

**OpenClaw feedback: “First 25 directories” returned only 10; pageToken didn’t advance (2026-03)**  
User asked for “first 25 directories” in Drive root. OpenClaw called `drive.listFiles` with Drive API-style args: `query`, `fields`, `maxResults: 25`, and later `pageToken`. It got only 10 items and “same 10 repeatedly” when using pageToken. **Root cause:** Our MCP tools use `max` and `page`, not `maxResults`/`pageSize`/`pageToken`. Those were ignored, so default page size applied and pagination never advanced. **Fix:** In `internal/mcp/providers/google/tools.go`, `driveListFiles` and `driveSearchFiles` accept aliases: `pageToken` → `page`, `maxResults` or `pageSize` → `max` (capped for gateway). Tool descriptions and TOOLS/runbook note that `page` = previous response’s `nextPageToken`, and that Drive API names are accepted.

---

### 2. AGENTS.md and Agent Discipline

**State:** `AGENTS.md` had been deleted from the repo.

**What we did:** Recreated `AGENTS.md` with the full repository guidelines (project structure, build/test, coding style, testing, commit/PR, security, agentic workflow, batch vs sedmat). Added a new section at the bottom:

**## Agent Discipline**

- Do not perform broad refactors unless explicitly requested.
- Do not upgrade dependencies or change toolchain versions without approval.
- Prefer minimal diffs over cleanup-only changes.
- For bug fixes: identify root cause before proposing changes.
- Do not restate large diffs or logs in responses unless necessary.

Cursor loads workspace rules from `AGENTS.md` when present, so the agent now gets these discipline rules in this project.

---

### 3. Cursor rules reference and user-rule edits

**What we did:**

- **Created `docs/CURSOR-RULES-FULL-TEXT.md`** — A single markdown file that contains the full text of every rule applied in Cursor (workspace rule from AGENTS.md + user rules). Use it as a reference; update it whenever someone changes a rule so the doc stays in sync.

- **Playwright / Chromium (user rule):** Replaced with:
  - “Always use Chromium with Playwright. Add: `--browser=chromium`. Or when using npx: `--project=chromium`.”  
  User rules live in **Cursor Settings → Rules**; they are global (all projects). We updated the text in `docs/CURSOR-RULES-FULL-TEXT.md` and provided the exact text to paste into Settings, because we cannot edit Settings from the repo.

- **Universal LLM-Driven Development Cheat-Sheet (user rule):** Same situation — it’s a user rule in Settings → Rules. To replace it entirely and keep it global: paste the new full text into Settings → Rules (over the old cheat-sheet). The reference copy in `docs/CURSOR-RULES-FULL-TEXT.md` can be updated whenever the rule text changes so the doc remains accurate.

**Takeaway for new devs:** Workspace rules = files in the repo (e.g. `AGENTS.md`). User rules = Cursor Settings → Rules (global). We can edit repo files; for user rules we update the reference doc and give paste-ready text.

---

## Suggested Milestone Order (Practical)

1. **Milestone A:** Shared foundation complete
2. **Milestone B:** Sheets edit commands + safety rails complete
3. **Milestone C:** Slides batch edit + safety rails complete
4. **Milestone D:** Cross-service hardening + docs complete

---

## Scope Guardrails (Important)

To keep this achievable:

- Do not add rich formatting/UI editing tools yet unless required.
- Do not redesign auth; reuse existing auth and scopes.
- Keep PRs small and feature-focused.
- Prefer backward-compatible behavior over command churn.

---

## First 3 Tickets to Create

1. **Ticket 1: Shared Agentic Edit Helpers**
   - reusable safety flags and request normalization/hash/error helpers.
2. **Ticket 2: Sheets Edit Batch + Validate/Preview Flow**
   - include execute-from-file, structured errors, tests.
3. **Ticket 3: Slides Edit Batch MVP**
   - same agentic contract and tests.

---

## Definition of Done (for each command)

A command is done only if:

1. supports JSON mode with deterministic fields,
2. has validate-only and/or dry-run behavior (as appropriate),
3. has structured JSON errors (`error_code` minimum),
4. has unit tests for success + safety + error paths,
5. passes `make test` and `make lint`.

---

## Final Note for Junior Dev

When unsure, copy the behavior contract of existing `docs edit` commands first, then adapt to Sheets/Slides APIs. Consistency is more important than cleverness for agentic systems.
