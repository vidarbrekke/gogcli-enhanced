# OpenClaw + gog on Linode (headless)

Use this when running OpenClaw on a Linode server and want the agent to create/edit Google Docs and Drive (e.g. “Create a new Google Doc called Test1 in a new Drive folder called testing123”).

## 1. Automatic MCP registration during setup

**No manual config is required** if you run the repo’s setup script where OpenClaw can see the config:

- **Command:** From the repo root, run `./scripts/setup.sh`.
- **What it does:** The script builds/installs `gog`, runs auth setup, and **registers the gog MCP server automatically** by writing (or merging) a `gog-agentic` entry into `config/mcporter.json` under a detected “workspace” directory.
- **Workspace detection:**
  - If `OPENCLAW_WORKSPACE` is set, that path is used.
  - Else if the repo lives under a path containing `/repositories/` (e.g. `.../workspace/repositories/gogcli-enhanced`), the workspace is the parent of `repositories` (e.g. `.../workspace`).
  - Else the repo root is used.
- **Config path:** `$workspace_dir/config/mcporter.json` (e.g. `/root/openclaw-stock-home/.openclaw/workspace/config/mcporter.json`).
- **Fallback:** If `~/openclaw-stock-home/.openclaw/workspace` or `~/.openclaw/workspace` exists and is different from the detected workspace, the script also merges the same `gog-agentic` entry there, so OpenClaw finds gog even when the repo is cloned elsewhere.
- **Env:** If `GOG_KEYRING_BACKEND` is set during setup (e.g. file keyring), the script adds `"env": {"GOG_KEYRING_BACKEND": "file"}` to the server entry so the spawned `gog mcp serve` process uses the file keyring. For headless MCP, the script can also create a keyring password file and add `GOG_KEYRING_PASSWORD_FILE` to that `env`: it will do so automatically when `GOG_KEYRING_PASSWORD` is set, or when you confirm at the prompt “Save keyring password to a file so the MCP server can unlock without a TTY?”. Otherwise add `GOG_KEYRING_PASSWORD_FILE` manually (see §4 and §8.2).

After setup, the script has already started the mcporter daemon and restarted OpenClaw (when the gateway runs under the same user). You do not need to run any further commands—the agent should see gog-agentic tools. If the gateway runs under a different user, restart it manually so it picks up the config.

## 2. Why it failed before

- **No browser:** Linode is headless; gog needs a **refresh token** (no interactive OAuth).
- **Missing tool:** The MCP layer had no `docs.create`; the agent could create a folder with `drive.ensureFolder` but could not create a Doc. That is now added: use `docs.create` with optional `parentId` (from `drive.ensureFolder`).

## 3. Why the agent gave manual steps instead of using tools

If the agent responds with “create the folder manually” or suggests CLI commands instead of calling MCP tools, usually:

- **MCP not connected:** OpenClaw is not configured to use the gog MCP server, so it never sees `drive.ensureFolder` or `docs.create`.
- **Credentials missing:** gog is not configured on the server (no keyring / refresh token), so even if the agent called the tools, they would fail and the agent may fall back to manual instructions.

To have the agent **use the tools**, ensure (1) OpenClaw’s MCP config runs `gog mcp serve` and (2) gog has valid credentials on the Linode host (see §4 and §5 below). You can also add a system or project instruction that when the user asks to create a folder or Doc, the agent should call `drive.ensureFolder` and `docs.create` (and use the returned `folderId` as `parentId`) rather than giving manual steps.

## 4. One-time credential setup on Linode

gog must have a valid refresh token for your Google account.

**Option A – Copy keyring from a machine that already did OAuth**

1. On a machine where you’ve already run `gog` and completed OAuth (e.g. your laptop), locate the keyring file (see `gog` / INSTALL docs for your OS).
2. Copy that keyring file to the Linode server (e.g. under `~/.config/gog/` or the path used by `GOG_CONFIG_ROOT`).
3. On Linode, use the file keyring so no GUI is needed:
   - `export GOG_KEYRING_BACKEND=file`
   - `export GOG_KEYRING_PASSWORD=<password used when creating the keyring>`
   - Ensure the same password is used as on the source machine if the keyring is encrypted.

