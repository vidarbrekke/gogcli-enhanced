# Developer Handoff: gogcli-enhanced Agentic Edit Services

**Date:** 2026-02-17  
**Repository:** vidarbrekke/gogcli-enhanced  
**Status:** Phase 1 Complete, Phase 2 Complete (Sheets + Docs), Phase 3 In Progress, Phases 4–5 Queued

---

## ✅ What's Complete

### Phase 1: Shared Foundation (VID-91)
**File:** `internal/cmd/edit_helpers.go`

Unified helpers for agent-safe editing across Docs/Sheets/Slides:
- `AgenticEditSafetyFlags` (dry-run, validate-only, pretty, output-request-file, execute-from-file, require-revision)
- `EditError` with service/operation/resource/error code/HTTP/reason/request_index
- `RequestHash`, `NormalizedRequestString`, `NormalizedRequestForOutput`, `DryRunOutput`, `NewEditError`, `IsNotFound`

### Phase 2A: Sheets Edit Integration (VID-92, VID-93, VID-94, VID-95) — **Done**

**Summary:** Legacy Sheets edit code in `sheets.go` was removed; `sheets_edit_cmd.go` and `sheets_edit_helpers.go` were refactored to use the shared helpers from `edit_helpers.go`. All four commands now follow the agentic pattern.

**Files changed:**
- `internal/cmd/sheets.go` — Removed duplicate type declarations and legacy `SheetsEditValuesCmd` Run implementation (~210 lines).
- `internal/cmd/sheets_edit_helpers.go` — Rewritten: `SheetsEditSafetyFlags` is an alias of `AgenticEditSafetyFlags`; `newSheetsEditError` delegates to `NewEditError("sheets", ...)`; `isSheetsNotFound` delegates to `IsNotFound`. Kept only Sheets-specific `sheetsRequestOperationCount` / `sheetsRequestOperationName`.
- `internal/cmd/sheets_edit_cmd.go` — All commands now use `RequestHash`, `NormalizedRequestForOutput`, `NormalizedRequestString`, `SheetsDryRunOutput` from `edit_helpers.go`. Batch command sets `RequestIndex` via `errors.As(err, &ee)` on `*EditError`.

**Commands:** `gog sheets edit values`, `append`, `clear`, `batch` — all support:
- `--validate-only` (no auth)
- `--dry-run` (no API)
- `--pretty`, `--output-request-file`, `--execute-from-file`
- Structured `EditError` on failure

**Tests:** `internal/cmd/sheets_edit_test.go` — all tests pass; `make test && make lint` green (no new lint in changed files).

### Phase 2B: Docs Edit Agentic Refactor (New) — **Done**

**Date:** 2026-02-17  
**Summary:** Four Docs edit commands refactored to use shared agentic helpers (pattern matching Sheets/Slides).

**Commands Refactored:**
1. **DocsReplaceCmd** — Find/replace text across doc
   - `gog docs edit replace <docId> --find X --replace Y [--match-case]`
   
2. **DocsInsertCmd** — Insert text at index
   - `gog docs edit insert <docId> <text> [--index N]`
   
3. **DocsDeleteCmd** — Delete text range
   - `gog docs edit delete <docId> <start> <end>`
   
4. **DocsInsertTableCmd** — NEW: Insert table
   - `gog docs edit insert-table <docId> [--rows N] [--cols M] [--index I]`

**All 4 Commands Support:**
- `--validate-only` (local validation, no auth)
- `--dry-run` (build request, no API call)
- `--pretty` (include normalized JSON)
- `--output-request-file` (write to file, use `-` for stdout)
- `--execute-from-file` (replay from file)
- `--require-revision` (concurrency guard)
- Structured `EditError` with error_code/http_status/google_reason

**Files Changed:**
- `internal/cmd/docs_cmd.go` — Added `InsertTable` to `DocsEditCmd`; `DocsEditSafetyFlags` aliased to `AgenticEditSafetyFlags`
- `internal/cmd/docs_edit_cmd.go` — Refactored all 4 commands to use shared helpers
- `internal/cmd/docs_edit_helpers.go` — Simplified to only contain service-specific helpers; delegates to shared `NewEditError`, `IsNotFound`
- `internal/cmd/edit_helpers.go` — Added backward-compatibility wrappers for legacy commands

**Commits:**
- `5d6273b` — Refactor: DocsReplaceCmd (foundation)
- `357eb35` — Feat: DocsInsertTableCmd (new operation)
- `5f0e175` — Refactor: DocsInsertCmd
- `83eb16d` — Refactor: DocsDeleteCmd

**Status:** All build cleanly, --validate-only and --dry-run tested. Ready for integration.

### Phase 3: Cross-Service Capability (VID-111, VID-112, VID-114) — **In Progress**

**Completed:**
- **VID-111** — Sheets DeleteRange: `gog sheets edit delete-range <spreadsheetId> <range> --shift-dimension ROWS|COLUMNS` (default ROWS). Full agentic flow; tests for validate-only, dry-run, force guard, success.
- **VID-112** — Docs InsertImage: `gog docs edit insert-image <docId> --uri <url> [--index 1] [--width-pt/--height-pt]`. Full agentic flow; tests for validate-only, dry-run, success, empty-uri error.
- **VID-114** — MergeData design doc: `docs/PHASE_3_MERGEDATA_DESIGN.md` — Slides pattern analysis, Docs/Sheets CLI sketches, data format, error matrix, shared components, implementation order (VID-115 → VID-116).
- **VID-115** — Docs Edit MergeData: `gog docs edit merge-data <templateId> --data-file <path>` (Drive copy + ReplaceAllText per record).
- **VID-116** — Sheets Edit MergeData: `gog sheets edit merge-data <templateId> --data-file <path>` (Drive copy + FindReplace all sheets per record).

