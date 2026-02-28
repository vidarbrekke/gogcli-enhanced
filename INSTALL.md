# gogcli — Installation and usage

Step-by-step guide to install **gog** (gogcli) and start using it.

## Prerequisites

- **Google account** (Gmail, Workspace, or both).
- **OAuth 2.0 credentials** from [Google Cloud Console](https://console.cloud.google.com/apis/credentials) (Desktop app type). You’ll create these in the setup steps below.
- For **build from source**: Go 1.21+ (`go version`). If Go is not installed, the install script will download it for you.

---

## 1. Installation

Choose one of the following.

### Option A: Homebrew (macOS / Linux)

```bash
brew install steipete/tap/gogcli
```

Verify:

```bash
gog --version
gog --help
```

### Option B: Arch User Repository (Arch Linux)

```bash
yay -S gogcli
```

Verify:

```bash
gog --version
gog --help
```

### Option C (recommended on Linux): Interactive setup wizard

From the repository root:

```bash
./scripts/setup.sh
```

What it does:

- Linux-only interactive install/reinstall flow
- Lets user choose install target (`~/.local/bin` recommended, or `/usr/local/bin`)
- Detects dependencies and asks permission before apt installs
- Supports reinstall modes:
  - preserve config
  - backup + reset config
  - clean reset (remove config + installed binary)
- Optional auth/keyring bootstrap
- Optional minimal MCP client template generation

### Option D: Build from source (manual)

1. Clone the repository:

   ```bash
   git clone https://github.com/steipete/gogcli.git
   cd gogcli
   ```

   For the enhanced fork (MCP, agentic features):

   ```bash
   git clone https://github.com/vidarbrekke/gogcli-enhanced.git
   cd gogcli-enhanced
   ```

2. Run the install script (checks for Go and downloads it if missing, then builds):

   ```bash
   ./scripts/install.sh
   ```

   This produces `./bin/gog`. If you already have Go 1.24+ installed, you can instead run `make` directly.

3. Run the CLI (from the repo root):

   ```bash
   ./bin/gog --version
   ./bin/gog --help
   ```

4. (Optional) Install to your PATH:

   ```bash
   sudo cp bin/gog /usr/local/bin/gog
   # or
   cp bin/gog ~/bin/gog   # ensure ~/bin is in your PATH
   ```

---

## 2. First-time setup

Before using Gmail, Drive, Calendar, etc., you must store OAuth credentials and authorize an account.

### Step 1: Create OAuth credentials (Google Cloud)

1. Open [Google Cloud Console](https://console.cloud.google.com/apis/credentials).
2. Create or select a project.
3. Enable the APIs you need (e.g. [Gmail API](https://console.cloud.google.com/apis/api/gmail.googleapis.com), [Drive API](https://console.cloud.google.com/apis/api/drive.googleapis.com)).
4. Go to **Credentials** → **Create credentials** → **OAuth client ID**.
5. Application type: **Desktop app**.
6. Download the JSON file (e.g. `client_secret_....apps.googleusercontent.com.json`).

### Step 2: Store credentials in gog

```bash
gog auth credentials ~/Downloads/client_secret_....json
```

Use the path to your downloaded file. For multiple OAuth clients (e.g. work vs personal):

```bash
gog --client work auth credentials ~/Downloads/work-client.json
gog auth credentials list
```

### Step 3: Authorize your Google account

```bash
gog auth add you@gmail.com
```

A browser window opens for Google sign-in and consent. The refresh token is stored in your system keychain (or configured keyring backend).

**Headless / remote server (no browser on the machine):**

```bash
gog auth add you@gmail.com --services user --manual
```

Follow the printed URL in a browser on another device, then paste the redirect URL back into the terminal when prompted.

### Step 4: Verify authentication

```bash
export GOG_ACCOUNT=you@gmail.com
gog gmail labels list
```

If you see your Gmail labels, installation and auth are complete.

---

## 3. How to use gog

### Running commands

- **Help:** `gog --help`, `gog gmail --help`, `gog drive --help`, etc.
- **Version:** `gog --version` or `gog version`.
- **Account:** Use `--account <email>` or set `GOG_ACCOUNT=you@gmail.com`; otherwise the default or only stored account is used.

### Output modes

- **Human-readable (default):** Tables and formatted text on stdout.
- **JSON (scripting):** `gog --json ...` or `gog ... --json`.
- **Plain TSV:** `gog --plain ...` for stable, parseable output.

Progress and hints go to stderr so you can pipe JSON to `jq` or other tools.

### Common workflows

| Task | Example |
|------|--------|
| Search Gmail | `gog gmail search 'newer_than:7d' --max 10` |
| List Drive files | `gog drive ls --max 20` |
| List calendars | `gog calendar calendars` |
| Today’s events | `gog calendar events primary --today` |
| Download a file | `gog drive download <fileId> --out ./file.pdf` |
| Edit a Doc | `gog docs edit replace <docId> "old" "new"` |

Full command reference: see **[README.md](README.md)** (Gmail, Calendar, Drive, Docs, Sheets, Slides, Tasks, Contacts, etc.).

### Multiple accounts

```bash
gog gmail labels list --account work@company.com
gog gmail labels list --account personal@gmail.com
```

Or set a default:

```bash
export GOG_ACCOUNT=work@company.com
gog gmail search 'is:unread'
```

Use `gog auth list` to see stored accounts and `gog auth alias set work work@company.com` for short names.

---

## 4. MCP server (optional)

If you use the **gogcli-enhanced** fork and want to expose gog as an MCP (Model Context Protocol) server for AI agents:

```bash
gog mcp serve
```

The server runs over stdio (newline-delimited JSON-RPC). Configure your MCP client to run `gog mcp serve` as the transport command. Tools include Docs plan/execute batch and Drive ensure-folder, untrash, and get-permission. See **docs/mcp-tooling.md** for tool schemas and behavior.

---

## 5. Configuration and security

- **Config path:** Shown by `gog auth keyring` or `gog --help` (e.g. macOS: `~/Library/Application Support/gogcli/config.json`).
- **Keyring:** Default is `auto` (OS keychain). For headless/CI use `GOG_KEYRING_BACKEND=file` and `GOG_KEYRING_PASSWORD=...`.
- **Best practice:** Never commit OAuth client JSON files; store them outside the repo and use `gog auth credentials <path>` once per machine/client.

For more on auth, keyring, service accounts, and scopes, see **README.md** → Authentication & Secrets and Configuration.

---

## 6. Getting help

- **CLI help:** `gog --help`, `gog <command> --help`, or `GOG_HELP=full gog --help`.
- **Make shortcut (from source):** `make gog -- --help` or `make gog -- gmail --help`.
- **Full docs:** [README.md](README.md) and files in **docs/** (e.g. `docs/editing.md`, `docs/auth-clients.md`).

---

## Quick reference

| Step | Command |
|------|--------|
| Install (Homebrew) | `brew install steipete/tap/gogcli` |
| Install (source) | `git clone ... && ./scripts/install.sh` (or `make` if Go is installed) |
| Store OAuth client | `gog auth credentials <path-to-json>` |
| Add account | `gog auth add you@gmail.com` |
| List accounts | `gog auth list` |
| Use an account | `gog ... --account you@gmail.com` or `GOG_ACCOUNT=...` |
| JSON output | `gog --json ...` |
| MCP server | `gog mcp serve` |
