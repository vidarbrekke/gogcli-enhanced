# Start here: next developer (GWS integration)

**You’re taking over the GWS integration and routing phase.** The app is a Go CLI and MCP server (gog-agentic) that acts as the agent-safe control plane for Google Workspace. Today every live request is served by gog (native APIs). The goal is to let **Tier A read commands** (e.g. Gmail labels list, Drive list) optionally use **gws** as a backend, with strict normalization so contracts and behavior stay stable. Writes and safety-critical flows stay on gog.

**What’s done:** The parity runner is implemented (`cmd/gog-parity`, `internal/parity/*`): it compares native vs gws **fixtures** (no live API calls), normalizes gws errors, validates schemas, and reports breaking vs drift. CI runs it; 401/404 are hard-gated. **Live routing** to gws is implemented for **Gmail labels list/get** (`GOG_BACKEND=gws`); extend to more Tier A commands per the matrix.

**What to do next:**
1. Read **`handover.md`** (repo root), then **`gogcli-developer-handover/HANDOVER.md`** (current status, gotchas, key paths).
2. Implement per **`docs/merge/GWS-VS-GOG-ROUTING.md`** §2: backend switch (e.g. `GOG_BACKEND=gws`), invoke gws CLI for the chosen Tier A command, normalize output with existing parity logic, add tests.
3. Run **`make parity`** to confirm fixtures pass; after routing exists, add integration tests and manual smoke with gws on PATH.

**Conventions:** `AGENTS.md` for build, test, lint, and PR workflow. Do not parse error `message` for semantics; `google_reason` is drift-only; keep contracts minimal.
