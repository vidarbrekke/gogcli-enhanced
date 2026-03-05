# gws auth — start over (wrong OAuth app)

If you used an existing app’s OAuth client during `gws auth setup` and want to use the **new** project’s app instead, do this.

## 1. Log out (clear stored tokens)

```bash
gws auth logout
```

This removes the tokens gws had for the wrong app. It does **not** change `client_secret.json`.

## 2. Point gws at the new app’s OAuth client

gws reads the OAuth client from **`~/.config/gws/client_secret.json`**. Right now that file is for the old app. Replace it with the **new** project’s Desktop app credentials.

**Option A — Create new OAuth client in the correct project (gcloud-agent-cli)**

1. Open the Credentials page **with project gcloud-agent-cli preselected**:
   **https://console.cloud.google.com/apis/credentials?project=gcloud-agent-cli**
2. Click **Create credentials** → **OAuth client ID**.
3. If prompted, set **Application type** to **Desktop app** and complete the OAuth consent screen.
4. Click **Create**, then **download the JSON** (download icon on the new client row).  
   Save it to your Downloads folder (browser may name it `client_secret.json` or `client_secret_<id>.json`).
5. Replace gws’s config with the new file. From a terminal (use the path of the file you just downloaded):

   ```bash
   # If the new file is the only client_secret*.json from gcloud-agent-cli:
   cp "$(grep -l '"project_id":"gcloud-agent-cli"' ~/Downloads/client_secret*.json 2>/dev/null | head -1)" ~/.config/gws/client_secret.json
   ```
   Or manually:
   ```bash
   cp ~/Downloads/client_secret.json ~/.config/gws/client_secret.json
   ```
   (Adjust the source path if your browser used a different filename.)

**Option B — You already have the new project’s client JSON**

If you already created a Desktop OAuth client in **gcloud-agent-cli** and have its JSON:

```bash
cp /path/to/new_client_secret.json ~/.config/gws/client_secret.json
```

## 3. Log in again with the new app

```bash
gws auth login
```

Open the URL it prints, sign in, and approve. The redirect will go to the new app’s client; gws will receive the code and save tokens for that app.

**Then export credentials for API calls:**  
gws uses `GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE` (plain JSON) for Gmail/Drive/etc. commands; it does not use the encrypted `credentials.enc` for those. So after login, run:

```bash
gws auth export --unmasked > ~/.config/gws/credentials.json
export GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE=~/.config/gws/credentials.json
```

Add that `export` to your `~/.zshrc` if you want gws API commands to work in every terminal without re-exporting.

## 4. Verify

```bash
gws auth status
gws gmail users labels list --params '{"userId":"me"}'
```

You should see your email and a labels JSON response (not an auth error).

## If you see "Access blocked: ... has not completed the Google verification process"

For **External** (public) apps, Google only allows sign-in for **test users** until the app is verified. The project owner is not automatically a test user — you must add your email.

1. Open the **OAuth consent screen** for your project (e.g. gcloud-agent-cli):
   **https://console.cloud.google.com/apis/credentials/consent?project=gcloud-agent-cli**
2. Scroll to **Test users**.
3. Click **Add users** and add the Google account you use for `gws auth login` (e.g. `your@gmail.com`).
4. Save. Then try `gws auth login` again.

You can add up to 100 test users. No verification is required for test users.

## Summary

| Step | Command / action |
|------|-------------------|
| 1 | `gws auth logout` |
| 2 | In Console: project **gcloud-agent-cli** → Credentials → Create OAuth client ID (Desktop) → Download JSON → save as `~/.config/gws/client_secret.json` |
| 3 | `gws auth login` (open URL, sign in, allow) |
| 4 | `gws auth status` and run a small gws command to confirm |
