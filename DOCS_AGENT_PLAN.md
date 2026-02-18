# Google Docs Agent-Friendly Enhancements - Deep Analysis & Plan

**Created:** 2026-02-17  
**Context:** After implementing 7 Slides operations, analyzing what Docs needs for agent excellence

---

## 🧠 Deep Thinking: Agent-Friendly Principles

### What Makes Tools Truly Agent-Friendly?

1. **High-level intent, low-level control**
   - Agents think: "replace all company names"
   - Not: "loop through ranges, find text, call updateTextStyle 47 times"

2. **Idempotency & safety**
   - `--dry-run` → preview without commitment
   - `--execute-from-file` → replay deterministically
   - Request hashing → detect duplicates

3. **Structured I/O**
   - JSON with error_code, request_hash, resource_id
   - Agents can parse, reason, retry, log

4. **Semantic operations**
   - Domain concepts: "heading", "paragraph", "list"
   - Not just API primitives: "insertText at index 427"

5. **Composability**
   - Small focused commands that chain
   - Build complex workflows from simple parts

---

## 📊 Current State Gap Analysis

### What Docs Has (vs Sheets/Slides)
| Feature | Docs | Sheets | Slides |
|---------|------|--------|--------|
| Focused operations | ❌ **Only `batch`** | ✅ values/append/clear | ✅ 7 commands |
| Find/replace | ❌ | ✅ (via values) | ✅ replace-text |
| Mail-merge | ❌ | ❌ | ✅ **merge-data** 🚀 |
| Structured insert | ❌ | ✅ append | ✅ insert-table |
| Template ops | ❌ | N/A | ❌ |
| Data extraction | ❌ | ✅ (native) | ❌ |

**Critical insight:** Docs has the LEAST agent-friendly operations despite being the most template-heavy service.

---

## 🎯 Agent Use Cases for Docs (Real-World)

### Document Generation
- **Contracts:** 50 personalized NDAs from template
- **Offer letters:** Merge candidate data → formatted letters
- **Certificates:** Event attendee certificates
- **Reports:** Monthly reports with consistent structure

### Content Transformation
- **Format standardization:** Apply corporate style guide to 100 docs
- **Cleanup:** Remove direct formatting, enforce named styles
- **Update branding:** Replace old logo references, update footer

### Data Operations
- **Injection:** Insert tables from CSV, charts from data
- **Extraction:** Pull headings → outline JSON, tables → CSV
- **Cross-reference:** Update outdated links, citations

### Collaborative Workflow
- **Review cleanup:** Batch accept suggestions, resolve comments
- **Compliance:** Add "DRAFT" watermarks, standardize footers
- **Version prep:** Final draft → remove all comments/suggestions

---

## 💡 Proposed Operations (Prioritized)

### 🚀 Tier 1: Transformative (HIGHEST IMPACT)

#### 1. **`docs edit merge-data`** ⭐ TRANSFORMATIVE
**Impact:** Like Slides merge-data — generates N docs from 1 template

```bash
gog docs edit merge-data template_id \
  --data-file employees.json \
  --filename-format "Offer Letter - {{name}}" \
  --output-folder-id "..."
```

**Template:** "Dear {{name}}, your salary is {{salary}}..."  
**Result:** 50 personalized offer letters

**Use cases:**
- Employment contracts
- Client proposals
- Certificates & awards
- Personalized reports

**API:** Copy doc → ReplaceAllText for each {{placeholder}} → Export/Share

**Unique value:** No agent can currently do doc mail-merge programmatically with full formatting preservation.

---

#### 2. **`docs edit apply-template`** ⭐ TRANSFORMATIVE
**Impact:** Apply structure/styles from template to content

```bash
gog docs edit apply-template doc_id \
  --template-id template_doc_id \
  --preserve-content \
  --apply-styles \
  --apply-structure
```

**Use case:** Batch format 100 reports to match corporate template

**Modes:**
- `--apply-styles` → Named styles only
- `--apply-structure` → Heading levels, lists
- `--apply-all` → Full template (headers, footers, margins)

**Unique value:** Agents can't currently "restyle" a doc programmatically.

---

#### 3. **`docs edit extract-data`** ⭐ TRANSFORMATIVE
**Impact:** Turn unstructured docs into structured data

