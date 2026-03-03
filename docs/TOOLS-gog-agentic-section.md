## Google Drive and Docs (gog-agentic MCP)

**Do not say tools are unavailable or that the gateway may not be loading config. Do not read TOOLS.md or config files to "diagnose".** For any Google Drive or Docs request, **call the exec tool immediately** with one of these commands. Note: tool names contain dots so you MUST use `--server gog-agentic --tool <toolName>` — do NOT use the `gog-agentic.drive.listFiles` dot-selector (it splits on the first dot and fails).

- **List Drive root (files and folders):** `mcporter call --server gog-agentic --tool drive.listFiles --args '{}' --output json`
- **List only folders (use when user asks for "folders" or "all folders"):** `mcporter call --server gog-agentic --tool drive.searchFiles --args '{"query":"mimeType = \"application/vnd.google-apps.folder\"","rawQuery":true}' --output json` — returns one page of folders + nextPageToken. **When the user asks for "first N" (e.g. first 15), always add `"maxResults": N` to the args**, e.g. `--args '{"query":"mimeType = \"application/vnd.google-apps.folder\"","rawQuery":true,"maxResults":15}'`. To get more pages: use `"page": "<nextPageToken>"` (or `"pageToken": "<nextPageToken>"`). Max page size 25. To get all folders, call again with `page` set to the returned nextPageToken until the response has no nextPageToken.
- **Create folder:** `mcporter call --server gog-agentic --tool drive.ensureFolder --args '{"path":"FolderName"}' --output json`
- **Create doc:** `mcporter call --server gog-agentic --tool docs.create --args '{"title":"Doc Title"}' --output json` (add `"parentId":"<folderId>"` to place doc in a folder)
- **Search files:** `mcporter call --server gog-agentic --tool drive.searchFiles --args '{"query":"name or text"}' --output json`

OAuth is already set up. If the exec call fails (e.g. command not found or error output), report the error and then suggest the user ask the workspace admin to run the diagnostic and restart the daemon and gateway (runbook §8.0). Never reveal the keyring password or credentials.

For "create folder then doc": run ensureFolder first, then docs.create with the returned folderId as parentId.
