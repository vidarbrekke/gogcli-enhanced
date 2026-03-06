# Google Workspace CLI + gogcli Merge Plan

Status: Draft (handover-ready)  
Audience: New and existing developers, tech leads, QA, release engineers  
Horizon: 9-24 months  
Primary Goal: Preserve gogcli's agentic reliability guarantees while leveraging Google Workspace CLI (`gws`) for API breadth and velocity.

---

## 1) Executive Summary

This plan defines how to evolve `gogcli` from a single-stack CLI into a robust control-plane over multiple execution backends.

Long-term target architecture:

- **Control Plane (gogcli):** Stable UX, policy enforcement, safety semantics, deterministic request lifecycle, normalized output/error contracts, audit and observability.
- **Execution Backend(s):** Native Go API clients and `gws` provider, selected per command/workflow via routing policy.

Core principle:

- **Replace plumbing, not guarantees.**  
  We can swap execution engines, but we must not regress agentic correctness, safety, or output contracts.

Non-goals:

- Big-bang rewrite.
- Broad refactors without measurable reliability gains.
- Changing command UX/output formats without explicit versioning and migration.

---

## 2) Why This Plan Exists

`gws` provides excellent API surface velocity and broad Workspace coverage. `gogcli` already provides differentiated value in:

- Safety controls (`validate-only`, `dry-run`, revision guards).
- Deterministic request normalization (`requestHash`, normalized payload output).
- Cross-service contract consistency and testable machine-oriented envelopes.
- Operator-grade command behavior and policy controls.

This merger strategy combines both strengths:

- `gws` accelerates "how we call APIs."
- `gogcli` preserves "how reliable and safe agent behavior must be."

---

## 3) Design Principles (Must-Haves)

1. **Contract stability over implementation convenience**
   - Command outputs are treated as APIs for agents and automation.
   - No unversioned contract breaking changes.

2. **Determinism over dynamism**
   - Backend behavior differences must be normalized.
   - Same command + same input should produce equivalent semantic output.

3. **Safe-by-default**
   - For write/destructive operations, preflight and explicit guardrails remain mandatory.

4. **Progressive migration**
   - Migrate by risk tier (low to high), with canary and rollback.

5. **Evidence-driven decisions**
   - Promote backend routing only when parity + SLO criteria are met.

---

## 4) Current Strengths to Preserve

These are hard requirements and should be maintained regardless of backend:

- Shared agentic safety flags and workflow:
  - `--validate-only`
  - `--dry-run` / `--dry-run-edit`
  - `--pretty`
  - `--output-request-file`
  - `--execute-from-file`
  - `--require-revision` (where service supports it)
- Deterministic metadata:
  - `requestHash`
  - operation IDs (`opId`)
- Normalized error envelopes:
  - stable `error_code` taxonomy
  - service + operation + resource context
- Cross-service contract tests that assert behavior consistency.

If a backend cannot support these guarantees directly, `gogcli` control plane must emulate/compensate.

---

## 5) Target Architecture

### 5.1 Layered Model

1. **CLI UX Layer**
   - Existing command grammar, flags, and user-facing behavior.
2. **Command Intent Layer**
   - Parse and validate command intent into canonical internal request models.
3. **Policy/Safety Layer**
   - Enforce preflight constraints, write guards, revision constraints, allowlists.
4. **Provider Adapter Layer**
   - Translate canonical intent to backend-specific requests.
5. **Execution Layer**
   - Native Go client or `gws`.
6. **Normalization Layer**
   - Convert backend responses/errors into stable output envelopes.
7. **Telemetry/Audit Layer**
   - Structured logs, request fingerprints, correlation IDs, backend metadata.

### 5.2 Provider Interface (Conceptual)

Each provider must implement:

- `Validate(intent) -> normalized validation result`
- `DryRun(intent) -> normalized request preview`
- `Execute(intent) -> normalized success payload`
- `MapError(error) -> normalized error envelope`
- `Capabilities() -> operation/service feature support map`

The interface must hide backend specifics from command handlers.

### 5.3 Routing Strategy

Routing input:

- Command/op name
- Risk tier
- Feature flags
- Capability support
- Runtime health (optional future)

Routing output:

- Selected provider (`native`, `gws`)
- Optional fallback chain
- Policy mode (`strict parity`, `best effort`, etc.)

---

## 6) Migration Scope and Risk Tiers

### Tier A: Low Risk (Migrate Early)

- Read/list/get/search operations without side effects.
- Metadata and status endpoints.
- Export/read-only utility commands.

Success criteria:

- Output parity (schema and core field semantics).
- Error mapping parity for common failures.
- No reliability regression under load.

