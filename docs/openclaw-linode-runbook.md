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

After setup, **restart OpenClaw** (or ensure it is started with the same `mcporter.json` path) so it picks up the new server. No refactor or manual MCP config is needed.

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

### 6.1 Prefer gog-agentic MCP (system/project instruction)

To ensure the agent **always** uses the gog MCP server for Google Drive and Docs (and only falls back to CLI or other methods when an action has no MCP tool), add a system or project instruction in OpenClaw. Example:

**Instruction text (copy into your OpenClaw project/system instructions):**

> For Google Drive and Google Docs actions (create folder, create document, edit document, list files, etc.), **always use the gog-agentic MCP tools** when available. Call `drive.ensureFolder` for folders, `docs.create` for new docs (with `parentId` from the folder result), `docs.insertText`, `docs.replaceAllText`, and other `drive.*` / `docs.*` tools. Do not invoke the `gog` CLI directly or use other Google integrations for these actions unless the required tool does not exist in gog-agentic. If a user asks to create a folder and document, use `drive.ensureFolder` then `docs.create` with the returned `folderId` as `parentId`.

This reduces ambiguity and keeps behavior consistent with the headless setup (keyring, `GOG_CLIENT=default`, etc.).

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

## 9. Verifying that OpenClaw used gog-agentic MCP

When the agent creates a folder or document, it is not always obvious whether it used the gog MCP tools or another path (e.g. another Google Workspace integration, or the gog CLI via a shell tool).

**How to tell:**

- **MCP was used** if the conversation or OpenClaw’s tool-call log shows invocations of **gog-agentic** tools such as `drive.ensureFolder`, `docs.create`, `docs.insertText`, etc. Some UIs show “Tool: drive.ensureFolder” or similar in the thread.
- **Unclear / possibly not MCP** if the agent only says something like “I’ll create it programmatically using the Google Workspace” or “using the API” without naming `drive.ensureFolder` or `docs.create`. That can mean another integration (e.g. another MCP server or a built-in Google API) was used.

**To enforce MCP usage:** Add the system/project instruction from §6.1 so the agent is explicitly told to use gog-agentic tools for Drive and Docs. Then, if your UI exposes tool calls, you can confirm by seeing `drive.ensureFolder` and `docs.create` (or other `drive.*` / `docs.*` tools) in the conversation or logs.
