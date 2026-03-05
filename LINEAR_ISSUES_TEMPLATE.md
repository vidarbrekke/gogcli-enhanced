# Linear Issues Template: Phase 3 Cross-Service Capability Expansion

**Copy-paste these into Linear to create issues VID-111 through VID-116**

---

## Issue VID-111: Sheets DeleteRange (Quick Win)

**Title:** Sheets Edit: DeleteRange - Delete cell range with shift options  
**Priority:** High  
**Team:** Engineering  
**Estimate:** 1.5 hours  
**Status:** Todo  
**Related Issues:** VID-107 (Docs Edit Refactor), VID-112 (Docs InsertImage)

**Description:**

Implement `gog sheets edit delete-range` to complete the delete operation family across all services.

This fills a capability gap: Delete operations exist in Docs (`delete`), but not in Sheets. However, the Google Sheets API supports DeleteRangeRequest, so the gap is CLI-only.

**Command Signature:**
```bash
gog sheets edit delete-range <spreadsheetId> \
  --range "A1:C10" \
  --shift-dimension ROWS|COLUMNS \
  [--sheet-id <id>]
```

**Features:**
- `--range` (required) — Range to delete (A1 notation)
- `--shift-dimension` (required) — How to shift: ROWS or COLUMNS
- `--sheet-id` (optional) — Target specific sheet
- Standard agentic flags: `--validate-only`, `--dry-run`, `--pretty`, etc. (inherited)

**API Reference:**
- Google Sheets API v4: `DeleteRangeRequest`
- Batch operation: `sheets.Spreadsheets.BatchUpdate`

**Success Criteria:**
- [ ] Command compiles with `go build ./internal/cmd`
- [ ] `--validate-only` returns structured JSON with requestHash
- [ ] `--dry-run` shows request structure without API call
- [ ] Returns `OccurrencesChanged` (cells deleted) in response
- [ ] Error handling: invalid range, 404, permissions
- [ ] Unit tests for success, error, dry-run paths
- [ ] `make test && make lint` passes

**Implementation Notes:**
- Pattern: Copy from `DocsDeleteCmd`, adapt to Sheets API
- Reuse: `RequestHash()`, `NormalizedRequestForOutput()`, `DryRunOutput()` from `edit_helpers.go`
- File: `internal/cmd/sheets_edit_cmd.go` + register in `sheets.go`

**Testing:**
```bash
# Validate-only
gog sheets edit delete-range test-id --range "A1:C10" --shift-dimension ROWS --validate-only --json

# Dry-run
gog sheets edit delete-range test-id --range "A1:C10" --shift-dimension COLUMNS --dry-run --json
```

---

## Issue VID-112: Docs InsertImage (Quick Win + Polish)

**Title:** Docs Edit: InsertImage - Insert image at specific position  
**Priority:** High  
**Team:** Engineering  
**Estimate:** 2 hours  
**Status:** Todo  
**Related Issues:** VID-107 (Docs Edit Refactor), VID-111 (Sheets DeleteRange)

**Description:**

Implement `gog docs edit insert-image` to complement existing `replace-image` (VID-4893e06) and enable complete image workflow in document templates.

This enables document template branding workflows: insert logo at beginning, then use in mail-merge templates with image replacement for dynamic branding.

**Command Signature:**
```bash
gog docs edit insert-image <docId> \
  --uri "https://example.com/logo.png" \
  --index 1 \
  [--width-pt <width>] [--height-pt <height>] \
  [--tab-id <id>]
```

**Features:**
- `--uri` (required) — Image URI (must be publicly accessible)
- `--index` (required) — Insertion index (1-based position in document)
- `--width-pt` (optional) — Width in points
- `--height-pt` (optional) — Height in points
- `--tab-id` (optional) — Target specific tab (omit for first tab)
- Standard agentic flags: `--validate-only`, `--dry-run`, `--pretty`, etc.

**Use Cases:**
- Insert logo/header image in mail-merge templates
- Add illustrations at specific positions in reports
- Template image injection workflows

**API Reference:**
- Google Docs API: `InsertInlineImageRequest`
- Batch operation: `docs.Documents.BatchUpdate`

**Success Criteria:**
- [ ] Command compiles
- [ ] `--validate-only` works without auth
- [ ] `--dry-run` shows request without API call
- [ ] Returns structured JSON with image metadata
- [ ] Complements `replace-image` for complete image workflow
- [ ] Error handling: invalid URI, 404 document, permissions
- [ ] Unit tests pass
- [ ] `make test && make lint` passes

**Implementation Notes:**
- Pattern: Similar to `DocsInsertTableCmd` (single operation, simple request)
- File: `internal/cmd/docs_edit_cmd.go` + register in `docs_cmd.go`
- Reference: `DocsReplaceImageCmd` for image-related patterns

