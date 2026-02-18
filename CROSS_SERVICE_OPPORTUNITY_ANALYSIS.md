# 🚀 Cross-Service Capability Analysis: Docs, Sheets, Slides

**Date:** 2026-02-17  
**Analysis:** 10x engineer review — identifying replicable patterns across services  
**Status:** Ready for implementation (6 high-value opportunities identified)

---

## 📊 Current State Matrix

| Operation | Docs | Sheets | Slides | API Support | Gap |
|-----------|------|--------|--------|-------------|-----|
| ReplaceText | ✅ | ❌ | ✅ | ✅ FindReplace | Sheets missing |
| ReplaceImage | ❌ | ❌ | ✅ | ✅ (Docs) ❌ (Sheets) | Both missing |
| InsertTable | ✅ | ❌ | ✅ | ✅ (Sheets has grid) | Sheets missing |
| MergeData | ❌ | ❌ | ✅ | ✅ (copy + batch) | Both missing |
| Append | ✅ | ✅ | ❌ | ✅ (content-level) | Slides N/A |
| Delete | ✅ | ❌ | ❌ | ✅ (Sheets DeleteRange) | Sheets missing |
| Batch | ✅ | ✅ | ✅ | ✅ | Complete |

---

## 🎯 TOP 6 IMMEDIATE OPPORTUNITIES (Ranked by Impact × Effort)

### 🏆 #1: **Sheets ReplaceText** (HIGH IMPACT, MEDIUM EFFORT)

**Status:** MISSING  
**Proven Pattern:** Docs ✅, Slides ✅  
**API Support:** `FindReplaceRequest` in Sheets v4 API ✅  
**Complexity:** Medium

**Why:** Fills obvious gap — users expect find/replace in sheets like in docs/slides.

**Implementation Plan:**
```bash
gog sheets edit replace-text <spreadsheetId> \
  --find "Q4 2025" \
  --replace "Q1 2026" \
  --sheet-id 0 \
  [--match-case] [--all-sheets] [--regex] [--formulas]
```

**Advantage Over Docs/Slides:** Sheets has additional capabilities:
- `--all-sheets` — Replace across entire workbook
- `--regex` — Full regex support (Java pattern)
- `--formulas` — Include formula cells
- `--match-entire-cell` — Exact cell matching

**Effort Estimate:** 2-3 hours (command + tests)

---

### 🏆 #2: **Docs ReplaceImage** (HIGH IMPACT, LOW EFFORT)

**Status:** MISSING  
**Proven Pattern:** Slides ✅  
**API Support:** `ReplaceImageRequest` in Docs API ✅  
**Complexity:** Low

**Why:** API already exists in Docs. Copy/adapt Slides pattern.

**Implementation Plan:**
```bash
gog docs edit replace-image <docId> \
  --image-id <existing-image-id> \
  --uri "https://..." \
  [--replace-method CENTER_CROP|UNSPECIFIED]
```

**Use Case:** Update branded images, product screenshots, logos in templates.

**Effort Estimate:** 1-2 hours (straightforward copy from Slides)

---

### 🏆 #3: **Docs MergeData** (TRANSFORMATIVE, MEDIUM EFFORT)

**Status:** MISSING  
**Proven Pattern:** Slides ✅ (copy + batch replace)  
**API Support:** Yes (Drive copy + batch update)  
**Complexity:** Medium

**Why:** Mail-merge is THE killer use case for document automation. Docs needs this.

**Implementation Plan:**
```bash
gog docs edit merge-data <templateId> \
  --data-file employees.json \
  --filename-format "Offer Letter - {{name}}" \
  --output-folder-id <folder>
```

**Example Data:**
```json
[
  {
    "name": "Alice",
    "title": "Senior Engineer",
    "salary": "$200k"
  },
  {
    "name": "Bob",
    "title": "Product Manager",
    "salary": "$180k"
  }
]
```

