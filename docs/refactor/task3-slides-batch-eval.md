# Task 3: Slides batch completeness — Strategy Evaluation

## 1. Evaluate 4 Strategies

| Criterion   | Strategy A (doc in editing.md) | Strategy B (add tests) | Strategy C (add flags) | Strategy D (no change) |
|------------|--------------------------------|-------------------------|------------------------|------------------------|
| Complexity | Low: one doc section           | Medium: test coverage   | Low–medium             | None |
| DRY        | N/A                            | Reuse test patterns     | Reuse edit flags       | N/A |
| YAGNI      | Doc only                       | Tests always useful     | Only if flags missing  | Fails "implement" |
| Scalability| Doc scales with more examples  | More tests later        | More flags             | No |

## 2. Describe Each

**Strategy A — Document Slides batch in editing.md:** Add a "Google Slides editing" subsection to docs/editing.md that describes `gog slides edit batch` with --requests-file, --validate-only, --dry-run, --output-request-file, --execute-from-file. No code change. Trade-off: parity with Docs/Sheets sections; agents and users discover Slides batch in one place.

**Strategy B — Add tests for Slides batch:** Add unit or integration tests for Slides batch validate-only and dry-run paths. Improves coverage but code already exists and works; tests are optional per "don't fix what isn't broken."

**Strategy C — Add missing flags:** Audit SlidesEditBatchCmd for any missing agentic flags vs Docs/Sheets batch. If none missing, no change.

**Strategy D — No change:** Violates "implement" for the task.

## 3. Compare & Choose

- **A** delivers completeness as "documented parity" with minimal change and no risk.
- **B** is good practice but not required for "completeness" if behavior is already correct.
- **C** is redundant if flags already match (they do: Safety embed, RequestHash, NormalizedRequestForOutput).
- **D** is out.

**Choice: Strategy A** — Add Slides batch subsection to docs/editing.md so Slides batch is documented and parity with Docs/Sheets is explicit.

## 4. Implement

Add "Google Slides editing" section to docs/editing.md: prerequisites, batch command, safety flags, example.

## 5. Note

No code or test changes; documentation only.
