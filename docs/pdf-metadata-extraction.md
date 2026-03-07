# PDF metadata extraction strategy

This strategy replaces inline, duplicated instructions in setup/tooling docs with one
maintained policy.

Goal:
- Determine page count for PDF files in Drive when Drive metadata does not include it.

Preferred chain (ordered):
1. Download + local `pdfinfo` parse from a temporary file.
2. HTTP range read of Drive binary (`startxref` + `xref` + `/Pages`/`/Count`) as fallback.

Important:
- Never trust fallback results unless parser says `ok`.
- Treat `error`, `ambiguous`, or `fallback_required` as non-authoritative and report those conditions.

Implementation contract used by tooling:

- `status: ok`
  - `pageCount` is concrete and can be used directly.
- `status: ambiguous`
  - PDF structure exists but was not confidently resolved.
- `status: fallback_required`
  - A non-authoritative result or missing data path suggests another path (if any) is needed.
- `status: unavailable`
  - No strategy path configured.

Default implementation hooks:

- `internal/pdfmeta`
- `Resolve` executes:
  1. PDFInfo strategy (if `FilePath` + runner are configured).
    2. Range parser strategy (if `RangeClient` is configured).
  - Returns ordered `attempts` with method, status, reason and duration.

Operational guidance:
- For runtime instructions in agent-facing docs, keep only:
  - “Resolve by download + `pdfinfo`, fallback to range parser”
  - “No non-`ok` result is authoritative”
  - “If neither strategy is available, surface `fallback_required`/`unavailable` with reasons”

Why this layout is preferable:
- DRY: one place for policy and status semantics.
- YAGNI: only production-ready two-step strategy is in default chain.
- Scalable: more resolvers can be added behind explicit config without changing command contracts.
