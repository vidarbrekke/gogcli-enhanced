# External Review: Agentic Workflow Feedback

**Date:** 2025-02  
**Scope:** Full codebase, emphasis on docs-edit additions and agentic workflows (deterministic behavior, machine-readable output, safety rails, low-friction composition).

---

## What You Already Did Well (Agent-Friendly)

- **First-class safety flags:** `--dry-run` and `--require-revision` (optimistic concurrency) are exactly what agents need.
- **`docs edit batch` is excellent:**
  - Reads JSON from file/stdin
  - Validates "exactly one operation per request"
  - Supports `--validate-only`
  - Supports "normalized pretty JSON" + hash (great for caching/dedup and audit trails)
- **Structured JSON error envelopes** already exist in root command runner; `docsEditError` implements `JSONErrorFields()` so agents can parse `error_code`, `http_status`, `google_reason`, etc.

*If you stopped here, it's already usable in agent pipelines.*

---

## High-Impact Improvements for Agentic Workflows

### 1) Make `--dry-run` Not Require Auth

**Problem:** Several edit commands call `requireAccount()` before they get to dry-run, so agents cannot build plans without credentials.

**Why it matters:** Agents often build a plan first (dry-run/validate) without having secrets loaded or without wanting to touch the network.

**What to change:**

- For `docs edit insert`, `delete`, `replace`: build the request and emit dry-run output *before* `requireAccount()` / service creation.
- For `docs edit append`: you do need a read to compute the end index. Agent-friendly alternatives:
  - Allow `append --index <n>` (explicit index) OR
  - `append --assume-end` that inserts at 1 with a warning in dry-run output, OR
  - Provide a separate helper: `docs positions end <docId>` that returns the computed append index so the agent can compose: `(positions → insert)`.

**Net result:** Agents can safely generate exact `BatchUpdateDocumentRequest` payloads without credentials.

---

### 2) Add "Marker-Based" Insert/Delete Mode

**Problem:** Indexes in Google Docs are non-intuitive for humans and fragile for agents.

**Proposal:** Options that let agents target by text markers:

- `insert --after-text "Marker" --text "..." [--match-case]`
- `insert --before-text "Marker" ...`
- `delete --between "StartMarker" "EndMarker"` or `delete --match "text"`

**Implementation pattern:**

1. `Documents.Get` once (or use existing plain-text extraction logic)
2. Find marker ranges
3. Compute indices
4. Apply `InsertText` / `DeleteContentRange`

**Impact:** Single biggest improvement to reduce "agent makes wrong index" failures.

---

### 3) Standardize JSON Outputs Across Commands

**Proposal:** Formalize an output envelope for every command:

```json
{
  "ok": true,
  "operation": "docs.edit.replace",
  "documentId": "...",
  "stats": {...},
  "requestHash": "...",
  "requiredRevisionId": "...",
  "dryRun": false
}
```

Agents can treat every command uniformly; error envelope already standardizes failure.

---

### 4) Add `--timeout` and `--retries` (Agent Robustness)

**Problem:** Agents run in variable networks. Let them declare policy instead of ad-hoc wrapper scripts.

- `--timeout 30s` → passed into context deadline
- `--retries 3 --retry-backoff 250ms` for transient errors (429/5xx)

Especially helpful for batch edits.

---

### 5) Add `--output-request-file` for All Edit Subcommands (Not Just Batch)

**Problem:** Batch has great "emit normalized JSON" features, but replace/insert/delete/append do not.

**Proposal:**

- Implement shared helper: "build request → optionally write normalized request → execute or dry-run"
- Give each subcommand: `--output-request-file`, `--pretty`

**Benefits:** Request caching/dedup, review/audit, later replay with `docs edit batch`.

---

## Refactor Suggestions

### 6) Split `internal/cmd/docs.go`

`internal/cmd/docs.go` is now doing a lot (export/info/cat + full editing system + helpers).

**Refactor into:**

- `docs_cmd.go` — command structs + wiring
- `docs_edit_cmd.go` — edit subcommands
- `docs_edit_helpers.go` — hash/normalize/operation detection, dry-run output
- `docs_read_cmd.go` — export/info/cat

Makes future enhancements (marker insert/delete, regex replace, etc.) easier and reduces merge conflicts with upstream.

---

### 7) Remove Reflection-Based Request Operation Counting (Optional)

**Current:** Reflection used to verify "exactly one operation field set" per request.

**Suggestion:** Replace with explicit list or switch (generated or hand-written) for performance and maintainability. If keeping reflection, add a short comment explaining why it's safe and bounded.

---

## Security / Least-Privilege

### 8) Scope Escalation UX (Agent-Safe)

**Problem:** Docs editing requires write scopes. For agents, the failure mode should be crisp.

**Proposal:** Detect "insufficient scopes" error and return structured JSON:

```json
{
  "error_code": "insufficient_scope",
  "required_scopes": [...],
  "hint": "run gog auth ..."
}
```

Prevents agents from looping uselessly.

---

### 9) Default to Safe Destructive Behavior

**Current:** `docs edit delete` requires `--force` unless dry-run or JSON output. Good.

**Proposal:** Apply the same logic to any destructive modes added later (e.g., delete-by-marker, batch requests containing deletes).

---

## Agent Workflow Features

### 10) "Plan → Apply" First-Class Workflow

**Pattern:** Make it official:

```bash
docs edit replace ... --output-request-file ops.json --dry-run
docs edit batch <docId> --requests-file ops.json  # apply
```

Canonical agent pattern: build → review → apply.

---

### 11) Add `docs positions` Helper Command

To support index-based commands without fragile guessing:

- `docs positions search <docId> --text "foo"` → returns candidate ranges/indices
- `docs positions end <docId>` → returns append index
- `docs positions headings <docId>` → returns positions of headings

Turns "indices are hard" into a composable workflow agents can manage.

---

## Prioritized Implementation Plan

| Priority | Item | Effort | Status |
|----------|------|--------|--------|
| **P0** | 1. Reorder `requireAccount()` so dry-run works without auth (insert/delete/replace) | Small | ✅ Done |
| **P0** | 2. Add `--output-request-file` + `--pretty` to all edit subcommands | Medium | ✅ Done |
| **P0** | 3. Split `docs.go` into separate files | Medium | ✅ Done |
| **P1** | 4. Add marker-based insert/delete (start with `--after-text`) | Medium | Planned |
| **P1** | 5. Standardize JSON success envelope (ok, operation, documentId, requestHash, etc.) | Medium | Planned |
| **P2** | 6. Add `--timeout` and `--retries` | Medium | Planned |
| **P2** | 7. Scope escalation UX (`insufficient_scope` error envelope) | Small | Planned |
| **P2** | 8. Add `docs positions` helper (search, end, headings) | Medium | Planned |
| **P3** | 9. Replace reflection with explicit operation list (optional) | Small | Optional |
| **P3** | 10. `append --index` or `docs positions end` for auth-free append planning | Small | Planned |

---

## Quick Wins (Smallest Diffs, Biggest Payoff)

1. Reorder `requireAccount()` in insert/delete/replace so dry-run works without auth
2. Add `--output-request-file` + `--pretty` to all edit subcommands
3. Split `docs.go` into separate files
4. Add marker-based insert/delete (even just `--after-text` initially)

---

## References

- `handover.md` — Sheets/Slides rollout roadmap
- `docs/editing.md` — Docs editing user guide
- `AGENTS.md` — Agentic workflow rules
