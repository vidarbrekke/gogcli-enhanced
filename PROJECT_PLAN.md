# Google Docs Editing Feature - Complete Project Plan (Archived) / Google Sheets Editing - Next Plan

**Repository:** `gogcli-enhanced` (fork of steipete/gogcli)  
**Feature:** Docs edit completed; next implementation target is inline edit capabilities for Google Sheets  
**Version:** 1.0  
**Last Updated:** 2026-02-11  
**Status:** ✅ Docs Complete → 🎯 Next Active Phase: Sheets Editing

---

## 📋 Table of Contents

1. [Transition Update (Current Priority)](#transition-update-current-priority)
2. [Active Next Phase: Sheets Editing](#active-next-phase-sheets-editing)
3. [Historical Docs Plan (Archived Reference)](#historical-docs-plan-archived-reference)

---

## 📖 Project Overview

### Transition Update (Current Priority)

Google Docs editing work described in this document has been completed in this repository.

The next set of edit functionality we want is for **Google Sheets**:

- Keep the same agent-safe workflow introduced for Docs (`--dry-run`, `--validate-only`, `--pretty`, `--output-request-file`, `--execute-from-file`, structured JSON errors).
- Extend command surface under Sheets with a consistent `sheets edit ...` experience.
- Reuse existing Sheets capabilities (`update`, `append`, `clear`, `format`) as the implementation base.

Use this plan as pattern/reference for delivery approach and quality gates, but treat **Sheets editing** as the active execution target.

## 🎯 Active Next Phase: Sheets Editing

### Scope now

- Build `gog sheets edit ...` with Docs-level agentic safety and machine contracts.
- Reuse existing Sheets operations (`update`, `append`, `clear`, `format`) where possible.
- Prioritize deterministic agent workflows:
  - `--validate-only`
  - `--dry-run`
  - `--pretty`
  - `--output-request-file`
  - `--execute-from-file`
  - structured JSON errors with `error_code`

### Minimum deliverables

1. `sheets edit values`
2. `sheets edit append`
3. `sheets edit clear` (guarded destructive behavior)
4. `sheets edit batch`
5. parity tests + docs updates (`README.md`, `docs/editing.md`, `AGENTS.md`)

### Primary implementation references

- `handover.md`
- `docs/refactor/agentic-edit-rollout-checklist.md`
- `internal/cmd/docs.go` (completed pattern source)

## 📚 Historical Docs Plan (Archived Reference)

The remaining sections in this document are the completed Docs implementation plan retained for context and pattern reuse.

### Vision
Enable developers and automation scripts to programmatically edit Google Sheets content with the same reliability and safety now available for Google Docs edits.

### Problem Statement
Docs editing is now complete, but Sheets does not yet have the same unified agentic edit framework. This limits cross-service automation parity for use cases like:
- Automated report generation
- Template population
- Batch find/replace operations
- Programmatic content updates

### Solution
Implement a `gog sheets edit` command suite that mirrors Docs edit safety and machine contracts, while leveraging existing Sheets update/append/clear/format foundations and the Sheets batch update API where appropriate.

### Key Benefits
- **Automation-Friendly:** Scriptable document editing for CI/CD pipelines
- **Batch Operations:** Multiple edits in a single API call
- **Consistent UX:** Follows existing `gogcli` command patterns
- **No Manual Intervention:** Fully headless operation

---

## 🔍 Current State Analysis

### Repository Information
- **Original:** https://github.com/steipete/gogcli
- **Fork:** https://github.com/vidarbrekke/gogcli-enhanced
- **Local Path:** `/Users/vidarbrekke/Dev/ClawBackup/gogcli-enhanced`
- **Language:** Go (using Google APIs Client Library)
- **Architecture:** CLI built with Kong command parser

### Existing Docs Commands (Completed)

**File:** `internal/cmd/docs.go` (232 lines)

| Command | Functionality | Implementation |
|---------|--------------|----------------|
| `export` | Export to PDF/DOCX/TXT | Via Drive API export |
| `info` | Get document metadata | Docs API GET |
| `create` | Create new document | Drive API create |
| `copy` | Copy existing document | Drive API copy |
| `cat` | Print plain text content | Docs API GET + text extraction |

### Current Limitations (Now Applies to Sheets Extension Work)
- ❌ No `sheets edit` umbrella command with Docs-style agentic flags
- ❌ No unified preflight/normalize/execute-from-file workflow for Sheets edits
- ❌ No request hash output for Sheets edit preflight
- ❌ No consistent structured JSON error envelope for Sheets edit operations

### Technical Dependencies
```go
import (
    "google.golang.org/api/docs/v1"      // Google Docs API client
    "google.golang.org/api/drive/v3"     // Google Drive API client
    "github.com/steipete/gogcli/internal/googleapi"  // Service factories
    "github.com/steipete/gogcli/internal/outfmt"     // Output formatting
)
```

### Authentication
- OAuth2 refresh tokens (primary method)
- Service account support (Workspace domain-wide delegation)
- Docs scope requirement was `https://www.googleapis.com/auth/documents` (already included, completed feature).
- Next phase should target Sheets write scope requirements (`https://www.googleapis.com/auth/spreadsheets`) and preserve least-privilege behavior.

---

## 🎯 Goals & Success Criteria (Reference Baseline + Next Target)

### MVP Features (Version 1.0)

#### Must Have
1. ✅ Insert text at specific index
2. ✅ Append text to document end
3. ✅ Prepend text to document start
4. ✅ Replace text (find/replace all)
5. ✅ Delete text by range
6. ✅ JSON output support
7. ✅ Error handling (not found, out of bounds, API errors)
8. ✅ Unit test coverage >80%

#### Should Have
1. ✅ Batch operations (multiple edits in one request)
2. ✅ Dry-run mode (preview without executing)
3. ✅ Detailed error messages
4. ✅ Integration tests

#### Won't Have (Future Enhancements)
- Text formatting (bold, italic, colors) → v2.0
- Structure operations (tables, lists, headings) → v2.0
- Revision-based conflict detection → v2.0
- Interactive editing mode → v3.0

### Success Criteria

**Functional:**
- Developer can insert/replace/delete text via CLI
- Commands work with both OAuth and service accounts
- Output supports JSON for scripting automation
- Error messages are actionable and helpful

**Non-Functional:**
- Operations complete in <2 seconds for typical documents
- Test coverage >80% for new code
- No breaking changes to existing commands
- Follows existing code style and patterns

**Business:**
- Enable automation use cases (report generation, batch updates)
- Reduce manual document editing time
- Maintain backward compatibility

---

## 🏗️ Technical Architecture

### Command Structure

```
gog docs
├── export     (existing)
├── info       (existing)
├── create     (existing)
├── copy       (existing)
├── cat        (existing)
└── edit       (NEW)
    ├── insert    (insert text at index)
    ├── append    (insert at end)
    ├── prepend   (insert at start)
    ├── replace   (find/replace all)
    ├── delete    (delete range)
    └── batch     (multiple operations from JSON)
```

### File Structure

```
internal/
├── cmd/
│   ├── docs.go                  # Main commands (UPDATE)
│   └── docs_edit_test.go        # New test file (CREATE)
└── googleapi/
    └── docs.go                  # Service factory (NO CHANGE)
```

### Data Flow

```
User Input (CLI)
    ↓
Command Validation
    ↓
Service Factory (googleapi.NewDocs)
    ↓
Build BatchUpdateDocumentRequest
    ↓
Google Docs API v1
    ↓
Response Parsing
    ↓
Output Formatting (JSON or human-readable)
```

### API Integration

**Google Docs API v1 - batchUpdate Method**

**Endpoint:** `POST https://docs.googleapis.com/v1/documents/{documentId}:batchUpdate`

**Request Structure:**
```go
type BatchUpdateDocumentRequest struct {
    Requests []*Request  // Array of operations to perform
}

type Request struct {
    InsertText         *InsertTextRequest          // Insert text at index
    DeleteContentRange *DeleteContentRangeRequest   // Delete text range
    ReplaceAllText     *ReplaceAllTextRequest       // Find/replace all
    // ... many more for formatting, structure, etc.
}
```

**Key Concepts:**
- **1-based indexing:** Index 1 = start of document
- **Trailing newline:** Documents always end with `\n` (cannot delete)
- **Batch execution:** Requests execute in order, indexes auto-adjust
- **Atomic operation:** All requests succeed or all fail

---

## 📝 Implementation Plan

### Phase 1: Core Infrastructure (Days 1-3)

**Goal:** Get basic insert command working end-to-end

**Tasks:**

1. **Add Command Structure** (1 hour)
   - Add `Edit DocsEditCmd` to `DocsCmd` struct
   - Create `DocsEditCmd` with `Insert` subcommand
   - Define `DocsInsertCmd` struct with args/flags

2. **Implement Insert Command** (2 hours)
   ```go
   func (c *DocsInsertCmd) Run(ctx context.Context, flags *RootFlags) error {
       // 1. Validate inputs (docID, text, index)
       // 2. Create Docs service
       // 3. Build BatchUpdateDocumentRequest
       // 4. Execute batchUpdate
       // 5. Format and return response
   }
   ```

3. **Add Helper Functions** (1 hour)
   - `validateDocID(id string) error`
   - `validateIndex(index, maxIndex int64) error`
   - `getDocumentEndIndex(ctx, svc, docID) (int64, error)`

4. **Write Initial Tests** (2 hours)
   - Mock HTTP server test for insert
   - Test JSON output format
   - Test error cases (invalid ID, out of bounds)

5. **Manual Testing** (1 hour)
   - Create test document
   - Test insert at start, middle, end
   - Verify with `gog docs cat`

**Deliverable:** Working `gog docs edit insert <docId> <text> --index N` command

---

### Phase 2: Additional Operations (Days 4-7)

**Goal:** Complete all text editing operations

#### 2.1 Append Command (Day 4)
```go
type DocsAppendCmd struct {
    DocID string `arg:"" name:"docId" help:"Doc ID"`
    Text  string `arg:"" name:"text" help:"Text to append"`
}

func (c *DocsAppendCmd) Run(ctx context.Context, flags *RootFlags) error {
    // Get document end index
    endIndex := getDocumentEndIndex(ctx, svc, c.DocID)
    
    // Insert at end - 1 (before trailing newline)
    req := &docs.BatchUpdateDocumentRequest{
        Requests: []*docs.Request{
            {InsertText: &docs.InsertTextRequest{
                Location: &docs.Location{Index: endIndex},
                Text:     c.Text,
            }},
        },
    }
    // Execute...
}
```

#### 2.2 Prepare Command (Day 4)
```go
type DocsPrependCmd struct {
    DocID string `arg:"" name:"docId" help:"Doc ID"`
    Text  string `arg:"" name:"text" help:"Text to prepend"`
}

func (c *DocsPrependCmd) Run(ctx context.Context, flags *RootFlags) error {
    // Always insert at index 1 (start of document)
    req := &docs.BatchUpdateDocumentRequest{
        Requests: []*docs.Request{
            {InsertText: &docs.InsertTextRequest{
                Location: &docs.Location{Index: 1},
                Text:     c.Text,
            }},
        },
    }
    // Execute...
}
```

#### 2.3 Replace Command (Day 5)
```go
type DocsReplaceCmd struct {
    DocID      string `arg:"" name:"docId" help:"Doc ID"`
    Find       string `arg:"" name:"find" help:"Text to find"`
    Replace    string `arg:"" name:"replace" help:"Replacement text"`
    MatchCase  bool   `name:"match-case" help:"Case-sensitive match" default:"false"`
    ReplaceAll bool   `name:"all" help:"Replace all occurrences" default:"true"`
}

func (c *DocsReplaceCmd) Run(ctx context.Context, flags *RootFlags) error {
    req := &docs.BatchUpdateDocumentRequest{
        Requests: []*docs.Request{
            {ReplaceAllText: &docs.ReplaceAllTextRequest{
                ContainsText: &docs.SubstringMatchCriteria{
                    Text:      c.Find,
                    MatchCase: c.MatchCase,
                },
                ReplaceText: c.Replace,
            }},
        },
    }
    
    // Execute and report occurrences changed
    resp, err := svc.Documents.BatchUpdate(c.DocID, req).Context(ctx).Do()
    if err != nil {
        return err
    }
    
    // Extract count from response
    occurrences := resp.Replies[0].ReplaceAllText.OccurrencesChanged
    u.Out().Printf("replaced %d occurrences", occurrences)
    return nil
}
```

#### 2.4 Delete Command (Day 6)
```go
type DocsDeleteCmd struct {
    DocID      string `arg:"" name:"docId" help:"Doc ID"`
    StartIndex int64  `arg:"" name:"start" help:"Start index (1-based)"`
    EndIndex   int64  `arg:"" name:"end" help:"End index (exclusive)"`
}

func (c *DocsDeleteCmd) Run(ctx context.Context, flags *RootFlags) error {
    // Validate range
    if c.StartIndex < 1 {
        return usage("start index must be >= 1")
    }
    if c.EndIndex <= c.StartIndex {
        return usage("end must be > start")
    }
    
    req := &docs.BatchUpdateDocumentRequest{
        Requests: []*docs.Request{
            {DeleteContentRange: &docs.DeleteContentRangeRequest{
                Range: &docs.Range{
                    StartIndex: c.StartIndex,
                    EndIndex:   c.EndIndex,
                },
            }},
        },
    }
    
    // Execute...
    u.Out().Printf("deleted %d chars", c.EndIndex - c.StartIndex)
    return nil
}
```

#### 2.5 Testing (Day 7)
- Unit tests for append, prepend, replace, delete
- Integration tests (create → edit → verify)
- Error handling tests

**Deliverable:** All basic edit commands functional with tests

---

### Phase 3: Advanced Features (Days 8-10)

#### 3.1 Batch Operations (Day 8-9)

**Command:**
```go
type DocsBatchCmd struct {
    DocID     string `arg:"" name:"docId" help:"Doc ID"`
    BatchFile string `name:"batch" help:"JSON file with operations (use - for stdin)"`
}

func (c *DocsBatchCmd) Run(ctx context.Context, flags *RootFlags) error {
    // Read batch file (or stdin)
    var reader io.Reader
    if c.BatchFile == "-" {
        reader = os.Stdin
    } else {
        f, err := os.Open(c.BatchFile)
        if err != nil {
            return fmt.Errorf("open batch file: %w", err)
        }
        defer f.Close()
        reader = f
    }
    
    // Decode batch request
    var req docs.BatchUpdateDocumentRequest
    if err := json.NewDecoder(reader).Decode(&req); err != nil {
        return fmt.Errorf("decode batch JSON: %w", err)
    }
    
    // Validate and execute
    if len(req.Requests) == 0 {
        return usage("batch request has no operations")
    }
    
    resp, err := svc.Documents.BatchUpdate(c.DocID, &req).Context(ctx).Do()
    if err != nil {
        return fmt.Errorf("batch update failed: %w", err)
    }
    
    // Output summary
    u.Out().Printf("executed %d operations", len(resp.Replies))
    return nil
}
```

**Batch JSON Format:**
```json
{
  "requests": [
    {
      "insertText": {
        "location": {"index": 1},
        "text": "Title\n"
      }
    },
    {
      "updateTextStyle": {
        "range": {"startIndex": 1, "endIndex": 6},
        "textStyle": {"bold": true},
        "fields": "bold"
      }
    },
    {
      "insertText": {
        "location": {"index": 7},
        "text": "Body content..."
      }
    }
  ]
}
```

#### 3.2 Helper Functions (Day 9)

**Index Management:**
```go
// getDocumentEndIndex returns the index before the trailing newline
func getDocumentEndIndex(ctx context.Context, svc *docs.Service, docID string) (int64, error) {
    doc, err := svc.Documents.Get(docID).Context(ctx).Do()
    if err != nil {
        return 0, err
    }
    
    if len(doc.Body.Content) == 0 {
        return 1, nil  // Empty document
    }
    
    lastElement := doc.Body.Content[len(doc.Body.Content)-1]
    return lastElement.EndIndex - 1, nil
}

// validateIndexRange checks if indexes are within document bounds
func validateIndexRange(start, end, maxIndex int64) error {
    if start < 1 {
        return fmt.Errorf("start index must be >= 1 (got %d)", start)
    }
    if end > maxIndex {
        return fmt.Errorf("end index %d exceeds document length %d", end, maxIndex)
    }
    if end <= start {
        return fmt.Errorf("end index (%d) must be > start index (%d)", end, start)
    }
    return nil
}

// validateDocID performs basic sanity checks on document ID
func validateDocID(id string) error {
    id = strings.TrimSpace(id)
    if id == "" {
        return errors.New("document ID cannot be empty")
    }
    if len(id) < 20 {  // Google Doc IDs are typically 40+ chars
        return errors.New("document ID appears invalid (too short)")
    }
    return nil
}
```

#### 3.3 Enhanced Error Messages (Day 10)
```go
// Custom error types for better messages
type IndexOutOfBoundsError struct {
    Index    int64
    MaxIndex int64
}

func (e *IndexOutOfBoundsError) Error() string {
    return fmt.Sprintf("index %d is out of bounds (valid range: 1-%d)", e.Index, e.MaxIndex)
}

// Error handling wrapper
func handleDocsError(err error, docID string) error {
    var apiErr *googleapi.Error
    if errors.As(err, &apiErr) {
        switch apiErr.Code {
        case http.StatusNotFound:
            return fmt.Errorf("document not found (id=%s), or you lack permissions", docID)
        case http.StatusForbidden:
            return fmt.Errorf("insufficient permissions to edit document (id=%s)", docID)
        case http.StatusBadRequest:
            return fmt.Errorf("invalid request: %s", apiErr.Message)
        }
    }
    return err
}
```

**Deliverable:** Batch operations working, comprehensive helpers, better errors

---

### Phase 4: Testing (Days 11-13)

#### 4.1 Unit Tests (`internal/cmd/docs_edit_test.go`)

**Test Coverage:**
- ✅ Insert at start, middle, end
- ✅ Append to empty and non-empty documents
- ✅ Prepend to documents
- ✅ Replace single vs all occurrences
- ✅ Replace case-sensitive vs case-insensitive
- ✅ Delete valid ranges
- ✅ Delete edge cases (start, end, entire content)
- ✅ Batch operations (multiple requests)
- ✅ Error cases (invalid ID, out of bounds, API errors)
- ✅ JSON output format validation

**Test Structure:**
```go
func TestDocsInsert_Start(t *testing.T) {
    srv := setupMockDocsServer(t)
    defer srv.Close()
    
    ctx := testContext()
    flags := &RootFlags{Account: "test@example.com"}
    
    cmd := &DocsInsertCmd{
        DocID: "doc123",
        Text:  "Hello",
        Index: 1,
    }
    
    if err := cmd.Run(ctx, flags); err != nil {
        t.Fatalf("Run failed: %v", err)
    }
    
    // Verify request was correct
    // Verify output format
}

func TestDocsReplace_CaseSensitive(t *testing.T) {
    // Test case-sensitive replacement
}

func TestDocsDelete_OutOfBounds(t *testing.T) {
    // Test error handling for invalid ranges
}
```

#### 4.2 Integration Tests (Shell Script)

**File:** `scripts/test-edit-integration.sh`
```bash
#!/bin/bash
set -e

# Create test document
echo "Creating test document..."
DOCID=$(gog docs create "Edit Integration Test" --json | jq -r '.file.id')
echo "Document ID: $DOCID"

# Test insert
echo "Testing insert..."
gog docs edit insert $DOCID "Line 1" --index 1

# Verify
CONTENT=$(gog docs cat $DOCID)
if [[ "$CONTENT" != "Line 1" ]]; then
    echo "FAIL: Insert didn't work"
    exit 1
fi
echo "PASS: Insert"

# Test append
echo "Testing append..."
gog docs edit append $DOCID "\nLine 2"
CONTENT=$(gog docs cat $DOCID)
if [[ "$CONTENT" != "Line 1\nLine 2" ]]; then
    echo "FAIL: Append didn't work"
    exit 1
fi
echo "PASS: Append"

# Test replace
echo "Testing replace..."
gog docs edit replace $DOCID "Line" "Row" --all
CONTENT=$(gog docs cat $DOCID)
if [[ "$CONTENT" != "Row 1\nRow 2" ]]; then
    echo "FAIL: Replace didn't work"
    exit 1
fi
echo "PASS: Replace"

# Test delete
echo "Testing delete..."
gog docs edit delete $DOCID 1 6
CONTENT=$(gog docs cat $DOCID)
if [[ "$CONTENT" != "\nRow 2" ]]; then
    echo "FAIL: Delete didn't work"
    exit 1
fi
echo "PASS: Delete"

# Cleanup
echo "Cleaning up..."
gog drive delete $DOCID

echo "All tests passed!"
```

#### 4.3 Performance Testing

**Load Test:**
```bash
# Create large document (10,000 lines)
for i in {1..10000}; do echo "Line $i"; done > /tmp/large.txt
DOCID=$(gog docs create "Large Doc Test" --json | jq -r '.file.id')

# Measure insert performance
time gog docs edit append $DOCID "$(cat /tmp/large.txt)"

# Measure replace performance
time gog docs edit replace $DOCID "Line" "Row" --all

# Should complete in <5 seconds for 10k lines
```

**Deliverable:** >80% test coverage, passing integration tests, performance benchmarks

---

### Phase 5: Documentation (Days 14-15)

#### 5.1 README.md Updates

**Add to Features Section:**
```markdown
- **Docs** - export to PDF/DOCX/TXT, create/copy docs, **edit inline** (insert/append/replace/delete text, batch operations)
```

**Add Usage Section:**
```markdown
### Edit Google Docs

Insert text at a specific position:
```bash
gog docs edit insert <docId> "New text" --index 1
```

Append to end of document:
```bash
gog docs edit append <docId> "Footer text"
```

Find and replace:
```bash
gog docs edit replace <docId> "old text" "new text" --all --match-case
```

Delete text range:
```bash
gog docs edit delete <docId> 10 50
```

Batch operations from JSON:
```bash
cat operations.json | gog docs edit batch <docId> --batch -
```
```
```

#### 5.2 Create `docs/editing.md`

**Table of Contents:**
1. Introduction to Docs Editing
2. Understanding Document Indexes
3. Insert Operations
4. Replace Operations
5. Delete Operations
6. Batch Operations
7. Error Handling
8. Examples and Use Cases
9. API Limits and Best Practices

**Content Outline:**
```markdown
# Google Docs Editing Guide

## Introduction
The `gog docs edit` command suite enables programmatic editing of Google Docs...

## Understanding Indexes
Google Docs uses 1-based indexing...

[Diagram of document structure]

## Insert Operations
### Insert at Specific Index
...

### Append to End
...

## Batch Operations
### JSON Format
...

### Example: Template Population
...

## Best Practices
- Use batch operations for multiple edits
- Test with copies before production edits
- Handle API errors gracefully
```

#### 5.3 Inline Code Documentation

**Ensure all public functions have godoc comments:**
```go
// DocsEditCmd groups all document editing operations.
// It provides subcommands for inserting, replacing, and deleting text.
type DocsEditCmd struct { ... }

// Run executes the insert command, adding text to the document at the specified index.
// Indexes are 1-based (1 = start of document).
// Returns an error if the document is not found, the index is out of bounds, or the API call fails.
func (c *DocsInsertCmd) Run(ctx context.Context, flags *RootFlags) error { ... }
```

#### 5.4 CHANGELOG.md Entry

```markdown
## [Unreleased]

### Added
- `gog docs edit insert` - Insert text at specific index
- `gog docs edit append` - Append text to document end
- `gog docs edit prepend` - Prepend text to document start
- `gog docs edit replace` - Find and replace text
- `gog docs edit delete` - Delete text range
- `gog docs edit batch` - Execute multiple operations from JSON file
- Helper functions for document index management
- Comprehensive error messages for edit operations

### Changed
- Updated README with docs editing examples
- Added docs/editing.md guide

### Fixed
- N/A

## [0.9.0] - Previous version...
```

**Deliverable:** Complete documentation for all edit features

---

## 🧪 Testing Strategy

### Test Pyramid

```
           ┌─────────┐
           │  Manual │  (5%)
           │   E2E   │
           └─────────┘
         ┌─────────────┐
         │ Integration │  (15%)
         │   Tests     │
         └─────────────┘
     ┌───────────────────┐
     │   Unit Tests      │  (80%)
     │ (Mock HTTP Server)│
     └───────────────────┘
```

### Test Coverage Goals

| Component | Target Coverage | Priority |
|-----------|----------------|----------|
| Command logic | >90% | High |
| Helper functions | 100% | High |
| Error handling | >85% | High |
| Output formatting | >75% | Medium |
| Integration | Manual + scripts | Medium |

### Testing Tools

**Unit Tests:**
- Go standard `testing` package
- Mock HTTP server (`httptest.NewServer`)
- Existing `gogcli` test patterns

**Integration Tests:**
- Bash scripts
- Real Google Docs API calls (test account)
- Automated cleanup

**Linting:**
- `golangci-lint` (already configured in `.golangci.yml`)
- Run via `make lint`

### Continuous Testing

**Pre-commit:**
```bash
make lint
make test
```

**CI/CD (if applicable):**
- Run full test suite on PR
- Integration tests on merge to main

---

## 📚 Documentation Requirements

### User-Facing Documentation

1. **README.md**
   - Updated features list
   - Quick start examples
   - Link to detailed editing guide

2. **docs/editing.md** (NEW)
   - Comprehensive editing guide
   - Index management explained
   - Batch operation examples
   - Error handling guide
   - FAQ and troubleshooting

3. **CHANGELOG.md**
   - All new features documented
   - Breaking changes (if any)
   - Migration guide (if needed)

### Developer Documentation

1. **Inline Code Comments**
   - All public types have godoc comments
   - Complex logic explained
   - Error cases documented

2. **PROJECT_PLAN.md** (THIS FILE)
   - Complete implementation plan
   - Architecture decisions
   - Testing strategy

3. **Example Files**
   - `examples/batch-operations.json`
   - `examples/edit-workflow.sh`
   - `examples/template-population.sh`

### Help Text

**Ensure `--help` is comprehensive:**
```bash
gog docs edit --help
# Should show: Available subcommands with brief descriptions

gog docs edit insert --help
# Should show: Full usage, argument descriptions, examples
```

---

## 📅 Timeline & Milestones

### 3-Week Development Plan

**Week 1: Core Implementation**
- **Days 1-3:** Phase 1 (Insert command)
- **Days 4-5:** Phase 2 (Append, Prepend, Replace commands)

**Milestone 1:** Insert and append commands working end-to-end

**Week 2: Complete Features**
- **Days 6-7:** Phase 2 (Delete command, polish)
- **Days 8-10:** Phase 3 (Batch operations, helpers, error handling)

**Milestone 2:** All commands implemented, helpers complete

**Week 3: Quality & Release**
- **Days 11-13:** Phase 4 (Comprehensive testing)
- **Days 14-15:** Phase 5 (Documentation)

**Milestone 3:** Production-ready release

### Daily Checklist Template

```markdown
## Day N Checklist

**Goal:** [One-line goal]

**Tasks:**
- [ ] Code changes (2-3 hours)
- [ ] Unit tests (1-2 hours)
- [ ] Manual testing (30 min)
- [ ] Documentation updates (30 min)
- [ ] Code review prep (30 min)

**Deliverable:** [Specific outcome]

**Blockers:** [None / List issues]

**Notes:** [Learnings, decisions made]
```

---

## 👥 Developer Onboarding

### Getting Started (30 Minutes)

#### Prerequisites
- Go 1.21+ installed
- Git configured
- Google account with Docs API enabled
- `gog` OAuth credentials set up

#### Setup Steps

1. **Clone Repository**
   ```bash
   cd /Users/vidarbrekke/Dev/ClawBackup  # Or your dev folder
   # Fork is already cloned at gogcli-enhanced
   cd gogcli-enhanced
   ```

2. **Create Feature Branch**
   ```bash
   git checkout -b feature/docs-editing
   ```

3. **Build and Test**
   ```bash
   make          # Build binary
   make test     # Run tests
   make lint     # Check code style
   ```

4. **Set Up Test Environment**
   ```bash
   # Ensure you have gog auth set up
   export GOG_ACCOUNT=your@email.com
   
   # Create a test document
   ./bin/gog docs create "Edit Test Doc" --json | jq -r '.file.id' > /tmp/test-doc-id
   export TEST_DOC_ID=$(cat /tmp/test-doc-id)
   
   echo "Test doc: https://docs.google.com/document/d/$TEST_DOC_ID/edit"
   ```

5. **Verify Setup**
   ```bash
   # These should work:
   ./bin/gog docs info $TEST_DOC_ID
   ./bin/gog docs cat $TEST_DOC_ID
   ```

### Development Workflow

```
Edit Code → Build → Test Manually → Write Unit Test → Run Tests → Lint → Commit
```

**Detailed Steps:**

1. **Edit files** in `internal/cmd/docs.go`
2. **Build:** `make`
3. **Test manually:** `./bin/gog docs edit insert $TEST_DOC_ID "test"`
4. **Write unit test** in `internal/cmd/docs_edit_test.go`
5. **Run tests:** `make test`
6. **Lint:** `make lint`
7. **Commit:** `git commit -m "feat(docs): add insert command"`

### Key Files to Understand

| File | Purpose | Lines |
|------|---------|-------|
| `internal/cmd/docs.go` | All docs commands | ~232 (will grow to ~450) |
| `internal/googleapi/docs.go` | Service factory | ~15 |
| `internal/outfmt/*.go` | Output formatting | Various |
| `.golangci.yml` | Linting config | ~100 |

### Coding Standards

**Follow existing patterns:**
- Use Kong for command definitions
- Service initialization via `newDocsService`
- Error handling with custom helpers
- Output supports `--json` and human-readable
- Test with mock HTTP servers

**Commit Messages:**
- Use Conventional Commits format
- Examples:
  - `feat(docs): add insert command`
  - `test(docs): add unit tests for replace command`
  - `docs: update README with edit examples`
  - `fix(docs): handle edge case in index calculation`

**Code Style:**
- Run `make lint` before committing
- Follow existing patterns in `docs.go`
- Add godoc comments to all public types/functions
- Keep functions focused and small

---

## 🤝 Contributing Guidelines

### Before Starting Work

1. **Check this plan** - Ensure you're aligned with the architecture
2. **Coordinate** - If multiple developers, claim a phase/command
3. **Branch naming** - Use `feature/docs-editing-<command>` or `feature/docs-editing`

### Pull Request Process

1. **Small, Focused PRs**
   - One command per PR preferred
   - Include tests and docs in same PR

2. **PR Description Template**
   ```markdown
   ## What
   Brief description of changes
   
   ## Why
   What problem does this solve?
   
   ## How
   Technical approach summary
   
   ## Testing
   - [ ] Unit tests added
   - [ ] Manual testing completed
   - [ ] Integration test updated
   
   ## Checklist
   - [ ] Code follows existing patterns
   - [ ] Tests pass (`make test`)
   - [ ] Linter passes (`make lint`)
   - [ ] Documentation updated
   - [ ] Changelog entry added
   ```

3. **Review Criteria**
   - Follows existing code patterns
   - Has unit tests with >80% coverage
   - Documentation updated
   - No breaking changes to existing commands
   - Linter passes

### Communication

**Questions?**
- Reference this PROJECT_PLAN.md
- Check existing `docs.go` for patterns
- Review Google Docs API documentation

**Stuck?**
- Check the [Reference Materials](#reference-materials) section
- Create an issue with `[question]` tag
- Review test files for examples

### Code Ownership

**Primary Maintainer:** Vidar (@vidarbrekke)

**Review Required:** Yes for all changes to `internal/cmd/docs.go`

---

## 📖 Reference Materials

### Google Docs API Documentation

**Official Resources:**
- [Docs API Overview](https://developers.google.com/docs/api)
- [batchUpdate Method](https://developers.google.com/docs/api/reference/rest/v1/documents/batchUpdate)
- [Request Types](https://developers.google.com/docs/api/reference/rest/v1/documents/request)
- [Go Client Library](https://pkg.go.dev/google.golang.org/api/docs/v1)

**Key API Methods:**

| Method | Purpose | Request Type |
|--------|---------|-------------|
| `documents.get` | Fetch document structure | GET |
| `documents.batchUpdate` | Modify document content | POST |
| `documents.create` | Create new document | POST (Drive API) |

**Request Types Reference:**

| Request | Purpose | Key Fields |
|---------|---------|------------|
| `InsertText` | Insert text at index | `Location.Index`, `Text` |
| `DeleteContentRange` | Delete text range | `Range.StartIndex`, `Range.EndIndex` |
| `ReplaceAllText` | Find/replace all | `ContainsText.Text`, `ReplaceText` |
| `UpdateTextStyle` | Format text | `Range`, `TextStyle`, `Fields` |
| `UpdateParagraphStyle` | Format paragraph | `Range`, `ParagraphStyle`, `Fields` |

### gogcli Code Patterns

**Service Initialization:**
```go
svc, err := newDocsService(ctx, account)
```

**Error Handling:**
```go
if isDocsNotFound(err) {
    return fmt.Errorf("doc not found (id=%s)", id)
}
```

**Output Formatting:**
```go
if outfmt.IsJSON(ctx) {
    return outfmt.WriteJSON(os.Stdout, data)
}
u := ui.FromContext(ctx)
u.Out().Printf("key\tvalue")
```

**Command Structure:**
```go
type MyCmd struct {
    RequiredArg string `arg:"" name:"argName" help:"Description"`
    OptionalFlag string `name:"flag-name" help:"Description" default:"value"`
}

func (c *MyCmd) Run(ctx context.Context, flags *RootFlags) error {
    // 1. Validate
    // 2. Get service
    // 3. Build request
    // 4. Execute
    // 5. Format output
    return nil
}
```

### Additional Planning Documents

This repository contains several planning documents:

1. **PROJECT_PLAN.md** (this file) - Complete implementation plan
2. **DEVELOPMENT_PLAN.md** - Detailed phase breakdown
3. **DOCS_API_REFERENCE.md** - Google Docs API quick reference
4. **IMPLEMENTATION_SUMMARY.md** - Progress tracker
5. **QUICKSTART.md** - 30-minute getting started guide

**Recommended Reading Order:**
1. Start with this PROJECT_PLAN.md (overview)
2. Use QUICKSTART.md for hands-on start
3. Reference DOCS_API_REFERENCE.md when writing code
4. Track progress with IMPLEMENTATION_SUMMARY.md

---

## 🎯 Definition of Done

A feature/phase is complete when:

### Code Complete
- ✅ All planned commands implemented
- ✅ Code follows existing patterns
- ✅ Linter passes with no errors
- ✅ No breaking changes to existing functionality

### Testing Complete
- ✅ Unit tests written and passing
- ✅ Test coverage >80%
- ✅ Integration tests passing
- ✅ Manual testing completed
- ✅ Edge cases handled

### Documentation Complete
- ✅ Inline comments added
- ✅ README.md updated
- ✅ docs/editing.md created
- ✅ CHANGELOG.md entry added
- ✅ `--help` text comprehensive

### Review Complete
- ✅ Code reviewed by maintainer
- ✅ Feedback addressed
- ✅ Approved for merge

### Release Ready
- ✅ No known critical bugs
- ✅ Performance acceptable (<2s for typical docs)
- ✅ Error messages helpful
- ✅ Works with OAuth and service accounts

---

## 🚀 Quick Reference Commands

### Build & Test
```bash
make                    # Build binary to ./bin/gog
make test              # Run all unit tests
make lint              # Run linter
go test ./internal/cmd -v -run TestDocs  # Run specific tests
```

### Development Commands
```bash
# Create test doc
gog docs create "Test" --json | jq -r '.file.id'

# Test insert
./bin/gog docs edit insert <docId> "text" --index 1

# Test with JSON output
./bin/gog docs edit insert <docId> "text" --index 1 --json

# View document
gog docs cat <docId>

# Cleanup
gog drive delete <docId>
```

### Common Issues

**"command not found: gog"**
```bash
export PATH="$PWD/bin:$PATH"
# Or use: ./bin/gog
```

**"insufficient authentication scopes"**
```bash
gog auth add $GOG_ACCOUNT --services docs --force-consent
```

**"build failed"**
```bash
go mod tidy
make clean
make
```

---

## 📞 Support & Questions

**For Implementation Questions:**
- Review this PROJECT_PLAN.md first
- Check QUICKSTART.md for hands-on guidance
- Review existing code in `internal/cmd/docs.go`
- Check Google Docs API documentation

**For Technical Issues:**
- Create GitHub issue with detailed description
- Include error messages and steps to reproduce
- Tag with `[question]` or `[bug]`

**For Design Decisions:**
- Reference this plan's Architecture section
- Discuss in PR before implementing major changes

---

## ✏️ Document Maintenance

**This document should be updated when:**
- Architecture decisions change
- New phases/tasks are added
- Timeline shifts significantly
- New reference materials are discovered

**Update Process:**
1. Make changes to this file
2. Update "Last Updated" date at top
3. Commit with message: `docs: update PROJECT_PLAN with <change>`

---

## 📋 Appendix: Code Snippets

### Complete Insert Command Example

```go
// DocsInsertCmd inserts text at a specific index in a Google Doc.
type DocsInsertCmd struct {
    DocID string `arg:"" name:"docId" help:"Document ID"`
    Text  string `arg:"" name:"text" help:"Text to insert"`
    Index int64  `name:"index" help:"Character index (1-based, 1=start)" default:"1"`
}

// Run executes the insert operation.
func (c *DocsInsertCmd) Run(ctx context.Context, flags *RootFlags) error {
    u := ui.FromContext(ctx)
    
    // 1. Get and validate account
    account, err := requireAccount(flags)
    if err != nil {
        return err
    }
    
    // 2. Validate inputs
    id := strings.TrimSpace(c.DocID)
    if id == "" {
        return usage("empty docId")
    }
    if c.Text == "" {
        return usage("empty text")
    }
    if c.Index < 1 {
        return usage("index must be >= 1")
    }
    
    // 3. Create Docs service
    svc, err := newDocsService(ctx, account)
    if err != nil {
        return fmt.Errorf("create docs service: %w", err)
    }
    
    // 4. Build batch update request
    req := &docs.BatchUpdateDocumentRequest{
        Requests: []*docs.Request{
            {
                InsertText: &docs.InsertTextRequest{
                    Location: &docs.Location{
                        Index: c.Index,
                    },
                    Text: c.Text,
                },
            },
        },
    }
    
    // 5. Execute batch update
    resp, err := svc.Documents.BatchUpdate(id, req).Context(ctx).Do()
    if err != nil {
        if isDocsNotFound(err) {
            return fmt.Errorf("doc not found or not a Google Doc (id=%s)", id)
        }
        return fmt.Errorf("batch update failed: %w", err)
    }
    
    // 6. Format and output response
    if outfmt.IsJSON(ctx) {
        return outfmt.WriteJSON(os.Stdout, resp)
    }
    
    u.Out().Printf("inserted %d chars at index %d", len(c.Text), c.Index)
    u.Out().Printf("documentId\t%s", resp.DocumentId)
    
    return nil
}
```

### Complete Test Example

```go
func TestDocsInsert_Success(t *testing.T) {
    origDocs := newDocsService
    t.Cleanup(func() { newDocsService = origDocs })
    
    // Set up mock HTTP server
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if strings.Contains(r.URL.Path, ":batchUpdate") {
            // Verify request
            var req docs.BatchUpdateDocumentRequest
            if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                t.Fatalf("decode request: %v", err)
            }
            
            if len(req.Requests) != 1 {
                t.Fatalf("expected 1 request, got %d", len(req.Requests))
            }
            
            insertReq := req.Requests[0].InsertText
            if insertReq == nil {
                t.Fatal("expected InsertText request")
            }
            if insertReq.Text != "test text" {
                t.Fatalf("expected 'test text', got %q", insertReq.Text)
            }
            if insertReq.Location.Index != 1 {
                t.Fatalf("expected index 1, got %d", insertReq.Location.Index)
            }
            
            // Return mock response
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(map[string]any{
                "documentId": "doc123",
                "replies":    []any{},
            })
            return
        }
        http.NotFound(w, r)
    }))
    defer srv.Close()
    
    // Mock service factory
    newDocsService = func(ctx context.Context, email string) (*docs.Service, error) {
        return docs.NewService(ctx,
            option.WithoutAuthentication(),
            option.WithHTTPClient(srv.Client()),
            option.WithEndpoint(srv.URL+"/"),
        )
    }
    
    // Set up context
    ctx := context.Background()
    ctx = outfmt.SetJSON(ctx, true)
    ctx = ui.NewContext(ctx, ui.New())
    
    // Run command
    flags := &RootFlags{Account: "test@example.com"}
    cmd := &DocsInsertCmd{
        DocID: "doc123",
        Text:  "test text",
        Index: 1,
    }
    
    if err := cmd.Run(ctx, flags); err != nil {
        t.Fatalf("Run failed: %v", err)
    }
}
```

---

**End of PROJECT_PLAN.md**

**Status:** 🟢 Ready to Begin Implementation  
**Next Action:** Assign phases to developers and begin Phase 1 (Insert command)
