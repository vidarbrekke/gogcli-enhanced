# Task 1: docs edit apply-style — Strategy Evaluation

## 1. Evaluate 4 Strategies

| Criterion   | Strategy A (new subcommand) | Strategy B (invoke batch) | Strategy C (extend sed) | Strategy D (docs only) |
|------------|------------------------------|---------------------------|--------------------------|-------------------------|
| Complexity | Low: one cmd, reuse helpers  | Medium: temp file, 2 invocations | High: sed already large | None |
| DRY        | Reuses edit_helpers, BatchUpdate | Reuses batch only        | Reuses sed style builders | N/A |
| YAGNI      | One focused command          | Orchestration we may not need | Another sed mode        | No UX |
| Scalability| Add more style types later   | Tied to batch             | Tied to sed grammar      | No |

## 2. Describe Each

**Strategy A — New subcommand `docs edit apply-style`:** Add a dedicated command that takes docId, startIndex, endIndex, and style (e.g. bold, italic, heading1). Build a single BatchUpdateDocumentRequest with one UpdateTextStyleRequest or UpdateParagraphStyleRequest. Reuse AgenticEditSafetyFlags, NormalizedRequestForOutput, DryRunOutput, applyDocsEditSafety, and the existing BatchUpdate flow. Trade-off: small amount of new code but clear UX and single responsibility.

**Strategy B — Implement by invoking batch:** Generate a minimal request JSON (UpdateTextStyle or UpdateParagraphStyle), write to temp file, run `gog docs edit batch <id> --requests-file <tmp>`. Reuses batch entirely but adds indirection, temp files, and two processes. Not ideal for CLI ergonomics.

**Strategy C — Extend docs sed:** Add a "style-only" expression or mode to sed (e.g. apply style to range without replace). Reuses sed’s style-building helpers but mixes concerns and increases sed’s surface. Harder for agents to discover and use.

**Strategy D — Documentation only:** Document how to achieve apply-style via batch JSON. Zero code, but no first-class UX and worse for automation.

## 3. Compare & Choose

- **A** wins on complexity (bounded), DRY (reuse edit path), YAGNI (one command), and scalability (easy to add styles).
- **B** is more DRY in "reuse batch" but worse complexity and UX.
- **C** avoids new command but increases sed complexity and doesn’t scale as well.
- **D** doesn’t meet the "implement" requirement.

**Choice: Strategy A** — New `docs edit apply-style` subcommand reusing existing edit helpers and BatchUpdate.

## 4. Implement

See: `DocsApplyStyleCmd` in `docs_edit_cmd.go`, registered in `DocsEditCmd` in `docs_cmd.go`. Style values: bold, italic, underline, strikethrough (text); heading1..heading6, normal (paragraph).

## 5. Note

No changes to existing sed or batch behavior; additive only.