```bash
# Extract outline
gog docs edit extract-data doc_id --extract outline > outline.json

# Extract all tables
gog docs edit extract-data doc_id --extract tables > tables.json

# Extract links
gog docs edit extract-data doc_id --extract links > links.json
```

**Outputs:**
- `outline`: Heading hierarchy as JSON tree
- `tables`: All tables as CSV/JSON arrays
- `links`: All URLs with anchor text
- `images`: Image metadata (URL, alt text, position)

**Use cases:**
- Document analysis
- Content audit
- Migration prep
- Data mining from reports

**Unique value:** Inverse of generation — structured data OUT of docs.

---

### ✅ Tier 2: High-Value Quick Wins (4-6 hours total)

#### 4. **`docs edit replace-text`**
Like Slides replace-text — simple find/replace

```bash
gog docs edit replace-text doc_id \
  --find "Q4 2025" \
  --replace "Q1 2026" \
  --match-case
```

**Use case:** Update dates, product names, terminology across docs

---

#### 5. **`docs edit insert-table`**
Insert formatted table from CSV/JSON

```bash
gog docs edit insert-table doc_id \
  --data-file summary.csv \
  --position end \
  --header-row \
  --style "BLUE_TABLE"
```

**Use case:** Financial summaries, data tables in reports

---

#### 6. **`docs edit apply-style`**
Bulk apply named styles to ranges

```bash
gog docs edit apply-style doc_id \
  --range "1:100" \
  --style "Heading 1" \
  --find-text "Chapter"
```

**Use case:** Format cleanup, enforce style guide

---

#### 7. **`docs edit insert-toc`**
Generate/update table of contents

```bash
gog docs edit insert-toc doc_id \
  --position 1 \
  --max-level 3 \
  --update-existing
```

**Use case:** Report generation, document navigation

---

### 🔄 Tier 3: Collaboration & Workflow (6-8 hours)

#### 8. **`docs edit resolve-comments`**
Batch comment resolution

```bash
# Resolve all
gog docs edit resolve-comments doc_id --all

# Resolve by author
gog docs edit resolve-comments doc_id --author editor@example.com

# Resolve containing text
gog docs edit resolve-comments doc_id --contains "LGTM"
```

**Use case:** Post-review cleanup, final draft prep

---

#### 9. **`docs edit accept-suggestions`**
Batch suggestion acceptance

```bash
gog docs edit accept-suggestions doc_id --all
gog docs edit accept-suggestions doc_id --author proofreader@example.com
```

**Use case:** Incorporate edits, finalize drafts

---

#### 10. **`docs edit watermark`**
Add diagonal watermark text

```bash
gog docs edit watermark doc_id --text "DRAFT" --color gray --opacity 0.3
gog docs edit watermark doc_id --text "CONFIDENTIAL" --remove
```

**Use case:** Document status, compliance

---

#### 11. **`docs edit update-footer`**
Standardize footers/headers

```bash
gog docs edit update-footer doc_id \
  --text "Acme Corp | Page {{page_number}}" \
  --align center
```

**Use case:** Brand compliance, corporate standards

---

### 🏗️ Tier 4: Structure & Navigation (8-10 hours)

#### 12. **`docs edit split-sections`**
Split doc by headings into separate docs

```bash
gog docs edit split-sections doc_id \
  --level 1 \
  --output-folder-id "..." \
  --filename-format "{{heading_text}}"
```

**Use case:** Extract chapters, modularize reports

---

#### 13. **`docs edit merge-docs`**
Combine multiple docs

```bash
gog docs edit merge-docs --doc-ids doc1,doc2,doc3 \
  --output-title "Combined Report" \
  --insert-page-breaks
```

**Use case:** Report compilation, chapter assembly

---

#### 14. **`docs edit reorder-sections`**
Move sections by heading

```bash
gog docs edit reorder-sections doc_id \
  --from-heading "Appendix A" \
  --to-index 5
```

**Use case:** Document restructuring

---

### 🚀 Tier 5: Advanced (12-16 hours)

#### 15. **`docs edit inline-images`**
Replace inline images by placeholder text

```bash
gog docs edit inline-images doc_id \
  --find-text "{{product_image}}" \
  --replace-url "https://cdn.example.com/product.png"
```

**Use case:** Dynamic image injection in templates

---

#### 16. **`docs edit apply-conditional-format`**
Format based on content rules

```bash
gog docs edit apply-conditional-format doc_id \
  --if-contains "TODO" \
  --then-highlight yellow \
  --then-bold
```

