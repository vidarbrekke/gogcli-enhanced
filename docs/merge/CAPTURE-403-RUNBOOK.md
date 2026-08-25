# Capture gws 403 golden (maintainer only)

The **403 (permission denied)** golden must be a real gws stdout capture (credentials without Gmail scope). Until capture lands, `docs/merge/goldens/gmail-labels-403-forbidden/gws/PLACEHOLDER.txt` keeps the case soft-skipped. The parity runner already **hard-gates** `gmail-labels-403-forbidden` once the placeholder is removed (`permission_denied` / HTTP 403).

**Preferred one-shot:** from repo root, run:

```bash
scripts/capture-403-golden.sh
```

That backs up your current gws export, opens a **drive-only** OAuth browser flow (`gws auth login -s drive --readonly`), writes the golden under `docs/merge/goldens/gmail-labels-403-forbidden/gws/`, removes `PLACEHOLDER.txt`, then runs full `gws auth login` to restore normal scopes. Then `make parity` and commit.

**Note:** A stale `~/.config/gws/credentials-no-gmail.json` (revoked refresh token) makes `GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE=...` look like “No credentials provided” (401). Re-export after a successful drive-only login; do not reuse an old no-gmail file blindly.

**If you see “The OAuth client was deleted”:** Restore a working Desktop OAuth client into `~/.config/gws/client_secret.json` before running the script.

## Manual steps (same as the script)

1. Export current auth, then drive-only login:
   ```bash
   gws auth export --unmasked > /tmp/gws-credentials-full.json
   gws auth login -s drive --readonly
   gws auth export --unmasked > ~/.config/gws/credentials-no-gmail.json
   ```

2. Capture into the case directory:
   ```bash
   cd /path/to/gogcli-enhanced
   set +e
   GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE=~/.config/gws/credentials-no-gmail.json \
     gws gmail users labels list --params '{"userId":"me"}' \
     > docs/merge/goldens/gmail-labels-403-forbidden/gws/stdout.json
   echo $? > docs/merge/goldens/gmail-labels-403-forbidden/gws/exit_code.txt
   set -e
   printf '{}\n' > docs/merge/goldens/gmail-labels-403-forbidden/gws/stderr.json
   rm -f docs/merge/goldens/gmail-labels-403-forbidden/gws/PLACEHOLDER.txt
   ```

3. Optional `capture-info.txt` beside the golden:
   ```
   gws version: <gws --version>
   command argv: gws gmail users labels list --params '{"userId":"me"}'
   OAuth scopes: https://www.googleapis.com/auth/drive.readonly
   profile/creds: ~/.config/gws/credentials-no-gmail.json
   ```

4. Restore full scopes: `gws auth login`

5. Verify and commit:
   ```bash
   make parity
   git add docs/merge/goldens/gmail-labels-403-forbidden/gws/ docs/merge/CAPTURE-403-RUNBOOK.md
   git commit -m "test(parity): capture real gmail labels 403 golden"
   git push
   ```

After the placeholder is gone, CI treats 403 like 401/404: normalize must yield `permission_denied` / HTTP 403 or the run fails.
