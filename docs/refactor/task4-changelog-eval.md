# Task 4: Cross-service / CHANGELOG — Strategy Evaluation

## 1. Evaluate 4 Strategies

| Criterion   | Strategy A (CHANGELOG + README) | Strategy B (JSON error hardening) | Strategy C (shared envelope) | Strategy D (no change) |
|------------|----------------------------------|------------------------------------|------------------------------|------------------------|
| Complexity | Low: doc only                    | Medium: touch many paths           | High: refactor                | None |
| DRY        | N/A                              | Could centralize                   | Shared type                   | N/A |
| YAGNI      | Document what we shipped         | Only if bugs exist                 | Only if inconsistency found   | Fails "implement" |
| Scalability| Doc scales                       | Hardening helps later              | One envelope format           | No |

## 2. Describe Each

**Strategy A — CHANGELOG and README:** Add CHANGELOG entries for docs edit apply-style, docs extract-data, and Slides batch documentation. Add a one-line mention in README for the new Docs commands. No code or behavior change. Trade-off: clear record of changes; users see new features.

**Strategy B — JSON error hardening:** Audit all edit commands for consistent error_code and envelope shape; add missing fields. Risk: "don't fix what isn't broken" — existing EditError already provides structured JSON; only add if we find a real gap.

**Strategy C — Shared JSON envelope type:** Introduce a single response envelope type used by all services. Large refactor; not YAGNI for this task.

**Strategy D — No change:** Fails "implement" requirement.

## 3. Compare & Choose

- **A** satisfies "implement" with minimal, safe change and no risk to existing behavior.
- **B** and **C** are only justified if we have evidence of broken or inconsistent JSON; we do not.

**Choice: Strategy A** — CHANGELOG entries for apply-style, extract-data, and Slides batch doc; brief README update.

## 4. Implement

Update CHANGELOG.md (new entries at top or under Unreleased). Update README.md to mention `docs edit apply-style` and `docs extract-data` where Docs editing is described.

## 5. Note

No changes to existing error handling or response shapes.
