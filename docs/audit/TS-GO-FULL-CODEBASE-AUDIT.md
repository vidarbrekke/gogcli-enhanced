# TypeScript + Go Full Codebase Audit

**Scope:** TypeScript (tracking worker, scripts), Go (CLI + internal packages), cross-language contracts and tooling.

**Focus:** Type safety, correctness, idiomatic patterns, error handling, concurrency, package/module boundaries, API contracts, security, build/deployment, DRY/YAGNI, observability.

---

## 1. Findings

### 1.1 TypeScript

#### 1.1.1 internal/tracking/worker

| Area | Finding | Risk |
|------|---------|------|
| **Type safety** | No `any` in application code. `crypto.ts` uses `Record<string, unknown>` then runtime checks for `r`, `s`, `t`. `strict: true` in tsconfig. | Low |
| **Unsafe cast** | `(request as Request & { cf?: CfGeo }).cf` in `index.ts` — intentional Cloudflare binding; comment present. Acceptable. | Low |
| **Async** | Single `async fetch` entry; handlers async and awaited. No fire-and-forget. | Low |
| **Module boundaries** | Clear: `types` → `crypto`/`bot`/`pixel` → `index`. No cycles. | Low |
| **Runtime validation** | `decrypt()` validates payload shape after `JSON.parse`; throws on invalid. No Zod/AJV; sufficient for current surface. | Low |
| **Admin limit** | `parseInt(url.searchParams.get('limit') || '100', 10)` — no clamp; `NaN` or negative could reach D1. Default 100 is safe; if `limit` is ever user-controlled, cap it (e.g. 1–1000). | **Medium** (defense in depth) |
| **D1 row types** | `OpenRecord.id: number` — D1 may return bigint for auto-increment; runtime type gap if IDs grow large. | Low |
| **Tests** | `crypto.test.ts`, `bot.test.ts` cover main paths. No handler/integration tests (would need D1 mocks). | Medium (optional) |

#### 1.1.2 scripts/maton-capabilities-capture.mjs

| Area | Finding | Risk |
|------|---------|------|
| **Typing** | Plain ESM; no TypeScript. No type safety. | Low (script, not production service) |
| **Error handling** | `catch (e) { console.error(e); process.exit(1); }` then in loop `catch (e) { console.error(name, e.message); }` — `e` may not be `Error`; `e.message` can be undefined. | **Low** (script) |

#### 1.1.3 docs/assets/site.js

| Area | Finding | Risk |
|------|---------|------|
| **Role** | Static site script; minimal. Not part of core product. | N/A |

---

### 1.2 Go

#### 1.2.1 Correctness & error handling

| Area | Finding | Risk |
|------|---------|------|
| **Tracking client** | `gmail_track_opens.go`: `url.Parse` and `http.NewRequestWithContext` errors are **handled** (already fixed). | None |
| **Idiomatic** | Errors wrapped with `fmt.Errorf("...: %w", err)`; sentinels where appropriate. `errfmt` uses `errors.As`/`errors.Is`. | Low |
| **Ignored returns** | Isolated cases in tests (e.g. `resp.Body.Close()`) or intentional; no production paths ignoring critical errors in reviewed files. | Low |

#### 1.2.2 Concurrency

| Area | Finding | Risk |
|------|---------|------|
| **Goroutines** | `calendar_team.go`, `gmail.go`, `gmail_messages.go`: `sync.WaitGroup` + semaphore/channel; `defer wg.Done()`. Lifecycle clear. | Low |
| **googleauth** | Server goroutines: `<-ctx.Done()` then `Close()`; `Serve` in separate goroutine. Correct shutdown. | Low |
| **secrets/store.go** | Documented: if keyring timeout occurs, spawned goroutine can outlive and block. Acceptable for CLI (process exits). | Low (documented) |
| **MCP transport** | `transport_stdio.go`: goroutines for read/write; `sem` limits concurrency; `wg.Wait()`. No obvious leak. | Low |

#### 1.2.3 context.Context

| Area | Finding | Risk |
|------|---------|------|
| **Propagation** | Commands receive `context.Context`; HTTP uses `NewRequestWithContext(ctx, ...)`. Cancellation propagates. | Low |
| **Background/TODO** | Many tests use `context.Background()` for isolated runs — appropriate. No misuse observed in production paths. | Low |

#### 1.2.4 Package structure & testability

