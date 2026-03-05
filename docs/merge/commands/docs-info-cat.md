# Command Dossier: Docs Read Commands

Scope:

- `gog docs info`
- `gog docs cat`
- `gog docs list-tabs`

Tier: A (low risk, but output-shape sensitive)  
Target: `candidate-gws`  
DRI: TBD

---

## 1) Why This Migration Matters

- These commands are frequently used as read primitives before edits.
- Migration provides broad backend leverage while keeping edit flows native.
- Establishes pattern for Docs read/write split strategy.

---

## 2) Contract Requirements (Must Stay Stable)

Critical compatibility rules:

- Existing JSON shape and field names remain stable.
- `docs cat` textual extraction behavior remains compatible for scripts.
- Tab-related outputs (`list-tabs`, `--tab`, `--all-tabs`) remain semantically identical.
- Error envelopes keep stable codes (`not_found`, `permission_denied`, etc.).

Non-negotiable:

- No regressions in machine-parseable behavior for agent workflows that inspect doc structure before edits.

---

## 3) Provider Mapping (Canonical -> gws)

Command intents:

- `docs info`: fetch document metadata and high-level structure.
- `docs list-tabs`: retrieve tab descriptors.
- `docs cat`: retrieve text content with existing truncation/max-byte semantics.

Adapter notes:

- If `gws` returns richer/raw docs payloads, normalization must preserve current gog output contract.
- If textual extraction is currently opinionated in gog, keep extraction logic in control plane and use backend as raw source.

---

## 4) Known Parity Risks

1. Differences in text flattening/extraction order.
2. Tab model representations may differ.
3. Large document truncation behavior may differ.
4. Optional structural fields may vary in presence.

Mitigation:

- Preserve gog extraction/formatting path where possible.
- Add explicit max-byte and truncation parity tests.
- Golden tests for multi-tab and nested-structure docs.

---

## 5) Test Plan

Unit tests:

- mapping: command flags -> provider arguments.
- normalization: raw provider payload -> gog contract.

Contract tests:

- golden snapshots for:
  - simple doc
  - multi-tab doc
  - large doc (truncation path)
  - missing doc / permission error

Integration tests:

- dual provider comparison for same document fixtures.

Shadow tests:

- passive diffing for read traffic only.
- critical diff classes:
  - content order drift
  - tab ID/name mismatches
  - truncation inconsistencies

---

## 6) Rollout Plan

1. Enable `docs info` first (lowest complexity).
2. Enable `docs list-tabs` second.
3. Enable `docs cat` last after extraction parity confidence.

Feature flags:

- `provider.gws.command.docs-info.enabled`
- `provider.gws.command.docs-list-tabs.enabled`
- `provider.gws.command.docs-cat.enabled`

---

## 7) Rollback Plan

- Disable command-specific `gws` flags immediately.
- Revert to native read backend.
- Preserve diff artifacts for root-cause analysis.

---

## 8) Done Criteria

- Read command parity proven on critical fields and semantics.
- No critical shadow diffs for agreed window.
- Canary error rate and latency within baseline thresholds.
- Migration matrix row updated with evidence links.
