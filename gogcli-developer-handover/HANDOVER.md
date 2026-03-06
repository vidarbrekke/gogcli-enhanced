# Developer Handover: Hybrid Provider Parity + Drift Control (gogcli-enhanced)

> **Canonical handover:** repo-root **`handover.md`** is the single source of truth.  
> This file is the **detailed implementation plan** and reference; do not duplicate full handover content here.

**Audience:** core maintainer/developer implementing the parity runner + provider normalization + CI gates.  
**Goal:** enable a **hybrid architecture** where `gogcli-enhanced` remains the **agent-safe control plane** while backends (notably `gws`) provide broad Workspace API coverage **without breaking determinism, safety semantics, or machine contracts**.

Date: 2026-03-05

---

## 0) North Star

We are not “building a Workspace API wrapper.”  
We are building an **agentic control plane** with stable contracts:

- deterministic request lifecycle (validate-only / dry-run / request files / replay)
- stable JSON error classification (`error_code`) and safety semantics
- cross-service consistency for machine consumption

**Breadth is commodity. Determinism is product.**

So the plan is: **Hybrid**  
- Tier A (read/list/get): can route to `gws` via a strict adapter  
- Tier C (edit/merge/write): remain native (reference implementation)

---

## 1) What you are implementing (deliverables)

### 1.1 Parity runner (fixtures-only)
A small tool to compare provider outputs (native vs `gws`) using:
- outcome classification (success/error)
- normalization (esp. `gws` errors on stdout)
- schema validation (envelope + command payload)
- diff + reporting (breaking vs drift)

**YAGNI constraints**
- no live API calls
- no dynamic discovery pinning inside the runner
- no auto-upgrade logic
- keep it fixture-driven and deterministic

### 1.2 Provider normalization: `gws` → canonical gog envelope
Validated facts:
- `gws` emits **error JSON on stdout** with shape `{"error":{"code":<int>,"message":<string>,"reason":<string>}}`.

Normalization rules:
- classify as error if `stdout_json.error` OR `stderr_json.error` OR `exit_code != 0`
- map `http_status := error.code`
- map `google_reason := error.reason` (**drift-only**)
- map canonical `error_code` via HTTP status mapping table
- inject `service`, `operation`, `resource_id` from invocation context (argv/params), not from message text

### 1.3 CI gates
- Always enforce schema + parity for core Tier A commands with real goldens (401/404 now).
- 403 becomes **hard required** only after maintainer captures real 403, per runbook.

---

## 2) Artifacts you already have (and what each is for)

These are included verbatim in the attached handover zip under `artifacts/`:

1. **`gog-parity-specs.zip`**
   - Generated “executable parity spec” skeletons from dossiers
   - Includes schema placeholders, mapping docs, normalization stubs, and golden folders

2. **`envelope-artifacts-v2.zip`**
   - Provider-agnostic parity rules + explicit gws stdout-error handling
   - Includes gws normalization doc + Gmail labels parity rules

3. **`gmail-error-taxonomy-lock.zip`**
   - Consolidated Gmail taxonomy lock bundle:
     - real 401 + 404 goldens
     - 403 placeholder + runbook + capture-info guidance
     - “google_reason is drift-only forever” posture

> **DO NOT** invent new output fields or expand contracts unless a consumer needs it.  
> Keep schemas minimal and stable.

---

## 3) Implementation plan (detailed, DRY & YAGNI)

### Phase 1 — Wire the parity runner skeleton (1–2 sessions)
1. Add `cmd/gog-parity` (skeleton provided in `templates/`).
2. Implement fixture loading:
   - discover cases under `docs/merge/goldens/<case>/<provider>/`
   - load `stdout.json` (may be empty), `stderr.json` (may be empty), `exit_code.txt`
3. Implement outcome classification:
   - parse JSON if possible
   - ERROR if:
     - exit_code != 0 OR
     - stderr_json has top-level `error` OR
     - stdout_json has top-level `error` (needed for gws)

**Acceptance:** runner can enumerate fixtures and classify outcomes deterministically.

### Phase 2 — Implement normalization (Gmail first) (1–2 sessions)
1. Implement provider normalization interface:
   - `Normalize(provider, invocationCtx, stdout_json, stderr_json, exit_code) -> canonicalOutcome`
