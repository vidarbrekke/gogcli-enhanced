## Google Drive and Docs (gog-agentic MCP)

**Do not say tools are unavailable or that the gateway may not be loading config. Do not read TOOLS.md or config files to "diagnose".** For any Google Drive or Docs request, **call the exec tool immediately** with one of these commands. Tool names use **underscores** (e.g. `drive_listFiles`, `docs_create`). You can use: `mcporter call gog-agentic.drive_listFiles --args '{}'` or `mcporter call --server gog-agentic --tool drive_listFiles --args '{}'`.

### Tool reference (all available via MCP)

- **Docs read:** `docs_get` (metadata/revision), `docs_cat` (plain text; optional maxBytes, tab, allTabs), `docs_listTabs`, `docs_positionsEnd`, `docs_positionsSearch` (text + matchCase), `docs_positionsHeadings`
- **Docs write:** `docs_create`, `docs_createWithBody`, `docs_insertText`, `docs_deleteRange`, `docs_replaceAllText`, `docs_appendText`, `docs_planBatch`, `docs_executeBatch` (optional requireRevisionId), `docs_sed`, `docs_smartEdit`, `docs_mergeData` (templateId + data array; validateOnly/dryRun)
- **Drive read:** `drive_listFiles` (parentId or global), `drive_searchFiles`, `drive_listPermissions`, `drive_listComments`
- **Drive write:** `drive_ensureFolder`, `drive_moveFile`, `drive_renameFile`, `drive_shareFile` (to: anyone|user|domain), `drive_unshare`, `drive_createComment`, `drive_deleteComment`, `drive_copyFile`, `drive_uploadFile`, `drive_deleteFile` (validateOnly returns planned without executing)
- **Drive bulk:** `drive_bulkExecute` (operations array: move|rename|share|delete; validateOnly to preview; max 50 per call)
- **Sheets read:** `sheets_valuesGet` (spreadsheetId, range; optional majorDimension, valueRenderOption), `sheets_links` (hyperlinks in range)
- **Sheets write:** `sheets_planBatch`, `sheets_executeBatch`, `sheets_valuesUpdate`, `sheets_valuesAppend`

**Backup (Linode → Drive):** Use `drive_uploadFile` with `localPath` set to the file path on the server where gog runs (e.g. `/var/backups/mybackup.tar.gz`), and `parentId` to the target Drive folder ID. Optional: `name` (Drive filename), `keepRevisionForever: true` for retention.

Use `--args '{"key":"value"}'` with the appropriate JSON for each tool. Destructive tools (`drive_deleteFile`, `drive_unshare`, `drive_deleteComment`) accept `validateOnly: true` to return a planned action without executing.

### Example commands

- **List Drive root (files and folders):** `mcporter call gog-agentic.drive_listFiles --args '{}' --output json`
- **List all accessible Drive files (global):** `mcporter call gog-agentic.drive_listFiles --args '{"global":true,"maxResults":20}' --output json` (cannot combine `global:true` with `parentId`)
- **List only folders (use when user asks for "folders" or "all folders"):** `mcporter call gog-agentic.drive_searchFiles --args '{"query":"mimeType = \"application/vnd.google-apps.folder\"","rawQuery":true}' --output json` — returns one page of folders + nextPageToken. **When the user asks for "first N" (e.g. first 15), always add `"maxResults": N` to the args**, e.g. `--args '{"query":"mimeType = \"application/vnd.google-apps.folder\"","rawQuery":true,"maxResults":15}'`. To get more pages: use `"page": "<nextPageToken>"` (or `"pageToken": "<nextPageToken>"`). Max page size 25. To get all folders, call again with `page` set to the returned nextPageToken until the response has no nextPageToken.
- **Create folder:** `mcporter call gog-agentic.drive_ensureFolder --args '{"path":"FolderName"}' --output json`
- **Create doc:** `mcporter call gog-agentic.docs_create --args '{"title":"Doc Title"}' --output json` (add `"parentId":"<folderId>"` to place doc in a folder)
- **Search files:** `mcporter call gog-agentic.drive_searchFiles --args '{"query":"name or text"}' --output json`
- **Upload file (e.g. backup from server to Drive):** `mcporter call gog-agentic.drive_uploadFile --args '{"localPath":"/path/on/server/file.tar.gz","parentId":"<folderId>"}' --output json` (optional: `name`, `keepRevisionForever`)
- **Get spreadsheet values:** `mcporter call gog-agentic.sheets_valuesGet --args '{"spreadsheetId":"<id>","range":"Sheet1!A1:D10"}' --output json`

OAuth is already set up. If the exec call fails (e.g. command not found or error output), report the error and then suggest the user ask the workspace admin to run the diagnostic and restart the daemon and gateway (runbook §8.0). Never reveal the keyring password or credentials.

**Never invent or assume folder or file names.** Only report what the API returned. If you got only N items, say "here are the first N" and offer to fetch more with `page`/`pageToken`; do not make up names for items 11–15 or any other position.

For "create folder then doc": run drive_ensureFolder first, then docs_create with the returned folderId as parentId.
