# Phase 3: Docs & Sheets MergeData — Design Document (VID-114)

**Date:** 2026-02-17  
**Status:** Design complete, ready for VID-115 (Docs) and VID-116 (Sheets) implementation  
**Reference:** Slides implementation `SlidesEditMergeDataCmd` in `internal/cmd/slides_edit_cmd.go`

---

## 1. Slides MergeData Pattern (Reference)

### Current behavior
- **Input:** `templateId` (presentation ID), `--data-file` (JSON array of objects), `--filename-format` (e.g. `"Generated - {{name}}"`), optional `--output-folder-id`, `--include-timestamp`, `--export-pdf`.
- **Flow:** For each record: (1) create new presentation with title = formatted filename, (2) batch ReplaceAllText for each `{{key}}` → value, (3) optionally move to output folder via Drive, (4) optionally export as PDF and delete the Slides copy.
- **Data format:** JSON array of objects; keys become placeholders `{{key}}`. Filename formatting uses the same placeholder syntax; unreplaced `{{...}}` are stripped.
- **Safety:** `--validate-only` (no API, preview first 3 records + sample filename), `--dry-run` (same plus requestHash), standard agentic flags.
- **Output:** JSON with `templateId`, `recordCount`, `generated`, `failed`, `results[]` (per-record status, documentId/title, optional pdfFileId).

**Note:** Slides currently creates new presentations via Presentations.Create (blank); for true template cloning, Drive.Files.Copy would be used. Docs and Sheets designs below assume **copy template via Drive, then batch update** so the template content is preserved.

---

## 2. Docs MergeData — Design

### Goal
Mail-merge for Google Docs: one template Doc with `{{placeholder}}` text; one output Doc per data record with placeholders replaced and file renamed.

### CLI (sketch)
```bash
gog docs edit merge-data <templateId> \
  --data-file <path> \
  --filename-format "Offer Letter - {{firstName}} {{lastName}}" \
  [--output-folder-id <folderId>] \
  [--include-timestamp] \
  [--validate-only] [--dry-run] [--pretty] [--output-request-file <path>]
```

### Execution flow
1. **Parse & validate:** Read `--data-file` as JSON array of objects; reject empty array or empty objects. Validate `--filename-format` contains at least one `{{key}}` or allow default.
2. **Validate-only:** Return recordCount, sample filename(s), requestHash, optional preview of replace ops (no Drive/Docs calls).
3. **Dry-run:** Same as validate-only plus dryRun: true; no account needed.
4. **Execute (per record):**
   - **Copy template:** `Drive.Files.Copy(templateId, &drive.File{Name: formattedTitle})` → newDocId. Requires Drive scope.
   - **Replace text:** Build `BatchUpdateDocumentRequest` with one `ReplaceAllText` per key (find `{{key}}`, replace with value). Use `docs.Documents.BatchUpdate(docId, req)`.
   - **Optional:** Move to `--output-folder-id` via Drive.Files.Update AddParents/RemoveParents.
5. **Response:** Summary (generated, failed, results[] with documentId, title, webViewLink).

### Data file format (same as Slides)
```json
[
  { "firstName": "Alice", "lastName": "Engineer", "title": "Senior Engineer", "salary": "$200,000" },
  { "firstName": "Bob", "lastName": "Manager", "title": "Product Manager", "salary": "$180,000" }
]
```

### Docs-specific considerations
- **Placeholders:** Plain text `{{key}}` in body; ReplaceAllText with MatchCase: false (or configurable) so casing in template is flexible.
- **Formatting:** Docs API ReplaceAllText preserves paragraph/style; no extra handling unless we need to support structured content.
- **Output:** Always Google Docs (no built-in “export as PDF” in first version; user can run `gog docs export <id> pdf` separately).
- **Template integrity:** Copy is a full Drive copy; formulas/tables/images in the Doc are preserved.

### Error handling (Docs)
| Scenario | error_code | Action |
|----------|------------|--------|
| Empty templateId | invalid_argument | Return, no API call |
| Empty/missing data-file | invalid_argument / input_open_failed | Return |
| Invalid JSON | invalid_json | Return |
| Empty records array | invalid_argument | Return |
| Template not found (Drive copy) | template_not_found | Per-record or fail-fast |
| Copy failed | api_error | Record in results as failed, continue |
| BatchUpdate failed | api_error | Record in results as failed, continue |
| Output folder not found | output_folder_not_found | Record as failed or fail-fast |

---

## 3. Sheets MergeData — Design

### Goal
Report generation: one template Sheet with `{{placeholder}}` in cells; one output Sheet per record with placeholders replaced and file renamed.