**Option A2 – Password from file (for MCP / headless)**

When OpenClaw (or mcporter) spawns `gog mcp serve`, it only passes the `env` from `mcporter.json` to the child process. It does not pass your shell’s `GOG_KEYRING_PASSWORD`, so the keyring cannot be unlocked and gog suggests “authorize the app”. To fix this without putting the password in the config file:

- **During setup:** When you run `./scripts/setup.sh` with file keyring, the script can create the password file and add `GOG_KEYRING_PASSWORD_FILE` to mcporter.json for you—either automatically if `GOG_KEYRING_PASSWORD` is set, or by answering yes to “Save keyring password to a file so the MCP server can unlock without a TTY?” (see §1).
- **Manually:** Otherwise:
  1. Create a file that contains only the keyring password (one line), e.g. `/root/.config/gogcli/keyring.password`.
  2. Restrict access: `chmod 600 /root/.config/gogcli/keyring.password`
  3. In `mcporter.json`, in the gog-agentic entry’s `env`, add `GOG_KEYRING_PASSWORD_FILE` pointing at that path. Example (merge into existing `env`):
     `"GOG_KEYRING_PASSWORD_FILE": "/root/.config/gogcli/keyring.password"`
  4. Restart OpenClaw. The spawned `gog mcp serve` will read the password from the file and unlock the keyring without a TTY.

**Option B – Run setup wizard once with a tunneled browser (advanced)**

If you can expose a browser to the Linode box (e.g. SSH port-forward + run setup in that session), run `./scripts/setup.sh` and complete OAuth once; then use that keyring for headless runs.

## 5. OpenClaw MCP configuration

OpenClaw must run the gog MCP server so it can call `drive.*` and `docs.*` tools.

- **Transport command:** `gog mcp serve` (or full path to `gog` plus `mcp serve`).
- **Environment:** The child process needs the keyring password. Either:
  - Set `GOG_KEYRING_BACKEND=file` and `GOG_KEYRING_PASSWORD` in the environment that runs OpenClaw (if your MCP host passes parent env to children), or
  - Set `GOG_KEYRING_BACKEND=file` and `GOG_KEYRING_PASSWORD_FILE=/path/to/password-file` in the gog-agentic `env` in mcporter.json (recommended for headless; see §4 Option A2).
- **Working directory:** Run `gog` from the repo or a directory where the `gog` binary and config/keyring are available.

After that, the agent should see tools such as `drive.ensureFolder`, `docs.create`, `docs.insertText`, etc., in `tools/list`.

## 6. Tool sequence for “Create a Doc in a new folder”

The intended flow:

1. **Create folder (idempotent):**  
   `drive.ensureFolder` with `path: "testing123"` (and optional `parentId` for a parent folder).  
   Response includes `folderId` (and `path`, `created`).

2. **Create Doc in that folder:**  
   `docs.create` with `title: "Test1"` and `parentId: "<folderId from step 1>"`.

So the agent should call `drive.ensureFolder` first, read `folderId` from the result, then call `docs.create` with that `parentId`. With `docs.create` now available in MCP, this flow is supported end-to-end.

### 6.1 Prefer gog-agentic MCP first (automatic via setup)

Setup **automatically** injects a directive into the OpenClaw workspace bootstrap file `TOOLS.md` so the agent uses gog-agentic for Google Drive and Docs **before** trying other paths. OpenClaw includes `TOOLS.md` in the system prompt for every run, so no manual instruction is required.

- **What setup does:** Creates or appends to `$workspace_dir/TOOLS.md` a section *"Google Drive and Docs (gog-agentic MCP)"* that instructs the agent to call gog-agentic MCP tools directly and **not** to try mcporter-to-CLI, browser automation, or GOG_KEYRING_PASSWORD—so the agent does not waste time and tokens probing those paths. Idempotent: re-running setup does not duplicate the section.
- **If you need to add it manually** (e.g. different workspace or TOOLS.md was removed), add a system or project instruction with this text:

