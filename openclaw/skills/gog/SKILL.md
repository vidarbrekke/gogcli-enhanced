---
name: gog
description: Use gog-agentic MCP for Google Workspace requests; use raw gog CLI only for gaps like auth status or Gmail labels.
homepage: https://gogcli.sh
metadata:
  {
    "openclaw":
      {
        "emoji": "🎮",
        "requires": { "bins": ["gog", "gog-agentic-call"] }
      }
  }
---

# gog

For Google Drive and Docs requests, prefer the installed `gog-agentic-call` wrapper and follow `TOOLS.md`.

Rules

- Do not claim `gog-agentic` is missing until `gog-agentic-call` or raw `mcporter` actually fails.
- Do not use dotted raw mcporter tool names like `drive.listFiles` with `--tool`; use `gog-agentic-call drive.listFiles '{}'` or raw `mcporter call gog-agentic.drive_listFiles --args '{}'`.
- Do not fall back to raw `gog drive search` for simple Drive listing requests when `gog-agentic-call` is available.
- Do not run interactive OAuth in chat. Never use `mcporter auth gog-agentic`. On auth failures, run `gog auth status --json` and tell the user to run `./scripts/setup.sh` in a real terminal.

Preferred commands

- Drive root files and folders: `gog-agentic-call drive.listFiles '{}'`
- Drive search: `gog-agentic-call drive.searchFiles '{"query":"name or text"}'`
- Create folder: `gog-agentic-call drive.ensureFolder '{"path":"FolderName"}'`
- Create doc: `gog-agentic-call docs.create '{"title":"Doc Title"}'`

Use raw `gog` CLI only when there is no MCP tool for the task.

Examples of valid raw gog CLI usage

- Auth status: `gog auth status --json`
- Gmail labels list: `gog gmail labels list -a ACCOUNT@gmail.com --json`

If a Drive or Docs request fails, report the exact error instead of saying the server is unavailable.
