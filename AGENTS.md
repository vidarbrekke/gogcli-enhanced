# Repository Guidelines

## Project Structure

- `cmd/gog/`: CLI entrypoint.
- `internal/`: implementation (`cmd/`, Google API/OAuth, config/secrets, output/UI).
- Tests: `*_test.go` next to code; opt-in integration suite in `internal/integration/` (build-tagged).
- `bin/`: build outputs; `docs/`: specs/releasing; `scripts/`: release helpers + `scripts/gog.mjs`.

## Build, Test, and Development Commands

- `make` / `make build`: build `bin/gog`.
- `make tools`: install pinned dev tools into `.tools/`.
- `make fmt` / `make lint` / `make test` / `make ci`: format, lint, test, full local gate.
- Optional: `pnpm gog …`: build + run in one step.
- Hooks: `lefthook install` enables pre-commit/pre-push checks.

## Coding Style & Naming Conventions

- Formatting: `make fmt` (`goimports` local prefix `github.com/steipete/gogcli` + `gofumpt`).
- Output: keep stdout parseable (`--json` / `--plain`); send human hints/progress to stderr.
- Gmail labels: treat label IDs as case-sensitive opaque tokens; only case-fold label names for name lookup.

## Testing Guidelines

- Unit tests: stdlib `testing` (and `httptest` where needed).
- Integration tests (local only):
 - `GOG_IT_ACCOUNT=you@gmail.com go test -tags=integration ./internal/integration`
 - Requires OAuth client credentials + a stored refresh token in your keyring.

## Commit & Pull Request Guidelines

- Create commits with `committer "<msg>" <file...>`; avoid manual staging.
- Follow Conventional Commits + action-oriented subjects (e.g. `feat(cli): add --verbose to send`).
- Group related changes; avoid bundling unrelated refactors.
- PRs should summarize scope, note testing performed, and mention any user-facing changes or new flags.
- PR review flow: when given a PR link, review via `gh pr view` / `gh pr diff` and do not change branches.

### PR Workflow (Review vs Land)

- **Review mode (PR link only):** read `gh pr view/diff`; do not switch branches; do not change code.
- **Landing mode:** temp branch from `main`; bring in PR (squash default; rebase/merge when needed); fix; update `CHANGELOG.md` (PR #/issue + thanks); run `make ci`; final commit; merge to `main`; delete temp; end on `main`.
- If we squash, add `Co-authored-by:` for the PR author when appropriate; leave a PR comment with what landed + SHAs.
- New contributor: thank in `CHANGELOG.md` (and update README contributors list if present).

## Security & Configuration Tips

- Never commit OAuth client credential JSON files or tokens.
- Prefer OS keychain backends; use `GOG_KEYRING_BACKEND=file` + `GOG_KEYRING_PASSWORD` only for headless environments.

## Agentic Workflow Rules

- Prefer `--json` for machine-driven workflows; treat text output as human-only.
- For Docs editing, default sequence:
 1. `gog docs edit batch <docId> --validate-only --pretty --output-request-file <normalized.json>`
 2. inspect/approve normalized payload + `requestHash`
 3. execute with `gog docs edit batch <docId> --execute-from-file <normalized.json> --require-revision <revId>`
- Use `--dry-run` before destructive or high-impact operations.
- `gog docs edit delete` requires explicit intent (`--force`) in non-JSON human mode.
- Parse JSON stderr error envelopes and branch on `error.error_code` (avoid message-string matching where possible).
- Persist `requestHash`, `doc_id`, and request file path in agent logs for replayability and audit trails.

### Third-party comparison (Maton)

- For Maton vs gog parity (both use native Google APIs; gog adds MCP/convenience tools), see `docs/maton-vs-gog-parity.md`.

### When to use batch vs sedmat (MCP / CLI)

- **Batch (`docs.planBatch` / `docs.executeBatch`, or `gog docs edit batch`):** Use for structured, multi-op Docs API requests when you need `requestHash`, revision safety, and a strict validate → approve → execute flow. Best for deterministic agent runs and audit trails.
- **Sedmat (`docs.sed` or `gog docs sed`):** Use for expression-based find/replace, positional edits (^, $), table/image ops, or when the intent is a small number of sed-like expressions. Prefer `--json` and `-n` (dry-run) before running for real.
- **Smart edit (`docs.smartEdit`):** Use when the agent should let the stack choose. Low-risk edits (e.g. single plain replace) are auto-executed; high-risk (delete, clear doc, table/image delete, merge/split) returns an assessment with `riskLevel`, `decisionReason`, and `requiresConfirmation: true` and does not write until the caller confirms or uses an explicit tool with dry-run first.
- **High-impact edits (safe two-step):** For destructive or broad changes (delete commands, clear document, table/image delete, merge/split), always call with `validateOnly: true` first, then branch on `riskLevel` and `requiresConfirmation`; if high, run `docs.sed` with `dryRun: true` or use batch validate → execute flow before performing the write.

## Agent Discipline

- Do not perform broad refactors unless explicitly requested.
- Do not upgrade dependencies or change toolchain versions without approval.
- Prefer minimal diffs over cleanup-only changes.
- For bug fixes: identify root cause before proposing changes.
- Do not restate large diffs or logs in responses unless necessary.
