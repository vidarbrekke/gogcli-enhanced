# Development / Refactoring Plan

This plan outlines the next stages of development and refactoring for gogcli-enhanced, based on a codebase sanity check. It prioritizes maintainability and correctness without changing behavior unnecessarily, and defers speculative work (YAGNI).

**High-level approach:** Phase 1 modularizes the MCP provider (single 3k-LOC file) and extracts shared schema/arg helpers to reduce cognitive load and duplication. Phase 2 adds targeted input validation where silent coercion is risky. Phase 3 is optional (Drive listing performance). Each phase is independently shippable; tests and `make ci` gate every change.

---

## Scope

- **In:**
  - Splitting `internal/mcp/providers/google/tools.go` by domain (docs, drive, sheets, slides) plus shared helpers
  - Extracting common MCP input-schema fragments and argument-building helpers (policyArgs, maybeOpIDArgs, maybeAccountArgs, sheets base schema)
  - Adding explicit validation for sheet tool inputs where conversion failures currently default to 0 (column, keyColumns, groupBy, metricColumn)
  - Optional: opt-in larger Drive list/search page size for MCP when transport allows
- **Out:**
  - Next.js / Firebase refactors (not in this repo)
  - Declarative/codegen tool registry (defer unless tool churn justifies it)
  - Broad CLI refactors (drive.go, slides_edit_cmd.go, etc.) unless a concrete bug or feature requires it
  - Dependency or toolchain upgrades without explicit approval

---

## Phase 1: Modularize MCP provider and DRY helpers

**Goal:** Reduce cognitive load and duplication in the Google MCP provider without changing behavior.

**Strategy evaluation (4 options):**  
**A — Same-package domain files + shared helpers:** Split into docs_tools.go, drive_tools.go, sheets_tools.go, slides_tools.go, helpers.go. Each domain exports *Specs(p) and its handlers; helpers.go has policyArgs, maybeOpIDArgs, maybeAccountArgs, cleanArgs, asString, asInt, asBool, etc.; tools.go keeps provider, runCLI, Register. Complexity low, DRY good, YAGNI satisfied, scalability high.  
**B — Subpackages per domain:** google/docs, google/drive, etc. Complexity high (runner wiring), DRY same, YAGNI overkill, scalability high.  
**C — Extract only sheets:** Minimal split. Complexity lowest, DRY partial, tools.go still large.  
**D — Schema-only DRY, no split:** Common schema builders only; single file unchanged. Complexity low, DRY schema-only, file stays huge.  
**Choice: A** (best balance; implemented below).

[ ] Add shared schema/arg helpers in `internal/mcp/providers/google/` (e.g. `schema.go` or `helpers.go`): common property maps for `account`, `opId`, `timeoutMs`, `retries`, `retryBackoffMs`; keep existing `policyArgs`, `maybeOpIDArgs`, `maybeAccountArgs` in one place
[ ] Extract docs tools: move docs_planBatch, docs_executeBatch, docs_* handlers and their spec entries to `docs_tools.go`; keep Register() in tools.go calling into the new file or a single registration helper
[ ] Extract drive tools: move drive_* handlers and spec entries to `drive_tools.go`
[ ] Extract sheets tools: move sheets_* handlers and spec entries to `sheets_tools.go`
[ ] Extract slides tools: move slides_* handlers and spec entries to `slides_tools.go`
[ ] Keep `tools.go` as the registration entrypoint (build the toolSpecs slice from domain files and call RegisterToolSpec) or move registration into a single init-style flow
[ ] Run `make fmt` and `make lint`; fix any new issues
[ ] Run `make test` and confirm all MCP tests pass (e.g. `TestGoogleTools_SheetsToolsRegistered`, `TestGoogleTools_SuccessEnvelope_HasServiceAndOperation`, docs/drive/sheets/slides mapping tests)
[ ] Optionally add a small test that ensures the number of registered tools is unchanged after the split (regression guard)

**Concrete step-by-step plan (file moves + guard tests only, no behavior change):** [REFACTOR-PLAN-PHASE1-SPLIT.md](REFACTOR-PLAN-PHASE1-SPLIT.md) (8 steps).

