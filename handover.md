# handover.md — gogcli-enhanced

This is the **single source of truth** for a new developer takeover.

## Handover: gogcli-enhanced (Hybrid Provider Parity Pivot)

## Developer Quickstart (First 60 Minutes)

## Repo layout pointers (from PROJECT-LAYOUT.md)

- Canonical takeover doc: `handover.md` (this file, repo root)
- Supporting bundle folder: `gogcli-developer-handover/` (contains artifacts + templates)
- Merge/parity docs and fixtures live under: `docs/merge/`
- Golden fixtures and schemas live under: `docs/merge/goldens/` and `docs/merge/schemas/`
- CLI entrypoints live under: `cmd/`
- Implementation lives under: `internal/` (including `internal/mcp/`)


Use this if you are new and need to execute without prior context.

### 1) Understand the mission (5 min)
- `gogcli-enhanced` remains the **control plane** (determinism, stable contracts, safety).
- `gws` is a **backend provider** for selected Tier A reads.
- We are explicitly preventing “keep up with Google” tax via parity + drift controls.

### 2) Read only what matters first (10 min)
Start here, in this order:
1. `handover.md` (this file)
2. `AGENTS.md` (repo conventions, build/test, PR workflow)
3. `docs/merge/discovery-drift-policy.md` (drift stance + CI philosophy)
4. `docs/merge/GWS-SAMPLES.md` (validated gws stdout/stderr behaviors)
5. `docs/merge/CAPTURE-403-RUNBOOK.md` (one-time 403 capture)
6. `gogcli-developer-handover/templates/PARITY-RUNNER-README.md` (parity runner scope)

### 3) Validate local setup (5 min)
- Run:
  - `make test`
  - `make lint`
- If either fails, stop and fix environment/toolchain first.

### 3.1) Smoke-run parity locally (5 min)
Once PR #1 lands (runner skeleton), you should be able to run:
- `go run ./cmd/gog-parity --fixtures docs/merge/goldens --schemas docs/merge/schemas --provider gws > parity-report.json`

Expected:
- Exit code 0
- `parity-report.json` is created
- Report contains cases and correct error classification for gws (error JSON may be on **stdout**)

### 4) Execute in 3 PRs (30+ min kickoff)
- **PR #1:** parity runner skeleton + fixture loader + classification.
- **PR #2:** gws normalization (Gmail first) + schema validation.
- **PR #3:** diff/reporting + CI workflow + 401/404 hard gating.

Use section `## 7) PR Plan (Explicit)` in this document as your checklist.

PR review rule:
- Reviewers must open the generated `parity-report.json` (local file or CI artifact) and confirm that any differences are correctly classified as `breaking` vs `drift` before approving.

### 5) Keep these invariants (always)
- `google_reason` is drift-only forever.
- Never parse error `message` for semantics.
- No live API calls in parity runner.
- No schema/contract expansion without explicit consumer need.
- Treat list ordering as non-contractual unless explicitly documented (prefer set-by-id comparisons).

### 6) Definition of success for this pivot
- 401 + 404 are hard-gated in parity CI.
- 403 is soft-gated until maintainer captures real 403 golden, then promoted to hard.
- Parity report clearly separates `breaking` vs `drift`.

### 7) Common pitfalls (avoid these)
- **gws error location:** gws can emit error JSON on **stdout**; classification must check `stdout.error`, not only stderr/exit code.
- **Stream precedence:** if both stdout and stderr contain JSON errors, prefer stderr (native convention), otherwise use whichever stream has the JSON error object.
- **False contract churn:** do not treat `google_reason` or freeform `message` as contractual; they are drift-only.
- **Over-scoped schemas:** only require minimal fields consumers need (`id`, `name`, etc.); avoid strictness without a real consumer requirement.
- **Unstable diffs:** compare Gmail labels as set-by-id (order drift allowed) instead of raw list order.
- **Runner scope creep:** no live API calls, dynamic discovery pinning, or auto-upgrade logic inside parity runner.
- **CI gate timing:** 401+404 are hard now; 403 becomes hard only after real 403 golden is captured and committed.

---

## 0) Read This First