> For Google Drive and Google Docs actions (create folder, create document, edit document, list files, etc.), **use the gog-agentic MCP tools first.** Call `drive.ensureFolder` for folders, `docs.create` for new docs (with `parentId` from the folder result), `docs.insertText`, `docs.replaceAllText`, and other `drive.*` / `docs.*` tools. Do not try mcporter to run the gog CLI, browser automation, or GOG_KEYRING_PASSWORD—use MCP tools directly. Only if gog-agentic tools are missing from your tool list, report that gog-agentic is unavailable. If a tool call fails, report the error and do not fall back to gog CLI or exec.

*Why TOOLS.md:* OpenClaw injects bootstrap files (e.g. `TOOLS.md`) into the system prompt automatically. Writing the directive during setup gives zero manual steps and scales to all users.

This reduces ambiguity and keeps behavior consistent with the headless setup (keyring, `GOG_CLIENT=default`, etc.).

### 6.2 Faster flows (fewer round-trips, fewer tokens)

To reduce latency and token usage use: `docs.createWithBody` when creating a doc with initial content (one tool call); `docs.executeBatch` to insert text and apply styling in one batch; `drive.searchFiles` with `query` to get folder ID when the folder may already exist. Setup injects this into `TOOLS.md` (§6.1).

### 6.3 Agent says a folder “does not exist” but you see it in Drive

Tools use the default account in the keyring. If the folder exists under a **different Google account** than the one on the server, the agent will not see it—add that account on the server or pass `account` to the tool. Use `drive.searchFiles` with the folder name as `query`; search includes My Drive and shared drives by default.

## 7. Verify

- From the Linode server (or the same env OpenClaw uses), run:
  - `gog drive ensure-folder testing123 --json`
  - From the JSON output, copy `folderId` (the Drive folder ID, not the name).
  - Run: `gog docs create Test1 --parent <folderId> --json` (use the actual folder ID, not the string `"testing123"`).
- If both succeed, OpenClaw can perform the same via MCP once credentials and MCP config are correct.

### Correct CLI example (do not use folder name in `--parent`)

Wrong (will not put the doc in the folder by name):

```bash
gog docs create Test1 --parent testing123   # WRONG: --parent must be a folder ID
```

Right:

```bash
# 1) Create folder and get folderId from JSON
gog drive ensure-folder testing123 --json
# Example output: {"folderId":"1abc...","name":"testing123","created":true,"path":"testing123"}

# 2) Use that folderId as --parent (replace 1abc... with the real ID from step 1)
gog docs create Test1 --parent 1abc... --json
```

Alternatively you can use `gog drive mkdir testing123 --json` for step 1; the response shape may differ but you still must pass the returned folder **ID** to `docs create --parent`, not the folder name.

## 8. If gog-agentic is in mcporter.json but still doesn’t work

When the MCP config file already lists **gog-agentic** but the agent doesn’t use the tools (or OpenClaw says there is no MCP server), check the following.

### 8.0 Agent says “gog-agentic tools are not showing up in my tool list”

If the agent reads TOOLS.md but reports that gog-agentic tools are not in its tool list, the gateway is not seeing the MCP server. **Restarting only the gateway is not enough** if the mcporter daemon is not running or the gateway uses a different config.

**Checklist (on the Linode server):**

1. **Run the diagnostic** (as the same user that runs OpenClaw):
   ```bash
   /root/openclaw-stock-home/.openclaw/workspace/repositories/gogcli-enhanced/scripts/mcp-diagnose-gog.sh /root/openclaw-stock-home/.openclaw/workspace/config/mcporter.json
   ```
   - If it prints **"OK: gog-agentic responds and exposes tools"**, gog is fine; the issue is how the gateway loads MCP (step 3).
   - If it prints **"ERROR: gog mcp serve exited"** or **"gog-agentic not found"**, fix the config or keyring (see §8.1, §8.2) then retry.

