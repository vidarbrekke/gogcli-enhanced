# Task 2: docs extract-data — Strategy Evaluation

## 1. Evaluate 4 Strategies

| Criterion   | Strategy A (one cmd, modes) | Strategy B (3 subcommands) | Strategy C (batch in one read) | Strategy D (MCP only) |
|------------|------------------------------|----------------------------|---------------------------------|------------------------|
| Complexity | Low: one command, one fetch   | Medium: 3 entries, 3 runs  | Low: single doc read            | No CLI |
| DRY        | Reuse docsHeadingRanges, walk | Reuse same helpers         | Single walk, emit all           | N/A |
| YAGNI      | One entry point, --sections  | More surface than needed   | One code path                   | Fails "implement" |
| Scalability| Add section types later      | Add subcommands            | Add keys to output              | No |

## 2. Describe Each

**Strategy A — One command with mode/sections:** Add `gog docs extract-data <docId>` with optional `--sections outline|tables|links|all` (default all). Single Documents.Get, then walk Body.Content once; populate outline from existing docsHeadingRanges, tables from Table elements (reuse or mirror collectAllTables pattern), links from TextRun.Link. Output one JSON object with outline, tables, links. Trade-off: one code path, clear UX, easy to extend.

**Strategy B — Three subcommands:** `docs extract outline`, `docs extract tables`, `docs extract links`. Each does its own Get and walk. More discoverable but three doc fetches if user wants everything; more registration and duplication.

**Strategy C — Single batch read, always emit all:** Same as A but no --sections; always return outline + tables + links. Simplest API; if we add more section types later we extend the payload. YAGNI-friendly.

**Strategy D — MCP tool only:** Expose extract via MCP only, no CLI. Fails the requirement to "implement" with runnable code for the platform; CLI is primary.

## 3. Compare & Choose

- **A** balances flexibility (--sections) with single fetch and one code path; scales by adding section types.
- **B** duplicates fetch/walk and surface area.
- **C** is slightly simpler than A; we choose A to allow future filtering without breaking changes.
- **D** is out of scope.

**Choice: Strategy A** — One command `docs extract-data <docId>` with optional `--sections outline|tables|links|all` (default all), single fetch, reuse docsHeadingRanges and existing doc-walk patterns.

## 4. Implement

See: `DocsExtractDataCmd` under DocsCmd (read-only; no edit safety flags). Reuse docsHeadingRanges for outline; add extractTablesFromDoc and extractLinksFromDoc walking Body.Content. Output JSON: outline, tables, links.

## 5. Note

No changes to existing positions/headings or read paths; additive only.
