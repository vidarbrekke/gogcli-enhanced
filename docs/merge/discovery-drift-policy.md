# Discovery Drift Policy

When comparing native gog output to another backend (e.g. gws), responses will differ. This document defines how we treat those differences: **pin/capture** vs **accept+detect**.

## Goals

- **Stable contracts** for agents and scripts that parse `--json` output.
- **Clear rules** so we can implement normalization and regression tests without ambiguity.
- **Evidence-based** decisions: only treat a difference as “acceptable” if we document and detect it.

---

## 1) Pin / Capture (strict parity)

**Use when:** The field or shape is part of the **documented contract** and consumers rely on it. Backend output must be normalized to match native (or we keep native for that path).

- **Pin:** This value/structure is fixed in the schema and goldens. Any backend must produce it (via normalization if needed).
- **Capture:** We record the native output as the golden; diffs in this area are **critical** and must be fixed before promoting the backend.

**Examples:**

- Root keys: `labels`, `label`, `files`, `nextPageToken` — names and presence must match.
- Required fields on labels: `id`, `name`, `type`.
- Required fields on drive file in ls: `id`; commonly used: `name`, `mimeType`, `size`, `modifiedTime`.
- Type consistency: `size` as string (Drive API), counts as integers (Gmail).

**Process:**

1. Add the rule to the schema (required vs optional) and to the golden fixture.
2. In the adapter, normalize the backend response to this shape.
3. In conformance tests, assert normalized output matches golden (or schema).

---

## 2) Accept + Detect (documented variance)

**Use when:** We explicitly allow the backend to differ (e.g. extra fields, different order, or optional field presence), but we **record** the variance so we can detect regressions and document behavior.

- **Accept:** We do not require normalization for this difference; both shapes are valid.
- **Detect:** We have a test or diff rule that knows about this variance (e.g. ignore these paths in diff, or assert one of two allowed shapes).

**Examples:**

- **Order of items:** Label or file list order may differ between APIs. We accept either order but may assert “same set of ids” or “sorted comparison”.
- **Optional fields:** e.g. `messageListVisibility` present in one backend, absent in another. Schema marks it optional; diff tool ignores or allows both.
- **Numeric vs string:** If we ever allow a field to be either (for compatibility), we document and detect both.

**Process:**

1. Document the allowed variants in the schema (e.g. optional, or oneOf) or in a separate “allowed-diffs” list for the command.
2. In the diff/conformance step, apply rules that treat these as non-critical (e.g. ignore path, or match with a comparator that allows both).
3. Do not normalize for accept+detect unless we later change policy to pin.

---

## 3) Decision flow

```
For each difference between native and backend:
  → Is this field/structure in the published contract or used by known consumers?
    YES → Pin/capture: require normalization or keep native.
    NO  → Is it safe to allow variance (no parsing/script impact)?
           YES → Accept + detect: document and add diff rule.
           NO  → Re-evaluate: either add to contract (pin) or defer migration for this path.
```

---

## 4) Schema and goldens

- **Schema** defines required vs optional and types. Tighter schema = more pin/capture.
- **Goldens** are the canonical native output. Backend output is normalized to match goldens for pinned fields; for accept+detect, goldens document “one valid shape” and diff rules encode the other(s).

---

## 5) When to tighten (pin) later

- We discover a consumer that relies on a field we previously marked accept+detect → move to pin and add normalization.
- We add a new backend and find its output is more reliable for a given field → we can change golden and pin the new shape (with a contract version bump if needed).

---

## 6) Summary table

| Situation | Policy | Action |
|-----------|--------|--------|
| Contract field, must be stable | Pin/capture | Normalize backend to match golden; fail conformance if mismatch. |
| Optional or order variance, no consumer dependency | Accept+detect | Document; diff/conformance allows both; no normalization. |
| Unknown or high-impact variance | Block | Do not promote backend until resolved (pin or accept+detect). |

This policy should be referenced in the merge plan and in each command dossier that compares native vs gws.

---

## 7) Parity runner: error envelope — treat `reason` as drift-only

**Rule:** In the parity runner, treat **`google_reason`** (the gws/Google `error.reason` string) as **drift-only by default**, even after the 403 golden is captured.

**Why:** Reason strings can vary subtly across auth flows and environments. Automation value is in **stable classification** (`error_code` / HTTP `code`) and **safety controls**, not mirroring Google's reason vocabulary. Avoid burning time on reason-string diffs.

**Implementation:** Normalize gws `error.code` (and optionally `error.reason`) to a stable `error_code` for classification and CI. In diff/conformance, do **not** fail on `reason` string mismatches; only require that the error is correctly classified (e.g. 401 → authError, 403 → forbidden, 404 → notFound). Optionally record reason in logs or drift report for debugging.
