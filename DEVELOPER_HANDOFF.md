# Developer Handoff: gogcli-enhanced Sheets & Slides Agentic Edit

**Date:** 2026-02-17  
**Repository:** vidarbrekke/gogcli-enhanced  
**Status:** Phase 1 Complete, Phases 2-5 Ready for Implementation

---

## ✅ What's Complete (VID-91)

### Phase 1: Shared Foundation
**File:** `internal/cmd/edit_helpers.go` (305 lines)

Created unified helpers for agent-safe editing across Docs/Sheets/Slides:

```go
// Core safety flags - embed in all edit commands
type AgenticEditSafetyFlags struct {
    DryRun            bool   // Build request without executing
    ValidateOnly      bool   // Local validation only, no API/auth
    Pretty            bool   // Include normalized JSON in output
    OutputRequestFile string // Write request JSON to file
    ExecuteFromFile   string // Execute from pre-saved request
    RequireRevision   string // Prevent conflicts (Docs only)
}

// Unified error type with structured metadata
type EditError struct {
    Service      string // "docs", "sheets", "slides"
    Operation    string // "values", "append", "clear", "batch"
    ResourceID   string // doc_id, spreadsheet_id, presentation_id
    ErrorCode    string // machine-readable error code
    HTTPStatus   int
    GoogleReason string
    RequestIndex *int   // for batch operations
}
```

**Key Functions:**
- `RequestHash(req)` - deterministic SHA256 hash for request validation
- `NormalizedRequestString(req)` - pretty-printed JSON for output
- `NormalizedRequestForOutput(ctx, path, req)` - respects JSON context
- `RequestOperationCount(r)`, `RequestOperationName(r)` - reflection utilities
- `DryRunOutput(ctx, u, service, id, req, extra, pretty)` - unified dry-run
- `NewEditError(service, op, id, code, msg, cause)` - structured errors

**Pattern to Follow:**
```go
// 1. Embed safety flags
type SheetsEditValuesCmd struct {
    SpreadsheetID string `arg:"" name:"spreadsheetId"`
    Range         string `arg:"" name:"range"`
    Safety        AgenticEditSafetyFlags `embed:""`
}

// 2. Validate and build request (NO API calls yet)
req := &myRequest{...}

// 3. Handle --validate-only (no auth needed)
if c.Safety.ValidateOnly {
    return outputValidateOnly(ctx, req)
}

// 4. Handle --dry-run (no API calls)
if c.Safety.DryRun {
    return DryRunOutput(ctx, u, "sheets", spreadsheetID, req, extra, c.Safety.Pretty)
}

// 5. Execute with proper error handling
account, _ := requireAccount(flags)
svc, _ := newSheetsService(ctx, account)
resp, err := svc.Spreadsheets.Values.Update(...).Do()
if err != nil {
    return NewEditError("sheets", "values", spreadsheetID, "api_error", "...", err)
}
```

---

## 🚧 Integration Challenge (Needs Your Work)

### The Problem
There's existing WIP code for Sheets editing from before this session:
- `internal/cmd/sheets.go` - Has legacy `SheetsEditValuesCmd`, `SheetsEditAppendRowsCmd`, etc.
- `internal/cmd/sheets_edit_cmd.go` - Has newer WIP with some agentic flags
- `internal/cmd/sheets_format.go` - SheetsFormatCmd implementation
- `internal/cmd/sheets_validation.go` - copyDataValidation helper

These have **duplicate declarations** and **different function signatures** than the new standard.

### Your Task
Integrate the new shared helpers from `edit_helpers.go` into the existing code. You'll need to:

1. **Standardize error handling** - Replace `newSheetsEditError()` with `NewEditError("sheets", ...)`
2. **Merge command implementations** - Combine the best of old and new
3. **Add missing safety flags** - Ensure all commands have full agentic support
4. **Update tests** - Make `internal/cmd/sheets_edit_test.go` pass

### Conflicting Files
```
internal/cmd/sheets.go              # Legacy edit commands (lines ~40-620)
internal/cmd/sheets_edit_cmd.go       # WIP refactored version
internal/cmd/sheets_format.go         # Format command
internal/cmd/sheets_validation.go   # Validation helpers
```

### Recommended Approach
1. Study `edit_helpers.go` - understand the shared utilities (30 min)
2. Look at `internal/cmd/docs_edit_cmd.go` - see the reference implementation (30 min)
3. Pick ONE sheets edit command (e.g., `sheets edit values`)
4. Rewrite it using the shared pattern above
5. Remove the duplicate old implementation
6. Run tests: `go test ./internal/cmd/... -v -run Sheets`
7. Repeat for other commands

---

## 📋 Linear Issue Status

### ✅ Completed
- **VID-91** - Phase 1: Shared Agentic Edit Helpers Foundation
- **VID-99** - Commit WIP Sheets Edit Code (committed as 75270aa before cleanup)

### 🚧 In Progress / Blocked by Integration
- **VID-92** - Sheets Edit: Values Command (WIP exists, needs merge)
- **VID-93** - Sheets Edit: Append Command (WIP exists, needs merge)
- **VID-94** - Sheets Edit: Clear Command (needs implementation)
- **VID-95** - Sheets Edit: Batch Command (needs implementation)

### 📋 Pending
- **VID-96** - Slides Edit: Batch MVP (4-6 days)
- **VID-97** - Cross-Service Agentic Hardening (2-3 days)
- **VID-98** - Documentation & Handoff (1-2 days)

---

## 🎯 Success Criteria

For each command to be done:
1. ✅ Uses `AgenticEditSafetyFlags` embedded struct
2. ✅ `validate-only` mode works without auth
3. ✅ `dry-run` mode builds request without API calls
4. ✅ `--pretty` includes normalized JSON output
5. ✅ `--output-request-file` writes request to file
6. ✅ `--execute-from-file` executes pre-saved request
7. ✅ Returns `EditError` with all fields populated on failure
8. ✅ Unit tests for success, dry-run, validate-only, error paths
9. ✅ `make test && make lint` passes

---

## 📚 Reference Implementation

**Best example:** `internal/cmd/docs_edit_cmd.go`
- Shows full agentic workflow
- Has comprehensive error handling
- Uses `DocsEditSafetyFlags` (alias of `AgenticEditSafetyFlags`)
- See how `dry-run` runs without `requireAccount()`

---

## 🔧 Quick Start for New Dev

```bash
# 1. Clone and setup
git clone https://github.com/vidarbrekke/gogcli-enhanced
cd gogcli-enhanced

# 2. Check current state
git log --oneline -5
make test  # Will fail initially due to conflicts

# 3. Study the foundation
less internal/cmd/edit_helpers.go
less internal/cmd/docs_edit_cmd.go

# 4. Pick a command to implement
# Recommended: start with 'sheets edit values' (VID-92)

# 5. Create feature branch
git checkout -b feat/sheets-edit-values

# 6. Implement following the Pattern above

# 7. Test
go test ./internal/cmd/... -v -run "SheetsEditValues"
make lint

# 8. Commit and PR
git commit -m "feat(sheets): implement edit values with agentic safety"
```

---

## 📞 Questions?

- Check `handover.md` - Original handover from 2026-02-11
- Check `docs/refactor/external-review-feedback.md` - Prioritized improvements
- Review Linear issues VID-92 through VID-98 for full scope

---

**Bottom line:** The hard part (shared infrastructure) is done. The integration work (merging old WIP with new standards) needs careful attention. Budget 2-3 days for full Sheets + Slides completion.
