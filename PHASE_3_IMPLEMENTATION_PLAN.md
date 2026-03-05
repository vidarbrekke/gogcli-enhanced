# Phase 3: Cross-Service Capability Expansion — Detailed Implementation Plan

**Date:** 2026-02-17  
**Status:** Ready for execution  
**Duration Estimate:** 12.5 hours (4-5 sprints)  
**Linear Issues:** VID-111 through VID-116 (6 issues)

---

## 🎯 Mission Statement

Transform gogcli-enhanced from a collection of point operations into a **cohesive enterprise-grade document automation platform** by implementing proven patterns across Docs, Sheets, and Slides.

**Key Principle:** Leverage agentic safety as an orthogonal layer — all new operations inherit `--validate-only`, `--dry-run`, `--pretty` for free.

---

## 📋 DETAILED IMPLEMENTATION ROADMAP (6 Issues, Ranked by Execution Order)

### 🟢 Issue VID-111: Sheets DeleteRange (Quick Win)

**Type:** Feature (gap fill)  
**Duration:** 1.5 hours  
**Priority:** High  
**Effort:** Low  
**Dependencies:** None

**Description:**
Implement `gog sheets edit delete-range` to complete the delete operation family across all services.

**API Reference:**
```
Google Sheets API v4: DeleteRangeRequest
- range: GridRange (A1:C10, etc.)
- shiftDimension: ROWS | COLUMNS
```

**Command Signature:**
```bash
gog sheets edit delete-range <spreadsheetId> \
  --range "A1:C10" \
  --shift-dimension ROWS|COLUMNS \
  --sheet-id <id>
```

**Features to Implement:**
- `--range` (required) — Range to delete (A1 notation)
- `--shift-dimension` (required) — How to shift: ROWS or COLUMNS
- `--sheet-id` (optional) — Target specific sheet
- Standard agentic flags: `--validate-only`, `--dry-run`, `--pretty`, etc.

**Success Criteria:**
1. ✅ Command compiles and runs
2. ✅ `--validate-only` returns structured JSON with requestHash
3. ✅ `--dry-run` shows request structure
4. ✅ Returns `OccurrencesChanged` (cells deleted)
5. ✅ Unit tests for success, error, dry-run paths
6. ✅ `make build && make test` passes

**Implementation Pattern:**
Copy from DocsDeleteCmd, adapt to Sheets DeleteRangeRequest.

**Checklist:**
- [x] Add `SheetsEditDeleteRangeCmd` struct to `sheets_edit_cmd.go`
- [x] Register in `SheetsEditCmd` (sheets.go)
- [x] Implement Run method with agentic pattern
- [x] Test --validate-only
- [x] Test --dry-run
- [x] Test with actual API (if account available)
- [x] Commit and push

---

### 🟢 Issue VID-112: Docs InsertImage (Quick Win + Polish)

**Type:** Feature (new operation)  
**Duration:** 2 hours  
**Priority:** High  
**Effort:** Low  
**Dependencies:** None (but pairs well with ReplaceImage)

**Description:**
Implement `gog docs edit insert-image` to complement ReplaceImage and enable complete image workflow in documents.

**API Reference:**
```
Google Docs API: InsertInlineImageRequest
- uri: String (image URL)
- location: Location (index in document)
- width/height: Optional dimensions
```

**Command Signature:**
```bash
gog docs edit insert-image <docId> \
  --uri "https://example.com/logo.png" \
  --index 1 \
  [--width-pt <width>] [--height-pt <height>]
```

**Features to Implement:**
- `--uri` (required) — Image URI
- `--index` (required) — Insertion index (1-based)
- `--width-pt` (optional) — Width in points
- `--height-pt` (optional) — Height in points
- Standard agentic flags

**Use Cases:**
- Insert logo/header image in mail-merge templates
- Add illustrations at specific positions in reports
- Template image injection workflows

**Success Criteria:**
1. ✅ Command compiles and runs
2. ✅ `--validate-only` works without auth
3. ✅ `--dry-run` shows request
4. ✅ Returns structured JSON with image metadata
5. ✅ Tests pass
6. ✅ Complements ReplaceImage for complete image workflow

**Implementation Pattern:**
Similar to DocsInsertTableCmd (simple operation, single request).