**Testing:**
```bash
# Validate-only
gog docs edit insert-image test-doc --uri "https://example.com/logo.png" --index 1 --validate-only --json

# With dimensions
gog docs edit insert-image test-doc --uri "https://example.com/logo.png" --index 1 --width-pt 200 --height-pt 100 --dry-run --json
```

---

## Issue VID-113: Code Review - Sheets DeleteRange & Docs InsertImage

**Title:** Code Review & Testing - Phase 3 Quick Wins (VID-111, VID-112)  
**Priority:** Medium  
**Team:** Engineering  
**Estimate:** 0.5 hours  
**Status:** Todo  
**Depends On:** VID-111, VID-112

**Description:**

Code review, testing validation, and quality assurance for VID-111 (Sheets DeleteRange) and VID-112 (Docs InsertImage).

**Checklist:**
- [ ] Review both implementations for agentic pattern compliance
- [ ] Verify error handling (invalid input, 404s, permissions)
- [ ] Verify dry-run works without API calls
- [ ] Verify validate-only works without auth
- [ ] Check lint/formatting (`make lint`)
- [ ] Run unit tests (`make test`)
- [ ] Spot-check documentation strings
- [ ] Verify git commits are clean and descriptive

**Success Criteria:**
- [ ] Both commands pass all quality checks
- [ ] Ready for merge to main branch

---

## Issue VID-114: Design Review - Docs/Sheets MergeData Pattern

**Title:** Design Review - Mail-Merge Pattern for Docs & Sheets  
**Priority:** Very High  
**Team:** Engineering  
**Estimate:** 1.5 hours  
**Status:** Todo  
**Blockers:** None

**Description:**

Design the MergeData pattern for Docs and Sheets. This is a critical design phase that will determine the quality and consistency of the transformative VID-115 and VID-116 implementations.

MergeData is proven in Slides (VID-100-106), but adapting it to Docs and Sheets requires thoughtful design because:
1. Docs templates can be arbitrarily complex (hundreds of pages, formatting)
2. Sheets templates have formulas that must not break during replacement
3. Both need consistent CLI interfaces

**Design Tasks:**

1. **Analyze Slides MergeData Implementation**
   - Read `SlidesEditMergeDataCmd` implementation (internal/cmd/slides_edit_cmd.go)
   - Understand: copy semantics, batch replace flow, placeholder detection ({{key}})
   - Document pattern for replication

2. **Docs-Specific Design**
   - How to preserve complex formatting during {{key}} replacement?
   - Should output be Google Docs or PDF?
   - How to handle multi-tab documents?
   - Error scenarios: template broken, invalid placeholders, etc.

3. **Sheets-Specific Design**
   - How to preserve formulas during {{key}} replacement?
   - How to handle multi-sheet templates?
   - Should we replace only values or also formula references?
   - How to handle named ranges?

4. **CLI Design (Both Services)**
   - Data file format: JSON array of objects (consistent with Slides)
   - Filename/title templating: {{placeholder}} syntax
   - Output folder: optional (default to same folder as template)
   - Timestamp option: append to filenames for uniqueness

5. **Error Handling Matrix**
   - Invalid JSON (unparseable, empty, etc.)
   - Missing placeholders in template
   - Template not found (404)
   - Output folder not found
   - Permission errors (read template, write to folder)
   - Large data files (>10MB)

**Deliverable:**
- Design document (300+ words) covering all above topics
- CLI sketches for both Docs and Sheets
- Data file format examples
- Error handling matrix
- Document in repo for reference

**Success Criteria:**
- [ ] Slides implementation thoroughly analyzed
- [ ] Docs design finalized (documented, no unknowns)
- [ ] Sheets design finalized (formula handling solved)
- [ ] CLI sketches match pattern across both services
- [ ] Data format examples provided
- [ ] Error handling matrix complete
- [ ] Design doc checked into repo
- [ ] Ready for implementation (VID-115, VID-116)

**Reference Documents:**
- `SlidesEditMergeDataCmd` — Proven pattern
- `CROSS_SERVICE_OPPORTUNITY_ANALYSIS.md` — Strategic context
- `PHASE_3_IMPLEMENTATION_PLAN.md` — Implementation approach

---

## Issue VID-115: Implement Docs MergeData (Transformative)

**Title:** Docs Edit: MergeData - Mail-merge documents from template + data  
**Priority:** Very High  
**Team:** Engineering  
**Estimate:** 3.5-4 hours  
**Status:** Todo  
**Depends On:** VID-114 (design complete)

**Description:**

Implement `gog docs edit merge-data` — mail-merge for Google Docs. This is a transformative feature that enables enterprise document generation at scale.

Given a template document and JSON data file, generate N personalized documents by:
1. Copying the template for each record
2. Replacing all {{placeholder}} with data values
3. Renaming documents according to format string
4. Organizing in output folder