2. **Start or restart the mcporter daemon** (required when using `"lifecycle": { "mode": "keep-alive" }`). If you ran `./scripts/setup.sh`, it already started the daemon and restarted the OpenClaw gateway; if the agent still has no tools, run:
   ```bash
   mcporter --config /root/openclaw-stock-home/.openclaw/workspace/config/mcporter.json daemon restart
   mcporter --config /root/openclaw-stock-home/.openclaw/workspace/config/mcporter.json daemon status
   ```
   You should see **gog-agentic** in the status. If not, the daemon may be using a different config path or gog may be failing to start (check diagnostic from step 1).

3. **Confirm the OpenClaw gateway uses this config** when it fetches the tool list. The gateway must be started with `--config` (or equivalent) pointing at the same `mcporter.json`. If OpenClaw uses another config file or workspace, it will not load gog-agentic. See §8.4.

4. **Then** restart the OpenClaw gateway so it reconnects to mcporter and refreshes the tool list.

### 8.1 Command must be an absolute path

If `command` in the gog-agentic entry is relative (e.g. `gog` or `./bin/gog`), mcporter may start the process with a different working directory and fail to find the binary. Re-run setup from the repo so it writes an absolute path, or edit `mcporter.json` and set `mcpServers.gog-agentic.command` to the full path to the `gog` binary (e.g. `/root/openclaw-stock-home/.openclaw/workspace/repositories/gogcli-enhanced/bin/gog`).

### 8.2 Environment for the process that runs OpenClaw

The process that starts OpenClaw (and thus mcporter / gog-agentic) must have:

- `GOG_KEYRING_BACKEND=file` if you use the file keyring on Linode.
- A way for the spawned `gog mcp serve` to get the keyring password. MCP hosts often pass only the `env` from mcporter.json to the child, not the parent’s environment, so:
  - **Recommended:** Put the password in a file (e.g. `/root/.config/gogcli/keyring.password`, `chmod 600`) and add to the gog-agentic `env` in mcporter.json: `"GOG_KEYRING_PASSWORD_FILE": "/root/.config/gogcli/keyring.password"`. Then the child can unlock the keyring without a TTY. See §4 Option A2.
  - **Alternatively:** If your setup passes the parent process environment to MCP children, set `GOG_KEYRING_PASSWORD` when starting OpenClaw.

Without the password (or password file), `gog mcp serve` cannot open the keyring and will prompt for auth or suggest “authorize the app”, which leads to a dead end on headless.

### 8.3 Run the MCP diagnostic script

From the Linode server (same user that runs OpenClaw), run:

```bash
/path/to/gogcli-enhanced/scripts/mcp-diagnose-gog.sh /root/openclaw-stock-home/.openclaw/workspace/config/mcporter.json
```

This script reads the gog-agentic entry, runs the same `command` and `args` with the same `env`, sends `initialize` and `tools/list` over stdio, and reports whether tools are returned. If it prints “ERROR: gog mcp serve exited with code …”, fix the binary path or the keyring env (see above). If it prints “OK: gog-agentic responds and exposes tools”, then gog is fine and the issue is likely how OpenClaw loads or uses the config (see 8.4).

### 8.4 Confirm OpenClaw uses this config

Ensure OpenClaw (or mcporter) is actually started with `--config /root/openclaw-stock-home/.openclaw/workspace/config/mcporter.json` (or the equivalent path). If OpenClaw uses a different config file or workspace, it will not load gog-agentic. Check how you start OpenClaw and that no other config overrides this one.

### 8.5 gog-agentic shows as “offline” or MCP tool calls don’t work

The gog MCP server talks over **stdio** (stdin/stdout). If the MCP host spawns it per-request without keeping the process attached, gog can exit or appear offline.

**Fix (recommended):** Use the **mcporter keep-alive daemon** so one long-lived process holds the gog stdio connection:

1. In `mcporter.json`, add to the gog-agentic entry: `"lifecycle": { "mode": "keep-alive" }`.
2. Start (or restart) the daemon: `mcporter --config /path/to/workspace/config/mcporter.json daemon restart`.
3. Ensure OpenClaw (or whatever runs mcporter) uses the same config path so it talks to the daemon; the daemon will then list/call tools via the held gog process.

