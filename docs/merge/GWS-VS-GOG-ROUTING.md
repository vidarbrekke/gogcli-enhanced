# gws vs gog: routing logic and how to implement/test

## 1. Logic: when gws vs gog?

| Context | What runs | Role of gws |
|--------|------------|-------------|
| **Agent / MCP request path (default)** | **gog only** | gws is not in the path. Every tool call (Drive, Gmail, Docs, etc.) is handled by the gog binary and native Google APIs. |
| **Parity / CI** | **Fixtures only** | gws is a fixture provider. The parity runner loads fixtures and compares to native goldens (and schemas). No live API calls in the runner. |
| **Agent / MCP with GOG_BACKEND=gws** | **gog** invokes **gws** for selected Tier A commands | When GOG_BACKEND=gws, gog runs the gws CLI for Tier A commands that support it (e.g. gmail labels list/get), captures stdout/stderr/exit, normalizes with parity logic, returns result. Agent still talks only to gog-agentic. |

So:

- **Use gog + MCP for all live agent actions** — default and as the control plane. Determinism, stable contracts, and safety live in gog.
- **Use gws only in parity** — to produce fixtures and validate that, when normalized, gws meets the same contract. When GOG_BACKEND=gws, gog can also invoke gws for Tier A commands that support it (e.g. Gmail labels list/get); gog normalizes and returns. Tier C (writes, safety-critical) stays native gog.
- **gws + MCP** in the sense of "agent calls an MCP server that runs gws" is not the design. The design is a single MCP (gog-agentic); gog may call gws as a backend for some Tier A reads. The agent still talks only to gog-agentic.

Reference: `handover.md` (§1 pivot, §2.1 glossary), `docs/merge/command-migration-matrix.md` (Tier A/B/C, rollout stages).

---

## 2. How to implement (extend live gws routing)

Prerequisites (in place):

- Parity runner that loads fixtures, classifies, normalizes, and diffs (`cmd/gog-parity`, `internal/parity/*`).
- Goldens for Tier A commands we care about (e.g. Gmail labels 401, 404, list success), with both `native` and `gws` fixtures.
- Schemas and drift policy so we don’t gate on `google_reason` / message text.

To actually **route** a Tier A read to gws:

1. **Backend switch**  
   In gog, for a specific Tier A command (e.g. `gmail labels list`), add a way to choose backend:
   - e.g. env `GOG_BACKEND=gws` or flag `--backend=gws` (or per-command in config).
   - Default remains `native` (current behavior).

2. **Invoke gws**  
   When backend is `gws`, run the gws CLI (or equivalent) with the same logical request (account, scopes, query). gws must be on PATH or at a configured path. Capture stdout, stderr, exit code.

3. **Normalize**  
   Run the same normalization used in the parity runner (e.g. `internal/parity/normalize`) on the live gws output so the response shape and `error_code` match the contract. Return the normalized result to the caller (CLI or MCP tool).

4. **Promotion gates**  
   Before defaulting any command to gws, satisfy the matrix gates: contract parity tests pass, error mapping complete, rollback flag tested, etc. (`docs/merge/command-migration-matrix.md` §1).

DRY: Reuse the parity normalizer and error taxonomy so live gws responses are normalized the same way as fixtures. No second normalization path.

---

## 3. How to test

### 3.1 Parity (fixtures only) — available now

- **What it does:** Ensures gws fixtures normalize to the same contract as native and that breaking vs drift are classified correctly. No live API calls.
- **How to run:**
  ```bash
  make parity
  # or
  go run ./cmd/gog-parity --fixtures docs/merge/goldens --schemas docs/merge/schemas --provider gws
  ```
- **Success:** Exit 0; `parity-report.json` has no breaking diffs for hard-gated cases (401, 404); drift is allowed.
- **CI:** Parity workflow runs on fixtures and uploads the report; reviewers confirm no breaking diffs.

### 3.2 Live gws routing (implemented for Gmail labels list/get, drive ls, drive get, drive search)

