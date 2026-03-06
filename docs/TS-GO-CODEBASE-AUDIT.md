# TypeScript + Go Codebase Audit

Audit scope: **TypeScript** (tracking worker), **Go** (CLI and libraries), **cross-language** (tracking API and payloads). Focus: correctness, idiomatic patterns, type safety, error handling, concurrency, boundaries, and maintainability.

---

## 1. Findings

### 1.1 TypeScript (internal/tracking/worker)

| Area | Finding | Risk |
|------|---------|------|
| **Type safety** | No `any` or unsafe casts in application code. `crypto.ts` uses `Record<string, unknown>` then runtime checks for `r`, `s`, `t` — good. | Low |
| **Platform cast** | `(request as Request & { cf?: CfGeo }).cf` in `index.ts` is an intentional Cloudflare Workers binding cast; documented in comment. | Low |
| **Async** | Single `async fetch` entrypoint; handlers are async and consistently awaited. No fire-and-forget or mixed patterns. | Low |
| **Module boundaries** | Worker is self-contained: `types`, `crypto`, `bot`, `pixel`. No cycles; clear dependency direction. | Low |
| **Runtime validation** | `decrypt()` validates payload shape after `JSON.parse`; throws on invalid. No schema library (YAGNI: current check is sufficient). | Low |
| **Tests** | `crypto.test.ts` and `bot.test.ts` cover main paths; no tests for `index.ts` handlers (integration would require D1/mocks). | Medium (optional) |

**Summary:** TS surface is small and well-typed. No critical gaps.

---

### 1.2 Go

| Area | Finding | Risk |
|------|---------|------|
| **Error handling** | **Ignored errors in gmail_track_opens.go:** `url.Parse(cfg.WorkerURL + "/opens")` and `http.NewRequestWithContext(...)` errors discarded (`reqURL, _`, `req, _`). If `WorkerURL` is malformed or request build fails, code can panic or send wrong URL. | **Medium** |
| **Idiomatic** | Otherwise errors are wrapped with `fmt.Errorf("...: %w", err)` and returned. Sentinels used where appropriate (e.g. `errTrackingNotConfigured`). | Low |
| **Context** | Commands take `context.Context`; HTTP calls use `NewRequestWithContext(ctx, ...)`. Cancellation propagates. | Low |
| **Concurrency** | Goroutines in `gmail.go`, `gmail_messages.go`, `calendar_team.go`, `mcp/transport_stdio.go`, `googleauth` use `sync.WaitGroup` / `sync.Mutex` / `sync.Once`; no obvious leaks. `secrets/store.go` documents a goroutine that may outlive timeout. | Low–Medium (review lifecycle if adding more concurrency) |
| **Package structure** | `internal/cmd` is large but organized by domain (gmail, drive, docs, etc.). `internal/tracking` is focused (config, crypto, pixel, deploy). No circular imports observed. | Low |
| **Testability** | Many `*_test.go` files; test helpers and table-driven tests present. Some tests use `captureStdout` and context-based stdout (documented in ROOT-CAUSE-AUDIT). | Low |
| **Lint** | `.golangci.yml` is strict; 73 existing issues (e.g. wsl_v5, err113, gosec) are documented separately; not introduced by this audit. | Known |

**Summary:** One concrete correctness issue (ignored errors in track opens). Rest is solid.

---

### 1.3 Cross-Language (Go ↔ TypeScript Worker)

| Area | Finding | Risk |
|------|---------|------|
| **Data contract** | **PixelPayload** is aligned: Go `PixelPayload` (json `r`, `s`, `t`) and TS `PixelPayload` (r, s, t; t number). Go uses `int64` for `SentAt`, TS uses number (unix). Compatible. | Low |
| **Crypto** | Both use AES-GCM; Go prepends nonce, TS splits IV/ciphertext; same layout. Go uses `base64.RawURLEncoding` for blob; TS uses URL-safe base64 decode with padding fix. Key: standard base64 in both. | Low |
| **Subject hash** | Go `hashSubject` returns first 6 chars of SHA256 hex; worker expects 6-char subject hash. Consistent. | Low |
| **API contract** | **Query `/q/:blob`:** Go expects JSON with `tracking_id`, `recipient`, `sent_at`, `opens`, `total_opens`, `human_opens`, `first_human_open`. Worker returns exactly that. **Admin `/opens`:** Go expects `opens` array with `tracking_id`, `recipient`, etc.; worker returns same. No formal schema (OpenAPI); structure is stable and mirrored in Go structs. | Low |
| **Error format** | Worker returns 400/401/404/500 with plain text or JSON body. Go parses success JSON; on non-200 it returns `fmt.Errorf("tracker returned %d: %s", ...)`. No shared error code enum. | Low |
| **CI** | Main `make ci` runs `pnpm-gate` (if package.json present) and Go lint/test. Worker has its own `make worker-ci` (tsc, vitest) but **worker-ci is not part of default `ci`**. Root `package.json` has only Playwright; worker lives under `internal/tracking/worker` with its own package.json. So CI does not currently run worker lint/build/test unless `worker-ci` is invoked explicitly. | Medium |

**Summary:** Contracts are consistent and documented in code. CI could run worker checks by default.

---

### 1.4 DRY, YAGNI, Observability

- **DRY:** Tracking payload and URL building live in one place (Go: `tracking/pixel.go`, `tracking/crypto.go`). Worker does not duplicate Go logic; it consumes the blob. No unnecessary duplication found.
- **YAGNI:** No speculative abstractions. Worker uses minimal deps (Cloudflare types, Vitest for tests).
- **Observability:** Worker uses `console.error` for failures; Go uses `ui` and structured errors. No distributed tracing or metrics; acceptable for current scope.
- **Config:** Go tracking config in `tracking.json`; worker env (D1, TRACKING_KEY, ADMIN_KEY). No cross-language config file; each side has its own. Clear.

