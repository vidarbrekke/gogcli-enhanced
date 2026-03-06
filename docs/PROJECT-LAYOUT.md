# gogcli-enhanced — Directory and File Layout

Generated snapshot of the repository structure. Excludes: `.git`, `node_modules`, `.tools`, `bin`, and heavy build artifacts.

**Canonical handover:** `handover.md` (repo root). **Parity runner (planned):** `cmd/gog-parity/`. **Parity assets:** `docs/merge/` (permanent). See handover §12 for full checklist and outcome.

---

## Root

| Path | Description |
|------|-------------|
| `AGENTS.md` | Repository guidelines, build/test, coding style, PR workflow |
| `handover.md` | **Single source of truth** for developer takeover (parity pivot, quickstart, PR plan) |
| `README.md` | Project intro and usage |
| `INSTALL.md` | Install and upgrade instructions |
| `CHANGELOG.md` | Release history |
| `Makefile` | Build, fmt, lint, test, ci |
| `go.mod` / `go.sum` | Go module |
| `.golangci.yml` | Linter config |
| `.goreleaser.yaml` | Release build config |
| `.lefthook.yml` | Git hooks (pre-commit/pre-push) |
| `.gitignore` | Ignored paths |
| `gogcli-developer-handover/` | Parity handover bundle (HANDOVER = reference; canonical handover is `handover.md`) |
| `docs/` | Specs, runbooks, merge/parity assets, TOOLS reference |
| `cmd/` | CLI entrypoints: `gog`, (planned) `gog-parity` |
| `internal/` | All implementation; (planned) `internal/parity/` for parity runner |
| `scripts/` | Deploy, live tests, release, auth diagnostics |

---

## cmd/

| Path | Description |
|------|-------------|
| `cmd/gog/main.go` | CLI entrypoint |
| `cmd/gog/main_test.go` | Entrypoint tests |
| `cmd/gog-parity/` | **(Planned)** Parity runner CLI (fixture load, classify, diff); not yet implemented |

---

## internal/ (high level)

Implementation lives under `internal/` with tests colocated (`*_test.go`).

| Package | Purpose |
|---------|---------|
| `internal/cmd/` | Command implementations (account, auth, calendar, contacts, docs, drive, gmail, sheets, slides, tasks, etc.) — ~470+ files |
| `internal/config/` | Config, aliases, credentials, paths, keys |
| `internal/googleapi/` | Google API clients (calendar, drive, gmail, docs, sheets, slides, tasks, etc.) and transport |
| `internal/googleauth/` | OAuth flow, accounts server, token handling, embedded HTML templates |
| `internal/secrets/` | Keychain/store (darwin + other) |
| `internal/errfmt/` | Error formatting |
| `internal/outfmt/` | Output formatting |
| `internal/input/` | Prompt, readline |
| `internal/ui/` | UI helpers |
| `internal/ops/` | Operations helpers |
| `internal/timeparse/` | Time parsing |
| `internal/integration/` | Opt-in integration tests (build-tagged) |
| `internal/mcp/` | MCP server, transport, Google tools provider |
| `internal/parity/` | **(Planned)** Parity runner internals (io, classify, normalize, schema, diff); no dedicated providers package until second provider |
| `internal/tracking/` | Email tracking (worker, config, pixel, deploy) |
| `internal/authclient/` | Auth client helpers |

---

## internal/cmd/ (command areas)

Commands are grouped by service; each area has many `*_test.go` files.

- **Account / auth:** `account*.go`, `auth*.go`, `config_cmd.go`, `confirm*.go`
- **Calendar:** `calendar*.go`
- **Contacts:** `contacts*.go`
- **Docs:** `docs_cmd.go`, `docs_*.go`, `docs_sed*.go`, `docs_edit*.go`, `docs_import*.go`, etc.
- **Drive:** `drive*.go`, `info_via_drive*.go`
- **Gmail:** `gmail*.go`
- **Sheets:** `sheets*.go`
- **Slides:** `slides*.go`
- **Tasks:** `tasks*.go`
- **People:** `people*.go`
- **Chat:** `chat*.go`
- **Classroom:** `classroom*.go`
- **Keep:** `keep.go`
- **Groups:** `groups*.go`
- **AppScript:** `appscript.go`
- **MCP / agent:** `mcp.go`, `agent*.go`, `completion*.go`
- **Root / help:** `root.go`, `help_printer.go`, `version.go`, `schema.go`, `open.go`, etc.

---

## internal/mcp/

| Path | Description |
|------|-------------|
| `internal/mcp/server/` | MCP server, errors, types |
| `internal/mcp/providers/google/` | Google provider: `tools.go`, `drive_tools.go`, `gmail_tools.go`, `docs_tools.go`, `sheets_tools.go`, `slides_tools.go`, `calendar_tools.go`, `contacts_tools.go`, `helpers.go`, `sedmat_policy*.go` |
| `internal/mcp/default.go` | Default MCP config |
| `internal/mcp/transport_stdio.go` | STDIO transport |
| `internal/mcp/google_tools_test.go` | Google tools tests |

---

## internal/tracking/

| Path | Description |
|------|-------------|
| `internal/tracking/config.go` | Tracking config |
| `internal/tracking/crypto.go` | Crypto helpers |
| `internal/tracking/deploy.go` | Deploy |
| `internal/tracking/pixel.go` | Pixel handling |
| `internal/tracking/secrets.go` | Secrets |
| `internal/tracking/worker/` | Worker app (TypeScript: `src/`, `package.json`, `wrangler.toml`, `schema.sql`) |

