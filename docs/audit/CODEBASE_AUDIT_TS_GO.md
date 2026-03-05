# TypeScript + Go Codebase Audit

**Scope:** gogcli-enhanced (Go CLI + MCP server; TypeScript Cloudflare Worker for email tracking).  
**Review areas:** TypeScript (type safety, async, modules, validation), Go (idioms, errors, concurrency, packages), Cross-language (API contracts, errors, config, security, CI).

---

## 1. Findings

### TypeScript (internal/tracking/worker)

| Area | Finding | Severity |
|------|---------|----------|
| **Type safety** | Single unsafe cast: `(request as Request & { cf?: CfGeo }).cf` in `index.ts` L55. Cloudflare Workers extend `Request` with `cf`; types package may not expose it. | Low |
| **Runtime validation** | No runtime validation of `PixelPayload` after `JSON.parse(text)` in `crypto.ts`; malformed blob could yield missing `r`/`s`/`t` and downstream DB/query errors. | Medium |
| **Async patterns** | Consistent async/await; no unhandled rejections in handlers. Top-level `catch` in fetch returns 500 with no structured error code. | Low |
| **Module boundaries** | Clean: types.ts, crypto.ts, bot.ts, pixel.ts, index.ts; no cycles. Worker is self-contained. | OK |
| **Error handling** | `handlePixel` swallows decrypt errors and still returns pixel (by design); `handleQuery` returns 400 "Invalid tracking ID"; `handleAdminOpens` returns 401 for bad auth. No shared error envelope (e.g. `{ error: { code, message } }`) for JSON responses. | Low |
| **SQL** | All D1 queries use `.bind(...)` with parameterized values; `handleAdminOpens` builds WHERE with string concatenation but pushes only user input into `params` and binds them. No SQL injection. | OK |
| **Config** | `Env` (DB, TRACKING_KEY, ADMIN_KEY) is typed; no validation that keys are non-empty before use. | Low |

### Go

| Area | Finding | Severity |
|------|---------|----------|
| **Idioms** | Generally idiomatic. Some handlers use `map[string]any` for tool results; consistent with MCP envelope. | OK |
| **Error handling** | EditError and MCP server.ErrorCode used; some provider paths return `errors.New("...")` (err113 in lint). No change recommended without product approval. | Known (lint) |
| **Concurrency** | `transport_stdio.go`: goroutine correctly receives `req` by value `(req)`; semaphore and writeCh lifecycle (close after wg.Wait()) are correct. Context passed to handleRPC; no leak. | OK |
| **Context** | Context passed through ServeStdio → handleRPC → ExecuteTool; exec.CommandContext(ctx, ...) in mcp.go. No cancellation propagated to long-running tool runs (by design: subprocess owns lifecycle). | OK |
| **Package structure** | `internal/cmd` is large but organized by domain (docs_*, sheets_*, gmail_*, etc.). MCP under internal/mcp with server, transport, providers. No circular deps observed. | OK |
| **Debug logging** | `mcpDebugLog` opens file on every call and does not defer close on error path (only after write). File handle could leak on write failure. | Low |
| **TOML config** | `tracking/replaceTomlString` injects `value` into TOML without escaping quotes; if `workerName`/`dbName`/`dbID` contain `"`, output TOML can be invalid or inject keys. | Medium |

### Cross-language (Go ↔ Worker)

| Area | Finding | Severity |
|------|---------|----------|
| **Data contracts** | Go `gmail_track_opens.go` decodes worker JSON into structs; field names match worker (`tracking_id`, `opens`, `first_human_open`, etc.). No shared schema (OpenAPI). | Low |
| **Error format** | Worker returns HTTP status + body string (e.g. "Invalid tracking ID"); Go parses 401/500 and surfaces message. No shared `error_code` in JSON body for 4xx/5xx. | Low |
| **Serialization** | Worker uses `Response.json()`; Go uses `json.Unmarshal`. Both sides use snake_case. No validation that worker payload matches Go structs. | Low |
| **Config** | Worker secrets (TRACKING_KEY, ADMIN_KEY) from Wrangler; Go reads WorkerURL + AdminKey from keyring/config. Same keys used for encrypt (Go) and decrypt (worker)—documented in tracking flow. | OK |
| **CI** | `make ci` runs pnpm-gate (--if-present), fmt-check, lint, test. `worker-ci` runs pnpm -C internal/tracking/worker lint/build/test. Worker not exercised in main CI path unless pnpm-gate picks it up (root package.json has no scripts). | Medium |

---

## 2. Code Fixes

### Fix 1 — Worker: Validate PixelPayload after decrypt (TypeScript)

**Rationale:** Ensures required fields exist before DB/query use; avoids opaque failures and improves error messages.

```ts
// crypto.ts - after JSON.parse(text)
export async function decrypt(blob: string, key: CryptoKey): Promise<PixelPayload> {
  // ... existing decrypt ...
  const parsed = JSON.parse(text) as Record<string, unknown>;
  if (typeof parsed.r !== 'string' || typeof parsed.s !== 'string' || typeof parsed.t !== 'number') {
    throw new Error('invalid payload: missing r, s, or t');
  }
  return { r: parsed.r, s: parsed.s, t: parsed.t };
}
```

