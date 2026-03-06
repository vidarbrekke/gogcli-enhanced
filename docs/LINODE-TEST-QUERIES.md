# Test queries for Linode (deployed gog)

Use these after deploying the latest branch to confirm the binary and config work. Replace `YOU@gmail.com` with your configured account.

---

## 1. No auth required

```bash
# Version and build hash
gog version

# Help (sanity check binary + Kong parsing)
gog --help
gog auth --help
gog gmail labels list --help

# Stable exit codes (agent doc)
gog exit-codes
```

---

## 2. Auth / config (requires at least one account)

```bash
# Auth status — text
gog auth status

# Auth status — JSON (exercises JSON output path)
gog auth status --json

# Default account
gog auth status -a YOU@gmail.com
```

---

## 3. Read-only API (one account)

Use `-a YOU@gmail.com` if you have multiple accounts.

**Gmail**

```bash
# Labels — text table
gog gmail labels list -a YOU@gmail.com

# Labels — JSON (exercises captureStdout/JSON path)
gog gmail labels list -a YOU@gmail.com --json

# Drafts list (first page)
gog gmail drafts list -a YOU@gmail.com --max 3 --json
```

**Drive**

```bash
# List files — text
gog drive ls -a YOU@gmail.com --max 2

# List files — JSON
gog drive ls -a YOU@gmail.com --max 2 --json

# Search (no write)
gog drive search "test" -a YOU@gmail.com --max 1 --json
```

**Calendar**

```bash
# Calendar list
gog calendar list -a YOU@gmail.com --json

# Events (next 7 days)
gog calendar events -a YOU@gmail.com --from today --to 7d --max 5 --json
```

**People / Tasks**

```bash
# Profile
gog people me -a YOU@gmail.com --json

# Task lists
gog tasks list -a YOU@gmail.com --json
```

**Sheets / Docs** (if you have a sheet or doc ID)

```bash
# Sheets metadata (read-only)
gog sheets metadata SPREADSHEET_ID -a YOU@gmail.com --json

# Docs info (read-only)
gog docs info DOC_ID -a YOU@gmail.com --json
```

---

## 4. Dry-run (no writes)

```bash
# Drive upload dry-run
gog drive upload /etc/hostname -a YOU@gmail.com --name "test-upload.txt" -n

# Docs edit dry-run (replace in a doc)
gog docs sed DOC_ID "s/foo/bar/" -a YOU@gmail.com -n
```

---

## 5. MCP server (if mcporter/Cursor uses gog)

If the Linode deploy runs the MCP daemon that exposes gog tools:

- In Cursor/IDE, list MCP tools and confirm `gmail`, `drive`, `docs`, etc. appear.
- Call a read-only tool, e.g. `gmail.labelsList` or `drive.filesList`, with `account: "YOU@gmail.com"` and check JSON result.

---

## Quick smoke sequence (copy-paste)

```bash
gog version && \
gog auth status --json | head -c 200 && echo "..." && \
gog gmail labels list -a YOU@gmail.com --json | head -c 300 && echo "..."
```

If all three succeed (version prints, auth status returns JSON, labels return JSON), the binary, config, and JSON output paths are working.

---

## 6. OpenClaw verification (gog-agentic MCP)

Use these **natural-language prompts in OpenClaw** to confirm the new code and Google integration are working and that requests are routed to gog-agentic.

### 6.1 Verify new code and auth (read-only)

Ask the agent:

1. **"List my Gmail labels."**  
   - **Success:** Agent calls a gog-agentic Gmail tool (e.g. `gmail_labelsList` or equivalent) and returns a list of labels (INBOX, SENT, custom labels).  
   - **Failure:** "unauthorized", "authorize the app", or agent suggests manual steps instead of calling a tool.

2. **"Show my Google Drive root — first 5 files or folders."**  
   - **Success:** Agent calls `drive_listFiles` (or similar) and returns file names/IDs.  
   - **Failure:** Auth error or agent does not call gog-agentic.

3. **"What is my gog auth status for my default Google account?"**  
   - **Success:** Agent uses gog/auth tool or exec to run auth status and reports config exists, credentials exist, account email.  
   - **Failure:** "no credentials" or tool not found.

### 6.2 Verify routing to gog-agentic (not fallback)

Ask:

4. **"Create a new Google Drive folder named OpenClawVerifyTest and tell me its ID."**  
   - **Success:** Agent calls `drive_ensureFolder` (or equivalent), returns a folder ID. You can delete the folder later in Drive.  
   - **Failure:** Agent suggests using the Drive UI or running a CLI command instead of calling the MCP tool.

5. **"List my next 3 calendar events from my primary calendar."**  
   - **Success:** Agent calls a gog-agentic calendar tool and returns event titles/dates.  
   - **Failure:** Tool missing or auth error.

If 1–3 work, the **new code and auth** are working. If 4–5 work, the agent is **routing to gog-agentic** for Drive/Calendar rather than falling back to other methods.

### 6.3 Google solution (gog vs gws)

The current deployment uses **gog** (native Google APIs via gog-agentic MCP). **gws** (Google Workspace CLI) is not wired into the command path yet; it is used only for parity fixtures. So:

- **Expected:** All Google actions (Gmail, Drive, Docs, Calendar, Sheets) go through **gog-agentic** MCP tools (backed by the gog binary and your OAuth tokens).
- **Not applicable yet:** "Verify gws is routed to" — gws is not in the request path for normal agent actions.

To confirm which binary is used: on the server, `which gog` and `gog version`; the MCP server runs that binary with `gog mcp serve`.

---

## 7. Audit trail for debugging

When you need to see what actions were taken (e.g. which tools were called, with what arguments, and what failed):

| Source | What it contains | Where to look |
|--------|------------------|---------------|
| **OpenClaw memory / conversation** | Agent reasoning, tool calls, and responses in the session. | OpenClaw UI: conversation history, memory view, or exported logs if your setup saves them. |
| **Gateway logs** | Incoming requests, which MCP servers were called, high-level errors. | Depends on OpenClaw version: e.g. gateway stdout/stderr, or logs under `~/.openclaw`, `workspace/logs`, or systemd journal if the gateway runs as a service. |
| **Mcporter daemon** | Tool invocations (e.g. `gog-agentic.drive_listFiles`), spawn of `gog mcp serve`, stdio to gog. | Mcporter daemon log (if enabled), or socket/stdio capture. On Linode: check how mcporter is started (e.g. `ensure-mcp-daemon.sh`) and whether it logs to a file or journal. |
| **gog binary** | API calls and errors from the gog CLI when it handles MCP tool requests. | gog does not write a separate audit log by default; errors and responses go back over MCP. For deeper debugging, run `gog mcp serve` in a terminal with verbose/debug if the build supports it, or inspect stderr of the gog subprocess. |

**Recommendation:** Rely on **OpenClaw memory and conversation history** as the primary audit trail for "what did the agent do?" For "why did the tool fail?", check gateway and mcporter logs next; then reproduce the same tool call via SSH (`mcporter call gog-agentic.<tool> --args '...'` or `gog ...` CLI) with the same account to see the raw error.
