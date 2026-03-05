# gws on Linode (one-time setup + capture)

**gws** (Google Workspace CLI) is installed globally on the Linode server via `npm install -g @googleworkspace/cli`. Use this flow to log in once (on a machine with a browser), then use the credentials on Linode to run gws and capture parity goldens.

---

## 0. One-time: install gws and authenticate (on your laptop)

**Standard way (recommended): use gcloud so you never touch the Cloud Console**

If you have the [Google Cloud SDK (gcloud)](https://cloud.google.com/sdk/docs/install) installed, gws can create the GCP project and OAuth client for you. You skip creating a project or downloading `client_secret.json` by hand.

```bash
# Install gws
npm install -g @googleworkspace/cli

# One command: creates project + OAuth client + opens browser to log in
gws auth setup
```

`gws auth setup` will (when gcloud is available):

- Create or select a GCP project  
- Create an OAuth client (Desktop app)  
- Enable the APIs gws needs  
- Open the browser for you to sign in and approve scopes  

After it finishes, you are logged in. Skip to §1 to export credentials for Linode.

**Fallback: no gcloud (manual OAuth client)**

If you don’t want to install gcloud, you must create the OAuth client yourself:

1. Open [Google Cloud Console → Credentials](https://console.cloud.google.com/apis/credentials).  
2. Create or select a project.  
3. **OAuth consent screen:** Configure if prompted (External, add your email as test user).  
4. **Credentials → Create credentials → OAuth client ID** → Application type: **Desktop app** → Create → Download the JSON.  
5. Save it so gws can find it:
   ```bash
   mkdir -p ~/.config/gws
   mv ~/Downloads/client_secret_*.json ~/.config/gws/client_secret.json
   ```
6. Then run:
   ```bash
   npm install -g @googleworkspace/cli
   gws auth login
   ```

---

## 1. Export credentials for Linode (on your laptop)

Once you’re logged in (via `gws auth setup` or `gws auth login`), export credentials so the headless Linode server can use gws without a browser:

```bash
# Export credentials (unmasked = full tokens; keep this file secret)
gws auth export --unmasked > ~/gws-credentials.json

# Copy to Linode (from repo root, with linode.env)
source linode.env   # or set SSH_HOST, SSH_USER, SSH_KEY_PATH yourself
ssh -i "$SSH_KEY_PATH" "$SSH_USER@$SSH_HOST" "mkdir -p /root/.config/gws"
scp -i "$SSH_KEY_PATH" ~/gws-credentials.json "$SSH_USER@$SSH_HOST:/root/.config/gws/credentials.json"
```

**Do not commit** `~/gws-credentials.json` or the file on Linode to git.

---

## 2. On the Linode server: set env and run gws

SSH to Linode, then:

```bash
# Point gws at the credentials file (use the path where you copied it)
export GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE=/root/.config/gws/credentials.json

# Verify
gws gmail users labels list --params '{"userId":"me"}'
```

If you see a JSON array of labels (not a 401 error), gws is authenticated.

---

## 3. Capture parity goldens (for gog vs gws merge)

From the repo’s `docs/merge` plan we need two gws outputs. Run these **on Linode** (with the env above set) and save stdout into the repo goldens:

```bash
export GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE=/root/.config/gws/credentials.json

# 1) Labels list (success)
gws gmail users labels list --params '{"userId":"me"}' > /tmp/gmail-labels-list-gws.json

# 2) Labels get not_found (404)
gws gmail users labels get --params '{"userId":"me","id":"Label_DoesNotExist_123"}' > /tmp/gmail-labels-get-not-found-gws.json
```

Then copy the two files from Linode to your laptop into the repo:

```bash
# From your laptop (repo root)
source linode.env
scp -i "$SSH_KEY_PATH" "$SSH_USER@$SSH_HOST:/tmp/gmail-labels-list-gws.json" docs/merge/goldens/
scp -i "$SSH_KEY_PATH" "$SSH_USER@$SSH_HOST:/tmp/gmail-labels-get-not-found-gws.json" docs/merge/goldens/
```

After that, the two golden files can be committed and the reviewer can produce `gmail-labels-list.result.schema.json` and `gmail-labels-get.error.schema.json`.

---

## 4. Where gws lives on Linode

- **Install:** `npm install -g @googleworkspace/cli` (already done).
- **Binary:** `gws` is on PATH (global npm bin).
- **Config dir (optional):** `~/.config/gws/` — put exported credentials here and set `GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE` to that path.

---

## 5. If you prefer to run everything on your laptop

You can skip Linode for the capture: on your laptop run `gws auth login`, then run the two `gws gmail users labels ...` commands and redirect stdout into `docs/merge/goldens/gmail-labels-list-gws.json` and `docs/merge/goldens/gmail-labels-get-not-found-gws.json`. Same result; Linode is only needed if you want gws available on the OpenClaw server too.