**Checklist:**
- [x] Add `DocsInsertImageCmd` struct to `docs_edit_cmd.go`
- [x] Register in `DocsEditCmd` (docs_cmd.go)
- [x] Implement Run method
- [x] Test complete image workflow (insert + replace)
- [x] Commit and push

---

### 🔵 Issue VID-113: Sheets DeleteRange (Code Review + Testing)

**Type:** Internal (code quality)  
**Duration:** 0.5 hours  
**Priority:** Medium  
**Effort:** Low (after VID-111)

**Description:**
Code review, testing, and documentation for DeleteRange implementation.

**Checklist:**
- [x] Review for agentic pattern compliance
- [x] Verify error handling (invalid ranges, 404s, etc.)
- [x] Unit tests (if applicable)
- [x] Documentation strings
- [x] Lint check

**Done:** DeleteRange (VID-111) reviewed; agentic pattern, force guard, validate-only/dry-run, structured errors, 404 handling, tests in place.

---

### 🟠 Issue VID-114: Design Review — Docs/Sheets MergeData Pattern

**Type:** Design/Planning  
**Duration:** 1.5 hours  
**Priority:** High  
**Effort:** Low (but critical)

**Description:**
Design the MergeData pattern for Docs and Sheets. This is **critical** because MergeData is the most complex operation and will set the pattern for any future cross-service "create copy" operations.

**Design Tasks:**

1. **Analyze Slides MergeData Implementation**
   - Read through `SlidesEditMergeDataCmd` implementation
   - Understand: copy API, batch update flow, data parsing, error handling
   - Document pattern

2. **Adapt Pattern for Docs**
   - Docs template needs: {{placeholder}} detection
   - How to handle complex formatting preservation?
   - Output: mail-merged PDFs or Google Docs?
   - Error handling: What if template is broken?

3. **Adapt Pattern for Sheets**
   - Sheets template needs: {{placeholder}} detection
   - Row/column structure: preserve template structure?
   - How to handle formulas in template?
   - Output: multiple sheets or separate spreadsheets?

4. **Create Design Document**
   - CLI interface for both
   - Data file format (JSON array of objects)
   - Filename/title templating ({{placeholder}})
   - Error scenarios
   - Success response format

**Deliverable:**
- Detailed design doc (500+ words)
- CLI sketches for both Docs and Sheets
- Data format examples
- Error handling matrix

**Checklist:**
- [x] Analyze Slides implementation thoroughly
- [x] Research Docs/Sheets template best practices
- [x] Sketch CLI for both services
- [x] Create design doc in repo
- [x] Get conceptual approval
- [x] Identify unknowns/blockers

**Done:** Design doc at `docs/PHASE_3_MERGEDATA_DESIGN.md` (2026-02-17). Ready for VID-115/116 implementation.

---

### 🔴 Issue VID-115: Implement Docs MergeData (Transformative)

**Type:** Feature (transformative)  
**Duration:** 3.5-4 hours  
**Priority:** Very High  
**Effort:** Medium  
**Dependencies:** VID-114 (design complete)

**Description:**
Implement `gog docs edit merge-data` — the first step to mail-merge capability.

**Command Signature:**
```bash
gog docs edit merge-data <templateId> \
  --data-file employees.json \
  --filename-format "Offer Letter - {{firstName}} {{lastName}}" \
  --output-folder-id <folderId>
```

**Data File Format (employees.json):**
```json
[
  {
    "firstName": "Alice",
    "lastName": "Engineer",
    "title": "Senior Software Engineer",
    "salary": "$200,000"
  },
  {
    "firstName": "Bob",
    "lastName": "Manager",
    "title": "Product Manager",
    "salary": "$180,000"
  }
]
```

**Template:** Google Doc with {{firstName}}, {{lastName}}, {{title}}, {{salary}} placeholders

**Execution Flow:**
1. Parse data file (JSON array validation)
2. For each record:
   - Copy template doc
   - Batch replace all {{key}} with values
   - Rename to filename-format
3. Return summary: N docs created, folder link

**Features:**
- `--data-file` (required) — JSON array of objects
- `--filename-format` (required) — Template with {{placeholders}}
- `--output-folder-id` (optional) — Destination folder (default: same as template)
- `--include-timestamp` (optional) — Append timestamp to filenames
- `--dry-run` — Preview operations without creating docs
- `--validate-only` — Validate data + template structure

