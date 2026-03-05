# Command Dossier: Sheets Read Commands

Scope:

- `gog sheets metadata`
- `gog sheets get`
- `gog sheets notes`
- `gog sheets links`

Tier: A (low risk with moderate schema sensitivity)  
Target: `candidate-gws`  
DRI: TBD

---

## 1) Migration Rationale

- Core read primitives used in automation/reporting.
- Good Tier A candidate to gain backend coverage benefits.
- Lets team validate normalization patterns for range/value outputs.

---

## 2) Contract Requirements

Critical behavior to preserve:

- Range parsing/echo semantics.
- Value table structure in JSON and plain outputs.
- Notes/links extraction format currently consumed by scripts.
- Stable error envelope for invalid range, not found, and permission cases.

Important:

- Preserve empty-cell, sparse-range, and mixed-type value behavior.

---

## 3) Provider Mapping (Canonical -> gws)

Command intent mapping:

- `metadata`: spreadsheet properties and sheet descriptors.
- `get`: range values retrieval.
- `notes`: note extraction for range.
- `links`: rich-text link extraction for range.

Adapter normalization duties:

1. Coerce provider response into current gog schema.
2. Preserve ordering and indexing assumptions where documented.
3. Normalize missing optional fields consistently.

---

## 4) Known Parity Risks

1. Value typing differences (string/number/boolean representation).
2. Empty trailing rows/columns handling.
3. Notes and rich-text links extraction nuances.
4. Range normalization differences (`Sheet1!A1:B10` variants).

Mitigation:

- Canonical conversion layer for cell values.
- Golden tests with mixed data types and sparse matrices.
- Dedicated tests for notes/links extraction edge cases.

---

## 5) Test Plan

Unit tests:

- mapping of command args and flags to provider invocations.
- value and range normalization helpers.

Contract tests:

- golden JSON for:
  - simple table
  - sparse data
  - mixed types
  - notes/links present and absent

Integration tests:

- native vs gws provider comparisons on identical fixture sheets.

Shadow tests:

- read-only shadow with diff report categorization:
  - value mismatch
  - range mismatch
  - notes/links mismatch

---

## 6) Rollout Plan

Suggested sequencing:

1. `sheets metadata`
2. `sheets get`
3. `sheets notes`
4. `sheets links`

Feature flags:

- `provider.gws.command.sheets-metadata.enabled`
- `provider.gws.command.sheets-get.enabled`
- `provider.gws.command.sheets-notes.enabled`
- `provider.gws.command.sheets-links.enabled`

---

## 7) Rollback Plan

- Disable affected command flags.
- Route all listed commands back to native implementation.
- Capture and archive latest diff/error artifacts.

---

## 8) Done Criteria

- Critical field parity confirmed for all listed commands.
- No unresolved critical diff classes in shadow phase.
- Canary performance and error profile meet baseline.
- Matrix and change log updated.