### Tier B: Medium Risk (Migrate After Soak)

- Non-destructive writes with straightforward rollback.
- Create/update operations with limited blast radius.

Success criteria:

- All Tier A criteria plus retry/idempotency confidence.
- Preflight safety behavior preserved.

### Tier C: High Risk (Migrate Last)

- Destructive edits/deletes/batch operations.
- Merge-data/template generation pipelines.
- Operations requiring revision control and strict determinism.

Success criteria:

- Full contract parity.
- Revision safety intact.
- Replay/audit behavior verified end-to-end.

---

## 7) Phased Plan (Detailed)

## Phase 0: Baseline and Decision Artifacts (2-4 weeks)

Goals:

- Capture what must never regress.
- Inventory command behavior and contract dependencies.

Tasks:

1. Build a command inventory table:
   - command path
   - service
   - operation type (read/write/destructive)
   - current backend
   - risk tier
   - critical consumers
2. Capture baseline contract docs:
   - success envelopes
   - error envelopes
   - validation/dry-run envelopes
3. Establish baseline SLOs:
   - command success rate
   - p95 latency per tier
   - retry rates
   - contract test pass rates
4. Write ADRs:
   - provider abstraction strategy
   - output contract versioning
   - rollback and traffic shaping policy

Deliverables:

- `docs/merge/command-inventory.md`
- `docs/merge/contract-baseline.md`
- `docs/merge/slo-baseline.md`
- ADR set (`docs/adr/`)

Exit criteria:

- Team agrees on frozen baseline.
- No ambiguity on what "parity" means.

---

## Phase 1: Provider Abstraction Foundation (3-6 weeks)

Goals:

- Introduce backend abstraction without changing user-visible behavior.

Tasks:

1. Add provider interfaces and canonical intent models.
2. Implement `native` adapter first (behavior-preserving shim).
3. Move command handlers to call provider abstraction.
4. Keep output formatting and error envelopes unchanged.
5. Add adapter-level unit tests and golden output tests.

Implementation notes:

- Do not migrate behavior and architecture simultaneously.
- First objective is an internal seam with zero behavior drift.

Deliverables:

- `internal/providers/` package (or equivalent)
- `native` provider implementation
- parity tests proving no behavior changes

Exit criteria:

- Existing tests pass.
- New abstraction introduces no contract diffs.

---

## Phase 2: gws Adapter MVP (4-8 weeks)

Goals:

- Build a minimal `gws` provider for Tier A commands.

Tasks:

1. Create command translation maps:
   - canonical intent -> `gws` invocation shape
2. Implement process runner and response ingestion.
3. Normalize `gws` output into existing `gogcli` contract.
4. Implement error taxonomy mapper:
   - map backend-specific messages/reasons to stable `error_code`.
5. Add conformance tests for every migrated command.

Implementation notes:

- Keep fallback to `native` available behind feature flags.
- Explicitly log backend chosen per command.

Deliverables:

- `gws` provider (Tier A coverage target)
- error mapping table docs
- conformance test suite for Tier A

Exit criteria:

- Tier A parity >= agreed threshold (target: 100% schema parity, near-100% semantic parity on critical fields).

---

## Phase 3: Dual-Run, Shadow, and Canary (4-10 weeks)

Goals:

- Validate production reliability before default routing changes.

Tasks:

1. Shadow mode:
   - Execute primary backend response path.
   - Optionally run secondary backend in non-user-impacting mode for diffing.
2. Diff engine:
   - compare normalized outputs
   - classify acceptable vs critical diffs
3. Canary rollout:
   - small percentage / specific commands / selected accounts
4. Automated rollback:
   - revert routing when error rate or latency breaches thresholds.

Operational controls:

- Feature flags:
  - global provider enable
  - per-service provider enable
  - per-command override
  - emergency disable

Deliverables:

- shadow diff reports
- canary dashboards
- rollback runbook

Exit criteria:

- SLOs equal or better than baseline for migrated scope.
- No unresolved critical diffs.

---

## Phase 4: Tier B Migration (6-12 weeks)

Goals:

- Expand provider routing to medium-risk operations.

Tasks:

1. Add idempotency strategy for retry-safe writes.
2. Validate preflight behavior equivalence.
3. Harden timeout/retry policies per operation class.
4. Expand contract + integration + chaos tests.

Reliability checkpoints:

- Backoff behavior verified under transient API failure.
- no duplicate side effects under retry paths.

Exit criteria:

- Tier B migrated commands meet parity and SLO gates.

---

## Phase 5: Tier C Selective Migration (8-20 weeks)

Goals:

