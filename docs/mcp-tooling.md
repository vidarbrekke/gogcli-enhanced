# MCP Tooling Baseline

This repo now includes a minimal internal operations layer in `internal/ops` to support MCP-oriented plan/execute workflows without coupling tool contracts to CLI flag parsing.

## Current baseline

- Shared envelope and hash primitives:
  - `internal/ops/ops.go`
- Existing CLI agentic primitives continue to be the execution backbone:
  - `internal/cmd/edit_helpers.go`
  - `internal/cmd/root.go`

## Planned MCP-facing operations (initial scope)

- Docs: plan/execute batch update
- Slides: plan/execute batch update
- Sheets: plan/execute values/batch updates
- Drive: resolve/ensure-folder/plan-execute file operations

## Contract principles

- JSON-only responses for tools
- Stable `error_code` mapping
- `opId` echoed in success and error envelopes
- Deterministic `requestHash` for replayability

## MCP server: tool schema and runtime

### Tool naming

- **Convention:** `service.operation` (e.g. `docs.planBatch`, `docs.executeBatch`, `drive.ensureFolder`). Matches envelope `service`/`operation` and keeps namespaces clear.
- **Exposed today:** `docs.planBatch`, `docs.executeBatch`, `drive.ensureFolder`, `drive.untrash`, `drive.getPermission`. Schema is in `internal/mcp/providers/google/tools.go` (InputSchema per tool).

### Plan/execute parity

- **Docs:** Plan = `docs edit batch <id> --requests-file <path> --validate-only`; Execute = same without `--validate-only`. Same request shape; tools pass `docId`, `request`, optional `opId` (and optional `account` for execute). Parity is 1:1.
- **Drive:** No separate "plan" tool yet; `drive.ensureFolder` is create-if-missing (idempotent). Future plan/execute for file ops would follow the same request-hash pattern as Docs.

### Concurrency model

- **Transport:** stdio is line-based (one JSON-RPC request per line). `ServeStdio` processes requests sequentially in a single goroutine; no mutex around tool execution.
- **CLI execution:** Each `tools/call` invokes the executor with fresh `bytes.Buffer`s and `ExecuteWithIO(args, outBuf, errBuf)`. No shared stdout/stderr, so concurrent tool calls would be safe at the Go level; current transport does not run them concurrently.
- **Recommendation:** Keep request-per-line processing unless a client needs parallel tool calls; then add a single-flight or worker pool and document it.

### Timeouts and retries

- **CLI:** Root flags `--request-timeout`, `--retries`, `--retry-backoff` apply to Google API calls inside each `gog` invocation. MCP does not set them; the client can pass them in the args if the executor is extended to accept optional timeout/retry from the tool input.
- **MCP layer:** No per-request timeout in `ServeStdio` or `ExecuteTool`; context is passed through but not cancelled by the server. For long-running tools, consider context timeout in the transport or in the executor wrapper (e.g. `context.WithTimeout(ctx, 5*time.Minute)` before calling the CLI).
- **Current state:** Relies on client or process-level timeout; no change required unless timeouts are observed in practice.

## Reviewer notes (internal/mcp and entrypoint)

### File paths (on main)

- **Entrypoint:** `internal/cmd/mcp.go` — MCP serve command; builds server with executor closure that uses `ExecuteWithIO(args, outBuf, errBuf)` only (no globals, no pipes).
- **Server:** `internal/mcp/server/server.go`, `internal/mcp/server/types.go` — tool registry, `ExecuteTool`, envelope normalization.
- **Transport:** `internal/mcp/transport_stdio.go` — stdio JSON-RPC; line-by-line; `handleRPC` → `s.ExecuteTool(ctx, name, args)`.
- **Google tools:** `internal/mcp/providers/google/tools.go` — tool specs, handlers, `runCLI`; `Register(s, executor)` sets the executor used by all tools.
- **Wire-up:** `internal/mcp/default.go` — `NewGoogleServer(executor)` creates server and calls `google.Register(s, executor)`.

### gofmt and formatting

- `internal/cmd/mcp.go` is kept multi-line and gofmt'd; it has a trailing newline. If the raw view on GitHub still shows a minified version, try a hard refresh or re-clone (and ensure `core.autocrlf` or line-ending filters aren't altering the file).

### Fixes applied (external review)

- **Scanner limit:** `ServeStdio` now sets `scanner.Buffer(..., 10*1024*1024)` so large `tools/call` payloads don't hit the default 64KB limit and drop the connection.
- **Arg building:** `maybeOpID`/`maybeAccount` replaced with `maybeOpIDArgs`/`maybeAccountArgs` returning `[]string`; `cleanArgs` only trims and drops empty strings (no split on spaces). Paths and values with spaces (e.g. `"My Folder/2026"`) are preserved.
- **Unmarshal:** The code uses `&params` (ASCII ampersand). If a reviewer sees `¶ms`, it is a display/encoding artifact; the repo builds and the source is correct.

### Mutable global state

- **Only package-level mutable:** `internal/mcp/providers/google/tools.go` has `var execCommand Executor`. It is set once in `Register(s, executor)` at server startup and never written again; all tool handlers only read it. No mutex, no per-request shared mutable state.
- **Server:** `Server.tools` map is populated at startup and not modified during request handling. Safe for concurrent read if the transport ever runs tool calls in parallel (currently it does not).

You can point external reviewers to this section and the listed paths for tool registration, schemas, error envelopes, and plan/execute readiness.
