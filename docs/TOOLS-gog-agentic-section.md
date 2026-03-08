## Google Drive and Docs (gog-agentic MCP)

**Do not say tools are unavailable or that the gateway may not be loading config. Do not read TOOLS.md or config files to "diagnose".** For any Google Drive or Docs request, **call the exec tool immediately**. Prefer the installed wrapper `gog-agentic-call`, which accepts both dotted and underscored tool names and resolves the right mcporter config automatically. Example: `gog-agentic-call drive.listFiles '{}'`. Raw mcporter still works as a fallback: `mcporter call gog-agentic.drive_listFiles --args '{}'`.

**Syntax (avoid common failures):** If you use `gog-agentic-call`, both `drive.listFiles` and `drive_listFiles` work. If you use raw `mcporter call`, tool names use **underscores**, not dots (e.g. `drive_listFiles`, not `drive.listFiles`). The `--server` flag belongs to the **`call`** subcommand: `mcporter call --server gog-agentic --tool TOOL_NAME --args '...'`. For actions with no MCP tool (e.g. **Gmail labels**), use exec with the gog CLI: `gog gmail labels list -a ACCOUNT@gmail.com --json`. CLI subcommand is `labels list`, not `list-labels`. **Auth status / default account:** No MCP tool; use exec: `gog auth status --json` (or `gog auth status`). Do not try `auth.status`, `auth.list`, or `mcporter auth gog-agentic` — they do not exist or are not the supported flow here.

**Comparison with gateway proxies (e.g. Maton):** See [maton-vs-gog-parity.md](maton-vs-gog-parity.md) for capability parity and a quick Maton→gog mapping.

### Tool reference (all available via MCP)

- **Docs read:** `docs_get` (metadata/revision), `docs_cat` (plain text; optional maxBytes, tab, allTabs), `docs_listTabs`, `docs_positionsEnd`, `docs_positionsSearch` (text + matchCase), `docs_positionsHeadings`, `docs_export` (export to pdf/docx/txt; docId, format, optional out — if out omitted writes to temp and returns path)
- **Docs write:** `docs_create`, `docs_createWithBody`, `docs_insertText`, `docs_deleteRange`, `docs_replaceAllText`, `docs_appendText`, `docs_planBatch`, `docs_executeBatch` (optional requireRevisionId), `docs_sed`, `docs_smartEdit`, `docs_mergeData` (templateId + data array; validateOnly/dryRun)
- **Drive read:** `drive_listFiles` (parentId or global), `drive_searchFiles`, `drive_getFile` (supports `pageCount`), `drive_listPermissions`, `drive_listComments`
- **Drive write:** `drive_ensureFolder`, `drive_moveFile`, `drive_renameFile`, `drive_shareFile` (to: anyone|user|domain), `drive_unshare`, `drive_createComment`, `drive_deleteComment`, `drive_copyFile`, `drive_uploadFile`, `drive_deleteFile` (validateOnly returns planned without executing)
- **Drive bulk:** `drive_bulkExecute` (operations array: move|rename|share|delete; validateOnly to preview; max 50 per call)
- **Sheets read:** `sheets_valuesGet`, `sheets_valuesRead` (alias; spreadsheetId, range; optional majorDimension, valueRenderOption), `sheets_links` (hyperlinks in range), `sheets_metadata` (spreadsheet title, locale, timeZone, sheet list)
- **Sheets write:** `sheets_planBatch`, `sheets_executeBatch`, `sheets_valuesUpdate`, `sheets_valuesAppend`, `sheets_clear` (clear values in range; optional dryRun), `sheets_sortRange` (sort range by column; sortByColumn 0-based, optional desc), `sheets_dedupeRows` (remove duplicate rows by key columns; keyColumns 0-based array, optional; keep first), `sheets_filterCopyRows` (filter rows by column op value, copy to targetSheet; op: eq, contains, gt, lt; optional destinationCell), `sheets_upsertRows` (upsert by keyColumns; update matching rows, append new; rows = 2D array), `sheets_moveRows` (filter by column op value, copy or move to targetSheet; mode: copy|move), `sheets_applyFormula` (apply formula to column range; formula template with {row} placeholder), `sheets_summarize` (group by columns + aggregate count/sum; optional targetSheet)

**Sheets — full data:** To read full spreadsheet cell contents (not just hyperlinks), use **`sheets_valuesGet`**. It is available on the server. If your tool list only shows `sheets_links`, `sheets_valuesUpdate`, `sheets_valuesAppend`, call `sheets_valuesGet` anyway via exec (see example below); the gateway may be showing a cached list.

**Sheets — tool names and range:** Use **underscores** in tool names: `sheets_valuesGet`, `sheets_sortRange`, `sheets_dedupeRows` (not dots like `sheets.valuesGet`). For **sort, dedupe, filter-copy, upsert, move-rows, apply-formula, summarize**, the `range` parameter **must include the sheet name** (e.g. `"range":"Sheet1!A2:J200"`). Using only `"range":"A2:J200"` will fail. **Sort by column:** Use `sheets_sortRange` with `sortByColumn` as 0-based index (0 = column A, 1 = column B; e.g. Due_Date in column B → `"sortByColumn":1`). **Duplicates:** Use `sheets_dedupeRows` to remove duplicate rows by key columns (keeps first). **valuesUpdate** requires a `values` parameter (2D array of cell data).

