# Capture gws 403 golden (maintainer only)

The developer does **not** have access to any app or environment. The **403 (permission denied)** golden must be captured by someone with gws + Google Cloud Console access (e.g. repo maintainer) and committed so the developer can freeze the Gmail error taxonomy.

**If you see “The OAuth client was deleted”:** You are using a client that was removed in Google Cloud Console. Restore your working gws client (e.g. `cp ~/.config/gws/client_secret-gcloud-agent-cli.json ~/.config/gws/client_secret.json` if you have that backup) and use a **new**, non-deleted OAuth client for the 403 capture below.

## One-time steps (on your Mac, where gws is already set up)

1. **Create a new OAuth client (do not reuse a deleted or unknown client)**
   - Open [Credentials](https://console.cloud.google.com/apis/credentials) and select a project (e.g. gcloud-agent-cli).
   - Create credentials → OAuth client ID → **Desktop app**.
   - Download the JSON and save as `~/.config/gws/client_secret-no-gmail.json`.
  - **Optional for “no Gmail”:** In the OAuth consent screen for that project, limit which scopes are available, or use a separate Desktop OAuth client dedicated to the Drive-only capture flow. Then export the resulting credentials to `credentials-no-gmail.json`. Otherwise, use the client for `gws auth login` and hope the consent screen only grants limited scope (gws normally requests Gmail too).

2. **Use that client for gws (without touching your main gws auth)**
   ```bash
   cp ~/.config/gws/client_secret.json ~/.config/gws/client_secret-backup.json
   cp ~/path/to/downloaded_client_secret_no_gmail.json ~/.config/gws/client_secret.json
   gws auth logout
   gws auth login
   ```
   Complete the browser flow (you’re now authorized with no Gmail scope).

3. **Export and capture 403**
   ```bash
   gws auth export --unmasked > ~/.config/gws/credentials-no-gmail.json
   cd /path/to/gogcli-enhanced
   GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE=~/.config/gws/credentials-no-gmail.json gws gmail users labels list --params '{"userId":"me"}' > docs/merge/goldens/gmail-labels-403-forbidden-gws.json
   ```

4. **Record capture metadata (optional, high ROI)**  
   Prevents “why does my 403 not match yours?” churn later without increasing contract strictness. In the same directory, create `gmail-labels-403-forbidden-gws.capture-info.txt`    with these lines (fill in the values from your run). The first three are the main ones; the fourth is optional — add only if present.
   ```
   gws version: <output of: gws --version>
   command argv: gws gmail users labels list --params '{"userId":"me"}'
   OAuth scopes: Drive-only (e.g. https://www.googleapis.com/auth/drive.readonly)
   profile/creds: <path or profile name, if gws uses one — e.g. GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE path or keyring profile>
   ```
   Example (three lines):
   ```
   gws version: 0.6.0
   command argv: gws gmail users labels list --params '{"userId":"me"}'
   OAuth scopes: https://www.googleapis.com/auth/drive.readonly
   ```
   Example (four lines, when you use a specific creds path):
   ```
   gws version: 0.6.0
   command argv: gws gmail users labels list --params '{"userId":"me"}'
   OAuth scopes: https://www.googleapis.com/auth/drive.readonly
   profile/creds: /Users/me/.config/gws/credentials-no-gmail.json
   ```

5. **Restore your main gws client**
   ```bash
   cp ~/.config/gws/client_secret-backup.json ~/.config/gws/client_secret.json
   gws auth login
   gws auth export --unmasked > ~/.config/gws/credentials.json
   export GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE=~/.config/gws/credentials.json
   ```

6. **Edit the golden**  
   Open `docs/merge/goldens/gmail-labels-403-forbidden-gws.json` and remove any `_replace_with_*` key so the file contains only the `error` object from gws stdout.

7. **Commit and push**
   ```bash
   git add docs/merge/goldens/gmail-labels-403-forbidden-gws.json docs/merge/goldens/gmail-labels-403-forbidden-gws.capture-info.txt docs/merge/CAPTURE-403-RUNBOOK.md
   git commit -m "docs(merge): add 403 golden and capture runbook for developer"
   git push
   ```

8. **Upload for promotion-ready bundle**  
   Whenever you’re ready, upload the **real 403 JSON file** (and optionally the capture-info file) to the developer. They’ll produce the **final promotion-ready bundle** in one shot (403 as hard CI requirement alongside 401 and 404).

After this, the developer has a real 403 golden and can complete the Gmail error taxonomy mapping without needing any app or environment.

---

## After you capture the real 403

**Upload** the maintainer's real 403 stdout (the contents of `gmail-labels-403-forbidden-gws.json` after capture) to the developer. Whenever you're ready, upload that JSON (and optionally `gmail-labels-403-forbidden-gws.capture-info.txt` if you recorded it) and they'll produce the **final promotion-ready bundle** in one shot — 403 becomes a hard CI requirement alongside 401 and 404.

**What the developer will do in the promotion-ready bundle (no new moving parts, no "keep up with Google" tax):**
- Replace the 403 placeholder golden with the real stdout JSON.
- Include the optional `gmail-labels-403-forbidden-gws.capture-info.txt` alongside that golden as non-test metadata.
- Update the CI gate doc so 403 is **hard required** (same as 401 + 404).
- Keep `google_reason` **drift-only**, per drift policy §7 (no change).
