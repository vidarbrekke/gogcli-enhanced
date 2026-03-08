## Google Drive and Docs (gog-agentic MCP)

For Google Drive or Docs requests, call tools directly. Prefer `gog-agentic-call`; fallback is raw `mcporter` with underscored tool names.

**Syntax:** `gog-agentic-call` accepts dotted or underscored names (`drive.listFiles` or `drive_listFiles`). Raw `mcporter` uses underscores only and needs `--server gog-agentic`.

### Tool list (all available)

- **Docs:** `docs_get`, `docs_cat`, `docs_listTabs`, `docs_positionsEnd`, `docs_positionsSearch`, `docs_positionsHeadings`, `docs_export`, `docs_create`, `docs_createWithBody`, `docs_insertText`, `docs_deleteRange`, `docs_replaceAllText`, `docs_appendText`, `docs_planBatch`, `docs_executeBatch`, `docs_sed`, `docs_smartEdit`, `docs_mergeData`
- **Drive:** `drive_listFiles`, `drive_searchFiles`, `drive_getFile`, `drive_listPermissions`, `drive_listComments`, `drive_ensureFolder`, `drive_moveFile`, `drive_renameFile`, `drive_shareFile`, `drive_unshare`, `drive_createComment`, `drive_deleteComment`, `drive_copyFile`, `drive_uploadFile`, `drive_deleteFile`, `drive_bulkExecute`
- **Sheets:** `sheets_valuesGet`, `sheets_valuesRead`, `sheets_links`, `sheets_metadata`, `sheets_planBatch`, `sheets_executeBatch`, `sheets_valuesUpdate`, `sheets_valuesAppend`, `sheets_clear`, `sheets_sortRange`, `sheets_dedupeRows`, `sheets_filterCopyRows`, `sheets_upsertRows`, `sheets_moveRows`, `sheets_applyFormula`, `sheets_summarize`
- **Others:** `gmail_search`, `gmail_send`, `calendar_events`, `contacts_list`

**Common runtime fields:** `account`, `opId`, `timeoutMs`, `retries`, `retryBackoffMs`.

**Drive pagination:** `max`/`maxResults` for page size, `page`/`pageToken` for next page, `fetchAllPages: true` for count-all.

**Result size cap:** set `GOG_MCP_RESULT_MAX_BYTES` to cap MCP outputs. `0` or unset = full output.

### Core usage reminders

- `docs_cat`: plain text read, optional `maxBytes`, `tab`, `allTabs`.
- `drive_listFiles` lists mixed files/folders; `drive_searchFiles` is query mode.
- `drive_getFile` with `pageCount: true` returns PDF metadata (`pdfMetadata` + `pdfMetadataEnvelope`).
- `sheets_valuesGet` is for full range values; range is `Sheet1!A1:D20` style.
- Gmail/Calendar/Contacts labels/auth are available via exec, not MCP.

### Minimal examples

- `gog-agentic-call drive.listFiles '{}'`
- `gog-agentic-call drive.getFile '{"fileId":"<id>","pageCount":true}'`
- `gog-agentic-call sheets_valuesGet '{"spreadsheetId":"<id>","range":"Sheet1!A1:D20"}'`
- `mcporter call --server gog-agentic --tool gmail_send --args '{"to":"user@example.com","subject":"Subject","body":"done"}' --output json`

### Reliable Drive folder + file lookup pattern

- Find folder: `gog-agentic-call drive.searchFiles '{"query":"name = \"Appraisal home valuation\"","rawQuery":true,"maxResults":10}'`
- List folder contents: `gog-agentic-call drive.listFiles '{"parentId":"<folderId>","maxResults":50}'`
- Query PDFs in a folder directly: `gog-agentic-call drive.searchFiles '{"query":"\"<folderId>\" in parents AND mimeType = \'application/pdf\'","rawQuery":true,"maxResults":50}'`
- Page through more results with `page:"<nextPageToken>"`.
- Use `name =` first, then fall back to `contains` when needed. Avoid `fields` unless the tool schema explicitly supports it.

If a folder is frequently accessed, keep IDs in a local cache file instead of injecting large lookup tables into prompt-visible memory.

### Auth policy

OAuth is already configured. If calls fail with auth issues:
1. Run `gog auth status --json`.
2. Report auth as missing/expired.
3. Ask the user to run `./scripts/setup.sh`.
4. For deeper repair, use `./scripts/setup-doctor.sh`.

**Never try to complete OAuth interactively inside chat/exec.** Never run `mcporter auth gog-agentic`, `gws auth login`, or `gog auth login` from this runner.

**Never invent names.** Report only returned fields and offer `page`/`pageToken` when more results exist.

### Response style (token-efficient)

- Start with one sentence answer.
- Keep caveats/errors in one short follow-up line.
- Offer at most two follow-up options.
