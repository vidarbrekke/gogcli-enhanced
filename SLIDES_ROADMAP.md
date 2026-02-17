# Slides Edit Operations Roadmap

**Status:** VID-91 through VID-97 complete. Now implementing extended operations.
**Branch:** `kimi`

---

## Proposed Issues (VID-100 through VID-106)

| VID | Title | Priority | Est. Time | Status |
|-----|-------|----------|-----------|--------|
| VID-100 | Replace Text | Top | 1h | 🚧 Starting |
| VID-101 | Replace Image | Top | 1h | Pending |
| VID-102 | **Merge Data** | **TRANSFORMATIVE** | 4-6h | Pending |
| VID-103 | Create Slide | Medium | 1h | Pending |
| VID-104 | Duplicate Slide | Medium | 1h | Pending |
| VID-105 | Refresh Charts | Medium | 1h | Pending |
| VID-106 | Insert Table | Medium | 1h | Pending |

---

## VID-100: Replace Text

**Description:** Find and replace text across all slides in a presentation.

**Use case:** Update company names, dates, pricing in templates

**Command:**
```bash
gog slides edit replace-text <presentationId> --find "Q4 2025" --replace "Q1 2026"
```

**API:** Uses ReplaceAllText batch operation

**Implementation notes:**
- Wrapper around batch ReplaceAllText operation
- Support regex matching (optional)
- Support case-sensitive vs insensitive
- Supports all agentic safety flags

---

## VID-101: Replace Image

**Description:** Replace images in slides while maintaining exact position, size, and formatting.

**Use case:** Update logos, product shots, headshots across standardized decks

**Command:**
```bash
gog slides edit replace-image <presentationId> \
  --object-id <imageId> \
  --source-image <url_or_file>
```

**API:** Uses ReplaceImage batch operation

---

## VID-102: Merge Data (TRANSFORMATIVE)

**Description:** Generate multiple presentations from a template by merging JSON data.

**Use case:** Generate proposals, reports, certificates from templates with placeholders

**Command:**
```bash
gog slides edit merge-data template_id \
  --data clients.json \
  --output-folder "Proposals 2026/" \
  --filename-format "proposal-{{client_name}}.pdf"
```

**Features:**
- Read array of data objects from JSON
- Replace {{placeholder}} syntax in text boxes
- Generate N presentations from 1 template
- Export as PDF with formatted filenames

**This is the killer feature for automated document generation at scale.**

---

## VID-103: Create Slide

**Description:** Add new slides to a presentation programmatically.

**Use case:** Build decks from scratch or append standardized closing slides

**Command:**
```bash
gog slides edit create-slide <presentationId> --layout TITLE --index 5
```

**Layouts:** TITLE, SECTION_HEADER, BLANK, TITLE_AND_BODY, etc.

---

## VID-104: Duplicate Slide

**Description:** Duplicate existing slides for template sections that repeat.

**Use case:** Template sections that repeat structure

**Command:**
```bash
gog slides edit duplicate-slide <presentationId> --slide-id <id> --count 3
```

---

## VID-105: Refresh Charts

**Description:** Refresh embedded Google Sheets charts when underlying data changes.

**Use case:** Monthly reports where data updates but layout stays fixed

**Command:**
```bash
gog slides edit refresh-charts <presentationId> --chart-id <id> or --all
```

---

## VID-106: Insert Table

**Description:** Insert formatted tables into slides from CSV/JSON data.

**Use case:** Financial summaries, comparison matrices, data visualizations

**Command:**
```bash
gog slides edit insert-table <presentationId> --slide-id <id> \
  --rows 5 --columns 3 \
  --data financial_summary.json
```

---

## Implementation Order

1. **VID-100** - Quick win, immediate value
2. **VID-102** - The transformative feature (highest user impact)
3. **VID-101** - Another quick win after 102
4. VID-103 through VID-106 - In parallel or as needed
