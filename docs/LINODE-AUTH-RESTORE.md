# Linode auth restore (2026-03-06)

## What was wrong

After deploying the then-current development branch (`merge-upstream-2026-03`, later merged and deleted), `gog auth status` showed no config and no credentials. **Root cause:** not a new repo or different app name — this codebase has always used `gogcli` and `/root/.config/gogcli/`. The problem was:

1. **`config.json` was missing** — so the CLI reported `config.exists: false` and never showed account/credentials even when `-a` was used.
2. **Keyring directory was empty** — no token files, so no refresh tokens for any account. Credentials.json was present but tokens live in the keyring.
3. **Backups existed** — `/root/.config/gogcli-backups/` had timestamped copies. Backup `20260228-195807` contained `config.json`, `credentials.json`, and a populated `keyring/` (tokens for `vidarbrekke@gmail.com`).

## What was done

- Restored from backup **20260228-195807** into `/root/.config/gogcli/`:
  - `config.json` (sets `keyring_backend: "file"`)
  - `credentials.json`
  - `keyring/` (including `token:default:vidarbrekke@gmail.com` and `token:vidarbrekke@gmail.com`)
- **Result:** `gog auth status -a vidarbrekke@gmail.com --json` now shows `config.exists: true`, `credentials_exists: true`, `email: vidarbrekke@gmail.com`, `keyring.backend: file`.

## What you still need to do

The **file** keyring is encrypted. To use `gog` non-interactively (SSH, cron, MCP), the process must have the keyring password:

- **Option A:** Set the password in the environment when you run gog:
  ```bash
  export GOG_KEYRING_PASSWORD='<the password you used when creating the keyring>'
  /root/.local/bin/gog gmail labels list -a vidarbrekke@gmail.com --json
  ```
- **Option B (recommended for headless):** Put the password in a file and point the CLI at it:
  1. Create a file containing only the keyring password (one line), e.g. `/root/.config/gogcli/keyring.password`.
  2. `chmod 600 /root/.config/gogcli/keyring.password`
  3. When running gog (or in mcporter.json `env` for the gog MCP server), set:
     ```bash
     export GOG_KEYRING_PASSWORD_FILE=/root/.config/gogcli/keyring.password
     ```

Then:

```bash
GOG_KEYRING_PASSWORD_FILE=/root/.config/gogcli/keyring.password /root/.local/bin/gog gmail labels list -a vidarbrekke@gmail.com --json
```

If you use mcporter/OpenClaw, add `GOG_KEYRING_PASSWORD_FILE` to the gog-agentic entry’s `env` in mcporter.json so the MCP server can unlock the keyring. See `docs/openclaw-linode-runbook.md` §4.

## Auth from SSH (Linode): paste redirect URL

When you run `gog auth add ...` over SSH, the CLI starts a callback server on **the server’s** 127.0.0.1. You open the auth URL in **your** browser; after login, Google redirects to that host:port on **your** machine, so the page “doesn’t load” and the server never sees the callback. You must use a flow where you **paste the redirect URL** back into the terminal.

**Option 1 — Manual (interactive paste, recommended)**  
Run with `--manual` so the CLI prompts you to paste the URL:

```bash
export GOG_KEYRING_PASSWORD='<keyring-password>'
gog auth add vidarbrekke@gmail.com --services gmail --manual
```

1. Open the URL it prints in your browser and sign in.
2. After redirect, the browser will show a URL that doesn’t load (e.g. `http://127.0.0.1:35709/oauth2/callback?code=...&state=...`).
3. Copy the **full** URL from the address bar and paste it into the SSH terminal at the “Paste redirect URL” prompt, then Enter.
4. The CLI will exchange the code for tokens and store them.

**Option 2 — Remote two-step (no prompt)**  
Step 1 on the server (prints URL only):

```bash
gog auth add vidarbrekke@gmail.com --services gmail --remote --step 1
```

Open the printed URL in your browser; after redirect, copy the full redirect URL. Then step 2 on the server:

```bash
gog auth add vidarbrekke@gmail.com --services gmail --remote --step 2 --auth-url 'http://127.0.0.1:35709/oauth2/callback?code=...&state=...'
```

(Paste your actual redirect URL in place of the example above.)

## Summary

- **Auth was not broken by the new repo** — same paths and app name.
- **Config and keyring had been lost or never fully present** on this box; backup contained them.
- **Restore is done**; you only need to supply the keyring password (env or file) for non-interactive use.