- Evaluate high-risk operations one by one.

Tasks:

1. Prioritize high-value high-risk commands.
2. Validate revision guard semantics and deterministic behavior.
3. Keep native as default where parity is not provable.
4. Adopt explicit "do not migrate" list where native remains superior.

Critical policy:

- Tier C commands require explicit approval to switch defaults.
- "No parity evidence, no promotion."

Exit criteria:

- Only proven-safe Tier C commands migrated.
- Remaining commands documented with rationale.

---

## Phase 6: Steady-State Governance (Ongoing)

Goals:

- Sustain reliability and prevent drift.

Tasks:

1. Quarterly parity audits.
2. Dependency health checks (security + breaking change review).
3. Contract compatibility CI gates remain mandatory.
4. Incident postmortems feed into adapter improvements.

Deliverables:

- quarterly health report
- updated capability/routing matrix

---

## 8) Data Contracts and Versioning

### 8.1 Contract Versioning Policy

- Add explicit contract version in machine-readable outputs if not already present.
- Semantic versioning:
  - Patch: non-breaking fields added (optional).
  - Minor: additive changes requiring documentation updates.
  - Major: breaking changes (rare, migration guide required).

### 8.2 Error Taxonomy

Maintain a stable error code set independent of backend wording:

- `invalid_argument`
- `unauthenticated`
- `permission_denied`
- `not_found`
- `conflict`
- `rate_limited`
- `backend_unavailable`
- `internal_error`
- service-specific codes where necessary, but documented.

### 8.3 Determinism Requirements

- Request normalization is canonical and stable.
- Hash generation rules must be unchanged across backends.
- Non-deterministic backend fields must be filtered or normalized.

---

## 9) Reliability Engineering Requirements

### 9.1 SLOs (Recommended Initial Targets)

- Tier A success rate: >= 99.9%
- Tier B success rate: >= 99.5%
- Tier C success rate: >= 99.0% (excluding external hard failures)
- p95 latency non-regression threshold: <= 15% over baseline unless accepted per-command.

### 9.2 Retry and Timeout Policy

- Only retry known transient errors.
- Exponential backoff with jitter.
- Max attempts per operation type configurable.
- Respect idempotency constraints.

### 9.3 Circuit Breakers (Optional but recommended)

- Per-provider/service breaker when upstream is unstable.
- Degrade gracefully to fallback provider when safe.

### 9.4 Fallback Rules

- Tier A: allow transparent fallback where parity risk is low.
- Tier B/C: fallback requires explicit safety check to avoid duplicate writes.

---

## 10) Testing Strategy (Mandatory)

### 10.1 Test Layers

1. Unit tests:
   - translation logic
   - error mapping
   - normalization functions
2. Contract tests:
   - golden JSON outputs/errors
   - validate-only and dry-run envelope checks
3. Integration tests:
   - native and gws provider runs
4. End-to-end tests:
   - critical workflows and agentic scenarios
5. Replay tests:
   - run historical anonymized traces against both backends
6. Chaos tests:
   - injected transient failures, timeout storms, partial responses

### 10.2 CI Gates

Required to merge:

- All existing unit tests pass
- all new provider tests pass
- contract parity tests pass
- no new lint/security failures
- migration-scope integration tests green

### 10.3 Golden Test Hygiene

- Goldens must include:
  - success payloads
  - validation payloads
  - dry-run payloads
  - representative error envelopes
- Golden updates require rationale in PR notes.

---

## 11) Observability and Audit

### 11.1 Structured Logging

Every execution event should include:

- `opId`
- command path
- provider selected
- request hash
- service + operation
- latency
- retry count
- outcome category

### 11.2 Metrics

Minimum metrics:

- requests total by command/provider/outcome
- error code distribution
- retries and fallback counts
- p50/p95/p99 latency
- parity diff rate in shadow mode

### 11.3 Traceability

- Ensure trace propagation from CLI invocation to provider execution.
- Record enough context for reproducible debugging without leaking secrets.

---

## 12) Security and Compliance

Requirements:

- Never log access tokens or secret material.
- Preserve current credential handling best practices.
- Validate subprocess invocation safety and argument escaping for `gws`.
- Enforce least-privilege scopes for new provider paths.
- Run dependency vulnerability scans on both Go and Node ecosystems when applicable.

---

## 13) Release and Rollout Playbook

### 13.1 Rollout Sequence

1. Internal alpha (developers only).
2. Controlled beta (selected accounts + low-risk commands).
3. Gradual GA by service/command.

### 13.2 Feature Flag Matrix