This file is the **single developer takeover guide** for the current workstream.
If you are new to this repository, start here, then use the links below for deeper reference.

Primary objective:
- Keep `gogcli-enhanced` as the **agent-safe control plane** (stable contracts, deterministic behavior, safety semantics).
- Allow `gws` to be used as a backend for selected Tier A read commands **without contract drift**.

---

## 1) Project Overview

`gogcli-enhanced` is a Go CLI and MCP-oriented control plane for Google Workspace operations.
Its value is not raw API breadth; it is:
- deterministic request lifecycle (`--dry-run`, `--validate-only`, replayability)
- stable machine-readable envelopes (`error_code`, consistent JSON)
- safety and predictable behavior across commands/services

### What changed (the pivot)

Google’s latest tooling release (via `gws`) provides broad API coverage quickly, but response envelopes and behavior can vary (notably error handling on stdout, reason-string drift, ordering/optional-field variance).

Because of that, we are pivoting to a **hybrid architecture**:
- **Tier A (read/list/get):** can route to `gws` through a strict normalization/parity layer.
- **Tier C (edit/write/merge/safety-critical):** stays native in `gogcli-enhanced`.

This preserves product value (determinism + safety) while benefiting from provider breadth.

---

## 2) Non-Negotiables

These are mandatory and override convenience:

1. `google_reason` is **drift-only forever**.
2. Never parse freeform `message` text for semantics.
3. Contracts/schemas stay minimal and stable (only fields consumers need).
4. No live API calls inside parity runner (fixtures-only).
5. No auto-upgrade or dynamic discovery pinning logic in parity runner.
6. `gws` upgrades are value-triggered/quarterly, not continuous churn.

Reference:
- `docs/merge/discovery-drift-policy.md`

---

## 2.1) Glossary (fast)

- **Control plane:** gogcli-enhanced’s stable contracts + safety semantics + deterministic workflows.
- **Provider:** execution backend (native gog, `gws`, future backends) whose outputs are normalized into canonical contracts.
- **Parity:** fixtures-only comparison of provider outputs against the canonical schema + diff rules.
- **Breaking vs drift:** breaking fails CI (contract/classification/safety). Drift is reported but non-blocking (e.g., `message`, `google_reason`, ordering).

---

## 3) Source of Truth Files

Read in this order:

1. `handover.md` (this file)
2. `AGENTS.md` (repo conventions and guardrails)
3. `gogcli-developer-handover/HANDOVER.md` (next-phase brief: GWS integration and routing; status and next steps)
4. `docs/merge/discovery-drift-policy.md`
5. `docs/merge/GWS-SAMPLES.md`
6. `docs/merge/GWS-VS-GOG-ROUTING.md` (when gws vs gog; how to implement and test routing)
7. `docs/merge/CAPTURE-403-RUNBOOK.md`
8. `gogcli-developer-handover/templates/PARITY-RUNNER-README.md` (parity runner scope)

Support material:
- `gogcli-developer-handover/artifacts/envelope-artifacts-v2.zip`
- `gogcli-developer-handover/artifacts/gmail-error-taxonomy-lock.zip`
- `gogcli-developer-handover/artifacts/gog-parity-specs.zip`

---

## 4) Current State (As Of This Handover)

### When gog binary+MCP vs gws is used

| Context | What runs | Notes |
|--------|-----------|--------|
| **Live agent / MCP (default)** | **gog binary** (`gog mcp serve` = gog-agentic) | All MCP tool calls (Drive, Gmail, Docs, Sheets, etc.) are served by the **gog** Go binary and native Google APIs. Single MCP server; agent talks only to gog-agentic. |
| **Live CLI with GOG_BACKEND=gws** | **gog** invokes **gws** (Rust CLI) for selected Tier A commands | Only when `GOG_BACKEND=gws` and only for commands that support it (currently `gmail labels list`, `gmail labels get`). gog runs the gws CLI, captures stdout/stderr/exit, normalizes with `internal/parity/normalize`, returns result. Default is `native` (gog only). |
| **Parity / CI** | **Fixtures only** | No live API calls. Runner loads `docs/merge/goldens/<case>/{native,gws}/` fixtures and compares; gws is a fixture provider for validation only. |