**Command Signature:**
```bash
gog docs edit merge-data <templateId> \
  --data-file employees.json \
  --filename-format "Offer Letter - {{firstName}} {{lastName}}" \
  --output-folder-id <folderId> \
  [--include-timestamp]
```

**Data File Format (employees.json):**
```json
[
  {
    "firstName": "Alice",
    "lastName": "Engineer",
    "title": "Senior Software Engineer",
    "salary": "$200,000",
    "startDate": "2026-03-01"
  },
  {
    "firstName": "Bob",
    "lastName": "Manager",
    "title": "Product Manager",
    "salary": "$180,000",
    "startDate": "2026-03-15"
  }
]
```

**Template Document:**
A Google Doc with {{firstName}}, {{lastName}}, {{title}}, {{salary}}, {{startDate}} placeholders.

**Execution Flow:**
1. Read and validate JSON (array of objects)
2. For each record:
   - Copy template document
   - Build batch replace request: {{key}} → value for each key in record
   - Execute batch update
   - Rename document: substitute {{placeholders}} in filename-format
3. Return summary: N docs created, output folder link

**Features:**
- `--data-file` (required) — JSON array of objects
- `--filename-format` (required) — Template with {{placeholders}} for naming
- `--output-folder-id` (optional) — Destination folder (default: same as template)
- `--include-timestamp` (optional) — Append timestamp to filenames for uniqueness
- `--validate-only` — Validate JSON + template structure (preview first N records)
- `--dry-run` — Show operations without API calls
- Standard agentic flags: `--pretty`, `--output-request-file`, etc.

**Use Cases:**
- Employment offer letters (personalized salary, role, start date)
- Client proposals (company-specific terms, pricing)
- Certificates of completion (student name, date, score)
- Personalized reports (executive summary, KPIs per division)
- Performance reviews (employee name, goals, feedback)

**API Operations:**
- `drive.Files.Copy()` — Copy template for each record
- `docs.Documents.BatchUpdate()` — ReplaceAllText for each {{placeholder}}
- `drive.Files.Update()` — Rename copied document

**Success Criteria:**
- [ ] Compiles and runs
- [ ] Validates JSON structure (array of objects)
- [ ] `--validate-only` works: previews first 3 records without API calls
- [ ] `--dry-run` shows all operations without API calls
- [ ] Creates N documents in output folder
- [ ] Renames docs using filename-format {{substitutions}}
- [ ] Returns summary with total created + folder link
- [ ] Error handling: invalid JSON, missing folder, template not found, permissions
- [ ] Tests: success path, dry-run, validate-only, error cases
- [ ] `make test && make lint` passes

**Implementation Approach:**
1. Create `DocsEditMergeDataCmd` struct (similar to `SlidesEditMergeDataCmd`)
2. Implement data file parsing with validation
3. Implement copy + batch replace loop
4. Implement filename formatting ({{key}} substitution)
5. Implement error handling and retry logic (if needed)
6. Test with and without actual API

**Key Implementation Notes:**
- Reuse `RequestHash()`, `NormalizedRequestForOutput()`, `DryRunOutput()` from `edit_helpers.go`
- Handle Drive rate limits gracefully (may need delays between copies)
- Preserve document formatting (ReplaceAllText does this automatically)
- Test with various placeholder formats: {{firstName}}, {{first_name}}, {{FIRST_NAME}}

**Testing:**
```bash
# Validate-only (preview)
gog docs edit merge-data <template-id> --data-file employees.json --filename-format "Offer - {{lastName}}" --validate-only --json

# Dry-run (show operations)
gog docs edit merge-data <template-id> --data-file employees.json --filename-format "Offer - {{lastName}}" --dry-run --json

# Actual execution
gog docs edit merge-data <template-id> --data-file employees.json --filename-format "Offer - {{lastName}}" --output-folder-id <folder-id> --json
```

---

## Issue VID-116: Implement Sheets MergeData (Transformative)

**Title:** Sheets Edit: MergeData - Mail-merge spreadsheets from template + data  
**Priority:** Very High  
**Team:** Engineering  
**Estimate:** 3.5-4 hours  
**Status:** Todo  
**Depends On:** VID-114 (design complete), VID-115 (Docs reference)

**Description:**

Implement `gog sheets edit merge-data` — mail-merge for Google Sheets. This enables dynamic report generation and multi-spreadsheet creation from a single template.

Similar to Docs MergeData (VID-115), but adapted for Sheets with special attention to formula preservation and multi-sheet templates.