**Use Cases:**
- Employment offer letters (with personalized salary, role)
- Client proposals (personalized pricing, timeline)
- Certificates of completion (name, date, score)
- Personalized reports (executive summary, KPIs)

**Advantage:** Unlike Slides, Docs templates can be **much more complex** (hundreds of pages, complex formatting). This unlocks serious enterprise use cases.

**Effort Estimate:** 3-4 hours (adapt Slides pattern, test)

---

### 🏆 #4: **Sheets MergeData** (TRANSFORMATIVE, MEDIUM EFFORT)

**Status:** MISSING  
**Proven Pattern:** Slides ✅ (copy + batch replace)  
**API Support:** Yes (Drive copy + batch update)  
**Complexity:** Medium

**Why:** Mail-merge for spreadsheets = dynamic report generation.

**Implementation Plan:**
```bash
gog sheets edit merge-data <templateId> \
  --data-file config.json \
  --filename-format "Report - {{date}}" \
  --output-folder-id <folder>
```

**Use Cases:**
- Financial reports with company-specific data
- Monthly dashboards with different metrics
- Budget templates filled with department-specific numbers
- Sales forecasts with territory-specific data

**Advantage:** Sheets is ideal for data transformation — generate N spreadsheets with different data subsets from one template.

**Effort Estimate:** 3-4 hours (adapt Slides pattern + Sheets peculiarities)

---

### 🏆 #5: **Sheets DeleteRange** (MEDIUM IMPACT, LOW EFFORT)

**Status:** MISSING  
**Proven Pattern:** Docs.DeleteContentRange ✅  
**API Support:** `DeleteRangeRequest` in Sheets batch API ✅  
**Complexity:** Low

**Why:** Fills operational gap. Docs and Slides both have delete. Sheets should too.

**Implementation Plan:**
```bash
gog sheets edit delete-range <spreadsheetId> \
  --range "A1:C10" \
  --shift-dimension ROWS|COLUMNS \
  --sheet-id <id>
```

**Effort Estimate:** 1.5 hours (straightforward)

---

### 🏆 #6: **Docs InsertImage** (MEDIUM IMPACT, MEDIUM EFFORT)

**Status:** MISSING  
**Proven Pattern:** None implemented yet  
**API Support:** `InsertInlineImage` in Docs API ✅  
**Complexity:** Medium

**Why:** Complement to ReplaceImage. Enable document templates with placeholders.

**Implementation Plan:**
```bash
gog docs edit insert-image <docId> \
  --uri "https://cdn.example.com/logo.png" \
  --index 1 \
  [--width-pt <width>] [--height-pt <height>]
```

**Use Cases:**
- Branding/logo injection in mail-merge templates
- Report illustrations at specific positions
- Certificate/award template images

**Effort Estimate:** 2 hours

---

## 🎓 PATTERN INSIGHTS (Why This Works)

### Pattern A: **Copy Semantics**
**Docs + Sheets:** Both services support Drive copy + batch update cycle.
```
1. Copy resource (Drive.Files.Copy)
2. Parse data source (JSON array)
3. Batch update copy with data (Docs/Sheets BatchUpdate)
4. Share/return resource ID
```
**This is:** MergeData, CompileReport, MailMerge, etc.

**Implication:** Any batch-updatable service can implement merge operations once.

---

### Pattern B: **FindReplace Universality**
**Docs:** ReplaceAllText (minimal options)  
**Sheets:** FindReplace (regex, formulas, all-sheets)  
**Slides:** ReplaceAllText (minimal options)

**Key:** Sheets has the most powerful implementation. Extract common subset for cross-service.

---

### Pattern C: **Agentic Safety as Orthogonal**
**Proven:** All 4 Docs commands + Sheets/Slides batch use same safety flags.

**Implication:** Safety is decoupled from operation semantics. This means:
- New operations inherit `--validate-only`, `--dry-run`, `--pretty`, etc. **for free**
- No per-operation safety work needed

---

## 📋 IMPLEMENTATION ROADMAP