**Success Criteria:**
1. ✅ Compiles and runs
2. ✅ Validates JSON structure
3. ✅ `--validate-only` works (preview first N records)
4. ✅ `--dry-run` shows operations without API calls
5. ✅ Creates N documents in output folder
6. ✅ Renames docs using filename-format
7. ✅ Returns summary with links
8. ✅ Tests cover: success, error, dry-run, validate-only
9. ✅ Error handling: invalid JSON, missing folder, template not found

**Implementation Approach:**
1. Create `DocsEditMergeDataCmd` struct
2. Parse and validate data file
3. Implement copy + batch replace loop
4. Implement output folder organization
5. Handle errors gracefully
6. Return structured summary

**Key Patterns:**
- Use existing `RequestHash()`, `NormalizedRequestForOutput()` from shared helpers
- Reuse `DryRunOutput()` for preview
- Handle Drive folder operations
- Batch replace like `DocsReplaceCmd`

**Checklist:**
- [ ] Implement command struct
- [ ] Parse data file with validation
- [ ] Test data parsing locally
- [ ] Implement copy + batch replace loop
- [ ] Implement --validate-only (preview)
- [ ] Implement --dry-run (show operations)
- [ ] Test with real template (if account available)
- [ ] Unit tests (mock API calls)
- [ ] Error handling tests
- [ ] Commit and push

---

### 🔴 Issue VID-116: Implement Sheets MergeData (Transformative)

**Type:** Feature (transformative)  
**Duration:** 3.5-4 hours  
**Priority:** Very High  
**Effort:** Medium  
**Dependencies:** VID-114 (design complete), VID-115 (Docs pattern reference)

**Description:**
Implement `gog sheets edit merge-data` — unlocks dynamic report generation and multi-sheet creation from template.

**Command Signature:**
```bash
gog sheets edit merge-data <templateId> \
  --data-file reports.json \
  --filename-format "Report - {{quarter}} {{year}}" \
  --output-folder-id <folderId>
```

**Data File Format (reports.json):**
```json
[
  {
    "quarter": "Q1",
    "year": "2026",
    "revenue": "$1.2M",
    "growth": "+15%",
    "forecast": "$1.5M"
  },
  {
    "quarter": "Q2",
    "year": "2026",
    "revenue": "$1.4M",
    "growth": "+17%",
    "forecast": "$1.6M"
  }
]
```

**Template:** Google Sheet with {{quarter}}, {{year}}, {{revenue}}, {{growth}}, {{forecast}} placeholders

**Execution Flow:**
1. Parse data file (JSON validation)
2. For each record:
   - Copy template spreadsheet
   - Batch replace all {{key}} with values
   - Rename to filename-format
3. Return summary: N sheets created, folder link

**Features:**
- Same as Docs: --data-file, --filename-format, --output-folder-id, --include-timestamp
- --validate-only, --dry-run
- Sheets-specific: handle multiple sheets in template (replace across all)

**Use Cases:**
- Generate monthly/quarterly financial reports
- Create dashboard templates with company-specific data
- Multi-region budgets from template
- Territory-specific sales forecasts