**Summary:** Default path for all live requests is **gog binary + gog-agentic MCP**. The **gws** (Google Workspace Rust CLI) is (1) used in parity as a fixture source and (2) optionally used at runtime for Tier A Gmail labels when `GOG_BACKEND=gws`. Writes and safety-critical flows stay on gog.

Completed:
- Parity runner implemented: `cmd/gog-parity`, `internal/parity/*` (io, classify, normalize, schema, diff). Fixture loading, outcome classification, gws error normalization (stdout + stderr), schema validation, breaking vs drift diff, deterministic report ordering. Hard-gate 401/404; 403 soft until real golden; runner failures and placeholders (PLACEHOLDER.txt) supported.
- Gmail goldens under `docs/merge/goldens/` (401, 404, list success, 403 placeholder); schemas under `docs/merge/schemas/`. CI parity workflow; `make parity` runs the runner.
- **Live gws routing (optional):** Backend switch via `GOG_BACKEND=gws`; `internal/backend/gws` runs gws CLI for `gmail labels list` and `gmail labels get`; `internal/cmd/backend.go` + `backend_error.go`; normalized errors via parity logic; Tier A Gmail labels use gws when env set, else native. See `gogcli-developer-handover/HANDOVER.md`.
- Drift policy and routing logic documented (`docs/merge/discovery-drift-policy.md`, `docs/merge/GWS-VS-GOG-ROUTING.md`). Linode deploy and OpenClaw verification docs updated (`docs/TOOLS-gog-agentic-section.md`, `docs/LINODE-TEST-QUERIES.md`, `docs/openclaw-linode-runbook.md`).

Next steps (see `gogcli-developer-handover/HANDOVER.md`): extend Tier A routing to more commands per matrix; add integration tests and manual smoke for `GOG_BACKEND=gws`; capture real 403 golden and promote to hard gate.

---

## 5) Execution Plan (Detailed, New-Developer Friendly)

### Phase A — Local Bootstrap (Day 1)

1. Verify toolchain and repo health:
   - `make test`
   - `make lint`
2. Read the source-of-truth files above.
3. Confirm fixture tree exists under `docs/merge/goldens` and schemas under `docs/merge/schemas`.
4. Create a short personal checklist from Section 6 and keep it updated in PR descriptions.

Exit criteria:
- You can explain the difference between **breaking parity** and **drift-only differences**.

### Phase B — PR #1 (Parity runner skeleton + fixture loader + classification)

Scope:
- Add `cmd/gog-parity` entrypoint.
- Implement fixture discovery/loading from:
  - `docs/merge/goldens/<case>/<provider>/stdout.json`
  - `docs/merge/goldens/<case>/<provider>/stderr.json`
  - `docs/merge/goldens/<case>/<provider>/exit_code.txt`
- Note: gws error JSON may be on stdout; always load and attempt to parse both streams.
- Implement deterministic outcome classification:
  - ERROR when:
    - `exit_code != 0`, OR
    - `stderr_json.error` exists, OR
    - `stdout_json.error` exists (critical for gws)

Recommended package split (YAGNI):
- `internal/parity/io`
- `internal/parity/classify`
- `cmd/gog-parity`

Tests to add:
- classifier table tests for all 3 error sources
- malformed/empty JSON fixture behavior tests

Exit criteria:
- runner enumerates fixtures and classifies outcomes correctly
- report file emitted (even if normalization/diff is stubbed)

### Phase C — PR #2 (Normalization + schema validation, Gmail first)

Scope:
- Add provider normalization interface:
  - `Normalize(provider, invocationCtx, stdout, stderr, exitCode) -> canonical`
- Implement gws error normalization:
  - map `http_status := error.code`
  - map `google_reason := error.reason` (drift-only)
  - map `error_code` by status:
    - 400 -> `invalid_argument`
    - 401 -> `unauthenticated`
    - 403 -> `permission_denied`
    - 404 -> `not_found`
    - 429 -> `resource_exhausted`
    - default -> `unknown`