### Fix 2 — Go: Close debug log file on error path (avoid leak)

**Rationale:** Ensures file handle is closed when write fails; reduces resource leak under debug.

```go
// internal/mcp/transport_stdio.go - mcpDebugLog
f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
if err != nil {
	return
}
defer f.Close()
line := time.Now().UTC().Format(time.RFC3339) + " " + method
// ... rest unchanged
```

### Fix 3 — Go: Escape double-quotes in TOML value (tracking deploy)

**Rationale:** Prevents invalid or injective TOML when worker/db names contain `"`.

```go
// internal/tracking/deploy.go - add helper and use in replaceTomlString
func escapeTomlString(s string) string {
	return strings.ReplaceAll(s, "\\", "\\\\")
}

// In replaceTomlString, use escapeTomlString(value) in the replacement.
re := regexp.MustCompile(fmt.Sprintf(`(?m)^%s\s*=\s*\"[^\"]*\"\s*$`, regexp.QuoteMeta(key)))
return re.ReplaceAllString(content, fmt.Sprintf(`%s = %q`, key, escapeTomlString(value)))
```
Note: Using `%q` in fmt.Sprintf already escapes the value for a double-quoted Go string, which is valid for TOML. So: `fmt.Sprintf("%s = %q", key, value)` with no extra regex capture.

---

## 3. Action Plan

### Quick wins
- **Formatting / lint:** Run `make fmt` and address remaining golangci-lint issues (err113, dupl, etc.) in a single pass; add `worker-ci` to `make ci` or document that full stack CI runs `make worker-ci` separately.  
  *Impact:* Consistent style; fewer CI surprises. *Risk:* Low.
- **Worker payload validation (Fix 1):** Add the 3-line validation in `crypto.ts` after parse.  
  *Impact:* Clearer errors; safer against malformed blobs. *Risk:* Low.
- **MCP debug log close (Fix 2):** Add `defer f.Close()` in `mcpDebugLog`.  
  *Impact:* No handle leak when debug log is enabled. *Risk:* Low.

### Medium improvements
- **TOML escaping (Fix 3):** Use `%q` (or explicit escape) in `replaceTomlString` so worker/db names with quotes don’t break deploy.  
  *Impact:* Safer deploy for non-alphanumeric names. *Risk:* Low.
- **Worker error envelope:** Define a small JSON shape for 4xx/5xx (e.g. `{ error: { code, message } }`) and use it in handleQuery/handleAdminOpens/top-level catch; Go can optionally decode it for richer errors.  
  *Impact:* Consistent API; better agent/CLI error handling. *Risk:* Low (additive).
- **TypeScript:** Add ESLint + typescript-eslint with strict recommended; run in worker `pnpm lint` next to `tsc --noEmit`.  
  *Impact:* Fewer type/quality issues. *Risk:* Low.

### Major refactors (defer unless needed)
- **OpenAPI/schema for Go–Worker API:** Only if more clients or stricter contracts are required.
- **Splitting internal/cmd:** Large but navigable; split only if ownership or build times demand it.
- **Propagating context cancellation into MCP tool runs:** Would require subprocess signalling; current design is acceptable.

---

## 4. Tooling Suggestions

### TypeScript
- **eslint** + **typescript-eslint** (strict): Already using `tsc --noEmit`; add ESLint for style and rules (no-explicit-any, no-floating-promises).
- **prettier:** Optional; single-formatter for worker and any future TS.
- **Vitest:** Already in use; keep.
- **Dependency cycles:** Worker has none; if more TS packages are added, use **madge** (e.g. `madge --circular src/`).

### Go
- **gofmt / goimports / gofumpt:** In use via `make fmt`.
- **golangci-lint:** In use; consider enabling **err113** (or a subset) and **dupl** with a short allow-list.
- **staticcheck:** Add to CI for extra checks.
- **govulncheck:** Run periodically (e.g. in CI or release).
- **gosec:** Already referenced in lint config; keep.
- **go test -race:** Run for mcp/transport and cmd tests periodically or in CI.

### Cross-language
- **CI:** Unify so that either `make ci` runs worker lint/build/test when present, or document and run `make worker-ci` in the same pipeline.
- **Contract tests:** Optional: small test that starts worker (or mocks), calls /q/ and /opens, and asserts JSON shape and status codes.

---

## 5. Additional Focus

- **DRY / YAGNI:** Worker is small and DRY. Go MCP provider has some repeated patterns (e.g. build args + runCLI); acceptable unless more tools amplify duplication.
- **Abstractions:** Edit helpers and MCP envelope are well-scoped; no over-abstraction.
- **Performance:** No hot spots identified; MCP transport uses bounded concurrency and buffered channel.
- **Concurrency:** Stdio transport and goroutine usage are correct; no races identified.
- **Config:** Go keyring and worker Env are separate; no cross-stack config drift beyond doc.
- **Observability:** MCP has optional debug log; worker uses console.error. For production, consider request IDs and structured logs (both sides).

**Summary:** Codebase is in good shape. Highest-value, low-risk improvements: validate Worker payload after decrypt, fix MCP debug log close, escape TOML in deploy, and align CI with worker (and optionally add ESLint + worker error envelope).