---

## docs/

| Path | Description |
|------|-------------|
| `docs/merge/` | **Parity/merge workstream (permanent home):** goldens, schemas, runbooks, drift policy, command dossiers. See `docs/merge/README.md`. |
| `docs/merge/README.md` | Explains "merge" = parity/contracts/drift-control; canonical handover = `handover.md` |
| `docs/merge/goldens/` | Native + gws fixture JSON (Gmail labels, drive-ls, 401/403/404, etc.); see `goldens/README.md` |
| `docs/merge/schemas/` | Draft JSON schemas (gmail-labels-get, gmail-labels-list, drive-ls) |
| `docs/merge/commands/` | Command dossiers: gmail-labels-read, drive-read, docs-info-cat, sheets-get, slides-info |
| `docs/merge/CAPTURE-403-RUNBOOK.md` | Maintainer 403 golden capture steps |
| `docs/merge/discovery-drift-policy.md` | Pin vs accept+detect; §7 google_reason drift-only |
| `docs/merge/GWS-SAMPLES.md` | gws stdout/stderr samples and capture commands |
| `docs/merge/NATIVE-ENVELOPE-SAMPLES.md` | Native envelope examples |
| `docs/merge/command-migration-matrix.md` | Per-command migration and risk |
| `docs/merge/HANDOFF-FOR-REVIEWER.md` | Paste-ready samples for reviewers |
| `docs/refactor/` | Refactor notes, task evals, external review feedback |
| `docs/editing.md` | Docs/Sheets editing (user-facing) |
| `docs/openclaw-linode-runbook.md` | Linode + OpenClaw MCP deploy |
| `docs/TOOLS-gog-agentic-section.md` | MCP tool reference |
| `docs/gws-on-linode.md` | gws setup on Linode |
| `docs/gws-auth-start-over.md` | gws auth reset / wrong app |
| `docs/spec.md` | API/spec notes |
| `docs/sedmat.md` | Sedmat routing/behavior |
| `docs/maton-vs-gog-parity.md` | Maton vs gog parity comparison |
| `docs/PROJECT-LAYOUT.md` | This file — directory and file structure |
| `docs/index.html` | Site entry |
| `docs/assets/` | site.css, site.js, site.more.css |
| `docs/examples/` | Example payloads (e.g. docs-edit-batch.json) |
| Other `docs/*.md` | Auth, contacts, email tracking, MCP plan, releasing, etc. |

---

## gogcli-developer-handover/

| Path | Description |
|------|-------------|
| `gogcli-developer-handover/HANDOVER.md` | **Reference** implementation plan (canonical handover is repo-root `handover.md`) |
| `gogcli-developer-handover/artifacts/` | Zips: `envelope-artifacts-v2.zip`, `gmail-error-taxonomy-lock.zip` (do not modify; treat as source) |
| `gogcli-developer-handover/templates/` | `PARITY-RUNNER-README.md`, `gog-parity-skeleton.go`, `github-actions-parity.yml` |

---

## scripts/

| Path | Description |
|------|-------------|
| `scripts/deploy.sh` | Deploy to Linode (pull, build, copy, restart) |
| `scripts/live-test.sh` | Live test runner |
| `scripts/live-tests/` | Per-service live test scripts (calendar, docs, drive, gmail, sheets, etc.) |
| `scripts/release.sh` | Release workflow |
| `scripts/install.sh`, `scripts/setup.sh` | Install/setup |
| `scripts/gog-auth-diagnostic-report.sh` | Auth diagnostics |
| `scripts/mcp-diagnose-gog.sh` | MCP diagnostics |
| `scripts/get-drive-only-credentials.py` | One-off Drive-only OAuth for 403 capture |
| Other `scripts/*.sh`, `*.go`, `*.mjs` | Helpers, generators, verification |

---

## .github/

| Path | Description |
|------|-------------|
| `.github/workflows/ci.yml` | CI workflow |
| `.github/workflows/release.yml` | Release workflow |

---

## Other root files (planning / deliverables)

- `DOCS_AGENT_PLAN.md` — Docs/agent roadmap
- `DELIVERABLES_SUMMARY.md` — Deliverables summary
- `google_gogcli_merge_plan.md` — Merge plan
- `PROJECT_PLAN.md`, `PLANNING_README.md` — Planning
- `PHASE_3_IMPLEMENTATION_PLAN.md` — Phase 3 plan
- `SLIDES_ROADMAP.md` — Slides roadmap
- `TS-Go-review.md` — TypeScript/Go review notes
- `linode.env` — Linode SSH env (optional; do not commit secrets)
- `package.json` / `package-lock.json` — Optional Node/pnpm (e.g. `pnpm gog`) |

---

## Summary

- **CLI:** `cmd/gog` → `internal/cmd` (per-service command code). **Parity (planned):** `cmd/gog-parity` → `internal/parity`.
- **APIs & auth:** `internal/googleapi`, `internal/googleauth`, `internal/config`, `internal/secrets`.
- **MCP:** `internal/mcp` (server + Google provider).
- **Handover:** `handover.md` (canonical); `gogcli-developer-handover/HANDOVER.md` (reference). PR checklist: no duplicate handover docs.
- **Merge/parity specs:** `docs/merge/` (permanent: goldens, schemas, runbooks, drift policy); see `docs/merge/README.md`.
- **Build/test:** `make`, `make test`, `make lint`, `make ci`; see `AGENTS.md`.
