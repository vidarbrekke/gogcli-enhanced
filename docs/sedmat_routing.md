# Sedmat routing matrix and risk taxonomy

Used by `docs.smartEdit` MCP tool to decide between auto-execute (low risk) and require plan/confirmation (high risk). Policy: **balanced (B)** — auto-execute low-risk edits; require validate/plan path for high-impact or destructive operations.

## Risk levels

| Level   | Description | Router behavior |
|--------|-------------|------------------|
| **low**   | Single plain replace/append/prepend; no structural or destructive ops. | Auto-execute (unless `validateOnly=true`). |
| **medium**| Regex or multi-line scope; single structural op (e.g. one table cell). | Auto-execute if `riskTolerance` allows; else return assessment and recommend plan. |
| **high**  | Delete commands, clear-document, table/image delete, merge/split, or multiple structural expressions. | Never auto-execute; return assessment with `requiresConfirmation: true` and recommend `docs.planBatch` / `docs.executeBatch` or explicit `docs.sed` with `dryRun` first. |

## Classification rules (expression-based)

- **High risk**
  - Any `d/pattern/` (delete) command.
  - `s/^$//` or equivalent (clear entire document).
  - Table delete: pattern/replacement that targets table with empty replacement (e.g. `s/|1|//`, `s/|*|//`).
  - Image replace with empty replacement (e.g. `s/!(1)//`).
  - Merge/split/unmerge in replacement (literal `merge`, `unmerge`, `split` in cell context).
  - Row/column insert or delete in tables (structural).
  - Multiple expressions where any is high-risk, or count &gt; 3 with any structural.
- **Medium risk**
  - Single `s/pattern/replacement/` with regex metacharacters in pattern (e.g. `.*`, `\d+`, `[a-z]`).
  - Single `a/` or `i/` (append/insert) that could match many lines.
  - Single table-cell or image replacement (non-delete).
  - Single table creation (e.g. `|2x3|`).
- **Low risk**
  - Single `s/literal/replacement/` or `s/literal/replacement/g` with no table/cell/image refs and no destructive repl.
  - Single positional: `s/^/prepend/` or `s/$/append/` with plain text.

## Intent types (smartEdit)

- `replace_all` — map to `docs.replaceAllText` or single sed `s/find/replace/g` (risk by expression).
- `append` — map to `docs.appendText` or sed `s/$/text/` (low).
- `insert` — map to `docs.insertText` or sed (risk by expression).
- `batch` — structured Docs API request; route to `docs.planBatch` / `docs.executeBatch` (existing flow).
- `sed` — explicit sed expressions; route to `docs.sed` (risk by classification above).

## Output envelope (smartEdit)

- `engineSelected`: `"batch"` | `"sed"`.
- `decisionReason`: short string (e.g. "single plain replace", "delete command requires confirmation").
- `riskLevel`: `"low"` | `"medium"` | `"high"`.
- `requiresConfirmation`: true when risk is high and validateOnly was not already requested.
