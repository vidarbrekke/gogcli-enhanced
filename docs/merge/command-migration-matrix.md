# Command Migration Matrix (gogcli <-> gws)

Status: Initial draft  
Owner: Platform / CLI team  
Purpose: Track per-command migration decisions, parity status, and rollout readiness.

Related:

- `google_gogcli_merge_plan.md`

---

## 1) How to Use This Matrix

Decision values:

- `keep-native`: Keep gogcli native implementation as default.
- `candidate-gws`: Good candidate to route through `gws` after parity validation.
- `hybrid`: Split behavior; some sub-operations native, others `gws`.
- `blocked`: Cannot migrate yet (missing capability/parity/safety guarantees).

Risk tiers:

- `Tier A` = read/list/get/search (low risk)
- `Tier B` = non-destructive writes (medium risk)
- `Tier C` = destructive edits/high-impact workflows (high risk)

Required promotion gates (must all pass before default routing change):

1. Contract parity tests pass.
2. Error mapping coverage complete.
3. Canary SLOs meet baseline.
4. Rollback flag tested.
5. DRI + lead signoff.

---

## 2) Matrix Template

Copy this table row structure for new commands:

| Command | Service | Tier | Current Backend | Recommended Target | Decision | Why | Required Work | Test Coverage Needed | Rollout Stage | DRI |
|---|---|---|---|---|---|---|---|---|---|---|
| `gog <service> <cmd>` | `<service>` | `A/B/C` | `native` | `native/gws/hybrid` | `keep-native/candidate-gws/hybrid/blocked` | one-line rationale | parity + adapter + mapping tasks | unit + contract + integration + canary | `not-started/shadow/canary/ga` | `<name>` |

---

## 3) Initial Classification (First Pass)

Notes:

- This is a planning-grade first pass and should be refined with usage telemetry.
- High-value differentiators (agentic safety + deterministic edit workflows) are intentionally biased toward native/hybrid.

