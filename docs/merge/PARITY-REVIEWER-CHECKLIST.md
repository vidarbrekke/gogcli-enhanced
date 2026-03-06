# Parity runner — reviewer checklist

Use this list to verify **policy + intent**, **core engine**, **schemas/goldens**, and **CI enforcement**. Paths are from the repo root.

---

## A) Policy + intent (are we building the right thing?)

| # | Path | Purpose |
|---|------|--------|
| 1 | **handover.md** | Single source of truth: mission, 3-PR plan, invariants, pitfalls, PR checklist. |
| 2 | **docs/merge/discovery-drift-policy.md** | When to pin vs accept+detect; `google_reason` drift-only; parity runner `reason` stance. |
| 3 | **docs/merge/GWS-SAMPLES.md** | Validated gws stdout/stderr behaviors (errors on stdout, reason strings). |
| 4 | **docs/merge/README.md** | What `docs/merge/` is for (parity, contracts, drift control). |
| 5 | **docs/merge/CAPTURE-403-RUNBOOK.md** | One-time capture of real 403 golden; 403 stays soft until then. |

---

## B) Parity runner code (is the core engine correct + DRY?)

| # | Path | Purpose |
|---|------|--------|
| 6 | **cmd/gog-parity/main.go** | CLI entrypoint: flags, case loop, classify → normalize/schema/diff, report JSON, hard-gate exit. |
| 7 | **internal/parity/io/io.go** | DiscoverCases, LoadFixture, ProvidersForCase, FixtureData. No live API calls. |
| 8 | **internal/parity/classify/classify.go** | Classify(FixtureData): ERROR if exit≠0 or top-level `error` in stderr or stdout JSON. |
| 9 | **internal/parity/normalize/normalize.go** | Gmail-only: NormalizeError (HTTP→error_code, google_reason drift-only), InvocationCtx. |
| 10 | **internal/parity/schema/schema.go** | LoadSchema, Validate (required + types, same-doc `$ref` to `#/$defs`). |
| 11 | **internal/parity/diff/diff.go** | Diff(canonical, baseline, rules): recursive diff, DriftPaths, labels set-by-id. |
| 12a | **internal/parity/io/io_test.go** | Discovery, load, missing file, invalid exit_code. |
| 12b | **internal/parity/classify/classify_test.go** | Exit, stderr error, stdout error, combinations, malformed JSON. |
| 12c | **internal/parity/normalize/normalize_test.go** | 401/404 normalization, google_reason present, prefer stderr. |
| 12d | **internal/parity/schema/schema_test.go** | Validate pass/fail, LoadSchema, required-field violations. |
| 12e | **internal/parity/diff/diff_test.go** | Breaking vs drift, message/google_reason drift, labels set-by-id, missing id. |
| 12f | **cmd/gog-parity/main_test.go** | reportPath, isHardGatedBreakingPath. |

---

## C) Schemas + goldens (are we avoiding “keep up with Google”?)

**Schemas**

| # | Path | Purpose |
|---|------|--------|
| 13 | **docs/merge/schemas/gmail-labels-list.json** | Root `labels[]`; items require `id`, `name`; `type` optional (reduces provider churn). |
| 13 | **docs/merge/schemas/gmail-labels-get.json** | Root `label` object; same required fields (`id`, `name`; `type` optional). |
| 13 | **docs/merge/schemas/drive-ls.json** | Root `files[]` with `id`; optional nextPageToken. |

*Note:* There is no separate “envelope” schema file. Error envelope shape is defined in **internal/parity/normalize** (CanonicalEnvelope) and documented in code (contractual vs drift fields). Policy: error_code contractual; message/google_reason drift-only.

**Goldens (Gmail labels)**

| # | Path | Purpose |
|---|------|--------|
| 14 | **docs/merge/goldens/README.md** | Layout: `<case>/<provider>/{stdout.json,stderr.json,exit_code.txt}`. |
| 14 | **docs/merge/goldens/gmail-labels-401-unauthenticated/gws/** | 401 unauthenticated (error on stdout). |
| 14 | **docs/merge/goldens/gmail-labels-403-forbidden/gws/** | 403 placeholder until real capture (see CAPTURE-403-RUNBOOK). |
| 14 | **docs/merge/goldens/gmail-labels-get-not-found/gws/** | 404 not_found. |
| 14 | **docs/merge/goldens/gmail-labels-list/gws/** | List success (many labels). |
| 14 | **docs/merge/goldens/gmail-labels-list/native/** | List success (minimal fixture). |

---

## D) CI enforcement (does this actually get reviewed?)

| # | Path | Purpose |
|---|------|--------|
| 15 | **.github/workflows/parity.yml** | On PR/push to main: build `gog-parity`, run on fixtures, upload `parity-report.json`. Go version matches go.mod (1.24). |
| 16 | **Makefile** | `make fmt` / `make fmt-check`, `make lint`, `make test`, `make ci`. `make parity` runs the parity runner with default fixture/schema paths (see below). |

**Reviewer contract:** Open the **parity-report.json** artifact (or run `make parity` or `./bin/gog-parity --fixtures docs/merge/goldens --schemas docs/merge/schemas --provider gws > parity-report.json` locally) and confirm **no breaking diffs** in hard-gated cases (401, 404). Drift-only differences are acceptable per policy.
