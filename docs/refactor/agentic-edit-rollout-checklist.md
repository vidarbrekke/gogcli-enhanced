# Agentic Edit Rollout Checklist (Sheets + Slides)

Companion to `handover.md`.  
Use this as a working board for implementation status.

## Phase 1: Shared Foundation

- [ ] Create shared agentic edit helper(s) in `internal/cmd`:
  - [ ] safety flags (`dry-run`, `require-revision`, `validate-only`, `pretty`, `output-request-file`, `execute-from-file`)
  - [ ] normalized request output helper
  - [ ] request hash helper
  - [ ] structured error helper (`error_code`, `operation`, resource fields)
- [ ] Add helper unit tests.
- [ ] Verify no behavior regression in Docs edit commands.

## Phase 2: Sheets Agentic Extension

- [ ] Add `gog sheets edit` command group.
- [ ] Implement `sheets edit values`.
- [ ] Implement `sheets edit append`.
- [ ] Implement `sheets edit clear` with destructive safety guard.
- [ ] Implement `sheets edit batch`.
- [ ] Add agentic flags support to all above commands.
- [ ] Add structured JSON error envelope support.
- [ ] Add tests:
  - [ ] happy paths
  - [ ] dry-run no-mutation
  - [ ] validate-only no-auth/no-network path (where feasible)
  - [ ] destructive guard behavior
  - [ ] JSON error field checks
- [ ] Run `make test`.
- [ ] Run `make lint`.

## Phase 3: Slides Edit MVP

- [ ] Add `gog slides edit` command group.
- [ ] Implement `slides edit batch`.
- [ ] Add agentic flags (`validate-only`, `dry-run`, `pretty`, `output-request-file`, `execute-from-file`).
- [ ] Add local request validation.
- [ ] Add structured JSON error envelope support.
- [ ] Add tests:
  - [ ] happy path
  - [ ] validate-only path
  - [ ] execute-from-file path
  - [ ] invalid request schema path
- [ ] Run `make test`.
- [ ] Run `make lint`.

## Phase 4: Cross-Service Hardening

- [ ] Standardize success JSON shape for Docs/Sheets/Slides edit commands.
- [ ] Standardize error JSON shape and required fields.
- [ ] Verify stdout contains a single JSON envelope per command in JSON mode.
- [ ] Ensure destructive commands have explicit guard behavior.
- [ ] Add contract tests for all three services:
  - [ ] validate-only output shape
  - [ ] dry-run output shape
  - [ ] error output shape

## Phase 5: Documentation + Handoff

- [ ] Update `README.md` examples for Sheets/Slides edit workflows.
- [ ] Update `docs/editing.md` with Sheets/Slides sections (or split docs by service).
- [ ] Update `AGENTS.md` with final cross-service agent workflow sequence.
- [ ] Update `CHANGELOG.md`.
- [ ] Final verification:
  - [ ] `make test`
  - [ ] `make lint`

## Ticket Starter (recommended order)

- [ ] Ticket 1: Shared Agentic Edit Helpers
- [ ] Ticket 2: Sheets Edit Batch + Validate/Preview Flow
- [ ] Ticket 3: Slides Edit Batch MVP

## Done Definition (per command)

- [ ] Deterministic JSON success output.
- [ ] Safety path available (`dry-run` and/or `validate-only`).
- [ ] Structured JSON errors with `error_code`.
- [ ] Unit tests for success + safety + failure.
- [ ] Full test/lint suite passes.