2. Implement gws error normalization:
   - if stdout_json.error exists: map to canonical error payload
   - apply HTTP→error_code mapping:
     - 401 unauthenticated
     - 403 permission_denied
     - 404 not_found
     - 400 invalid_argument
     - 429 resource_exhausted
     - else unknown
3. **Do not** treat `google_reason` as contractual; report as drift only.

**Acceptance:** gws 401 + 404 goldens normalize into canonical errors with correct `error_code` and `http_status`.

### Phase 3 — Schema validation (1–2 sessions)
1. Add shared envelope schema validation:
   - validate canonical errors against `schemas/gmail.error.schema.json` (Gmail scope)
   - validate success payloads against command schemas (e.g., `gmail-labels-list.result.schema.json`)
2. Schema stance for maintenance:
   - top-level payload strict (`additionalProperties: false`) where feasible
   - inner objects allow `additionalProperties: true` for controlled growth
   - require only minimal fields (`id`, `name`) for lists

**Acceptance:** the current goldens validate cleanly.

### Phase 4 — Diffing + reporting (1–2 sessions)
1. Implement a minimal recursive diff:
   - output json-pointer path + summary
   - support allowlists for:
     - ordering differences (lists compare as sets by key)
     - drift-only fields (message, google_reason)
2. Gmail labels specific rules:
   - compare `labels` as set keyed by `id` (order drift allowed)

**Acceptance:** report separates `breaking` vs `drift`.

### Phase 5 — CI integration (1 session)
1. Add a GitHub Actions workflow (template provided) that:
   - builds parity runner
   - runs it on fixtures only
   - uploads report artifact
2. Gate policy:
   - require 401 + 404 parity always
   - require 403 parity only for “promotion” PRs until 403 is captured

**Acceptance:** CI runs on PR and produces a parity report.

---

## 4) “Only the maintainer can do this” items (explicit guidance)

### 4.1 Capture real 403 permission denied (one-time)
- Follow `docs/merge/CAPTURE-403-RUNBOOK.md`
- Capture stdout JSON into `gmail-labels-403-forbidden-gws.json`
- Optionally capture:
  - `gmail-labels-403-forbidden-gws.capture-info.txt` (version, argv, scopes, optional creds/profile line)
- Commit both

**After capture:** upload the real 403 JSON (+ optional capture-info) and we will generate the final promotion-ready bundle where 403 becomes a hard CI requirement.

### 4.2 Discovery drift policy enforcement (quarterly / value-triggered)
- Upgrades to `gws` must run parity suite
- Do not auto-upgrade `gws` based on upstream churn
- Focus on value-driven upgrades only

---

## 5) File placement recommendations (minimal churn)

Recommended repo locations (adjust to match your tree):
- `docs/merge/schemas/` — JSON schemas
- `docs/merge/goldens/` — fixture goldens
- `docs/merge/normalization/` — human-readable normalization rules
- `cmd/gog-parity/` — parity runner
- `internal/parity/` — implementation modules

---

## 6) Acceptance checklist

### Must pass before routing defaults to `gws` for Gmail reads
- [ ] Parity runner classifies errors from stdout + stderr correctly
- [ ] 401 + 404 gws goldens normalize to canonical error_code + http_status
- [ ] Gmail labels list success validates against minimal schema
- [ ] Diff report shows zero breaking diffs (drift diffs allowed: message, google_reason, ordering)
- [ ] CI job runs and produces parity report

### Must pass before promoting 403 to hard requirement
- [ ] Real 403 golden captured + committed
- [ ] 403 normalizes to canonical `permission_denied`
- [ ] CI gate updated so 403 is required alongside 401 + 404

---

## 7) Appendix: What’s attached

See `artifacts/` in the zip:
- `gog-parity-specs.zip`
- `envelope-artifacts-v2.zip`
- `gmail-error-taxonomy-lock.zip`

See `templates/`:
- `PARITY-RUNNER-README.md`
- `gog-parity-skeleton.go`
- `github-actions-parity.yml`

---

## 8) Non-negotiables (to prevent “keep up with Google” tax)

- **google_reason is drift-only forever** (per drift policy §7)
- never parse `message` for semantics
- schemas promise only minimal fields needed by automation
- upgrade `gws` only on purpose (quarterly or value-triggered)
- no contract changes without goldens + schema updates + explicit review
