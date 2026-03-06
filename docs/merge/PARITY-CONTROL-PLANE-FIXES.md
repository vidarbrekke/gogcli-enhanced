# Parity runner: control-plane-grade fixes (external review plan)

This doc captures the fix plan for the three issues that separate “nice parity harness” from “control-plane-grade enforcement.” Implementation can follow in 3 PRs.

---

## 1) 401/404 hard-gated in code (not just docs)

**Problem:** We normalize 401/404 but never turn them into `breaking[]` or enforce schema+diff — CI can pass while the contract regresses.

**Fix:**
- Per-case `CaseResult` gets explicit **Status**: `PASS` | `DRIFT_ONLY` | `FAIL_BREAKING` | `FAIL_RUNNER`.
- **Policy:** Hard-gated cases = `gmail-labels-401-unauthenticated`, `gmail-labels-get-not-found`. CI fails if any breaking OR any runner failure on those cases.
- **Exit rule:** Non-zero if any `FAIL_RUNNER` OR any hard-gated case is `FAIL_BREAKING`.

**Tests to add:** Hard-gated + missing schema → non-zero; hard-gated + ErrorCode mismatch → breaking → non-zero; hard-gated + only message diff → drift only → zero.

---

## 2) No silent skips (schema/unmarshal/setup failures)

**Problem:** Schema load/validate and JSON unmarshal failures are skipped with `if err == nil`; no runner failure reported.

**Fix:**
- Add **runnerFailures[]** per case: `kind` (IO_ERROR | JSON_PARSE_ERROR | SCHEMA_NOT_FOUND | SCHEMA_INVALID | SCHEMA_VALIDATION_FAILED), `stream`, `details`.
- **Default:** Any runner failure fails CI (exit non-zero).
- **Exception:** Explicit placeholder cases. Marker: `docs/merge/goldens/<case>/<provider>/PLACEHOLDER.txt` (empty file). If present → `Status = SKIPPED_PLACEHOLDER`, no runner failure, meta.warning for reviewers.

**Tests to add:** Missing stdout.json → runner failure; malformed JSON → runner failure; schema not found → runner failure; PLACEHOLDER.txt present → skipped, runner passes.

---

## 3) Deterministic diff output

**Problem:** Map iteration order in diff makes report non-deterministic; CI diffs become noisy.

**Fix:**
- **A)** Canonicalize before diff: when traversing objects, iterate keys in sorted order (or canonicalize parsed JSON to sorted-key form before diffing).
- **B)** Sort resulting diff entries by path then summary before emitting report.

**Tests to add:** Same inputs → byte-identical parity-report.json; map key order in inputs doesn’t change diff order; set-by-id produces stable ordering (sorted by id).

---

## Patch plan (3 PRs)

| PR | Scope |
|----|--------|
| **#1** | runnerFailures[] + Status per case; exit non-zero on runner failure (except placeholder); case + diff entry ordering; PLACEHOLDER.txt support. |
| **#2** | Policy: hard-gated case list; enforce exit on breaking for hard-gated; ensure 401/404 normalized output is validated + diffed. |
| **#3** | Canonical JSON key ordering before diff; sort diff entries; byte-identical report test. |

---

## Optional: policy file

**docs/merge/parity-policy.json** (no hardcoding in Go):

```json
{
  "hard_gated_cases": [
    "gmail-labels-401-unauthenticated",
    "gmail-labels-get-not-found"
  ],
  "placeholder_cases": [
    "gmail-labels-403-forbidden"
  ]
}
```

---

## What to send external reviewer for surgical diffs

1. **cmd/gog-parity/main.go** (exit code logic + case loop)
2. **internal/parity/diff/*.go** (traversal + ordering)
3. **internal/parity/schema/*.go** (how schema failures are handled)
4. **internal/parity/io/*.go** (fixture load errors)
5. **One sample parity-report.json** (from `make parity` or CI)

Or paste just the runner loop + report emission snippets (~30–60 lines) so they can point to exact silent-skip and nondeterminism locations.
