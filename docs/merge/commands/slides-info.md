# Command Dossier: Slides Read Commands

Scope:

- `gog slides info`
- `gog slides list-slides`

Tier: A (low risk)  
Target: `candidate-gws`  
DRI: TBD

---

## 1) Migration Rationale

- Read-only commands with low blast radius.
- Useful proving ground before any Slides write/edit consideration.
- Supports broader strategy of migrating commodity reads first.

---

## 2) Contract Requirements

Must preserve:

- Presentation identity fields and metadata shape.
- Slide listing semantics and ordering.
- Existing JSON field naming conventions consumed by scripts.
- Stable error envelope behavior.

Do not change:

- Interpretation of slide IDs/titles where existing consumers rely on them.

---

## 3) Provider Mapping (Canonical -> gws)

Intent mapping:

- `slides info`: presentation metadata and high-level info.
- `slides list-slides`: list slide identifiers and relevant descriptors.

Adapter responsibilities:

- Parse provider output and normalize into current gog contract.
- Ensure consistent ordering and field availability.

---

## 4) Known Parity Risks

1. Slide order representation differences.
2. Optional slide metadata fields present/absent inconsistently.
3. Presentation property naming differences.

Mitigation:

- Apply deterministic ordering normalization if needed.
- Mark non-critical optional field diffs as informational.
- Keep critical fields required in contract tests.

---

## 5) Test Plan

Unit tests:

- command-to-provider mapping.
- payload normalization helpers.

Contract tests:

- golden outputs for:
  - standard deck
  - deck with edge-case slide structures
  - not found/permission denied errors

Integration tests:

- compare native vs gws normalized outputs.

Shadow tests:

- non-impacting mirror execution and diff classification.

---

## 6) Rollout Plan

1. Enable `slides info` first.
2. Enable `slides list-slides` second.

Feature flags:

- `provider.gws.command.slides-info.enabled`
- `provider.gws.command.slides-list-slides.enabled`

---

## 7) Rollback Plan

- Toggle off command-level gws flags.
- Route to native implementation.
- Archive diff/error context for follow-up fixes.

---

## 8) Done Criteria

- All contract tests green.
- No critical shadow diffs in agreed observation window.
- Canary SLOs stable.
- Migration matrix updated with final status.