- Inject `service`, `operation`, `resource_id` from invocation context only.
- Add schema validation for envelope + Gmail command schemas.

Recommended package split:
- `internal/parity/normalize`
- `internal/parity/schema`

Tests to add:
- gws 401 + 404 normalize to expected canonical error_code/http_status
- schema pass/fail tests for canonical outputs

Exit criteria:
- 401 and 404 goldens validate and normalize cleanly

### Phase D — PR #3 (Diff/reporting + Gmail set-by-id + CI)

Scope:
- Implement minimal recursive diff with json-pointer paths.
- Separate findings into:
  - `breaking`
  - `drift`
- Drift allowlist includes:
  - message text
  - `google_reason`
  - ordering variance where configured
- Gmail labels comparison as set keyed by `id`.
- Add GitHub Action using template `gogcli-developer-handover/templates/github-actions-parity.yml`.
- Upload parity report artifact (`parity-report.json`) in CI.

Gate policy now:
- hard required: 401 + 404
- 403 remains soft until real 403 golden is committed

Exit criteria:
- PR CI runs parity job on fixtures and uploads report
- breaking vs drift separation is explicit and deterministic

### Phase E — Maintainer follow-up (Not for general dev)

- Capture real 403 using `docs/merge/CAPTURE-403-RUNBOOK.md`.
- Commit real 403 golden.
- Promote 403 to hard CI requirement.

---

## 6) Implementation Checklist (Copy Into PR)

- [ ] **No duplicate handover docs edited; canonical handover is repo-root `handover.md`.**
- [ ] Read handover + drift policy + samples
- [ ] Implement fixture loader
- [ ] Implement classification (exit/stderr/stdout error)
- [ ] Add table tests for classification
- [ ] Implement gws normalization mapping (http -> error_code)
- [ ] Ensure `google_reason` is drift-only, never breaking
- [ ] Add schema validation for envelope + Gmail schemas
- [ ] Add recursive diff with breaking/drift split
- [ ] Add Gmail labels set-by-id comparator
- [ ] Emit `parity-report.json`
- [ ] Add CI workflow + artifact upload
- [ ] Confirm 401 + 404 hard gate; 403 soft gate pending real golden

---

## 7) PR Plan (Explicit)

### PR #1 — Parity Runner Foundation

Title suggestion:
- `feat(parity): add fixture loader and outcome classification skeleton`

Include:
- `cmd/gog-parity` basic command
- `internal/parity/io` + `internal/parity/classify`
- unit tests for fixture loading + classification
- initial report JSON shape

Do not include:
- schema validation logic
- provider normalization beyond stubs
- CI workflow

### PR #2 — Gmail Normalization + Schema Validation

Title suggestion:
- `feat(parity): normalize gws gmail errors and validate schemas`

Include:
- `internal/parity/normalize` (gws-focused)
- `internal/parity/schema`
- tests for 401/404 normalization and schema validation

Do not include:
- complex diff engine
- CI gating changes

### PR #3 — Diff/Reporting + CI

Title suggestion:
- `feat(parity): add drift-aware diff reporting and parity CI`

Include:
- drift vs breaking diff classification
- Gmail labels set-by-id compare
- GitHub Action parity workflow and artifact upload
- gate config for 401/404 hard, 403 soft

---

## 8) How To Add a New Command Spec (2–3 Steps)

1. Add fixtures for **both** providers (`native` and `gws`) under `docs/merge/goldens/<case>/<provider>/` with `stdout.json`, `stderr.json`, `exit_code.txt`.
2. Add or tighten the minimal command schema in `docs/merge/schemas/`.
3. Register normalization + diff rules (if needed) and run parity runner in CI.

Rule: add only the minimum required contract fields; keep drift fields non-breaking unless a consumer explicitly depends on them.

---

## 9) Reporting Format Expectations

`parity-report.json` should include at minimum:
- run metadata (timestamp, provider/version if known)
- per-case result
- `breaking[]` entries (path + reason)
- `drift[]` entries (path + reason)
- `normalizations_applied[]`

This report is the artifact uploaded by CI for PR review.
Reviewers must explicitly check the artifact and ensure there are **no breaking diffs** (drift is allowed per policy).

