# Maton vs gog-agentic: Google Workspace capability parity

**Source:** Investigation via Playwright (logged-in Maton session) and Clawhub API Gateway skill (https://clawhub.ai/byungkyu/api-gateway). Maton API key is used with `https://gateway.maton.ai` and `https://ctrl.maton.ai`.

## How Maton exposes Google Workspace

Maton does **not** expose high-level "actions" in the web UI. It exposes the **native Google REST APIs** through a proxy:

- **Base URL:** `https://gateway.maton.ai/{app}/{native-api-path}`
- **Auth:** `Authorization: Bearer $MATON_API_KEY` (Maton injects OAuth for the target service)
- **Connection control:** `https://ctrl.maton.ai` (list/create/get/delete connections)

### Google app names and proxied hosts

| Service         | Maton app name     | Proxied base URL           |
|----------------|--------------------|----------------------------|
| Google Docs    | `google-docs`      | docs.googleapis.com        |
| Google Drive   | `google-drive`     | www.googleapis.com         |
| Google Sheets  | `google-sheets`    | sheets.googleapis.com      |
| Google Slides  | `google-slides`    | slides.googleapis.com      |
| Gmail          | `google-mail`      | gmail.googleapis.com       |
| Google Calendar| `google-calendar`  | www.googleapis.com         |

Example Sheets call:

```http
GET https://gateway.maton.ai/google-sheets/v4/spreadsheets/{id}/values/Sheet1!A1:B2
Authorization: Bearer $MATON_API_KEY
```

So **capabilities = whatever the official Google Docs, Drive, Sheets, Slides REST APIs support**. No extra "Maton-only" actions; you call the same paths as in Google's API docs.

## Parity with gog-agentic

| Aspect | Maton | gog-agentic |
|--------|--------|-------------|
| **Model** | Passthrough to native Google APIs | Same underlying APIs, wrapped as MCP tools + CLI |
| **Docs** | Any `docs.googleapis.com` endpoint | `docs_get`, `docs_cat`, `docs_planBatch`, `docs_executeBatch`, `docs_sed`, `docs_smartEdit`, `docs_mergeData`, etc. |
| **Drive** | Any Drive endpoint via `google-drive` | `drive_listFiles`, `drive_searchFiles`, `drive_ensureFolder`, `drive_uploadFile`, `drive_deleteFile`, etc. |
| **Sheets** | Any `sheets.googleapis.com` endpoint (e.g. Values.Get, BatchUpdate) | `sheets_valuesGet`, `sheets_valuesUpdate`, `sheets_sortRange`, `sheets_dedupeRows`, `sheets_filterCopyRows`, `sheets_upsertRows`, `sheets_moveRows`, `sheets_applyFormula`, `sheets_summarize`, etc. |
| **Slides** | Any `slides.googleapis.com` endpoint | `slides_planBatch`, `slides_executeBatch`, `slides_replaceText`, `slides_createSlide`, etc. |

**Conclusion:** For "what's possible," we are at parity: both use the same Google APIs. gog-agentic adds **convenience tools** (e.g. dedupe, filter-copy, upsert, apply-formula, summarize) so agents don't have to compose raw BatchUpdate or multi-call flows. Maton users can achieve the same by calling the same Sheets/Docs/Slides endpoints through the gateway, but they must build the request payloads themselves.

### Auth model

- **gog:** Local OAuth (keyring-stored refresh token); credentials stay on the machine where `gog auth` was run. No central server; each environment needs its own auth.
- **Maton:** One API key; OAuth is completed once per connection at ctrl.maton.ai. The gateway injects the user's token server-side. One key can represent "this agent's Google connection" without distributing refresh tokens.

Tradeoff: gog gives more control and privacy (tokens never leave your environment); Maton simplifies agent-only or shared setups (paste API key, no keyring).

### Quick mapping (Maton → gog)

| Maton-style call | gog equivalent |
|------------------|----------------|
| `GET .../google-sheets/v4/spreadsheets/{id}/values/Sheet1!A1:B2` | `sheets_valuesGet` (MCP or `gog sheets values get`) |
| Docs `documents.batchUpdate` | `docs_planBatch` + `docs_executeBatch` (or `docs_sed` / `docs_smartEdit` for expression-based edits) |
| Drive list files | `drive_listFiles` or `drive_searchFiles` |
| Sheets append / update | `sheets_valuesAppend`, `sheets_valuesUpdate`, or `sheets_upsertRows` |

### What we're missing about Maton (for better insights)

- **Connected-account view:** With Google (Docs/Sheets/Drive) connected in Maton, we could see whether they surface a **curated list of actions** (e.g. "Read document", "Append rows") and which endpoints those map to; any **templates or presets** would imply preferred flows we could document as "do X in gog with Y".
- **Request/response shapes:** Example requests and responses from the gateway (success and error) would confirm they're byte-for-byte the same as Google's API or if Maton wraps errors (e.g. standard envelope); useful for a "migrating from Maton, errors in gog look like …" note.
- **Rate limits and quotas:** Whether Maton documents or enforces different limits than Google; would only affect runbook or scaling guidance.
- **Connection metadata:** Whether ctrl.maton.ai or an API exposes "supported operations" or capability flags per connection; that would give a checklist to ensure one-to-one coverage in gog.

## References

- Maton: https://www.maton.ai (API key under user menu)
- API Gateway skill: https://clawhub.ai/byungkyu/api-gateway (SKILL.md: Base URL, auth, supported services table, "Use native API docs")
- Connection control: https://ctrl.maton.ai (list/create connections; open returned URL to complete OAuth)
- Maton API reference (linked from skill): https://www.maton.ai/docs/api-reference