**Backup (Linode → Drive):** Use `drive_uploadFile` with `localPath` set to the file path on the server where gog runs (e.g. `/var/backups/mybackup.tar.gz`), and `parentId` to the target Drive folder ID. Optional: `name` (Drive filename), `keepRevisionForever: true` for retention.

**PDF / binary files (e.g. page count):** Drive metadata does not include PDF page count. Use the canonical policy in `docs/pdf-metadata-extraction.md` (download + `pdfinfo`, then `startxref/xref` fallback) and never treat non-`ok` metadata as authoritative.

Use `--args '{"key":"value"}'` with the appropriate JSON for each tool. Destructive tools (`drive_deleteFile`, `drive_unshare`, `drive_deleteComment`) accept `validateOnly: true` to return a planned action without executing.

### Example commands

- **Auth status / default account (no MCP tool — use exec):** `gog auth status --json` or `gog auth status`
- **List Drive root (files and folders):** `gog-agentic-call drive.listFiles '{}'`
- **List all accessible Drive files (global):** `gog-agentic-call drive.listFiles '{"global":true,"maxResults":20}'` (cannot combine `global:true` with `parentId`)
- **List only folders (use when user asks for "folders" or "all folders"):** `gog-agentic-call drive.searchFiles '{"query":"mimeType = \"application/vnd.google-apps.folder\"","rawQuery":true}'` — returns one page of folders + nextPageToken. **For "how many folders in root" use one call with fetchAllPages:** `gog-agentic-call drive.searchFiles '{"query":"mimeType = \\\"application/vnd.google-apps.folder\\\" and \\\"root\\\" in parents","rawQuery":true,"fetchAllPages":true}'` — response includes `totalCount`. When the user asks for "first N" (e.g. first 15), add `"maxResults": N`. To get more pages manually: use `"page": "<nextPageToken>"` (or `"pageToken"`) until the response has no nextPageToken. **Choice:** If the user says **"only folders"** or **"just folders"** (or clarifies they wanted folders only), use `drive_searchFiles` with the folder mimeType query above; for root only add ` and "root" in parents` to the query. If the user says **"files and folders"** or **"files or folders"**, `drive_listFiles` is correct (mixed list).
- **Create folder:** `mcporter call gog-agentic.drive_ensureFolder --args '{"path":"FolderName"}' --output json`
- **Create doc:** `mcporter call gog-agentic.docs_create --args '{"title":"Doc Title"}' --output json` (add `"parentId":"<folderId>"` to place doc in a folder)
- **Get file (including PDF metadata):** `mcporter call gog-agentic.drive_getFile --args '{"fileId":"<fileId>","pageCount":true}' --output json`
  - For PDFs, the response includes:
    - `pageCount` (top-level)
    - `pdfMetadata` (`status`, `source`, `confidence`, `attempts`, `pages`)
    - `pdfMetadataEnvelope.pdf` (same payload under a dedicated metadata namespace)
- **PDF content analysis (OpenClaw native tool):** if your OpenClaw instance has `pdf` enabled, you can analyze Drive PDFs directly with `pdf.analyze` (for text, tables, metadata):
  - `pdf.analyze` payload: `{"fileId":"<GOOGLE_DRIVE_ID>"}`
  - This is complementary to `drive_getFile`: use `drive_getFile` for fast page-count metadata, then `pdf.analyze` when you need content extraction.
- **Search files:** `mcporter call gog-agentic.drive_searchFiles --args '{"query":"name or text"}' --output json`
- **Generic folder+file lookup pattern (for any term):**
1. Find folders: `gog-agentic-call drive.searchFiles '{"query":"mimeType = 'application/vnd.google-apps.folder' and (name contains \'<TERM>\')","rawQuery":true}'`
2. For each returned folder ID, list PDFs inside it: `gog-agentic-call drive.listFiles '{"parentId":"<FOLDER_ID>","query":"mimeType = 'application/pdf'","maxResults":50}'`
3. If needed, continue with `"page":"<nextPageToken>"` using the same args.
4. For each PDF file ID, call: `gog-agentic-call drive.getFile '{"fileId":"<FILE_ID>","pageCount":true}'` to extract page counts.
   This flow is generic: replace `<TERM>`, `<FOLDER_ID>`, and `<FILE_ID>` with user-provided values.

 - Or run the helper script: `scripts/find-drive-pdfs-by-term.sh --term "<TERM>" --workspace-dir /path/to/openclaw/workspace [--json] [--max-results 50]`.

JSON mode output includes:
- `pdfMetadata` and `pdfMetadataEnvelope`
- `fileLookup.ok` and `fileLookup.error` when `drive_getFile` fails
- `pageCount`, `fileMimeType`, and `pageCountDisplay`