- **Implemented:** Gmail labels list/get; drive ls (single-page, non-global); **drive get** (no `--page-count`); **drive search** (single-page, no `--all`).

- **Unit / integration:** harness in `internal/cmd/gws_routing_parity_test.go` (fake `gws` via `GOG_GWS_PATH`):
  - Account policy: reject explicit `--account` / `GOG_ACCOUNT`; allow `auto`/`default`.
  - `TestGWS_RoutedCommands`: every live route (gmail labels list/get, drive ls/get/search) — JSON shape, argv, native constructor not called; text format checks where unique.
  - `TestGWS_KeepsNativeForBoundedFlags`: `--global` / `--all` / `--page-count` stay native (gws not invoked).
  - `TestGWS_NormalizesProviderError`: gws 401 → `BackendError` with stable `error_code`.

- **Manual smoke:** `scripts/smoke-gws-routing.sh` (or `make smoke-gws`). See §3.3.

- **Rollback:** included in the same script (`GOG_BACKEND=native` half). See §3.3.

### 3.3 Manual smoke and rollback (concrete steps)

**Preferred:** run the smoke script on a host with OAuth credentials and (for the gws half) authenticated `gws`:

```bash
make smoke-gws
# or
scripts/smoke-gws-routing.sh
scripts/smoke-gws-routing.sh --native-only --account you@gmail.com
scripts/smoke-gws-routing.sh --gws-only
```

The script checks Gmail labels list/get and Drive ls/get/search under both backends and asserts JSON shapes (`labels`, wrapped `label`/`file`, `files`).

**Manual equivalent** (default imported account for gws — do **not** pass `--account` / `-a` or set `GOG_ACCOUNT` when `GOG_BACKEND=gws`):

1. **Gmail labels (list):**  
   `GOG_BACKEND=gws gog gmail labels list --json`  
   Expect: exit 0; stdout is JSON with `labels` array; schema matches `docs/merge/schemas/gmail-labels-list.json` or only drift differences.

2. **Gmail labels (get):**  
   `GOG_BACKEND=gws gog gmail labels get INBOX --json`  
   Expect: exit 0; JSON has label details.

3. **Drive ls** (single-page, non-global):  
   `GOG_BACKEND=gws gog drive ls --json`  
   Expect: exit 0; stdout has `files` and optionally `nextPageToken`. For `--global` or `--all`, gog uses native (no gws path).

4. **Drive get** (no `--page-count`):  
   `GOG_BACKEND=gws gog drive get <fileId> --json`  
   Expect: exit 0; JSON has file metadata. With `--page-count`, gog uses native.

5. **Drive search** (single-page, no `--all`):  
   `GOG_BACKEND=gws gog drive search --raw 'trashed = false' --json`  
   Expect: exit 0; stdout has `files` and optionally `nextPageToken`. With `--all`, gog uses native.

**Rollback:**

- Unset or set `GOG_BACKEND=native`, then run the same command. Behavior must be unchanged (native API used; same JSON shape per contract).  
  Example: `GOG_BACKEND=native gog gmail labels list -a you@gmail.com --json` uses native Gmail API (`-a` is fine on native). Same for `gog drive ls`, `gog drive get`, `gog drive search`.

---

## 4. Summary

| Question | Answer |
|----------|--------|
| When do we use gws vs gog in the agent path? | **Default:** always gog. **When GOG_BACKEND=gws:** gog invokes gws for Tier A commands that support it (gmail labels list/get, drive ls single-page, **drive get**, **drive search** single-page), normalizes, returns. gws is also used in parity (fixtures only). |
| How is routing implemented? | Backend switch in gog (`GOG_BACKEND` env); `internal/backend/gws` invokes gws CLI; `internal/parity/normalize` normalizes; single MCP (gog-agentic). |
| How do we test? | **Parity:** `make parity` on fixtures. **Unit:** `TestGWS_*` fake-gws harness. **Live:** `make smoke-gws` / `scripts/smoke-gws-routing.sh` (authenticated native + gws). |