- `provider.global.enabled`
- `provider.gws.enabled`
- `provider.gws.service.<service>.enabled`
- `provider.gws.command.<command>.enabled`
- `provider.fallback.enabled`
- `provider.emergency.native_only`

### 13.3 Rollback Protocol

Trigger conditions:

- SLO breach over rolling window
- elevated parity critical diffs
- significant error-code drift

Actions:

1. flip emergency flag to native-only
2. capture failure artifacts
3. notify on-call/release channel
4. open incident doc and assign owner

---

## 14) Team Roles and Responsibilities

Suggested role split:

- **Tech Lead:** architecture, ADR approvals, parity gate owner.
- **Backend Engineer(s):** provider adapter implementation and tests.
- **QA/Automation:** conformance and end-to-end suite ownership.
- **SRE/Release Engineer:** canary, metrics, rollback automation.
- **Product/PM:** migration prioritization and customer communication.

Ownership model:

- Each migrated command has a DRI.
- No migration task is "done" without telemetry and rollback readiness.

---

## 15) Known Challenges and Mitigations

1. **Output mismatch across providers**
   - Mitigation: strict normalization + golden tests + shadow diff tooling.

2. **Error inconsistency**
   - Mitigation: centralized taxonomy mapper and required mapping tests.

3. **Dynamic upstream behavior changes**
   - Mitigation: scheduled compatibility checks + canary + alerting.

4. **Duplicate side effects from fallback/retry**
   - Mitigation: idempotency keys and operation-specific fallback policies.

5. **Team onboarding complexity**
   - Mitigation: this doc + architecture diagrams + playbooks + ADR index.

---

## 16) Opportunities to Capture

- Faster access to newly exposed Workspace endpoints.
- Reduced maintenance burden on low-differentiation command implementations.
- More development focus on agentic intelligence, safety, and workflow orchestration.
- Potential future multi-provider support using same control-plane abstraction.

---

## 17) New Developer Onboarding Guide

Week 1 goals:

1. Read architecture and contracts:
   - this document
   - existing agentic contract and edit helper code
2. Run baseline tests locally.
3. Execute one Tier A command through native provider path.
4. Implement one small adapter translation with tests.

Week 2 goals:

1. Add/extend one contract parity test.
2. Run shadow diff tooling for selected command.
3. Document findings and open one improvement PR.

Definition of "onboarded":

- Can explain routing policy and parity criteria.
- Can implement a small provider mapping safely.
- Can run and interpret conformance suite outputs.

---

## 18) Documentation Plan

Create/maintain:

- `docs/merge/architecture.md`
- `docs/merge/provider-interface.md`
- `docs/merge/error-taxonomy.md`
- `docs/merge/routing-policy.md`
- `docs/merge/shadow-diff-guide.md`
- `docs/merge/rollback-runbook.md`
- `docs/merge/command-migration-matrix.md`

Documentation standards:

- Every migration PR updates the command migration matrix.
- Every new mapping includes examples and test links.

---

## 19) Milestone Checklist

Use this as a tracker:

- [ ] Baseline contracts frozen and documented
- [ ] Provider abstraction merged with zero behavior drift
- [ ] gws Tier A adapter shipped behind flags
- [ ] Shadow mode and diff tooling operational
- [ ] Canary + rollback automation in place
- [ ] Tier A promoted to stable default where proven
- [ ] Tier B migration completed with SLO compliance
- [ ] Tier C selective migration decisions documented
- [ ] Governance and quarterly parity audit process established

---

## 20) Final Decision Rules

A command may move to `gws` default only if all conditions hold:

1. Contract parity proven (tests + golden outputs).
2. Error mapping coverage complete.
3. SLOs meet or exceed baseline in canary.
4. Rollback path tested and documented.
5. Owning engineer signs off and lead approves.

If any condition fails:

- Keep native as default.
- Document blocker and re-evaluate later.

---

## 21) Immediate Next Actions (Actionable Start)

1. Create `docs/merge/` directory and seed baseline docs listed above.
2. Produce first command migration matrix (top 30 most-used commands).
3. Draft provider interface ADR and seek team approval.
4. Implement provider abstraction seam with native adapter only.
5. Add first contract conformance harness that runs provider-agnostic tests.

This sequence de-risks the entire 9-24 month journey and avoids speculative over-engineering.

---

## 22) Closing Guidance

The robust long-term solution is not to chase maximum backend replacement.  
It is to preserve a stable, testable, agent-safe control plane while selectively adopting backend capabilities where they improve velocity and reliability.

In practical terms:

- Keep promises to agents and automation first.
- Move execution internals only when evidence says it is safe.
- Treat parity, observability, and rollback as first-class product features.
