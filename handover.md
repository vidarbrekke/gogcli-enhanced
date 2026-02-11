# Handover: Roadmap to Extend Agentic Edit Framework to Sheets and Slides

## Context

What is already done:

- Docs has a full edit suite:
  - `gog docs edit replace`
  - `gog docs edit append`
  - `gog docs edit insert`
  - `gog docs edit delete`
  - `gog docs edit batch`
- Docs edit also has agent-friendly safety and pipeline flags:
  - `--dry-run`
  - `--require-revision`
  - `--validate-only`
  - `--pretty`
  - `--output-request-file`
  - `--execute-from-file`
- JSON error handling was improved for agent workflows.

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
   - `internal/cmd/docs.go`
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