| Area | Finding | Risk |
|------|---------|------|
| **internal/cmd** | Large but domain-split (gmail, drive, docs, calendar, etc.). Some files are long; no circular imports. | Low |
| **internal/tracking** | Focused: crypto, config, deploy. Clear. | Low |
| **internal/mcp** | Server, transport, providers. Boundaries clear. | Low |
| **Testability** | Many `*_test.go`; test helpers (`runKong`, `captureStdout`); table-driven tests. Test context/stdout pattern documented (ROOT-CAUSE-AUDIT). | Low |

#### 1.2.5 Lint & style

| Area | Finding | Risk |
|------|---------|------|
| **golangci-lint** | Strict config; exclusions for wsl_v5, err113, etc. on parts of `internal/cmd`, `internal/mcp`, etc. Documented in LINT-74-10X-PLAN. | Known technical debt |

---

### 1.3 Cross-Language

#### 1.3.1 Data contracts (Go ↔ TS worker)

| Contract | Go | TS | Status |
|----------|----|----|--------|
| **PixelPayload** | `Recipient` (json `r`), `SubjectHash` (json `s`), `SentAt` (json `t`, int64) | `r: string`, `s: string`, `t: number` | Aligned |
| **Crypto** | AES-GCM; nonce 12 bytes; nonce\|ciphertext; base64.RawURLEncoding | AES-GCM; IV 12 bytes; same layout; URL-safe base64 with padding fix | Compatible |
| **Query response** | Struct with `tracking_id`, `recipient`, `sent_at`, `opens`, `total_opens`, `human_opens`, `first_human_open` | Same keys in Response.json | Aligned |
| **Admin /opens** | Expects `opens` array with `tracking_id`, `recipient`, etc. | Returns same shape | Aligned |

#### 1.3.2 API boundaries & errors

| Area | Finding | Risk |
|------|---------|------|
| **Error format** | Worker returns 400/401/404/500 with body; Go maps non-200 to `fmt.Errorf("tracker returned %d: %s", ...)`. No shared error code enum. | Low |
| **MCP (Go)** | Tools take `map[string]any`; no JSON schema validation at boundary. Normal for dynamic tools. | Low |

#### 1.3.3 Configuration & security

| Area | Finding | Risk |
|------|---------|------|
| **Config** | Go: `tracking.json` (WorkerURL, keys). Worker: env (D1, TRACKING_KEY, ADMIN_KEY). No shared file; each side owns its config. | Low |
| **Secrets** | Worker: Bearer token for admin; TRACKING_KEY for crypto. Go: reads from config written by `gog gmail track setup`. No secrets in repo. | Low |
| **Admin query** | Worker uses parameterized D1 `.bind(...params)`; no SQL injection. | Low |

#### 1.3.4 Build & deployment

| Area | Finding | Risk |
|------|---------|------|
| **CI** | `make ci` runs `pnpm-gate`, `fmt-check`, `lint`, `test`, **worker-ci**. Worker is included. | None |
| **Worker build** | `pnpm -C internal/tracking/worker build` runs `wrangler deploy --dry-run`. Lint = `tsc --noEmit`. | Low |

---

### 1.4 DRY, YAGNI, observability

- **DRY:** Tracking payload/crypto live in one place per language; no duplicated business logic across TS/Go.
- **YAGNI:** No speculative abstractions; worker deps minimal.
- **Observability:** Worker `console.error` on failure; Go uses `ui` and structured errors. No tracing/metrics; acceptable for scope.
- **Performance:** No hot-spot analysis done; CLI and worker are request-scoped; no major concerns flagged.

---

## 2. Code Fixes

### Fix 1: Worker — validate and clamp admin `limit` (TypeScript)

**Why:** `parseInt` can return `NaN`; a large or negative value could stress D1 or produce surprising results. Clamping improves robustness and sets a clear contract.

**Change:** Validate and clamp `limit` in `handleAdminOpens`.

```ts
// internal/tracking/worker/src/index.ts — in handleAdminOpens()

  const limitParam = url.searchParams.get('limit') || '100';
  const limitRaw = parseInt(limitParam, 10);
  const limit = Number.isNaN(limitRaw) || limitRaw < 1 ? 100 : Math.min(limitRaw, 1000);
```

**Rationale:** Ensures `limit` is always 1–1000; default 100 when missing or invalid. Reduces technical debt and risk if the admin API is ever exposed to more callers.

---

### Fix 2: maton-capabilities-capture.mjs — safe error logging (JavaScript)

**Why:** In `catch (e)`, `e` may not be an `Error` (e.g. thrown string). Accessing `e.message` can be undefined and may throw again.

**Change:** Use a small guard when logging.

```js
// scripts/maton-capabilities-capture.mjs — in the for loop catch and top-level catch

    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      console.error(name, msg);
    }
```

