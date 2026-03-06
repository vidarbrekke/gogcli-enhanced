# Upstream merge – Phase 1 (low-risk)

Branch: `merge-upstream-2026-03`  
Upstream: `steipete/gogcli` (main)  
Strategy: **Keep our behavior for all conflicted files**; take upstream’s new files and non-conflicting changes.

## What was merged (low-risk)

- **New files from upstream:** `.env.example`, `.github/actionlint.yaml`, `PR_DESCRIPTION.md`, `internal/cmd/gmail_archive.go` (+ test), `internal/cmd/docs_table_test.go`, `testdata/v7_*` through `v12_*`.
- **Non-conflicting updates:** `go.mod`/`go.sum` (Go 1.25, deps), `.github/workflows/ci.yml`, `Makefile`, and many `internal/cmd` / `internal/googleauth` / `internal/config` / `internal/googleapi` / `internal/tracking` files where we kept our version on conflict or reverted to our tests.
- **Explicitly kept ours (no upstream feature pull yet):**  
  `CHANGELOG.md`, `README.md`, `docs/sedmat.md`, `docs/spec.md`,  
  `internal/cmd/root.go` (MCP and our structure), `internal/cmd/drive.go`, `internal/cmd/sheets.go`,  
  `internal/secrets/store.go`, all `internal/cmd/docs_sed*.go` and our `docs_edit` test/behavior,  
  `internal/cmd/docs_validation_more_test.go`, `internal/cmd/drive_errors_test.go`, `internal/cmd/drive_export_format_test.go`.

## Removed to avoid conflicts

- Upstream’s `internal/cmd/docs_edit.go` (redeclares our `DocsEditCmd`).
- Upstream’s `docs_paragraphs.go`, `docs_paragraphs_test.go`, `docs_write_update_test.go` (type clash with our `docParagraph` in tests).

## Fixes applied

- `internal/cmd/auth_add_test.go`: `ensureKeychainAccess` kept as `func(bool) error` (our API).

## Known test flakiness (unchanged)

- Some `TestSedIntegration_*` and `TestExecute_ClassroomMoreCommands_JSON` can fail with `encode json: write |1: broken pipe` under parallel runs. They pass when run in isolation or with `-p 1`. Treated as pre-existing; fix in a follow-up.

## Phase 2 (done 2026-03-05)

- **Sheets links, drive ls --global, Gmail archive/read/unread/trash, calendar --to expansion:** Already present on our branch from Phase 1 merge; no code changes.
- **Drive flag:** We expose `drive ls --global` (same behavior as upstream’s `drive ls --all`). No alias added; documented in CHANGELOG.
- **CHANGELOG:** Merged upstream 0.12.0 entries (Drive ls --global, Gmail drafts update --quote, Auth --gmail-scope, Gmail archive|read|unread|trash --dry-run).
- **README:** Restored Gmail scope note and draft --quote examples; added `--gmail-scope` to least-privilege bullet and auth command reference.
- **Broken-pipe test flakiness:** Deferred; documented as known issue (run with `-p 1` or in isolation).

## Future (optional)

- Docs write/update reconciliation with our `DocsEditCmd` (if upstream adds more).
- Stabilize broken-pipe tests (capture/pipe handling when running with `--json`).
