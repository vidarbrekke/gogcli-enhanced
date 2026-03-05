# Command Dossier: Drive Read Commands

Scope:

- `gog drive ls`
- `gog drive search`
- `gog drive get`
- `gog drive url`

Tier: A (low risk)  
Target: `candidate-gws`  
DRI: TBD

---

## 1) Why This Is a Good First Migration

- High-usage, low-side-effect read operations.
- Strong leverage from broad backend coverage.
- Lower risk of irreversible failures vs write/edit commands.

---

## 2) Contract Requirements (Must Stay Stable)

For all migrated commands:

- Existing top-level output shape is unchanged.
- JSON field names remain backward-compatible.
- Existing pagination and max-item behavior preserved.
- Error envelopes still map to `error.error_code` taxonomy.

Critical fields by command:

- `drive ls/search`:
  - `files[]`
  - each file `id`, `name`, `mimeType`
- `drive get`:
  - file identity and metadata fields currently consumed by scripts.
- `drive url`:
  - exact URL construction semantics (including docs/sheets/slides handling if applicable).

---

## 2b) Schemas, goldens, and capture instructions

**Real native JSON sample (for schema tightening and golden fixtures):**

- **List:** `docs/merge/goldens/drive-ls-native.json` — exact native output for `gog --json drive ls` (test fixture: two items, one file + one folder, with `nextPageToken`).

**JSON Schema (contract):**

- `docs/merge/schemas/drive-ls.json` — drive ls (and search list shape) response.

**How to capture native output (for reviewers / one-off goldens):**

```bash
gog --json drive ls --max 5 > docs/merge/goldens/drive-ls-native.json
gog --json drive get <fileId> > docs/merge/goldens/drive-get-native.json   # add when needed
```

**Run diffs and classify normalization:**

1. Produce gws output for the same logical request (same account, same parent/max/page).
2. Diff native golden vs gws output (e.g. `diff drive-ls-native.json drive-ls-gws.json` or JSON-aware diff).
3. For each difference, classify per `docs/merge/discovery-drift-policy.md`: **pin/capture** (must normalize), **accept+detect** (document and allow), or block.
4. Document normalization rules in this dossier or in the schema.

---

## 3) Provider Mapping (Canonical -> gws)

Mapping intent:

- `drive ls`: list files with optional parent/global/all-drives options.
- `drive search`: query-based list with optional raw query mode.
- `drive get`: fetch file metadata by ID.
- `drive url`: may remain native if pure local formatting; otherwise use `get` + formatter.

Adapter responsibilities:

1. Build `gws` invocation args from canonical command intent.
2. Execute command safely (timeout, stderr capture, structured parse).
3. Normalize response payload to existing gog contract.
4. Normalize errors to stable taxonomy.

---

## 4) Known Parity Risks

1. Pagination defaults may differ.
2. All-drives/shared-drive inclusion semantics may differ.
3. Query syntax handling (`--raw-query`) may diverge on edge cases.
4. Optional metadata fields can be absent/present differently.

Mitigation:

- Golden tests on representative query cases.
- Explicit adapter defaults matching gog behavior.
- Diff classification rules for optional/non-critical fields.

---

## 5) Test Plan

Unit tests:

- intent -> provider arg mapping for each command variant.
- error mapper behavior for common API failures.

Contract tests:

- golden JSON snapshots for each command:
  - simple default call
  - filters/flags enabled
  - no results
  - permission denied/not found cases

Integration tests:

- execute both native and gws provider for same input fixtures.
- compare normalized output for critical fields.

Shadow tests:

- production-safe shadow run on read traffic.
- collect diff reports and classify critical vs informational.

---

## 6) Rollout Plan

1. Implement adapter + tests behind feature flag.
2. Enable shadow mode for these commands.
3. Resolve critical diffs.
4. Canary rollout (small percentage / selected accounts).
5. Promote to default if SLO and parity gates pass.

Feature flags:

- `provider.gws.service.drive.enabled`
- `provider.gws.command.drive-ls.enabled`
- `provider.gws.command.drive-search.enabled`
- `provider.gws.command.drive-get.enabled`

---

## 7) Rollback Plan

Trigger:

- Increased error rate, latency regression, or critical contract drift.

Action:

- Disable command-level `gws` flags and return to native path.
- Preserve artifacts:
  - failing request hashes
  - diff reports
  - normalized error envelopes

---

## 8) Done Criteria

- All listed commands pass contract parity tests.
- Shadow diff critical rate = 0 over agreed window.
- Canary SLOs non-regressive.
- Migration matrix updated with final decision and date.
