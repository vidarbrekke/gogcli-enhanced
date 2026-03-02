## Google Drive and Docs (gog-agentic MCP)

**Do not say tools are unavailable or that the gateway may not be loading config. Do not read TOOLS.md or config files to "diagnose".** For any Google Drive or Docs request, **call the exec tool immediately** with one of these commands. Note: tool names contain dots so you MUST use `--server gog-agentic --tool <toolName>` — do NOT use the `gog-agentic.drive.listFiles` dot-selector (it splits on the first dot and fails).

- **List Drive root:** `mcporter --server gog-agentic --tool drive.listFiles --args '{}' --output json`
- **Create folder:** `mcporter --server gog-agentic --tool drive.ensureFolder --args '{"path":"FolderName"}' --output json`
- **Create doc:** `mcporter --server gog-agentic --tool docs.create --args '{"title":"Doc Title"}' --output json` (add `"parentId":"<folderId>"` to put in a folder)
- **Search files:** `mcporter --server gog-agentic --tool drive.searchFiles --args '{"query":"name or text"}' --output json`

Wait — the correct mcporter invocation uses the `call` subcommand:

- **List Drive root:** `mcporter call --server gog-agentic --tool drive.listFiles --args '{}' --output json`
- **Create folder:** `mcporter call --server gog-agentic --tool drive.ensureFolder --args '{"path":"FolderName"}' --output json`
- **Create doc:** `mcporter call --server gog-agentic --tool docs.create --args '{"title":"Doc Title"}' --output json` (add `"parentId":"<folderId>"` to place doc in a folder)
- **Search files:** `mcporter call --server gog-agentic --tool drive.searchFiles --args '{"query":"name or text"}' --output json`

OAuth is already set up. If the exec call fails (e.g. command not found or error output), report the error and then suggest the user ask the workspace admin to run the diagnostic and restart the daemon and gateway (runbook §8.0). Never reveal the keyring password or credentials.

For "create folder then doc": run ensureFolder first, then docs.create with the returned folderId as parentId.
