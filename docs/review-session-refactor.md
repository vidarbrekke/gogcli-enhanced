# Session code review: objectives and refactoring options

## Scope of changes (this session)

- **Agentic/MCP foundations:** opId, requestHash, timeout/retry flags, error envelopes, contract tests.
- **Drive:** ensure-folder, untrash, permission get, path helpers.
- **Docs:** locator-based edit (anchor/marker), tests.
- **MCP server:** stdio transport, tool registry, Google tools (docs batch, drive ensure-folder/untrash/get-permission).
- **ExecuteWithIO:** command execution with injected stdout/stderr; MCP uses per-request buffers (no global stream swapping).

## Objectives: validated findings

| Area | Finding | Validated? |
|------|---------|------------|
| **normalizeError(nil, err)** | Handlers can return `(nil, err)` (e.g. writeTempJSON failure). In Go, reading from a nil map does not panic, but code explicitly dereferences `result` for Service/Operation/OpID; clearer and safer to treat nil result explicitly. | Yes – handlers return `(nil, err)` in tools.go. |
| **Concurrency** | Global stdout/stderr swapping removed; MCP uses ExecuteWithIO with buffers. No mutex added – protocol is line-by-line; no evidence concurrent tool calls are in use. | No change – already addressed by ExecuteWithIO. |
| **Pipe/deadlock** | Pipes removed from MCP path; no FD leak or deadlock risk. | No change. |
| **DRY in google/tools** | Repeated “invalid_argument” map construction (~5 similar blocks). Small duplication. | Acknowledged – minimal gain to extract; YAGNI. |
| **Performance** | No hot path or allocation issues identified; temp file per docs batch is required by CLI contract. | No change. |

## Refactoring options (Go/CLI best practices)

### Strategy 1: Minimal defensive fix only
- **Change:** Make `normalizeError` explicitly handle `result == nil` (do not read from result when nil).
- **Cognitive:** Low – one clear branch.
- **Performance:** Negligible.
- **DRY/YAGNI:** No new abstraction; fixes a real edge case.
- **Scalability:** Unchanged.
- **Verdict:** Validated, low-risk.

### Strategy 2: Fix + DRY error-map helper in google/tools
- **Change:** Strategy 1 + extract `toolError(service, operation, code, message string) map[string]any` in providers/google/tools.go.
- **Cognitive:** Slightly lower repetition.
- **DRY:** Fewer repeated maps.
- **YAGNI:** Repetition is small (~5 lines × 5); abstraction adds indirection.
- **Verdict:** Optional; slight over-engineering for current size.

### Strategy 3: Fix + MCP single-flight mutex
- **Change:** Strategy 1 + mutex in transport so only one `tools/call` runs at a time.
- **Cognitive:** Extra concurrency concept.
- **Performance:** Serializes tool calls.
- **YAGNI:** No evidence of concurrent tool calls; ExecuteWithIO already isolates output.
- **Verdict:** Unnecessary.

### Strategy 4: Broader refactor (context propagation, shared envelope types)
- **Change:** Thread context through executor, unify envelope types across cmd/mcp, etc.
- **Cognitive:** High.
- **YAGNI/Scalability:** Large change for no validated requirement.
- **Verdict:** Over-engineering.

## Recommendation and fix

- **Choose:** Strategy 1 only.
- **Reason:** Single validated improvement (nil-result handling in normalizeError); no over-engineering; no change to behavior beyond making one edge path robust and explicit.

## Validation

- normalizeError is called with `result, err` from `spec.Handler(ctx, input)`.
- Handlers in google/tools.go can return `(nil, err)` (e.g. `return nil, err` after writeTempJSON).
- Go does not panic on read from nil map, but explicitly handling nil avoids reliance on that and keeps the envelope consistent when result is nil.