**Next:** Phase 3 merge-data complete; optional VID-113 (DeleteRange code review), then Phase 4/5.

---

## 📋 Linear Issue Status

### ✅ Completed
- **VID-91** — Phase 1: Shared Agentic Edit Helpers Foundation
- **VID-92** — Sheets Edit: Values Command
- **VID-93** — Sheets Edit: Append Command
- **VID-94** — Sheets Edit: Clear Command
- **VID-95** — Sheets Edit: Batch Command
- **VID-99** — Commit WIP Sheets Edit Code (superseded by integration commit)
- **VID-107** — Docs Edit Agentic Refactor (NEW) — Replace, Insert, Delete, InsertTable
- **VID-111** — Sheets Edit DeleteRange
- **VID-112** — Docs Edit InsertImage
- **VID-113** — Sheets DeleteRange code review (agentic compliance, error handling, tests)
- **VID-114** — Phase 3 MergeData design doc (Docs/Sheets)

### 📋 Pending
- **VID-96** — Slides Edit: Batch MVP (4–6 days)
- **VID-97** — Cross-Service Agentic Hardening (2–3 days)
- **VID-98** — Documentation & Handoff (1–2 days)
- **VID-108** — Docs: Remaining Edit Commands (Append, Batch finalization)
- **VID-109** — Docs Phase 1 Quick Wins (apply-style, insert-toc, more)
- **VID-115** — Docs Edit MergeData ✅
- **VID-116** — Sheets Edit MergeData ✅

---

## 🎯 Success Criteria (Phase 2 — Met)

For each Sheets/Docs edit command:
1. ✅ Uses `AgenticEditSafetyFlags` (via `[Service]EditSafetyFlags` alias)
2. ✅ `--validate-only` works without auth
3. ✅ `--dry-run` builds request without API calls
4. ✅ `--pretty` includes normalized JSON output
5. ✅ `--output-request-file` writes request to file
6. ✅ `--execute-from-file` supported where applicable
7. ✅ Returns `EditError` (via `new[Service]EditError`) on failure
8. ✅ Unit tests for success, dry-run, validate-only, error paths (Sheets)
9. ✅ `make build` passes; --validate-only and --dry-run tested (Docs)

**Docs Commands Status:**
- ✅ Replace — Refactored, tested, working
- ✅ Insert — Refactored, tested, working
- ✅ Delete — Refactored, tested, working
- ✅ InsertTable — NEW, implemented, tested, working
- ⏳ Append — Blocked (requires no-auth strategy for validate-only)
- ⏳ Batch — Partially refactored, pending finalization

---

## 📚 Reference

- **Shared helpers:** `internal/cmd/edit_helpers.go`
- **Docs pattern:** `internal/cmd/docs_edit_cmd.go`
- **Sheets implementation:** `internal/cmd/sheets_edit_cmd.go`, `sheets_edit_helpers.go`
- **User guide:** `docs/editing.md` (Docs + Sheets)
- **Phase 3 MergeData design:** `docs/PHASE_3_MERGEDATA_DESIGN.md` (VID-114)

---

## 🔧 Next Steps (New Dev)

### Immediate (Docs Phase 2 Completion)
1. **Append & Batch Finalization (VID-108):** 
   - Refactor `DocsAppendCmd` (needs strategy for validate-only without fetch)
   - Finalize `DocsBatchCmd` cleanup
   - Estimated: 2–3 hours

### Phase 1 Quick Wins (Docs Operations)
2. **Tier 2 Operations (VID-109):**
   - `apply-style` — Bulk apply named styles to ranges
   - `insert-toc` — Generate/update table of contents (requires heading extraction)
   - `watermark` — Add diagonal watermark text
   - Estimated: 4–6 hours total

### Phase 2 Transformative (High Impact)
3. **Tier 1 Operations (VID-110):**
   - `merge-data` — Mail-merge template → N personalized docs
   - `apply-template` — Apply structure/styles from template doc
   - `extract-data` — Extract outline, tables, links → structured JSON
   - Estimated: 8–12 hours (requires template parsing)

### Cross-Service
4. **Slides (VID-96):** Complete remaining slides operations
5. **Hardening (VID-97):** Standardize JSON shape across all services
6. **Docs (VID-98):** Update README, AGENTS.md, CHANGELOG
7. **MergeData hardening follow-up:**
   - Extract shared helpers for Docs/Sheets merge-data (`parseMergeData`, replace-op builders, Drive copy/move helper)
   - Add failure-path tests (copy failure, template not found, batch-update failure)
   - Normalize merge-data response/error schema across Docs/Sheets/Slides
   - Run lint cleanup focused on files touched in Phase 3
   - Create a separate lint-debt cleanup pass for pre-existing repo issues

---

**Bottom line:** Phase 1 (shared foundation) ✅, Phase 2A (Sheets) ✅, Phase 2B (Docs core) ✅ complete. Phase 2C (Append/Batch), Phase 1 quick wins, and Phase 2 transformative operations remain.