- **Upload file (e.g. backup from server to Drive):** `mcporter call gog-agentic.drive_uploadFile --args '{"localPath":"/path/on/server/file.tar.gz","parentId":"<folderId>"}' --output json` (optional: `name`, `keepRevisionForever`)
- **Get spreadsheet values:** `mcporter call gog-agentic.sheets_valuesGet --args '{"spreadsheetId":"<id>","range":"Sheet1!A1:D10"}' --output json`
- **Sort sheet range by column:** `mcporter call gog-agentic.sheets_sortRange --args '{"spreadsheetId":"<id>","range":"Sheet1!A2:J200","sortByColumn":0,"desc":false}' --output json` (sortByColumn 0 = column A; use desc: true for descending). **To sort by Due_Date:** if Due_Date is in column B use `"sortByColumn":1`; range must include sheet name, e.g. `"range":"Sheet1!A2:J200"`.
- **Dedupe rows by key columns:** `mcporter call gog-agentic.sheets_dedupeRows --args '{"spreadsheetId":"<id>","range":"Sheet1!A2:J200","keyColumns":[0,1]}' --output json` (keeps first occurrence; omit keyColumns to use all columns)
- **Filter rows and copy to another sheet:** `mcporter call gog-agentic.sheets_filterCopyRows --args '{"spreadsheetId":"<id>","range":"Sheet1!A2:J200","targetSheet":"Filtered","column":1,"op":"eq","value":"yes"}' --output json` (op: eq, contains, gt, lt; optional destinationCell, default A1)
- **Upsert rows by key:** `mcporter call gog-agentic.sheets_upsertRows --args '{"spreadsheetId":"<id>","range":"Sheet1!A2:J200","keyColumns":[0,1],"rows":[["a","b"],["c","d"]]}' --output json`
- **Move or copy rows by condition:** `mcporter call gog-agentic.sheets_moveRows --args '{"spreadsheetId":"<id>","range":"Sheet1!A2:J200","targetSheet":"Out","column":1,"op":"eq","value":"x","mode":"move"}' --output json` (mode: copy or move)
- **Apply formula to column:** `mcporter call gog-agentic.sheets_applyFormula --args '{"spreadsheetId":"<id>","range":"Sheet1!C2:C10","formula":"=A{row}+B{row}"}' --output json` (use {row} for 1-based row number)
- **Summarize (group + aggregate):** `mcporter call gog-agentic.sheets_summarize --args '{"spreadsheetId":"<id>","range":"Sheet1!A2:D200","groupBy":[0],"metricColumn":1,"aggregate":"sum","targetSheet":"Summary"}' --output json` (aggregate: count or sum)

**Gmail — use MCP tools first.** `gmail_search` (query, max, page), `gmail_send` (to, subject, body or bodyHtml; optional cc, bcc, from). **Gmail labels:** No MCP tool; use exec: `gog gmail labels list -a ACCOUNT@gmail.com --json`.

- **Search Gmail:** `mcporter call gog-agentic.gmail_search --args '{"query":"from:user@example.com is:unread","max":10}' --output json`
- **Send email:** `mcporter call gog-agentic.gmail_send --args '{"to":"recipient@example.com","subject":"Subject","body":"Plain text body"}' --output json` (optional: cc, bcc, bodyHtml, from)

**Calendar — use MCP tools first.** `calendar_events` (from, to required; optional calendarId, max, page, query).

- **List calendar events:** `mcporter call gog-agentic.calendar_events --args '{"from":"2025-01-01T00:00:00Z","to":"2025-01-02T00:00:00Z","max":10}' --output json` (omit calendarId for primary)

**Contacts — use MCP tools first.** `contacts_list` (optional max, page).

- **List contacts:** `mcporter call gog-agentic.contacts_list --args '{"max":50}' --output json`

OAuth is already set up. If the exec call fails (e.g. command not found or error output), report the error and then suggest the user ask the workspace admin to run the diagnostic and restart the daemon and gateway (runbook §8.0). Never reveal the keyring password or credentials.

**Auth failure policy:** Never try to complete OAuth interactively inside chat/exec. Do **not** run `mcporter auth gog-agentic`, and do **not** run `gws auth login` or `gog auth login` from the chat tool runner because they may wait for browser interaction and produce a blank line or hang. If a Google request fails with unauthenticated/authError/invalid_grant/missing refresh token/no credentials:
1. Run `gog auth status --json`.
2. Briefly report that the workspace auth is missing or expired.
3. Tell the user to run `./scripts/setup.sh` in a real terminal as the single supported recovery path, then retry the original request.
4. If they need advanced repair, point them to `./scripts/setup-doctor.sh`.

**Never invent or assume folder or file names.** Only report what the API returned. If you got only N items, say "here are the first N" and offer to fetch more with `page`/`pageToken`; do not make up names for items 11–15 or any other position.

For "create folder then doc": run drive_ensureFolder first, then docs_create with the returned folderId as parentId.