After this, `mcporter daemon status` should list gog-agentic and `mcporter list gog-agentic` should return tools. The gog CLI (e.g. `gog drive ls`) works in a separate process with a TTY; MCP only works when the host (or daemon) keeps the gog process attached and pipes stdio.

**After deploy (git pull):** From the repo on the server, run `./scripts/ensure-mcp-daemon.sh` to restart the daemon (and optionally `RESTART_GATEWAY=1 ./scripts/ensure-mcp-daemon.sh` to also restart the OpenClaw gateway). That prevents "tools not in list" when the gateway was restarted without the daemon.

### 8.6 Agent says "gog CLI needs to be authenticated" or "no tokens stored" (5-whys root cause)

**Root cause:** The environment passed to the exec/shell when the agent runs a command did not include the gog keyring env vars (`GOG_KEYRING_BACKEND`, `GOG_KEYRING_PASSWORD_FILE`, `XDG_CONFIG_HOME`). OpenClaw builds exec env from the gateway's runtime env, which is seeded from `process.env` and merged with config `env.vars` when config is loaded. If neither had the gog vars, the shell's `gog` process sees no keyring and reports "no tokens" / "authenticate first".

**Fix (applied on the server):**

1. **Config:** In `openclaw.json` (e.g. `/root/openclaw-stock-home/.openclaw/openclaw.json`), add under the top-level `env`:
   ```json
   "env": {
     "vars": {
       "GOG_KEYRING_BACKEND": "file",
       "GOG_KEYRING_PASSWORD_FILE": "/path/to/workspace/.config/gogcli/keyring.password",
       "XDG_CONFIG_HOME": "/path/to/workspace/.config"
     }
   }
   ```
   Use the real path to your keyring password file and config directory. When the gateway loads config, these vars are merged into the runtime env used for exec.

2. **Systemd (optional):** In the gateway's systemd override, add the same `Environment=` lines so the gateway process has them at startup.

3. **Restart:** Restart the OpenClaw gateway so it reloads config and applies `env.vars`.

4. **Prefer MCP:** Continue to direct the agent to use gog-agentic MCP first (TOOLS.md) so it does not rely on the CLI for Drive/Docs.

## 8.7 Agent says “I can’t access Google Drive — authentication isn’t set up” (5-whys)

Use this when the agent responds with “authentication isn’t set up” or “I can’t access your Google Drive directly” and offers to set up gog CLI or share files instead of using tools.

**Why 1:** Why does the agent say authentication isn’t set up?  
Because it either has no gog-agentic tools in its tool list and infers it can’t access Drive, or it called a tool and got an auth/keyring error and is paraphrasing it.

**Why 2:** Why would it have no tools / get an auth error?  
Either (A) the OpenClaw gateway never loaded gog-agentic (wrong or missing MCP config path), or (B) the gateway loaded it but the gog process fails at runtime (keyring env, binary path), or (C) the agent has the tools but didn’t call them and defaulted to “auth not set up” from context.

**Why 3:** Why would the gateway not load gog-agentic?  
The gateway gets its MCP server list from a config file. If that config is not the same file we write to (e.g. we write to `workspace/config/mcporter.json` but the gateway uses `~/.mcporter/mcporter.json` or another path), gog-agentic will never appear in the tool list.

**Why 4:** Why would the config path differ?  
OpenClaw (or the skill that runs mcporter) may use a default path like `~/.mcporter/mcporter.json` or a path set in openclaw.json / systemd. Our setup writes to the workspace `config/mcporter.json` and to fallback workspaces; it does not know which path the gateway actually uses.

**Why 5 (root cause):** The gateway’s MCP config path and the path(s) we write to are not guaranteed to match, so the agent may never see gog-agentic and falls back to “authentication isn’t set up.”

**Fix:**

