# OpenClaw: gcloud vs gog — How to Proceed

Part of the solution is incorporating the official **Google Cloud SDK** (`gcloud`). The `gcloud` CLI is **not** installed by default in many OpenClaw environments. You can proceed in one of three ways. This doc also covers **when to use gog via MCP vs raw gog CLI** and **when gog uses native APIs vs gws** (see sections at the end).

## 1. Install the gcloud CLI (optional)

If you want to use Google Cloud services (including Drive, Docs, etc.) via `gcloud`, or to use `gws auth setup` to create a GCP project and OAuth client without touching the Cloud Console, install the official Google Cloud SDK:

```bash
curl https://sdk.cloud.google.com | bash
exec -l $SHELL   # Restart your shell to apply changes
gcloud init       # Initialize and authenticate
```

After that you can run `gcloud` commands directly (e.g. via OpenClaw's **exec** tool) and use **gws auth setup** for one-shot project + OAuth client creation (see [gws-on-linode.md](gws-on-linode.md)).

## 2. Use gog without gcloud (recommended for Drive/Docs/Sheets)

**gcloud is not required** for Google Drive, Docs, Sheets, Gmail, Calendar, etc. Use the **gog** CLI (and gog-agentic MCP) with your own OAuth client:

```bash
gog auth credentials /path/to/client_secret.json
gog auth add your-email@gmail.com --services drive,docs,sheets
```

- Get a Desktop OAuth client from [Google Cloud Console → Credentials](https://console.cloud.google.com/apis/credentials) (create project + OAuth client ID → Desktop app → download JSON).
- On OpenClaw/Linode, use the file keyring and (for MCP) `GOG_KEYRING_PASSWORD_FILE`; see [openclaw-linode-runbook.md](openclaw-linode-runbook.md) §4 and §8.2.

Once gog is authenticated, the agent uses **gog-agentic MCP** tools (`drive_*`, `docs_*`, `sheets_*`, etc.) or, when tools are not in the agent's list, **exec** with `mcporter call gog-agentic.<toolName> --args '...'`.

## 3. Integrate gcloud with OpenClaw (after installing)

Once `gcloud` is installed, you can:

- Run `gcloud` commands via OpenClaw's **exec** tool.
- Use **gws auth setup** (with gcloud) on a machine with a browser to create the GCP project and OAuth client, then export credentials for headless use (see [gws-on-linode.md](gws-on-linode.md)).

---

**Summary:** For Drive, Docs, Sheets, Gmail, and Calendar in OpenClaw, **use gog** (option 2). Install gcloud (option 1) only if you need `gcloud` or want `gws auth setup` to create the OAuth client for you.

---

## When to use gog via MCP vs raw gog CLI

- **Prefer gog-agentic MCP** for all Google Workspace requests: Drive, Docs, Sheets, Gmail (search/send), Calendar, Contacts. Call tools directly (e.g. `drive_listFiles`, `docs_create`, `sheets_valuesGet`) or via **exec** with `gog-agentic-call` / `mcporter call gog-agentic.<toolName> --args '...'`.
- **Use raw `gog` CLI only for gaps** where there is no MCP tool, for example:
  - Auth status: `gog auth status --json`
  - Gmail labels list: `gog gmail labels list -a ACCOUNT@gmail.com --json`
  - Other discovery or one-off commands that have no MCP equivalent.

Do not fall back to raw `gog drive search` (or similar) when MCP tools are available; use `gog-agentic-call` or the MCP tool list. See [TOOLS-gog-agentic-section.md](TOOLS-gog-agentic-section.md) and the **gog** skill (`skills/gog/SKILL.md` or workspace-injected copy) for the full tool list and auth policy.

## When gog uses native APIs vs gws (Google Workspace CLI)

By default, **all live requests** go through **gog** and its **native Google APIs**. The optional **gws** CLI is not in the agent path unless you enable backend routing:

- **Default:** gog uses native Go/API clients for every command. Single MCP (gog-agentic); agent talks only to gog.
- **With `GOG_BACKEND=gws`:** gog can invoke the **gws** CLI for Gmail labels list/get and Drive list/get/search. Drive list/search are single-page routes; global/all-pages and PDF page-count paths remain native. Tier C (writes, safety-critical) stays on native gog.

So: **gog vs gws** is a backend choice inside gog (env `GOG_BACKEND`), not a choice between two CLIs in the agent. For full routing logic, implementation, and tests, see [merge/GWS-VS-GOG-ROUTING.md](merge/GWS-VS-GOG-ROUTING.md).