### Phase 1 (This Week) — Quick Wins
1. **Sheets ReplaceText** ⚡ (2-3h) — highest-value gap fill
2. **Docs ReplaceImage** ⚡ (1-2h) — simple port
3. **Sheets DeleteRange** ⚡ (1.5h) — operational completeness

**Subtotal:** 4.5-6.5 hours

### Phase 2 (Next Week) — Transformative
4. **Docs MergeData** 🚀 (3-4h) — enterprise feature
5. **Sheets MergeData** 🚀 (3-4h) — reporting powerhouse

**Subtotal:** 6-8 hours

### Phase 3 (Optional) — Polish
6. **Docs InsertImage** (2h) — template completeness

---

## 🎯 ESTIMATED TOTAL IMPACT

| Item | Hours | Impact | Users |
|------|-------|--------|-------|
| Sheets ReplaceText | 2.5 | Replace operations consistent | All |
| Docs ReplaceImage | 1.5 | Template branding | 40% |
| Sheets DeleteRange | 1.5 | Operational completeness | 30% |
| Docs MergeData | 3.5 | **Mail-merge for docs** | **60%** |
| Sheets MergeData | 3.5 | **Report generation** | **50%** |
| **Total** | **12.5** | **Enterprise-grade** | **→** |

**Value Unlock:** 
- From scattered point operations → **Cohesive cross-service platform**
- From "sync to template" → **Data-driven document generation at scale**
- From user confusion → **Consistency: same patterns across all services**

---

## 🏗️ TECHNICAL DETAILS

### Sheets ReplaceText Signature
```go
type SheetsEditReplaceTextCmd struct {
    SpreadsheetID  string `arg:""`
    Find           string `name:"find"`
    Replace        string `name:"replace"`
    SheetID        int64  `name:"sheet-id"`
    AllSheets      bool   `name:"all-sheets"`
    MatchCase      bool   `name:"match-case"`
    MatchEntireCell bool  `name:"match-entire-cell"`
    UseRegex       bool   `name:"regex"`
    IncludeFormulas bool  `name:"formulas"`
    Safety         AgenticEditSafetyFlags `embed:""`
}
```

### Docs ReplaceImage Signature
```go
type DocsEditReplaceImageCmd struct {
    DocID       string `arg:""`
    ImageID     string `name:"image-id"`
    URI         string `name:"uri"`
    ReplaceMethod string `name:"replace-method"` // CENTER_CROP, UNSPECIFIED
    TabID       string `name:"tab-id"`
    Safety      AgenticEditSafetyFlags `embed:""`
}
```

### Docs/Sheets MergeData Signature (Unified)
```go
type [Service]EditMergeDataCmd struct {
    TemplateID       string `arg:""`
    DataFile         string `name:"data-file"`
    OutputFolderID   string `name:"output-folder-id"`
    FilenameFormat   string `name:"filename-format"` // {{placeholder}}
    IncludeTimestamp bool   `name:"include-timestamp"`
    Safety           AgenticEditSafetyFlags `embed:""`
}
```

---

## ✅ SUCCESS CRITERIA

For each new operation:
1. ✅ Uses `AgenticEditSafetyFlags` (validate-only, dry-run, pretty, etc.)
2. ✅ Structured JSON output with requestHash
3. ✅ Works locally with --validate-only (no auth)
4. ✅ Works with --dry-run (preview mode)
5. ✅ Comprehensive tests
6. ✅ Cross-service consistency

---

## 🎬 NEXT ACTION

**Recommend:** Start with **Sheets ReplaceText** (highest confidence, proven pattern, fills obvious gap).

This will:
1. Build confidence with Sheets edit refactoring
2. Prove the pattern works across services
3. Set momentum for MergeData implementations

**Then:** Do Docs ReplaceImage (quick win, high confidence, 1.5h).

**Then:** Attack MergeData pair (Docs + Sheets) for transformative impact.

---

**Prepared for:** 10x implementation sprint  
**Status:** Ready to proceed