---

## 2. Code Fixes

### Fix 1: Handle URL and request build errors in gmail_track_opens.go

**Why:** Ignoring `url.Parse` and `http.NewRequestWithContext` errors can cause panics or wrong requests if `WorkerURL` is malformed or empty.

**Change:** Check errors and return them with context.

```diff
--- a/internal/cmd/gmail_track_opens.go
+++ b/internal/cmd/gmail_track_opens.go
@@ -119,8 +119,14 @@ func (c *GmailTrackOpensCmd) queryAdmin(ctx context.Context, cfg *tracking.Conf
 		return fmt.Errorf("tracking admin key not configured; run 'gog gmail track setup' again")
 	}

-	reqURL, _ := url.Parse(cfg.WorkerURL + "/opens")
+	reqURL, err := url.Parse(cfg.WorkerURL + "/opens")
+	if err != nil {
+		return fmt.Errorf("tracking worker URL invalid: %w", err)
+	}
 	q := reqURL.Query()
 	if c.To != "" {
 		q.Set("recipient", c.To)
@@ -133,8 +139,12 @@ func (c *GmailTrackOpensCmd) queryAdmin(ctx context.Context, cfg *tracking.Conf
 	}
 	reqURL.RawQuery = q.Encode()

-	req, _ := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
+	req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
+	if err != nil {
+		return fmt.Errorf("build request: %w", err)
+	}
 	req.Header.Set("Authorization", "Bearer "+cfg.AdminKey)
```

**Rationale:** Improves correctness and fail-fast behavior; reduces risk of opaque runtime failures. Minimal, idiomatic Go.

---

### Fix 2 (optional): Limit parameter in admin query

**Finding:** Worker uses `parseInt(url.searchParams.get('limit') || '100', 10)`; Go never sets `limit` so it defaults to 100. If Go later adds a limit flag, it should be validated (e.g. cap at 1000) to avoid unbounded admin responses. **No code change required now** — YAGNI; document if you add a `--limit` flag later.

---

## 3. Action Plan

### Quick Wins

| Item | Action | Impact | Risk |
|------|--------|--------|------|
| Fix ignored errors | Apply Fix 1 in `gmail_track_opens.go` | Correctness; avoids rare misbehavior on bad config | None |
| Run worker in CI | Add `worker-ci` to `ci` target in Makefile (e.g. `ci: pnpm-gate fmt-check lint test worker-ci`) so TS worker is linted and tested on every run | Catches TS regressions | Low (may add ~10–30s) |
| Format/lint | Existing `make fmt` / `make lint`; address the 73 known lint issues in a separate pass or by relaxing specific rules (e.g. wsl_v5) per AGENTS.md | Consistency | Low |

### Medium Improvements

| Item | Action | Impact | Risk |
|------|--------|--------|------|
| Error-handling consistency | Audit other Go call sites that ignore `url.Parse` or `http.NewRequestWithContext` (e.g. in tests, or other commands) and fix only production paths | Reduces similar bugs | Low |
| Shared types / schema | If the tracking API grows, consider a small OpenAPI or JSON schema for `/q/:blob` and `/opens` and generate or document types on both sides | Contract clarity | Low (only when needed) |
| Worker integration tests | Add a small integration test that runs the worker (e.g. with Miniflare) and the Go client against it; optional and only if you want stronger cross-stack guarantees | Higher confidence | Medium (extra CI deps) |

### Major Refactors

| Item | Action | Impact | Risk |
|------|--------|--------|------|
| None recommended | Codebase is coherent; no architecture change or API redesign needed for current scope | — | — |

---

## 4. Tooling Suggestions

### TypeScript

- **Current:** `tsc -p tsconfig.json --noEmit` (lint script), Vitest for tests. No ESLint in worker.
- **Suggestions:**
  - **eslint** + **typescript-eslint**: Enable in `internal/tracking/worker` to catch style and type-related issues (e.g. no-explicit-any, consistent async).
  - **tsc --noEmit**: Already used; keep.
  - **Dependency cycle detection (e.g. madge):** Optional; worker is small and has no cycles.

### Go

- **Current:** gofmt, goimports, golangci-lint, staticcheck (via golangci-lint), govulncheck not in Makefile.
- **Suggestions:**
  - **govulncheck:** Add `make vuln` or run in CI to check for known vulnerabilities in deps.
  - **go test -race:** Run occasionally (e.g. in CI once per night or on demand) for concurrency bugs; not mandatory for every commit given current concurrency surface.
  - **gosec:** Already enabled in golangci-lint; keep. Existing nolint for acceptable cases (e.g. user-configured URLs) are fine.

### Cross-Language

- **OpenAPI / schema:** Only introduce when you add more endpoints or consumers; current contract is small and stable.
- **Contract tests:** Optional: script that builds worker, starts it, and calls from Go (or vice versa) to assert status and JSON shape. Not required for current maturity.
- **CI:** Run `worker-ci` as part of `ci` (see Quick Wins) so both stacks are validated on every run.

---

## 5. Summary

- **TypeScript:** Small, well-typed worker; no critical issues. Optional: add ESLint and include worker in main CI.
- **Go:** One concrete fix: handle `url.Parse` and `http.NewRequestWithContext` errors in `gmail_track_opens.go`. Rest is idiomatic and consistent.
- **Cross-language:** Payload and API contracts align; crypto and subject hash are consistent. Main gap is CI not running worker tests by default.
- **Recommendation:** Apply Fix 1, add `worker-ci` to `ci`, then address remaining lint in a dedicated pass. No major refactors recommended; avoid over-engineering.