**Success Criteria:**
1. ✅ Compiles and runs
2. ✅ Validates JSON
3. ✅ --validate-only works
4. ✅ --dry-run shows operations
5. ✅ Creates N spreadsheets
6. ✅ Replaces across all sheets in template
7. ✅ Handles formulas gracefully (don't break them)
8. ✅ Error handling: invalid JSON, formula conflicts, etc.

**Implementation Approach:**
1. Create `SheetsEditMergeDataCmd` struct (very similar to Docs)
2. Reuse data parsing logic
3. Adapt to Sheets API (copy spreadsheet, batch update cells)
4. Handle multi-sheet replacement
5. Preserve formulas while replacing values

**Sheets-Specific Considerations:**
- Template may have formulas → ensure replacement preserves formula structure
- May have multiple sheets → replace placeholder only in value cells
- Cell ranges → use A:Z notation for entire columns (safer)

**Checklist:**
- [ ] Implement command struct
- [ ] Reuse data parsing from Docs (DRY)
- [ ] Implement copy + batch update loop
- [ ] Handle multi-sheet templates
- [ ] Test formula preservation
- [ ] Implement --validate-only
- [ ] Implement --dry-run
- [ ] Unit tests
- [ ] Error handling tests
- [ ] Commit and push

---

## 📊 EXECUTION TIMELINE

```
Week 1 (This Week):
  ✅ VID-111: Sheets DeleteRange (1.5h)
  ✅ VID-112: Docs InsertImage (2h)
  ⏳ VID-114: Design Review — Docs/Sheets MergeData (1.5h)

Week 2:
  🔄 VID-115: Docs MergeData (3.5-4h)
  🔄 VID-116: Sheets MergeData (3.5-4h)

Total: 12.5 hours over 2 weeks
```

---

## 🎯 SUCCESS CRITERIA (Overall)

### Code Quality
- ✅ All commands follow agentic pattern (inherit safety flags)
- ✅ All build cleanly (`make build`)
- ✅ All pass tests (`make test`)
- ✅ All handle errors gracefully (structured error JSON)
- ✅ Zero breaking changes

### Feature Completeness
- ✅ 4 operations implemented (Delete, InsertImage, MergeData×2)
- ✅ 6/6 identified gaps addressed
- ✅ Cross-service consistency (same patterns across Docs/Sheets/Slides)
- ✅ Enterprise-grade capabilities (mail-merge + report generation)

### Documentation
- ✅ Each command has --help text
- ✅ Implementation documented in handover.md
- ✅ User guide examples in `docs/editing.md` (if exists)
- ✅ Git history clean and searchable

### Testing
- ✅ All --validate-only paths work
- ✅ All --dry-run paths work
- ✅ Error handling tested (404, invalid JSON, permissions)
- ✅ Happy path tests (if account available)

---

## 🚀 EXPECTED IMPACT

### Platform Transformation
| Capability | Before | After |
|-----------|--------|-------|
| Consistent pattern | ✓ (Docs) | ✓ (All 3 services) |
| ReplaceText | ✓✓ (Docs, Slides) | ✓✓✓ (All 3) |
| Delete operations | ✓ (Docs) | ✓✓ (Docs, Sheets) |
| Image operations | ✓ (Slides) | ✓✓ (Docs, Slides) |
| **Mail-merge** | ✓ (Slides) | ✓✓✓ (All 3 + use-cases) |

### User Value
- **60% of users** unlock with Docs MergeData (mail-merge for documents)
- **50% of users** unlock with Sheets MergeData (report generation)
- **100% of users** benefit from cross-service consistency
- **→ Platform cohesion: from point tools to automation platform**

---

## 📝 NOTES FOR IMPLEMENTERS

### Pattern Reference
- See `DocsReplaceCmd` for agentic pattern template
- See `SlidesEditMergeDataCmd` for MergeData pattern
- All use `RequestHash()`, `NormalizedRequestForOutput()`, `DryRunOutput()` from `edit_helpers.go`

### Common Pitfalls to Avoid
1. Don't re-implement safety flags — they're already in `AgenticEditSafetyFlags`
2. Don't call `requireAccount()` in validate-only path (skip API setup)
3. Don't forget error.As() for structured error JSON
4. Do test --validate-only with empty/invalid data
5. Do test --dry-run without making API calls

### Testing Strategy
- Unit tests: validate request structure locally
- Integration tests (if account): test full flow
- Mock tests: simulate API responses
- Error tests: invalid JSON, 404s, permissions

---

## 🔗 RELATED ISSUES

- **VID-91** — Phase 1: Shared Agentic Edit Helpers (foundation)
- **VID-92-95** — Phase 2A: Sheets Edit (reference pattern)
- **VID-107** — Phase 2B: Docs Edit Agentic Refactor (completed)
- **VID-100-106** — Phase 3A: Slides Edit Operations (reference pattern)
- **VID-111-116** — Phase 3B: Cross-Service Gap Fills (THIS PLAN)

---

## 📞 BLOCKERS & ASSUMPTIONS

**Assumptions:**
- Google API credentials available for testing
- Drive API access enabled (for copy operations)
- No breaking changes to underlying Google APIs

**Potential Blockers:**
- Sheets formula preservation during replace (may need special handling)
- Large data files (memory constraints with JSON parsing)
- Drive folder permission constraints

**Mitigation:**
- Design review (VID-114) will surface Sheets formula issues early
- Add data file size validation (warn if >10MB)
- Test with limited permissions, document requirements

---

**Document Status:** Ready for Linear intake  
**Prepared by:** 10x engineering review  
**Next Step:** Create Linear issues VID-111 through VID-116