**Command Signature:**
```bash
gog sheets edit merge-data <templateId> \
  --data-file reports.json \
  --filename-format "Report - {{quarter}} {{year}}" \
  --output-folder-id <folderId> \
  [--include-timestamp]
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

**Template Spreadsheet:**
A Google Sheet with {{quarter}}, {{year}}, {{revenue}}, {{growth}}, {{forecast}} placeholders. May contain formulas.

**Execution Flow:**
1. Read and validate JSON (array of objects)
2. For each record:
   - Copy template spreadsheet
   - Build batch replace request: {{key}} → value
   - Execute batch update (all sheets in template)
   - Rename spreadsheet
3. Return summary: N sheets created, folder link

**Features:**
- `--data-file` (required) — JSON array of objects
- `--filename-format` (required) — Template with {{placeholders}}
- `--output-folder-id` (optional) — Destination folder
- `--include-timestamp` (optional) — Append timestamp
- `--validate-only` — Validate data + template
- `--dry-run` — Show operations without API calls
- Standard agentic flags

**Use Cases:**
- Monthly/quarterly financial reports
- Dashboard templates with company-specific data
- Multi-region budgets from template
- Territory-specific sales forecasts
- Department-specific KPI reports

**Sheets-Specific Considerations:**
- Template may have formulas → ensure replacement preserves formula structure
- May have multiple sheets → replace placeholder only in value cells (not formulas)
- Use cell ranges carefully: A:Z notation for entire columns (safer than specific ranges)
- Formulas like `=SUM(A:A)` should not have {{placeholders}}

**API Operations:**
- `drive.Files.Copy()` — Copy template spreadsheet
- `sheets.Spreadsheets.BatchUpdate()` with `FindReplaceRequest` — Replace all {{placeholders}}
- `drive.Files.Update()` — Rename copied spreadsheet

**Success Criteria:**
- [ ] Compiles and runs
- [ ] Validates JSON
- [ ] `--validate-only` works (no API calls)
- [ ] `--dry-run` shows operations without API calls
- [ ] Creates N spreadsheets in output folder
- [ ] Replaces {{placeholders}} across all sheets
- [ ] Preserves formulas (doesn't break them)
- [ ] Renames spreadsheets using filename-format
- [ ] Error handling: invalid JSON, formula conflicts, permissions
- [ ] Tests: success, dry-run, validate-only, formula preservation, error cases
- [ ] `make test && make lint` passes

**Implementation Approach:**
1. Create `SheetsEditMergeDataCmd` struct (reuse data parsing from Docs/Slides)
2. Adapt copy + batch update loop for Sheets API
3. Use `FindReplaceRequest` instead of `ReplaceAllText` (works across all sheets)
4. Test formula preservation thoroughly
5. Handle multi-sheet templates correctly

**Key Implementation Notes:**
- Reuse data parsing from VID-115 (Docs implementation) for DRY
- Use `FindReplaceRequest` for batch replacement (works on all sheets)
- Test with formulas: ensure {{placeholder}} in formula cell doesn't break
- Consider: what if template has named ranges with {{placeholders}}?

**Formula Preservation Testing:**
```
Template Sheet:
  A1: "Revenue: {{revenue}}"
  A2: "Growth: {{growth}}"
  A3: "Forecast: =SUM(A1:A2)"

After replacement with revenue=$1.2M, growth=+15%:
  A1: "Revenue: $1.2M"
  A2: "Growth: +15%"
  A3: "Forecast: =SUM(A1:A2)"  ← Formula preserved!
```

**Testing:**
```bash
# Validate-only
gog sheets edit merge-data <template-id> --data-file reports.json --filename-format "Report - {{quarter}}" --validate-only --json

# Dry-run
gog sheets edit merge-data <template-id> --data-file reports.json --filename-format "Report - {{quarter}}" --dry-run --json

# Actual execution
gog sheets edit merge-data <template-id> --data-file reports.json --filename-format "Report - {{quarter}}" --output-folder-id <folder-id> --json
```

---

## Summary Table

| Issue | Title | Duration | Priority | Status |
|-------|-------|----------|----------|--------|
| VID-111 | Sheets DeleteRange | 1.5h | High | Todo |
| VID-112 | Docs InsertImage | 2h | High | Todo |
| VID-113 | Code Review | 0.5h | Medium | Todo |
| VID-114 | Design Review | 1.5h | Very High | Todo |
| VID-115 | Docs MergeData | 3.5-4h | Very High | Todo |
| VID-116 | Sheets MergeData | 3.5-4h | Very High | Todo |
| **Total** | **Phase 3** | **12.5h** | — | — |

**Related Documentation:**
- `CROSS_SERVICE_OPPORTUNITY_ANALYSIS.md` — Strategic context
- `PHASE_3_IMPLEMENTATION_PLAN.md` — Detailed implementation guide
- `handover.md` — Project overview and key docs

---

**Status:** Ready for Linear intake  
**Next Step:** Create issues in Linear using above templates
