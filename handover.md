# Handover: gogcli-enhanced

Single entry point for a new developer. See **Key documents** for details; this file states current status and next steps only.

---

## Key documents

| Purpose | Document |
|--------|----------|
| Repo rules, build, test, commit, agentic workflow | `AGENTS.md` |
| Docs/Sheets/Slides agent ops roadmap + MCP parity | `DOCS_AGENT_PLAN.md` |
| Docs/Sheets editing (user-facing) | `docs/editing.md` |
| Linode + OpenClaw MCP deploy & troubleshooting | `docs/openclaw-linode-runbook.md` |
| MCP tool reference (Drive/Docs/Sheets + Gmail/Calendar/Contacts when added) | `docs/TOOLS-gog-agentic-section.md` |
| Shared agentic edit helpers | `internal/cmd/edit_helpers.go` |
| External review feedback (prioritized) | `docs/refactor/external-review-feedback.md` |
| Cursor rules reference (workspace + user) | `docs/CURSOR-RULES-FULL-TEXT.md` |

---

## Context

- **Docs:** Full edit suite (replace, append, insert, delete, batch, insert-table, insert-image, merge-data) with agentic flags (`--dry-run`, `--validate-only`, `--pretty`, `--output-request-file`, `--execute-from-file`), structured `EditError`, JSON envelopes.
- **Sheets:** Edit commands (values, append, clear, batch, delete-range, merge-data) use same shared helpers and safety flags.
- **Slides:** Edit batch + replace-text + create-slide; merge-data design in Phase 3.
- **MCP (gog-agentic):** Drive, Docs, Sheets (and Slides) tools; token-efficient tools/call (content only). Gmail, Calendar, Contacts and small gaps (sheets_clear, sheets_metadata, docs_export) are planned—see `DOCS_AGENT_PLAN.md`.

---

## Current status

- **Done:** Phase 1 (shared foundation in `edit_helpers.go`). Phase 2A (Sheets edit). Phase 2B (Docs edit refactor: replace, insert, delete, insert-table). Phase 3 items: Sheets delete-range, Docs insert-image, MergeData design + Docs/Sheets merge-data CLI.
- **In progress / next:** Tier 2 (apply-style; insert-toc blocked by Docs API) or Tier 1 (apply-template, extract-data); Slides batch completeness; cross-service hardening; docs/CHANGELOG. MCP parity (Gmail, Calendar, Contacts, sheets_clear, sheets_metadata, docs_export) done.

---

## MCP & Linode

- **Drive “only N folders”:** Gateway/exec caps tool result length. We use paginated list/search (no `--all`): one page, `nextPageToken`; agent loops with `page`/`pageToken`. See `docs/openclaw-linode-runbook.md` §8.10.
- **Deploy (Linode):** From repo: `./scripts/deploy.sh` (set `WORKSPACE_DIR` if workspace not auto-detected). Pulls, builds, copies binary, restarts mcporter daemon.
- **Aliases:** MCP tools accept `pageToken`→`page`, `maxResults`/`pageSize`→`max` (capped). See `internal/mcp/providers/google/tools.go` and TOOLS doc.

---

## Agent discipline & rules

- Repo rules: `AGENTS.md`. User rules: Cursor Settings → Rules (global). Reference copy: `docs/CURSOR-RULES-FULL-TEXT.md`. Do not broad-refactor, upgrade deps, or change toolchain without approval; prefer minimal diffs; fix root cause, not symptoms.

---

## Next steps (new developer)

1. Read `AGENTS.md`, then `DOCS_AGENT_PLAN.md` for roadmap and MCP action items.
2. Run `make test` and `make lint`; fix any failures in your area.
3. Pick work from `DOCS_AGENT_PLAN.md` (Docs ops, MCP parity) or: Tier 2 (apply-style) or Tier 1 (apply-template, extract-data), Slides batch, cross-service JSON hardening, README/CHANGELOG.
4. One feature/PR at a time; tests with command; no server/config/version changes without approval.

---

## Scope guardrails

- No rich formatting/UI editing unless required. Reuse existing auth/scopes. Prefer backward compatibility. Keep PRs small and feature-focused.

---

## Definition of done (per command)

- JSON mode with deterministic fields; validate-only and/or dry-run where appropriate; structured JSON errors (`error_code`); unit tests for success + safety + errors; `make test` and `make lint` pass.
