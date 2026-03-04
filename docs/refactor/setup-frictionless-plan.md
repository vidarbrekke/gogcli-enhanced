# Frictionless setup plan (gogcli-enhanced)

Context: Feedback from using gogcli-enhanced in a backup app (separate repo with its own `setup.sh` / `backup.sh` that consume gog and write `.env.backup`). The “convoluted” feeling comes from OAuth + keyring + headless/cron requirements. This doc maps recommendations onto **this** repo only.

**Backup app repo:** The consumer that uses gog for Linode→Drive backup lives in a separate repo: **openclaw-backup-utils** (user’s GitHub). Keep backup logic and its `setup.sh`/`backup.sh` there; gogcli-enhanced only improves its own setup so that repo can rely on `./scripts/setup.sh --cli-only` and env for `GOG_ACCOUNT` / `GOG_KEYRING_PASSWORD`.

**Scope:** `scripts/setup.sh`, `scripts/setup-doctor.sh`, auth/keyring behavior, and docs — to make the golden-path and CLI-only/cron paths clearer and less fragile.

---

## 1. Clarify: two different setups

| Where | Purpose |
|-------|--------|
| **openclaw-backup-utils** (separate repo) | Backup app’s own `setup.sh` / `backup.sh`; writes `GOG_ACCOUNT` to `.env.backup`, calls gog for uploads. Should assume gog was configured once via gogcli-enhanced’s setup (e.g. `--cli-only`). |
| **gogcli-enhanced** (this repo) | `scripts/setup.sh`: build gog, OAuth credentials, keyring, account auth, and optionally MCP/OpenClaw registration. Single entry point for “get gog working.” |

Frictionless improvements below apply to **gogcli-enhanced** so that any consumer (backup script, cron, MCP) has a clear, one-time setup story.

---

## 2. Recommendations mapped to gogcli-enhanced

### 2.1 Streamline OAuth credentials import

**Current:** `scripts/setup.sh` reuses existing `credentials.json` or prompts for Client ID + Secret (paste). No guided “go to Cloud Console” flow.

**Ideas (no code yet):**
- **Interactive guide:** In setup, if no valid creds: print a strict numbered sequence: “1) Open URL X. 2) Create Desktop OAuth client. 3) Download JSON. 4) Either run `gog auth credentials /path/to/file` or paste path here.” Optionally open the Cloud Console URL in a browser when not headless.
- **Single paste:** Allow “paste path to downloaded JSON” as first-class option in the credentials step (already supported via `gog auth credentials <path>`; ensure setup.sh offers it clearly).

**Files:** `scripts/setup.sh` (credentials block in `configure_auth`), `docs/openclaw-linode-runbook.md` (headless credential steps).

---

### 2.2 Simplify headless account authorization

**Current:** `is_cloud_context` triggers `--remote --step 1` / `--step 2` with “paste redirect URL.” Prompts are one line each.

**Ideas:**
- Add a clear banner before the URL: **“ACTION REQUIRED ON YOUR LOCAL MACHINE”** and “Paste the **entire** URL from your browser’s address bar after you click Allow.”
- Optional: detect `--manual` and surface the same wording in `gog auth add` help so headless users see it when they run the command manually.

**Files:** `scripts/setup.sh` (block around “Open this URL in your browser”), `scripts/setup-doctor.sh` (same). Optionally `internal/cmd/auth.go` help text for `auth add --manual`.

---

### 2.3 Automate GOG_KEYRING_PASSWORD for cron / non-interactive

**Current:** Setup offers (1) persist to shell rc (`GOG_KEYRING_PASSWORD` in `.bashrc`/`.zshrc`), or (2) write `keyring.password` and set `GOG_KEYRING_PASSWORD_FILE` in mcporter.json for MCP. No explicit “cron” or “script” path.

