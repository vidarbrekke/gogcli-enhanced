## Google Drive and Docs (gog-agentic MCP)

**Do not say tools are unavailable or that the gateway may not be loading config. Do not read TOOLS.md or config files to "diagnose".** For any Google Drive or Docs request, **call the exec tool immediately** with one of these commands. Note: tool names contain dots so you MUST use `--server gog-agentic --tool <toolName>` — do NOT use the `gog-agentic.drive.listFiles` dot-selector (it splits on the first dot and fails).

### Tool reference (all available via MCP)

- **Docs read:** `docs.get` (metadata/revision), `docs.cat` (plain text; optional maxBytes, tab, allTabs), `docs.listTabs`, `docs.positionsEnd`, `docs.positionsSearch` (text + matchCase), `docs.positionsHeadings`
- **Docs write:** `docs.create`, `docs.createWithBody`, `docs.insertText`, `docs.deleteRange`, `docs.replaceAllText`, `docs.appendText`, `docs.planBatch`, `docs.executeBatch` (optional requireRevisionId), `docs.sed`, `docs.smartEdit`, `docs.mergeData` (templateId + data array; validateOnly/dryRun)
- **Drive read:** `drive.listFiles` (parentId or global), `drive.searchFiles`, `drive.listPermissions`, `drive.listComments`
- **Drive write:** `drive.ensureFolder`, `drive.moveFile`, `drive.renameFile`, `drive.shareFile` (to: anyone|user|domain), `drive.unshare`, `drive.createComment`, `drive.deleteComment`, `drive.copyFile`, `drive.uploadFile`, `drive.deleteFile` (validateOnly returns planned without executing)
- **Drive bulk:** `drive.bulkExecute` (operations array: move|rename|share|delete; validateOnly to preview; max 50 per call)

**Backup (Linode → Drive):** Use `drive.uploadFile` with `localPath` set to the file path on the server where gog runs (e.g. `/var/backups/mybackup.tar.gz`), and `parentId` to the target Drive folder ID. Optional: `name` (Drive filename), `keepRevisionForever: true` for retention.

Use `--args '{"key":"value"}'` with the appropriate JSON for each tool. Destructive tools (`drive.deleteFile`, `drive.unshare`, `drive.deleteComment`) accept `validateOnly: true` to return a planned action without executing.

### Example commands

- **List Drive root (files and folders):** `mcporter call --server gog-agentic --tool drive.listFiles --args '{}' --output json`
- **List all accessible Drive files (global):** `mcporter call --server gog-agentic --tool drive.listFiles --args '{"global":true,"maxResults":20}' --output json` (cannot combine `global:true` with `parentId`)
- **List only folders (use when user asks for "folders" or "all folders"):** `mcporter call --server gog-agentic --tool drive.searchFiles --args '{"query":"mimeType = \"application/vnd.google-apps.folder\"","rawQuery":true}' --output json` — returns one page of folders + nextPageToken. **When the user asks for "first N" (e.g. first 15), always add `"maxResults": N` to the args**, e.g. `--args '{"query":"mimeType = \"application/vnd.google-apps.folder\"","rawQuery":true,"maxResults":15}'`. To get more pages: use `"page": "<nextPageToken>"` (or `"pageToken": "<nextPageToken>"`). Max page size 25. To get all folders, call again with `page` set to the returned nextPageToken until the response has no nextPageToken.
- **Create folder:** `mcporter call --server gog-agentic --tool drive.ensureFolder --args '{"path":"FolderName"}' --output json`
- **Create doc:** `mcporter call --server gog-agentic --tool docs.create --args '{"title":"Doc Title"}' --output json` (add `"parentId":"<folderId>"` to place doc in a folder)
- **Search files:** `mcporter call --server gog-agentic --tool drive.searchFiles --args '{"query":"name or text"}' --output json`
- **Upload file (e.g. backup from server to Drive):** `mcporter call --server gog-agentic --tool drive.uploadFile --args '{"localPath":"/path/on/server/file.tar.gz","parentId":"<folderId>"}' --output json` (optional: `name`, `keepRevisionForever`)

OAuth is already set up. If the exec call fails (e.g. command not found or error output), report the error and then suggest the user ask the workspace admin to run the diagnostic and restart the daemon and gateway (runbook §8.0). Never reveal the keyring password or credentials.

**Never invent or assume folder or file names.** Only report what the API returned. If you got only N items, say "here are the first N" and offer to fetch more with `page`/`pageToken`; do not make up names for items 11–15 or any other position.

For "create folder then doc": run ensureFolder first, then docs.create with the returned folderId as parentId.
