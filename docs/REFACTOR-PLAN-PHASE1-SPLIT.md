# Phase 1 Split: No-Behavior-Change Refactor Plan

**Goal:** Decompose `internal/mcp/providers/google/tools.go` by domain (docs, sheets, slides, drive) so each domain’s **tool specs** live in a dedicated file. Handler implementations stay in `tools.go`; only the `ToolSpec` slice definitions move. No logic changes, no new behavior.

**Guard:** Existing test `TestGoogleTools_SheetsToolsRegistered` already asserts **57 tools**; it must keep passing after every step.

---

## Step 1 — Confirm guard test

- [ ] Run: `go test -v -run TestGoogleTools_SheetsToolsRegistered ./internal/mcp`
- [ ] Ensure it passes and expects exactly **57** registered tools.
- [ ] (Optional) Add a short comment in the test that the 57 count is the regression guard for the provider split (already present: “Regression: expect 57 tools total after provider modularization”).

**No code change required** if the test already exists and passes; this step is validation only.

---

## Step 2 — Create `docs_tools.go`

- [ ] In `internal/mcp/providers/google/`, create `docs_tools.go` (same package `google`).
- [ ] Add:
  - `func docsSpecs(p *provider) []server.ToolSpec`
  - Return a slice containing **only** the docs tool specs, in current order: `docs_planBatch`, `docs_executeBatch`, `docs_sed`, `docs_smartEdit`, `docs_create`, `docs_createWithBody`, `docs_insertText`, `docs_deleteRange`, `docs_replaceAllText`, `docs_appendText`, `docs_insertTable`, `docs_insertImage`, `docs_locatorEdit`, `docs_mergeData`, `docs_get`, `docs_cat`, `docs_listTabs`, `docs_positionsEnd`, `docs_positionsSearch`, `docs_positionsHeadings`.
- [ ] Copy the exact `Name`, `Description`, `Tier`, `Version`, `PolicyClass`, `InputSchema`, `Handler` from `tools.go` (lines 75–488 in current layout). Do not change any field values.
- [ ] Import only what’s needed: `github.com/steipete/gogcli/internal/mcp/server`.

**Do not** move any handler code; handlers stay in `tools.go`. Leave `tools.go` unchanged for now (still with the full inline slice).

---

## Step 3 — Create `sheets_tools.go`

- [ ] In `internal/mcp/providers/google/`, create `sheets_tools.go` (same package `google`).
- [ ] Add:
  - `func sheetsSpecs(p *provider) []server.ToolSpec`
  - Return a slice containing **only** the sheets tool specs in current order: `sheets_planBatch`, `sheets_executeBatch`, `sheets_valuesUpdate`, `sheets_valuesAppend`, `sheets_links`, `sheets_valuesGet`, `sheets_valuesRead`, `sheets_sortRange`, `sheets_dedupeRows`, `sheets_filterCopyRows`, `sheets_upsertRows`, `sheets_moveRows`, `sheets_applyFormula`, `sheets_summarize`.
- [ ] Copy the exact spec structs from `tools.go` (current block from first sheets spec through the last one before `slides_planBatch`).
- [ ] Same import as in Step 2.

---

## Step 4 — Create `slides_tools.go`

- [ ] In `internal/mcp/providers/google/`, create `slides_tools.go` (same package `google`).
- [ ] Add:
  - `func slidesSpecs(p *provider) []server.ToolSpec`
  - Return a slice containing **only** the slides tool specs in current order: `slides_planBatch`, `slides_executeBatch`, `slides_replaceText`, `slides_createSlide`.
- [ ] Copy the exact spec structs from `tools.go` (current block from first slides spec through the last one before `drive_ensureFolder`).
- [ ] Same import as in Step 2.

---

## Step 5 — Create `drive_tools.go`

- [ ] In `internal/mcp/providers/google/`, create `drive_tools.go` (same package `google`).
- [ ] Add:
  - `func driveSpecs(p *provider) []server.ToolSpec`
  - Return a slice containing **only** the drive tool specs in current order: `drive_ensureFolder`, `drive_untrash`, `drive_getPermission`, `drive_listFiles`, `drive_searchFiles`, `drive_getFile`, `drive_uploadFile`, `drive_downloadFile`, `drive_listPermissions`, `drive_listComments`, `drive_deleteFile`, `drive_moveFile`, `drive_renameFile`, `drive_shareFile`, `drive_unshare`, `drive_createComment`, `drive_deleteComment`, `drive_copyFile`, `drive_bulkExecute`.
- [ ] Copy the exact spec structs from `tools.go` (current block from first drive spec to the closing `}` of the big slice, i.e. the last `},` before `}` and `for _, spec := range toolSpecs`).
- [ ] Same import as in Step 2.

---

## Step 6 — Wire `tools.go` to domain slices

- [ ] In `tools.go`, in `Register`, **remove** the entire inline `toolSpecs := []server.ToolSpec{ ... }` literal (all spec entries).
- [ ] Replace with building the slice from the four domain functions, preserving **order**: docs → sheets → slides → drive.

  ```go
  toolSpecs := append(append(append(append([]server.ToolSpec{},
      docsSpecs(p)...),
      sheetsSpecs(p)...),
      slidesSpecs(p)...),
      driveSpecs(p)...)
  ```

- [ ] Keep the rest of `Register` unchanged: `for _, spec := range toolSpecs { s.RegisterToolSpec(spec) }`.
- [ ] Ensure no duplicate or missing specs; the only behavioral contract is “same 57 tools in same order.”

---

## Step 7 — Run gate and fix any issues

- [ ] Run: `make fmt`
- [ ] Run: `make lint`
- [ ] Run: `make test` (or `make ci`)
- [ ] Confirm `TestGoogleTools_SheetsToolsRegistered` passes (57 tools).
- [ ] Confirm all other MCP and provider tests pass (e.g. `TestGoogleTools_SuccessEnvelope_HasServiceAndOperation`, Drive list/search tests, docs/sheets/slides tool tests).
- [ ] Fix any compile or test failures **only** by correcting copy/paste or imports; do not change behavior or add features.

---

## Step 8 — Document the split (optional)

- [ ] In `tools.go`, add a one-line comment above `toolSpecs := append(...)` explaining that tool specs are split by domain (docs, sheets, slides, drive) for maintainability; see `docs_tools.go`, `sheets_tools.go`, `slides_tools.go`, `drive_tools.go`.
- [ ] In `docs/DEVELOPMENT-PLAN.md`, in Phase 1, add a bullet or link: “Concrete step-by-step plan: `docs/REFACTOR-PLAN-PHASE1-SPLIT.md`.”

---

## Summary

| Step | Action |
|------|--------|
| 1 | Confirm 57-tool guard test exists and passes |
| 2 | Add `docs_tools.go` with `docsSpecs(p)` |
| 3 | Add `sheets_tools.go` with `sheetsSpecs(p)` |
| 4 | Add `slides_tools.go` with `slidesSpecs(p)` |
| 5 | Add `drive_tools.go` with `driveSpecs(p)` |
| 6 | In `tools.go`, replace inline slice with `append(docsSpecs(p), sheetsSpecs(p), ...)` |
| 7 | `make fmt` / `make lint` / `make test`; fix only mechanical issues |
| 8 | (Optional) Comment in `tools.go` + link in `DEVELOPMENT-PLAN.md` |

**Out of scope for this plan:** Moving handler implementations into domain files, changing any handler logic, adding new tools or validation. Those can be done in a later change.