**Use case:** Review highlighting, document analysis

---

#### 17. **`docs edit extract-comments`**
Export comments to JSON/CSV

```bash
gog docs edit extract-comments doc_id > comments.json
```

**Use case:** Review tracking, analytics

---

## 📋 Implementation Plan

### Phase 1: Quick Wins (4-6 hours) - IMMEDIATE VALUE
**Goal:** Match Slides feature parity with focused operations

1. **replace-text** (1h) - Simple, high usage
2. **insert-table** (1.5h) - Data-driven docs
3. **apply-style** (1.5h) - Format standardization
4. **insert-toc** (1h) - Report generation

**Deliverable:** 4 new commands, full agentic safety

---

### Phase 2: The Transformative Trio (8-12 hours) - GAME CHANGERS
**Goal:** Unlock document generation & transformation at scale

1. **merge-data** (4-5h) ⭐ - Mail merge for docs
   - Template parsing ({{placeholder}} detection)
   - Copy + batch replace pipeline
   - Filename formatting
   - Export options

2. **apply-template** (2-3h) ⭐ - Style/structure application
   - Extract styles from template
   - Apply to target doc
   - Preserve content

3. **extract-data** (2-4h) ⭐ - Structured extraction
   - Heading outline → JSON
   - Tables → CSV/JSON
   - Links → structured list
   - Images → metadata

**Deliverable:** The three operations that make Docs truly agent-friendly

---

### Phase 3: Workflow Automation (6-8 hours)
**Goal:** Collaborative & compliance operations

1. **resolve-comments** (2h)
2. **accept-suggestions** (2h)
3. **watermark** (2h)
4. **update-footer** (2h)

**Deliverable:** Batch review & compliance ops

---

### Phase 4: Structure (8-10 hours)
**Goal:** Document composition/decomposition

1. **split-sections** (3-4h)
2. **merge-docs** (3-4h)
3. **reorder-sections** (2h)

**Deliverable:** Build/break docs programmatically

---

### Phase 5: Advanced (12-16 hours)
**Goal:** Power user & edge cases

1. **inline-images** (4-5h)
2. **apply-conditional-format** (4-5h)
3. **extract-comments** (2-3h)

---

## 🎯 Recommended First Sprint (This Week)

### **Phase 1 + Transformative #1** (10-12 hours total)

**Rationale:**
- Quick wins prove value fast
- merge-data is THE killer feature for Docs
- Parallel Slides success pattern

**Implementation order:**
1. replace-text (1h) ← Quick win
2. insert-table (1.5h) ← Data ops
3. **merge-data (5h)** ← TRANSFORMATIVE ⭐
4. apply-style (1.5h) ← Format ops
5. insert-toc (1h) ← Navigation

**After this sprint:**
- Docs matches Slides in focused operations
- **Mail-merge unlocks 100+ use cases**
- Template + compliance ops queued

---

## 🔑 Key Insights

1. **Docs has the MOST potential** for agent value (templates, structure, collaboration)
2. **merge-data is the #1 missing feature** (contracts, letters, reports)
3. **apply-template** is unique to Docs (no equivalent in Sheets/Slides)
4. **extract-data** turns docs into structured data (inverse of generation)
5. **Style operations** leverage Docs' named styles (unlike Sheets/Slides)

---

## 📊 Expected Impact

After Phase 1 + 2:
- **Document generation:** 1 template → N personalized docs (merge-data)
- **Format enforcement:** Batch apply corporate styles (apply-template)
- **Data extraction:** Mining structured data from reports (extract-data)
- **Content updates:** Find/replace at scale (replace-text)
- **Report assembly:** Auto-generate TOC, insert tables (insert-toc, insert-table)

**Agent capability unlock:** Go from "batch API primitive" → "document automation platform"

---

## ✅ Success Criteria

A Docs operation is agent-friendly if:
1. ✅ Solves a common task in one command (not 20 API calls)
2. ✅ Supports --dry-run and --validate-only
3. ✅ Returns structured JSON with error codes
4. ✅ Uses semantic concepts (heading, style) not indices
5. ✅ Composes with other operations
6. ✅ Deterministic (same input → same output)
7. ✅ Tested with agentic contract tests

---

**Next steps:** Implement Phase 1 (4 commands) + merge-data on current `kimi` branch, test against real docs, document patterns for future ops.
