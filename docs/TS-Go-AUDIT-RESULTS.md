# TypeScript + Go Codebase Audit — Results

This document is the outcome of reviewing the repo against the scope in `TS-Go-review.md`. The codebase is **predominantly Go** (CLI + MCP provider); **TypeScript** exists only in `internal/tracking/worker` (Cloudflare Worker). There is no Next.js or Firebase app; no shared API contracts between TS and Go beyond deploy/config.

---

## 1. Findings

### TypeScript (`internal/tracking/worker`)

| Area | Finding | Risk |
|------|---------|------|
| **Weak typing** | `(request as any).cf` and `row: any` in D1 result maps; `params: any[]`. | Medium: runtime shape not enforced; refactors can miss field renames. |
| **Async** | Consistent `async/await`; no mixed callbacks. | None. |
| **Module boundaries** | Clear split: `types.ts`, `crypto.ts`, `bot.ts`, `pixel.ts`, `index.ts`; no cycles. | None. |
| **Runtime vs compile-time** | `decrypt()` returns `PixelPayload` but `JSON.parse(text)` is not validated at runtime. | Low: payload is internal (encrypted by us). |
| **DRY** | D1 row→response mapping duplicated between query and admin handlers. | Low: only two call sites. |

### Go (representative areas)

| Area | Finding | Risk |
|------|---------|------|
| **Idiomatic / errors** | `golangci-lint` reports err113 (dynamic errors), wrapcheck (unwrapped external errors), predeclared (`cap`, `max`), prealloc, dupl. | Low–medium: existing debt; no critical correctness issues. |
| **Concurrency** | MCP transport uses channels and goroutines; no obvious leaks. Race detector not run in default `make test`. | Low: run `go test -race` periodically. |
| **Context** | Handlers receive `context.Context`; propagation in CLI/MCP is present. | None identified. |
| **Package layout** | Large `internal/mcp/providers/google/tools.go` recently split into domain files (docs/sheets/slides/drive); `internal/cmd` is large but domain-grouped. | Addressed by Phase 1 split. |
| **Testability** | Unit tests next to code; MCP tests in `internal/mcp`; integration behind build tag. | Adequate. |

### Cross-language

| Area | Finding | Risk |
|------|---------|------|
| **API / data contracts** | No shared TS↔Go API; worker uses D1 and env bindings; Go CLI talks to Google APIs and MCP. | N/A. |
| **Serialization** | Worker: JSON for pixel payload and query response. Go: JSON for MCP and CLI `--json`. No shared schema repo. | Low. |
| **Errors / logging** | Worker: `console.error`; Go: stderr and structured error envelopes in MCP. No unified tracing. | Low for current scope. |

---

## 2. Code Fixes Applied

### TypeScript (tracking worker)

- **`request.cf`**  
  Replaced `(request as any).cf` with a typed access: introduced `CfGeo` (country, region, city, timezone) and `(request as Request & { cf?: CfGeo }).cf ?? {}`.  
  **Rationale:** Removes `any` and documents the cf shape we use; avoids accidental use of other cf fields without types.

- **D1 result rows**  
  Introduced `OpenQueryRow` and `OpenAdminRow` in `types.ts` and used them in `.all<OpenQueryRow>()` / `.all<OpenAdminRow>()` and in the `.map()` callbacks.  
  **Rationale:** Typed row access catches field renames and wrong SELECT columns at compile time.

- **`params` in admin query**  
  Changed `const params: any[]` to `const params: string[]` and `params.push(String(limit))`.  
  **Rationale:** D1 `.bind(...)` expects string values for placeholders; makes the type explicit and avoids accidental non-string params.

No behavior changes; lint (`tsc --noEmit`) and tests (`pnpm test`) pass.

---

## 3. Action Plan

### Quick wins

| Item | Action | Impact | Risk |
|------|--------|--------|------|
| Worker CI | Ensure `make worker-ci` (or `pnpm -C internal/tracking/worker lint/build/test`) runs in CI so TS is gated. | Prevents regressions in worker. | None. |
| Go race tests | Add optional `make test-race` that runs `go test -race ./...` (or key packages). | Surfaces data races. | Low (may expose flaky tests). |
| Lint baseline | Document known golangci-lint findings (err113, dupl, wsl_v5, etc.) in `docs/` or ignore files where fixing is out of scope. | Clear expectations; avoids "fix everything" churn. | None. |

### Medium improvements

| Item | Action | Impact | Risk |
|------|--------|--------|------|
| Runtime validation (worker) | Optionally validate decrypted pixel payload (e.g. `r` string, `t` number) before use. | Safer if payload format ever changes or is reused. | Low. |
| Error handling (Go) | Gradually replace dynamic `errors.New("...")` in MCP handlers with sentinel errors where err113 flags. | Consistent, comparable errors. | Medium (touch many call sites). |
| Worker ESLint | Add `eslint` + `typescript-eslint` to the worker with `strict`-friendly rules. | Catches more TS issues. | Low. |

### Major refactors (defer)

| Item | Notes | Risk |
|------|--------|------|
| API contract / OpenAPI | Only if a formal TS↔Go API appears. | N/A today. |
| Shared DTO/schema repo | Only if multiple services share the same JSON contracts. | YAGNI for current layout. |
| Observability | Structured logging/tracing across Go and worker is a broader initiative. | Scope creep. |

---

## 4. Tooling Suggestions

### TypeScript (worker)

| Tool | Status | Suggestion |
|------|--------|------------|
| `tsc --noEmit` | In use (`pnpm lint`) | Keep. |
| eslint | Not in use | Add `eslint`, `@typescript-eslint/eslint-plugin`, `@typescript-eslint/parser` with recommended + no-`any` where feasible. |
| prettier | Not in use | Optional; `tsc` and manual style are acceptable for small surface. |
| dependency cycles | N/A | Worker has no TS dependencies; no madge needed. |

### Go

| Tool | Status | Suggestion |
|------|--------|------------|
| gofmt / goimports / gofumpt | In use (`make fmt`) | Keep. |
| golangci-lint | In use (`make lint`) | Keep; consider narrowing or excluding rules that are intentionally deferred. |
| staticcheck | Often included in golangci-lint | No change. |
| govulncheck | Not in use | Add `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` in CI or periodically. |
| gosec | In use (via golangci-lint) | Keep. |
| `go test -race` | Not in default gate | Add as optional target or nightly. |

### Cross-language

| Item | Suggestion |
|------|------------|
| CI | Run both `make ci` (Go) and `make worker-ci` (TS) so both stacks are checked. |
| Contract tests | Not required unless a formal API exists between TS and Go. |

---

## 5. Additional Focus (from TS-Go-review.md)

- **DRY / YAGNI:** Worker is small; no extra abstraction needed. Go MCP domain split (Phase 1) improved DRY and file size.
- **Concurrency:** Go MCP transport is structured; add `-race` to tests for confidence.
- **Configuration:** Worker uses env bindings (TRACKING_KEY, ADMIN_KEY, DB); Go uses config + keyring. No inconsistency.
- **Observability:** Worker logs with `console.error`; Go uses stderr and MCP error envelopes. Unified tracing would be a separate project.

---

**Summary:** TypeScript weak typing in the tracking worker was addressed with minimal, type-only changes. Go findings are documented as known lint debt and optional improvements. No architectural or cross-language changes were recommended for the current codebase.