---

## 10) Operational Guardrails

- Keep changes small and reversible.
- No dependency/toolchain/version changes without explicit approval.
- No broad refactors unrelated to parity implementation.
- Tests first/alongside for non-trivial logic.
- Preserve existing command contracts unless explicitly approved.

---

## 11) Immediate Next Action

For **parity runner and live gws routing** (current state): see §4 Current State and **When gog binary+MCP vs gws is used** there.

For **next phase** (more Tier A commands, integration tests, 403 hard gate): read **`gogcli-developer-handover/HANDOVER.md`** for status, key files, and gotchas. Implement per `docs/merge/GWS-VS-GOG-ROUTING.md` and `docs/merge/command-migration-matrix.md`.

---

## 12) Folder/File Structure Checklist (Decisions)

Goal: keep the repo DRY/YAGNI while enabling the parity + drift-control pivot.

### 1) One canonical handover doc
- [x] **Decision: YES** — Repo-root `handover.md` is the **only** canonical handover doc.
- Any other handover-like content (e.g. inside `gogcli-developer-handover/`) must be **pointer-only** or supporting reference (artifacts/templates). Do not maintain a second full handover document.

### 2) Parity runner placement
- [x] **Decision: Option A** — Parity runner lives in `cmd/gog-parity/` (first-class CLI).
- Rule: choose A unless there is a strong reason parity must be test-only.

### 3) Where parity assets live (schemas/goldens/runbooks)
- [x] **Decision:** `docs/merge/` is the **permanent home** for schemas, goldens, normalization rules, and capture runbooks.
- No rename until ≥5 command groups migrate; then consider `docs/parity/` only if it reduces confusion. See `docs/merge/README.md`.

### 4) Provider adapter location
- [x] **Decision:** For now only the gws adapter exists; keep normalization inside `internal/parity/` (e.g. `internal/parity/normalize`). When a **second** provider lands, introduce a dedicated boundary (e.g. `internal/parity/providers/` or `internal/providers/`).
- Rule: do not create a separate providers package until there are at least two provider implementations.

### 5) Naming clarity (“merge” vs “parity” vs “contracts”)
- [x] **Decision:** Keep `docs/merge/`; add `docs/merge/README.md` explaining it is parity/contracts/drift-control (done). Prefer README over renaming midstream.

### 6) MCP reuse boundary
- [x] **Decision: NO** for now — Keep parity normalization **local to the parity runner** (`internal/parity/`). Do not share canonical error normalization with CLI commands or MCP tools yet; reduces blast radius. Revisit when we have a clear need to share.

### 7) Bundle folder (`gogcli-developer-handover/`) policy
- [x] **Decision:** Keep `gogcli-developer-handover/` **in-repo** — It contains living templates used by CI (parity skeleton, GitHub Action) and artifact zips. Keep it versioned.
- Rule: do not maintain a second full handover file inside this folder; canonical entrypoint is `handover.md`.

### 8) Preventing future duplication (DRY guardrail)
- [x] **Guardrail chosen:** PR checklist item — *“No duplicate handover docs edited; canonical handover is repo-root `handover.md`.”* Add to PR template or to Section 6 (Implementation Checklist) so reviewers confirm.

### 9) Minimal churn principle
- [x] **Decision:** No moves/renames for PR #1–#3. Only document decisions (this section) and add `docs/merge/README.md`. Do the smallest possible change that improves shipping.

### Outcome (filled in)

| Item | Decision |
|------|----------|
| **Canonical handover file** | `handover.md` (repo root) |
| **Parity runner location** | `cmd/gog-parity/` |
| **Parity assets location** | `docs/merge/` (permanent) |
| **Provider adapter location** | `internal/parity/` (e.g. `normalize`); dedicated `internal/parity/providers/` when second provider lands |
| **Guardrail chosen** | PR checklist: “No duplicate handover docs edited; canonical handover is handover.md” |

---

### Single-source rule (DRY/YAGNI)

Maintain **only this file** as the developer handover going forward. Avoid maintaining other full handover documents; keep this file authoritative and treat anything else as pointers only.