**Ideas (with explicit consent):**
- **Cron:** After configuring file keyring, ask: “Will you run gog from cron or a script? If yes, we can add a one-line env to your crontab (you’ll see the line before it’s written).” If yes, show the exact `export GOG_KEYRING_PASSWORD='...'` line and offer to append to `crontab -e` or a small wrapper script.
- **One-time .env.gog:** Option to write `~/.config/gogcli/.env.gog` (chmod 600) with `GOG_KEYRING_BACKEND=file` and `GOG_KEYRING_PASSWORD=...`, and print: “For scripts/cron, source this file or export these variables.”
- **Warning:** If user declines persistence, print a clear line: “For cron/scripts you must set GOG_KEYRING_PASSWORD or GOG_KEYRING_PASSWORD_FILE in the environment; otherwise gog will fail when run non-interactively.”

**Files:** `scripts/setup.sh` (`persist_keyring_env_auto`, and after `configure_keyring_file`), `docs/openclaw-linode-runbook.md` (§4 Option A2, cron note). AGENTS.md already mentions `GOG_KEYRING_PASSWORD` for headless.

---

### 2.4 CLI-only path

**Current:** Setup always runs MCP registration, mcporter daemon, TOOLS.md injection, and gateway restart. Users who only want `gog drive upload` (e.g. from a backup script) still get the full OpenClaw path.

**Idea:**
- Add **`--cli-only`** to `scripts/setup.sh`: when set, skip `configure_openclaw_mcp_auto` and any mcporter/gateway steps. Flow becomes: build → credentials → keyring → account auth → path hint → done. Document in usage and runbook.

**Files:** `scripts/setup.sh` (parse `--cli-only`, conditional around `configure_openclaw_mcp_auto`), `docs/openclaw-linode-runbook.md` (when to use `--cli-only` vs full setup).

---

### 2.5 Better errors and self-healing

**Current:** Errors like keyring unwrap failures are technical; setup-doctor exists but isn’t always suggested.

**Ideas:**
- **User-facing messages:** Where we surface keyring/auth errors, add one line: “If this is a wrong keyring password or expired token, try: gog auth manage (or see docs/openclaw-linode-runbook.md).”
- **Diagnostics:** In setup-doctor or a small `scripts/auth-check.sh`, run `gog auth status --debug` (or equivalent) and suggest fixes for common failures (e.g. “GOG_KEYRING_PASSWORD not set in this environment”).

**Files:** Keyring error paths in `internal/` (if any user-facing strings), `scripts/setup-doctor.sh`, `scripts/mcp-diagnose-gog.sh`, runbook §8.

---

## 3. Suggested order of work

**Done (this release):**
- **`--cli-only`** added to setup.sh; documented in runbook and usage.
- **Headless prompts:** “ACTION REQUIRED ON YOUR LOCAL MACHINE” banner and “paste the **entire** URL from your browser’s address bar” in setup.sh and setup-doctor.sh.
- **Cron/keyring:** One sentence after keyring setup: “For cron or non-interactive scripts: ensure GOG_KEYRING_PASSWORD or GOG_KEYRING_PASSWORD_FILE is set in the environment.”

1. **Low risk, high clarity** (remaining)
   - ~~Add **`--cli-only`** and document it (backup/cron users run setup once with `--cli-only`).~~
   - ~~Harden headless prompts (banner + “paste entire URL”).~~
   - ~~Add one sentence after keyring setup: “For cron/scripts, ensure GOG_KEYRING_PASSWORD or GOG_KEYRING_PASSWORD_FILE is set in the environment.”~~
2. **Medium**
   - Optional cron/.env.gog offer (with consent and clear explanation).
   - OAuth credentials: numbered “go to Cloud Console” steps when no creds found.
3. **Ongoing**
   - Replace cryptic keyring errors with a single “try gog auth manage / runbook” line where appropriate.

---

## 4. What not to change (without approval)

- No change to server/package/app **versions** or toolchain.
- No broad refactor of setup.sh structure; only additive or minimal edits (flags, prompts, one extra question).
- Backup app’s own `setup.sh`/`backup.sh` stay in that repo; this plan only improves gogcli-enhanced so that app’s setup can assume “run gogcli-enhanced’s setup once with --cli-only, then source .env.backup / set GOG_ACCOUNT and GOG_KEYRING_PASSWORD in cron.”