---

## Phase 2: Targeted MCP input validation

**Goal:** Fail fast on invalid sheet (and optionally other) inputs instead of silently defaulting to 0 or wrong behavior.

**Strategy evaluation (4 options):**  
**A — In-handler validation only:** In each sheets* handler, add explicit checks (e.g. column required and numeric; keyColumns/groupBy non-empty int array; op in allowed set). No shared layer. Complexity low, DRY low, YAGNI yes, scalability repetitive.  
**B — Shared validation helpers:** Add validateRequiredInt, validateRequiredIntSlice, validateOp in helpers.go; sheet handlers call them and return invalid_argument envelope. Complexity low, DRY good, YAGNI yes, scalability good.  
**C — Central validation layer:** Pre-handler validation map (tool name → validation func) for all tools. Complexity medium, DRY high, YAGNI no (we only need sheet fixes), scalability high.  
**D — Validate only where silent coercion exists:** Add checks only for the specific cases (column, keyColumns, groupBy, op) without helpers. Complexity lowest, DRY partial, YAGNI yes.  
**Choice: B** (shared helpers in helpers.go; sheet handlers use them).

[ ] Add unit tests that expect `invalid_argument` for invalid sheet inputs: missing or non-numeric `column`, invalid `keyColumns`/`groupBy` (e.g. empty, non-integer), invalid `op` enum
[ ] Introduce a small validation layer for sheet tools: require numeric `column` where required; require non-empty integer arrays for `keyColumns`/`groupBy` where required; reject unknown `op` values
[ ] Ensure `metricColumn` and similar optional numerics either validate or document default (e.g. 0) in the tool description
[ ] Run `make test`; add any missing edge-case tests for new validation paths
[ ] Update TOOLS/runbook docs if any previously accepted input is now rejected (document valid values)

---

## Phase 3 (Optional): Drive listing performance

**Goal:** Reduce round-trips for large Drive list/search when the transport allows larger responses.

**Strategy evaluation (4 options):**  
**A — Optional MCP param (pageSizeMax):** Add optional `pageSizeMax` to drive_listFiles and drive_searchFiles; when set (e.g. ≤100), use it as cap instead of mcpDriveMaxCap; default unchanged. Complexity low, DRY good, YAGNI yes, scalability good.  
**B — Separate tools (e.g. drive_listFilesLarge):** New tools with higher default page size. Complexity low, DRY poor (duplicate tools), YAGNI no.  
**C — Env var (GOG_MCP_DRIVE_PAGE_CAP):** Global cap override. Complexity low, DRY good, not per-request.  
**D — No code change:** Document constraint only. Complexity none.  
**Choice: A** (optional param; default unchanged).

[ ] Document current constraint (gateway/exec result length) in runbook or internal comment
[ ] Add an optional MCP parameter (e.g. `pageSizeMax` or `preferLargerPage`) that, when set, allows page sizes up to a safe cap (e.g. 100) for list/search
[ ] Keep default behavior unchanged (small page size) so existing agents do not need to change
[ ] Run existing Drive MCP tests and integration checks; verify backward compatibility

---

## Open questions

- Preferred layout for split files: one file per domain (docs_tools.go, drive_tools.go, sheets_tools.go, slides_tools.go) with a shared helpers file, or a subpackage per domain (e.g. `google/docs.go`, `google/drive.go`) with shared code in `google/schema.go`. Subpackages may complicate the single `Register()` call.
- Whether Phase 2 validation should apply to all MCP tools (docs, drive, slides) or only sheets in the first iteration.
- Whether Phase 3 is needed soon; if no one reports slow Drive listing, it can stay deferred.

---

## References

- Sanity-check summary: logic/edge-case/performance/debt review (no Next.js/Firebase in repo); recommendation was Strategy 1 (modularize) + slice of Strategy 2 (validation).
- Repo guidelines: `AGENTS.md` (no broad refactors unless requested; minimal diffs; test before/after).
- MCP provider: `internal/mcp/providers/google/tools.go` (~3000 LOC); `internal/mcp/default.go` wires `google.Register(s, executor)`.
- Test gate: `make ci` (fmt, lint, test).