| Command Group | Tier | Current Backend | Recommended Target | Decision | Why | Required Work | Rollout Stage |
|---|---|---|---|---|---|---|---|
| `gog drive ls/search/get/url` | A | native | gws | candidate-gws | Commodity read APIs, good fit for breadth leverage | adapter mapping + output normalization + error mapping | not-started |
| `gog drive permissions` (read/list) | A | native | gws | candidate-gws | Low side-effect surface | contract parity + auth behavior tests | not-started |
| `gog docs info/cat/list-tabs` | A | native | gws | candidate-gws | Mostly read operations with manageable normalization | response normalization + tab behavior parity tests | not-started |
| `gog slides info/list-slides` | A | native | gws | candidate-gws | Read path, low risk | adapter + golden outputs | not-started |
| `gog sheets metadata/get/notes/links` | A | native | gws | candidate-gws | Read operations, good migration starter set | range parsing parity + output schema parity | not-started |
| `gog gmail labels list/get` | A | native | gws | candidate-gws | Low risk and high volume; suitable for early confidence | label field normalization + error mapping | not-started |
| `gog calendar calendars/events/get/search` (read-only paths) | A | native | hybrid | hybrid | Read paths are migratable, but date/time UX semantics are sensitive | strict timezone/day-of-week parity test suite | not-started |
| `gog people me/get/search` | A | native | gws | candidate-gws | Commodity endpoints, low blast radius | identity field parity + permission edge case tests | not-started |
| `gog tasks lists/list/get` | A | native | gws | candidate-gws | Low-risk reads | standard parity set | not-started |
| `gog forms get/responses list/get` | A | native | gws | candidate-gws | Straightforward read routes | schema parity + pagination parity | not-started |
| `gog contacts list/search/get` | A | native | hybrid | hybrid | Data shape can be nuanced; likely split by subcommand first | field mapping + contract tests per subtype | not-started |
| `gog chat spaces list/messages list/threads list` | A | native | gws | candidate-gws | Mostly list/get semantics | workspace auth + permission mapping tests | not-started |
| `gog groups list/members` | A | native | gws | candidate-gws | Workspace-specific read ops, low write risk | auth/scopes parity tests | not-started |
| `gog classroom ... list/get` | A | native | gws | candidate-gws | Large surface; start with read subcommands | staged service-by-service parity | not-started |
| `gog drive upload/copy/rename/move` | B | native | hybrid | hybrid | Medium risk writes; migrate only after Tier A success | idempotency + retry policy + write parity tests | not-started |
| `gog sheets update/append/format/insert` | B | native | hybrid | hybrid | Write semantics + data shape sensitivity | range/value coercion parity + dry-run semantics | not-started |
| `gog calendar create/update/respond` | B | native | keep native (initially) | keep-native | Existing opinionated UX and date handling are core value | only re-evaluate after strong read-path parity | not-started |
| `gog gmail send/drafts/batch modify` | B | native | keep native (initially) | keep-native | Email composition/headers/threading behavior is high-sensitivity | maintain native until mature parity evidence exists | not-started |
| `gog contacts create/update/delete` | B | native | keep native (initially) | keep-native | Field mask/update semantics can drift; high regress risk | mapping rigor + acceptance tests before any migration | not-started |
| `gog chat messages send/dm send` | B | native | hybrid | blocked | Workspace policy + formatting/permission edge cases likely complex | proof-of-parity spike first | not-started |
| `gog docs edit replace/append/insert/delete/batch` | C | native | native | keep-native | Core agentic safety contract and deterministic edit semantics | keep as control-plane reference implementation | n/a |
| `gog docs edit merge-data` | C | native | native | keep-native | High-impact workflow and differentiator | preserve native unless strict parity proven later | n/a |
| `gog sheets edit values/append/clear/batch/...` | C | native | native/hybrid | keep-native | Agent-safe edit model is strategic moat | keep native baseline; consider hybrid only for narrow ops | n/a |
| `gog slides edit replace-text/replace-image/batch/merge-data` | C | native | native | keep-native | Advanced edit + merge operations are high-risk and high-value | maintain native for reliability | n/a |
| `gog docs sed` / high-impact edit flows | C | native | native | keep-native | Safety and predictable execution are critical | maintain strict native path | n/a |
| `gog auth *` | C | native | native | keep-native | Security-critical domain, unique credential model and UX | do not migrate without dedicated security review | n/a |
| `gog config *` | B | native | native | keep-native | Local behavior; no API coverage advantage from gws | none | n/a |
| `gog gmail watch *` | C | native | native | keep-native | Pub/Sub/webhook workflow is operationally sensitive | keep native unless explicit re-architecture | n/a |
| `gog gmail track *` | C | native | native | keep-native | Product-specific feature outside generic API plumbing | keep native | n/a |
| `gog keep *` | B/C | native | native | keep-native | Specialized Workspace service-account behaviors | keep native pending focused spike | n/a |
| `gog appscript *` | B | native | candidate-gws | candidate-gws | Could benefit from broad API coverage once write parity is validated | staged parity + execution result normalization | not-started |
| `gog time now` | A | native | native | keep-native | Local utility, unrelated to Workspace API backend | none | n/a |

---

## 4) Priority Migration Queue (Suggested)

Phase order for practical execution:

1. Tier A quick wins:
   - Drive read commands
   - Docs/Slides/Sheets read commands
   - Gmail labels read commands
2. Tier A medium complexity:
   - Calendar read paths (timezone-sensitive)
   - Contacts/People read paths
3. Tier B selective hybrids:
   - Drive write-lite operations
   - Sheets non-destructive writes
4. Tier C deferred:
   - Keep native unless explicit parity proof project is approved.

---

## 5) Required Supporting Artifacts

For each command promoted beyond `not-started`, add:

- provider mapping spec
- contract parity test file references
- known diffs and their classification
- canary metrics snapshot
- rollback validation record

Suggested artifact path:

- `docs/merge/commands/<service>-<command>.md`

---

## 6) Acceptance Checklist (Per Command)

- [ ] Command intent mapped to provider adapter.
- [ ] Success output matches contract/golden.
- [ ] Error outputs mapped to stable taxonomy.
- [ ] Validate-only and dry-run semantics unchanged (if applicable).
- [ ] Integration tests pass for both providers.
- [ ] Shadow diff shows no critical deltas.
- [ ] Canary SLOs pass.
- [ ] Rollback tested.
- [ ] Decision updated in matrix with date and DRI.

---

## 7) Open Questions to Resolve Early

1. Which commands are most-used in real workflows (telemetry ranking)?
2. Which fields in outputs are considered contract-critical vs informational?
3. For hybrid commands, what is the fallback policy per risk tier?
4. What parity threshold is acceptable for non-critical informational fields?
5. Which Tier C commands are explicitly out-of-scope for migration in the next 12 months?

---

## 8) Change Log

- 2026-03-05: Initial matrix created with first-pass command classification.
