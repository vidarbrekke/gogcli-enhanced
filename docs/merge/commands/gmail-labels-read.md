# Command Dossier: Gmail Labels Read Commands

Scope:

- `gog gmail labels list`
- `gog gmail labels get`

Tier: A (low risk)
Target: `hybrid` (`GOG_BACKEND=gws` live for list/get)
DRI: TBD
Status: opt-in gws routing implemented; authenticated smoke and real 403 golden remain open (see `handover.md`)

---

## 1) Why This Is in the First Batch

- Simple read operations with clear response models.
- High value for quick confidence in Gmail provider mapping.
- Limited side-effect risk.

---

## 2) Contract Requirements

Must preserve:

- Label identity semantics (`id`, name, type where applicable).
- Label count fields in `labels get` output (where currently present).
- Existing sorting/output conventions expected by scripts.
- Stable error codes for missing labels and permissions.

Name vs ID behavior:

- Preserve current gog behavior for label lookup by ID/name.
- Do not introduce case sensitivity regressions unintentionally.

---

## 2b) Schemas, goldens, and capture instructions

**Real native JSON samples (for schema tightening and golden fixtures):**

- **List:** `docs/merge/goldens/gmail-labels-list-native.json` — exact native output for `gog --json gmail labels list` (test fixture: two labels).
- **Get:** `docs/merge/goldens/gmail-labels-get-native.json` — exact native output for `gog --json gmail labels get INBOX` (test fixture: INBOX with counts).

**JSON Schemas (contract):**

- `docs/merge/schemas/gmail-labels-list.json` — list response.
- `docs/merge/schemas/gmail-labels-get.json` — get response.

**How to capture native output (for reviewers / one-off goldens):**

```bash
gog --json gmail labels list > docs/merge/goldens/gmail-labels-list-native.json
gog --json gmail labels get INBOX > docs/merge/goldens/gmail-labels-get-native.json
```

**Run diffs and classify normalization:**

1. Produce gws output for the same logical request (same account, same label id/name).
2. Diff native golden vs gws output (e.g. `diff gmail-labels-list-native.json gmail-labels-list-gws.json` or a JSON-aware diff).
3. For each difference, classify per `docs/merge/discovery-drift-policy.md`: **pin/capture** (must normalize), **accept+detect** (document and allow), or block.
4. Document normalization rules in this dossier or in the schema’s description/`allowed-diffs` section.

---

Intent:

- `labels list`: fetch all labels and format according to gog contract.
- `labels get`: fetch specific label by identifier and normalize result.

Adapter behavior:

- Ensure system labels and user labels are represented consistently.
- Map backend-specific fields to stable contract fields.

---

## 4) Known Parity Risks

1. Count fields may be absent or named differently across providers.
2. Ordering differences in returned label arrays.
3. Name/ID resolution behavior drift for lookups.

Mitigation:

- Explicit label sort normalization where required.
- Fallback lookup logic in control plane for strict compatibility.
- Golden tests for both system and user labels.

---

## 5) Test Plan

Unit tests:

- mapping and normalization tests for both commands.
- label identifier resolution helper tests.

Contract tests:

- golden outputs:
  - list with mixed system/user labels
  - get by label ID
  - get by label name (if supported by current contract)
  - not found and permission denied envelopes

Integration tests:

- native vs gws comparison for representative mailboxes.

Shadow tests:

- run on read traffic and classify diffs (counts/order/field presence).

---

## 6) Rollout Plan

1. Launch `labels list` first.
2. Launch `labels get` second after lookup parity validation.

Feature flags:

- `provider.gws.command.gmail-labels-list.enabled`
- `provider.gws.command.gmail-labels-get.enabled`

---

## 7) Rollback Plan

- Disable command-specific gws flags.
- Return to native immediately.
- Preserve diff reports and failing input samples.

---

## 8) Done Criteria

- Contract and integration tests pass for both commands.
- Shadow and canary show no critical regressions.
- Migration matrix updated with rollout evidence.