### CLI (sketch)
```bash
gog sheets edit merge-data <templateId> \
  --data-file <path> \
  --filename-format "Report - {{quarter}} {{year}}" \
  [--output-folder-id <folderId>] \
  [--include-timestamp] \
  [--validate-only] [--dry-run] [--pretty] [--output-request-file <path>]
```

### Execution flow
1. **Parse & validate:** Same as Docs (JSON array, non-empty objects).
2. **Validate-only / dry-run:** Same pattern as Docs/Slides.
3. **Execute (per record):**
   - **Copy template:** `Drive.Files.Copy(templateId, &drive.File{Name: formattedTitle})` → newSheetId.
   - **Replace text:** Use Sheets FindReplace (or batch update with FindReplaceRequest). Replace across **all sheets** in the workbook so multi-sheet templates are supported. Option: `--all-sheets` (default true), or scope to specific sheet IDs.
   - **Optional:** Move to `--output-folder-id` via Drive.
5. **Response:** Summary (generated, failed, results[] with spreadsheetId, title, spreadsheetUrl).

### Data file format (same)
```json
[
  { "quarter": "Q1", "year": "2026", "revenue": "$1.2M", "growth": "+15%" },
  { "quarter": "Q2", "year": "2026", "revenue": "$1.4M", "growth": "+17%" }
]
```

### Sheets-specific considerations
- **FindReplace API:** Sheets v4 has `FindReplaceRequest` in BatchUpdateSpreadsheetRequest; can set `allSheets: true` to replace across all sheets. Match case and regex optional.
- **Formulas:** Replace in **value cells only** by default; do not replace inside formula text unless we add an explicit `--include-formulas` (risk of breaking formulas). First version: find/replace in values only, or document that placeholders must be in value cells.
- **Multi-sheet:** Default to all sheets; optional `--sheet-ids` to limit scope later.
- **Cell format:** Preserved by copy; replacement only changes text content.

### Error handling (Sheets)
| Scenario | error_code | Action |
|----------|------------|--------|
| Same as Docs for input/parse | (same) | (same) |
| Template not found (Drive copy) | template_not_found | Per-record or fail-fast |
| Copy failed | api_error | Record failed, continue |
| BatchUpdate (findReplace) failed | api_error | Record failed, continue |
| Formula conflict (if we add formula replace) | invalid_request | Document limitation |

---

## 4. Shared Components (DRY)

- **Data parsing:** Single helper to read JSON array of `map[string]any`, validate non-empty. Can live in `internal/cmd/merge_data.go` or next to each command.
- **Filename formatting:** Reuse `formatMergeFilename(format, record, includeTimestamp)` — move to shared helper (e.g. `edit_helpers.go` or `merge_data.go`) and use from Docs and Sheets.
- **Drive copy + optional move:** Shared helper `copyDriveFileToFolder(ctx, driveSvc, fileId, newName, outputFolderID)` used by both Docs and Sheets.
- **Safety flags:** Both commands embed `AgenticEditSafetyFlags`; validate-only and dry-run require no auth.

---

## 5. Success Response Format (unified)

```json
{
  "templateId": "<id>",
  "recordCount": 10,
  "generated": 9,
  "failed": 1,
  "outputFolderId": "<optional>",
  "results": [
    { "index": 0, "status": "success", "documentId": "<id>", "title": "Offer Letter - Alice Engineer" },
    { "index": 5, "status": "failed", "error": "...", "stage": "batch-update" }
  ]
}
```

For Sheets, use `spreadsheetId` instead of `documentId` where appropriate.

---

## 6. Implementation order

1. **VID-115 (Docs):** Implement `DocsEditMergeDataCmd`, Drive copy + Docs BatchUpdate replace loop, shared `formatMergeFilename` (or keep in package), tests for validate-only, dry-run, one success path with mock.
2. **VID-116 (Sheets):** Implement `SheetsEditMergeDataCmd`, reuse data parsing and filename formatting, Drive copy + Sheets BatchUpdate with FindReplace (allSheets: true), same response shape, tests.

---

## 7. Open questions / blockers

- **Slides template copy:** Current Slides implementation uses Create (blank) not Drive copy; confirm whether Slides should be refactored to copy template first for consistency.
- **Sheets formulas:** Decide whether to support replace in formula cells at all; first version = value cells only.
- **Rate limits:** Large data files (e.g. 500+ records) may hit Drive/Docs/Sheets rate limits; consider `--limit` or chunking in a future iteration.
- **Idempotency:** Same data file run twice creates duplicate files; no built-in dedup. Users can use `--output-folder-id` and clear folder before run, or add `--require-empty-folder` later.

---

**Document status:** Ready for implementation. VID-115 and VID-116 can proceed using this design.