1. **Confirm where the gateway gets its MCP config.** Check how the OpenClaw gateway is started (systemd unit, openclaw.json, or env). If it uses `~/.mcporter/mcporter.json`, ensure that file contains gog-agentic (setup can merge into it; see below).
2. **Ensure gog-agentic is in that config.** We write to:
   - `$workspace_dir/config/mcporter.json` (e.g. `/root/openclaw-stock-home/.openclaw/workspace/config/mcporter.json`)
   - Fallbacks: `~/openclaw-stock-home/.openclaw/workspace/config/mcporter.json`, `~/.openclaw/workspace/config/mcporter.json`
   - If your gateway uses `~/.mcporter/mcporter.json`, run setup from the repo—it now merges gog-agentic there when that path exists (see §8.8). Or merge the gog-agentic entry from the workspace config into `~/.mcporter/mcporter.json` manually.
3. **Restart daemon and gateway.** Run `./scripts/ensure-mcp-daemon.sh` with the workspace config, then `RESTART_GATEWAY=1 ./scripts/ensure-mcp-daemon.sh` (or restart the gateway however you normally do). See §8.0.
4. **If the agent has tools but reports auth:** The gog process may be failing to unlock the keyring. Check §8.2 (GOG_KEYRING_PASSWORD_FILE in mcporter.json env) and run the diagnostic (§8.3).

**Why 6–10 (if still broken):** If the agent still says auth isn’t set up after the fix:
- **Why 6:** Gateway might not have reloaded the config (restart gateway after changing config).
- **Why 7:** Multiple gateway processes; only one was restarted (stop all, start one with the correct config).
- **Why 8:** Tool list is cached and not refreshed (restart gateway so it re-fetches tools from mcporter).
- **Why 9:** The model is inferring “auth not set up” from TOOLS.md or a previous turn without calling tools (ensure TOOLS.md says “use gog-agentic tools first” and that the agent sees the tools in its list).
- **Why 10:** A different OpenClaw profile or workspace is active in the UI, and that profile uses a different gateway/config (switch workspace or ensure the active one uses the config we write to).

## 8.8 Ensure gateway finds gog-agentic: merge into default mcporter path

Many OpenClaw installs use the default mcporter config path `~/.mcporter/mcporter.json`. Setup merges the gog-agentic entry into that file when it exists, or creates it in cloud context, and **starts the mcporter daemon with that path** so the gateway finds gog-agentic. If your gateway uses a different config path, run the daemon with that path (e.g. `./scripts/ensure-mcp-daemon.sh` for workspace config). See §8.0.

## 9. Verifying that OpenClaw used gog-agentic MCP

When the agent creates a folder or document, it is not always obvious whether it used the gog MCP tools or another path (e.g. another Google Workspace integration, or the gog CLI via a shell tool).

**How to tell:**

- **MCP was used** if the conversation or OpenClaw’s tool-call log shows invocations of **gog-agentic** tools such as `drive.ensureFolder`, `docs.create`, `docs.insertText`, etc. Some UIs show “Tool: drive.ensureFolder” or similar in the thread.
- **Unclear / possibly not MCP** if the agent only says something like “I’ll create it programmatically using the Google Workspace” or “using the API” without naming `drive.ensureFolder` or `docs.create`. That can mean another integration (e.g. another MCP server or a built-in Google API) was used.

**To enforce MCP usage:** Setup injects the directive into `TOOLS.md` by default (§6.1). If your UI exposes tool calls, you can confirm MCP was used by seeing `drive.ensureFolder` and `docs.create` (or other `drive.*` / `docs.*` tools) in the conversation or logs.

### 9.1 Never expose keyring password in agent responses

If the agent explains how it authenticated and **includes the keyring password** (e.g. `GOG_KEYRING_PASSWORD="..."`) or any credential value in the conversation, that is a security issue: the password is now in chat history and possibly logs. Setup injects a directive into TOOLS.md: *Never reveal, echo, or include the keyring password or credential values in your response; say "used stored credentials" or "authenticated via existing keyring" without exposing secrets.* If a password was already exposed, **rotate it**: create a new keyring password, re-run auth (e.g. `gog auth add <email> --services user --remote --step 1` then step 2), update the password file used by MCP, and restart the daemon/gateway.
