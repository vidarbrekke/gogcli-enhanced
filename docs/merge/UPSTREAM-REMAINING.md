# Remaining upstream features and optional merges

Branch: `merge-upstream-2026-03`  
Upstream: `steipete/gogcli` (main)  
Last sync: Phase 1 merge + Phase 2 doc/CHANGELOG; our branch is ahead of upstream/main.

This doc tracks what we **did not** take from upstream (by choice or conflict) and what is **left to investigate or optionally merge**.

---

## 1. Status overview

| Item | Status | Action |
|------|--------|--------|
| New upstream commits | None | We are up to date; re-fetch periodically. |
| Docs edit (`docs_edit.go`) | **No merge needed** | We supersede upstream (richer edit subcommands). |
| Docs paragraphs (`docs_paragraphs*.go`) | **Merged** | Renamed our test types to `sedTestParagraph`/`sedTextRun`/`sedPara()`; added upstream `docs_paragraphs.go` + `docs_paragraphs_test.go`. |
| Drive `--all` | **No change** | We have `--global`; adding `--all` would conflict with our `--all` (paginate). Document parity. |
| CHANGELOG / README | Phase 2 merged 0.12.0 | Optional: re-sync when upstream adds more. |
| Broken-pipe tests | Fixed (stdout from context) | Remaining tests: fix captureStdout+UI pattern (see ROOT-CAUSE-AUDIT.md). |
| Lint (73 issues) | Not addressed | Separate pass: fix or relax rules. |

---

## 2. Explicitly kept ours (no upstream pull yet)

- **Root / MCP:** `internal/cmd/root.go` (our structure and MCP).
- **Drive / Sheets:** `internal/cmd/drive.go`, `internal/cmd/sheets.go` (we have `--global`, sheets links; same behaviour as upstream).
- **Docs:** All `internal/cmd/docs_sed*.go`, our `DocsEditCmd` and edit tests; `docs/sedmat.md`, `docs/spec.md`.
- **Secrets:** `internal/secrets/store.go`.
- **Tests:** `docs_validation_more_test.go`, `drive_errors_test.go`, `drive_export_format_test.go`.

---

## 3. Removed to avoid conflicts (candidates for investigation)

- **Upstream `internal/cmd/docs_edit.go`** — Redeclared our `DocsEditCmd`. We kept our edit flow (batch, apply-style, merge-data, MCP). Upstream may add `docs update` / `docs write` behaviour we want to align with.
- **Upstream `docs_paragraphs.go`, `docs_paragraphs_test.go`, `docs_write_update_test.go`** — Type clash with our `docParagraph` in tests. Option: adopt upstream logic under different type names or keep our implementation only.

---

## 4. Optional follow-ups

- **Periodic re-sync:** When upstream adds CHANGELOG/README entries, merge those sections (keep our content).
- **Remaining captureStdout tests:** Tests that still use UI with `io.Discard` inside `captureStdout` get empty JSON. Fix by using context without UI when capturing (see ROOT-CAUSE-AUDIT.md). Affected: gmail_sendas, gmail_thread, info_via_drive, sheets_links, gmail_watch, etc.
- **Lint:** Address wsl_v5, err113, gosec, etc., or adjust `.golangci.yml`.

---

## 5. Investigation notes (see below)

- §6 Docs edit reconciliation  
- §7 Docs paragraphs  
- §8 Drive `--all` alias  

---

## 6. Investigation: Docs edit reconciliation

**Upstream `docs_edit.go`:** Single command `DocsEditCmd` with find/replace only: `DocID`, `Find`, `ReplaceStr`, `MatchCase` (default true). Runs one `ReplaceAllText` request and prints status/replaced count.

**Ours:** We have a richer `DocsEditCmd` in `docs_edit_cmd.go` with **subcommands**: batch, replace, append, insert, delete, apply-style, merge-data, etc. Our `docs edit replace` already covers find/replace (with more options). We also have batch, validate-only, dry-run, and MCP-oriented flows.

**Conclusion:** No merge needed. Upstream’s docs_edit is a **subset** of our functionality. We supersede it. If we wanted CLI parity for a minimal “find + replace” invocation, we could document that `gog docs edit replace <docId> "<find>" "<replace>"` is the equivalent of upstream’s `gog docs edit <docId> --find X --replace Y`.

---

## 7. Investigation: Docs paragraphs

**Upstream `docs_paragraphs.go`:** Defines `docParagraph` (Num, StartIndex, EndIndex, Type, IsBullet, NestLevel, Text, ElemType, TableRows, TableCols) and `paragraphMap` with `buildParagraphMap(doc, tabID)` to build a numbered view of a doc’s structure. Used for doc introspection (e.g. list paragraphs/tables).

**Ours:** We have a **different** `docParagraph` in `docs_sed_integration_test.go`: a test helper with `runs []textRun` used to build mock docs for sed tests (`buildDoc(para(plain("hello")), ...)`). Same type name, different shape → **clash** if we add upstream’s file as-is.

**Conclusion:** Merge is **possible** if we rename one of the two types:
- **Option A:** Rename our test helper to e.g. `sedTestParagraph` (or `mockDocParagraph`) in `docs_sed_integration_test.go` and `docs_sed_integration_edge_test.go`, then add upstream’s `docs_paragraphs.go` (and optionally `docs_paragraphs_test.go`). We don’t currently use `buildParagraphMap` anywhere; we could add it for future use (e.g. richer `docs cat` or list-tabs).
- **Option B:** Copy upstream’s logic but rename their type to e.g. `paragraphMapEntry` and keep our `docParagraph` for tests.

**Recommendation:** Done. We renamed our **test-only** types so upstream’s paragraph map can live in-tree: `docParagraph` → `sedTestParagraph`, `textRun` → `sedTextRun`, `para()` → `sedPara()` in `docs_sed_integration_test.go` and all call sites (`docs_sed_integration_edge_test.go`, `docs_sed_boost1_test.go`, `docs_sed_boost2_test.go`, `docs_sed_boost3_test.go`, `docs_sed_boost3b_test.go`). Added `internal/cmd/docs_paragraphs.go` and `internal/cmd/docs_paragraphs_test.go` from upstream.

---

## 8. Investigation: Drive `--all` alias

**Upstream `drive.go`:** `DriveLsCmd` has a **single** flag: `All bool` with `name:"all" aliases:"global"`. So `--all` and `--global` both mean “list all accessible files” (mutually exclusive with `--parent`). No separate “fetch all pages” flag.

**Ours:** We have **two** flags on `DriveLsCmd`:
- `All bool` with `name:"all"` → “fetch all pages” (paginate until no nextPageToken).
- `Global bool` with `name:"global"` → “list across all accessible files” (same semantics as upstream’s `--all`).

So we use `All` for pagination and `Global` for “list all files”; upstream uses one flag `All` (with alias `global`) for “list all files” only.

**Conclusion:** Adding `--all` as an alias for our `Global` would require Kong to bind both `--all` and `--global` to the same field. Our `All` already has `name:"all"`. So we **cannot** have both: either `--all` means “fetch all pages” (current) or “list all files” (upstream). To match upstream we’d have to rename our pagination flag (e.g. `AllPages bool` with `name:"all-pages"`) and give `Global` the name `"all"` and alias `"global"`. That would be a **breaking change** for anyone using `--all` for pagination.

**Recommendation:** Keep current behaviour. Document in README/CHANGELOG that we use `--global` for “list all accessible files” (equivalent to upstream’s `--all`). No code change.