```js
main().catch((e) => {
  const msg = e instanceof Error ? e.message : String(e);
  console.error(msg);
  process.exit(1);
});
```

**Rationale:** Improves correctness of error reporting and avoids secondary throws in catch blocks. Minimal change.

---

### Fix 3 (optional): Worker — type payload in crypto.test.ts

**Why:** Test uses object literal `{ r, s, t }`; typing it as `PixelPayload` improves consistency and catches interface drift.

**Change:** In `crypto.test.ts`:

```ts
import type { PixelPayload } from './types';
// ...
const payload: PixelPayload = { r: 'test@example.com', s: 'abc123', t: 1704067200 };
```

**Rationale:** Better type safety in tests; no behavior change.

---

## 3. Action Plan

### Quick wins

| Item | Action | Impact | Risk |
|------|--------|--------|------|
| Worker limit clamp | Apply Fix 1 in `internal/tracking/worker/src/index.ts` | Robustness of admin API | None |
| Maton script errors | Apply Fix 2 in `scripts/maton-capabilities-capture.mjs` | Safer error logging | None |
| Worker test type | Apply Fix 3 (optional) in `crypto.test.ts` | Type consistency | None |
| Format/lint | Keep `make fmt` / `make lint`; address known golangci issues in a dedicated pass (see LINT-74-10X-PLAN) | Consistency | Low |

### Medium improvements

| Item | Action | Impact | Risk |
|------|--------|--------|------|
| Worker ESLint | Add `eslint` + `typescript-eslint` in `internal/tracking/worker` (e.g. no-explicit-any, no-floating-promises) | Catches more TS issues | Low |
| Error-handling audit | Grep for ignored errors (`_, err :=` or `_ =`) in production Go code paths; fix only where correctness matters | Reduces similar bugs | Low |
| Shared API schema | If tracking API grows: add OpenAPI or JSON schema for `/q/:blob` and `/opens`; generate or document types both sides | Contract clarity | Low (only when needed) |
| Worker integration test | Optional: Miniflare + Go client test to assert status and JSON shape | Higher confidence | Medium (CI deps) |

### Major refactors

| Item | Action | Impact | Risk |
|------|--------|--------|------|
| None recommended | Codebase is coherent; no architecture or API redesign needed for current scope | — | — |

---

## 4. Tooling Suggestions

### TypeScript

| Tool | Status | Suggestion |
|------|--------|------------|
| **tsc --noEmit** | In use (`pnpm lint`) | Keep. |
| **eslint** | Not in use | Add in worker: `eslint`, `@typescript-eslint/eslint-plugin`, `@typescript-eslint/parser` with recommended + strict-friendly rules. |
| **prettier** | Not in use | Optional; single formatter for worker and future TS. |
| **typescript-eslint** | Not in use | Add with eslint; enable no-explicit-any where feasible, no-floating-promises. |
| **Dependency cycles (madge)** | Not in use | Optional; worker is small and has no cycles. |

### Go

| Tool | Status | Suggestion |
|------|--------|------------|
| **gofmt / goimports** | In use (`make fmt`) | Keep. |
| **golangci-lint** | In use (`make lint`) | Keep; consider narrowing or documenting deferred rules (e.g. wsl_v5). |
| **staticcheck** | Via golangci-lint | No change. |
| **govulncheck** | Not in Makefile | Add `make vuln` or run in CI to check dependencies. |
| **gosec** | Via golangci-lint | Keep; existing nolint for acceptable cases. |
| **go test -race** | Not default | Run occasionally (e.g. nightly or on-demand) for concurrency; optional per commit. |

### Cross-language

| Tool | Status | Suggestion |
|------|--------|------------|
| **OpenAPI / schema** | Not in use | Introduce only if tracking API grows or gets more consumers. |
| **Contract tests** | Not in use | Optional: script that runs worker + Go client and asserts status/JSON. |
| **CI** | `make ci` includes `worker-ci` | Already runs TS worker lint/build/test. No change. |

---

## 5. Summary

- **TypeScript:** Worker is small and well-typed; one robustness fix (admin limit clamp) and one script fix (maton error guard). Optional: ESLint + typescript-eslint, type in crypto test.
- **Go:** Error handling in tracking client is correct. Concurrency and context usage are appropriate; package layout is clear. Known lint debt is documented elsewhere.
- **Cross-language:** PixelPayload and crypto are aligned; API shapes match. Config and secrets are separated by side. CI runs both stacks.
- **Recommendation:** Apply Fix 1 and Fix 2; optionally Fix 3 and worker ESLint. No major refactors recommended.
