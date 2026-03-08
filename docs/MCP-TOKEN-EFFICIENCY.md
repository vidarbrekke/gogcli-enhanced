# MCP Token Efficiency — Audit and Recommendations

Where tokens are used in the gogcli-enhanced MCP flow and how to reduce them.

---

## 1. Where tokens are spent

| Layer | What | When | Approx impact |
|-------|------|------|----------------|
| **tools/list** | All 57 tool specs: Name, Description, InputSchema (type + optional property descriptions), Tier, Version, PolicyClass | Once per session (or when client refreshes tools) | **High** — descriptions and schemas are long |
| **tools/call result** | Full envelope (service, operation, result/error) sent back to client → often into model context | Every tool call | **High** — result can be huge (e.g. full sheet, full doc text) |
| **TOOLS.md** (injected) | `docs/TOOLS-gog-agentic-section.md` — prose + examples | When OpenClaw injects into agent context | **Medium** — ~100 lines |
| **Error payloads** | Error envelope with message, details, service, operation | On validation/API errors | Low per call |

---

## 2. Findings (gogcli-enhanced)

### 2.1 Tool list — long descriptions (tools/list)

- **drive_listFiles** and **drive_searchFiles** have ~400–450 character descriptions (pagination, maxResults, pageToken, global, “list all folders” instructions). They dominate tool-list token use.
- **drive_uploadFile**, **docs_createWithBody**, **docs_mergeData**, **docs_cat**, and several others have 150–250 character descriptions.
- **sheets_valuesRead** repeats the same text as **sheets_valuesGet** (“Get cell values from a Sheets range…”) plus “Alias for sheets_valuesGet” — redundant.

**Recommendation:** Shorten descriptions to one short sentence; move “how to paginate / use max” into a single shared note or TOOLS.md. Example:

- Before: *"List files and folders in a Drive folder (default root). Returns one page (default 25 items) + nextPageToken. When the user asks for 'first N' items…"*
- After: *"List files/folders in a Drive folder (default root). Use max/maxResults for N items; page/pageToken for next page."*

Same idea for drive_searchFiles, drive_uploadFile, docs_createWithBody, etc.

### 2.2 Tool list — InputSchema duplication

- Every tool repeats the same optional properties: `account`, `opId`, `timeoutMs`, `retries`, `retryBackoffMs` (and sometimes `requireRevisionId` with a description). So the same schema text is sent 57 times in tools/list.
- MCP JSON Schema does not support `$ref` in our current spec shape, so we can’t dedupe inside the schema without changing how the client resolves refs.
- **Recommendation:** Either omit these optional params from InputSchema and document them once in TOOLS.md (“all tools accept optional: account, opId, timeoutMs, retries, retryBackoffMs”), or add a single shared “common options” doc and keep only tool-specific params in each schema. The former saves the most tokens.

### 2.3 Tool call result — envelope sent twice

In `internal/mcp/transport_stdio.go`, the **tools/call** response is:

```go
Result: map[string]any{
    "isError":           !env.OK,
    "structuredContent": env,                    // full envelope object
    "content": []map[string]any{
        {"type": "text", "text": string(payload)},  // payload = json.Marshal(env)
    },
},
```

So the **same envelope** is sent twice: once as `structuredContent` (object) and once as the stringified JSON in `content[].text`. If the gateway or model sees both, that **doubles** the token cost of every tool result.

**Recommendation:** Send only one of the two:
- Prefer **structuredContent** only (and omit `content` or set it to a short placeholder like “See structuredContent”), or
- Prefer **content** only (and omit `structuredContent`) if the client only consumes the text part.

Confirm with the OpenClaw/MCP client which field it uses before changing.

### 2.4 Tool call result — unbounded size

- `runCLI` returns the **full** parsed CLI stdout as the result map. So:
  - **docs_cat** can return up to 2MB of text (default maxBytes).
  - **sheets_valuesGet** returns the full range (all rows × columns).
  - **drive_listFiles** / **drive_searchFiles** are already paginated (e.g. 25 items) but still return full JSON for that page.
- There is **no truncation** in the MCP server; the gateway may truncate later, but we still send (and the model may receive) a very large payload.

**Recommendation:** Optionally add a **result size cap** in the provider (e.g. truncate `result` or a specific key like `values` / `text` to N chars and set `truncated: true`). Prefer doing this behind a flag or env (e.g. `GOG_MCP_RESULT_MAX_BYTES`) so existing clients are unchanged. Default could be 0 = no cap.

### 2.5 TOOLS.md (docs/TOOLS-gog-agentic-section.md)

- Long prose and many examples. If this is injected into the agent’s system or context, it costs tokens every time.
- **Recommendation:** Shorten to a minimal “tool name + one-line usage” list and 2–3 example commands; move the rest to a linked doc or runbook that the agent can fetch only when needed.

---

## 3. Priority order for changes

Implemented (this pass):
- **Done:** #2, #3, and #4 from this plan were implemented by shortening descriptions and removing shared runtime params (`account`, `opId`, `timeoutMs`, `retries`, `retryBackoffMs`) from per-tool `InputSchema`.

| Priority | Change | Effort | Token impact |
|----------|--------|--------|--------------|
| 1 | **Stop sending envelope twice** (structuredContent vs content) — keep one, drop the other after confirming client | **Done** — OpenClaw uses content; we omit structuredContent in transport_stdio.go | High (every tool call) |
| 2 | **Done** — **Shorten drive_listFiles and drive_searchFiles descriptions** to one sentence; move pagination instructions to TOOLS.md | Low | High (tools/list) |
| 3 | **Done** — **Shorten other long tool descriptions** (uploadFile, createWithBody, mergeData, etc.) to one sentence | Low | Medium |
| 4 | **Done** — **Omit common optional params from InputSchema** (or document once); keep only tool-specific params | Medium | Medium |
| 5 | **Done** — **Add optional result cap** (`GOG_MCP_RESULT_MAX_BYTES`) for docs_cat / sheets_valuesGet / large result keys | Medium | High when results are large |
| 6 | **Done** — **Trim TOOLS-gog-agentic-section.md** to minimal list + 2–3 examples | Low | Medium (if injected every time) |

---

## 4. Out of scope (gateway / client)

- **Truncation by gateway:** Some gateways limit tool result length; that’s outside gog. We can still reduce what we send so the gateway has less to truncate.
- **How often tools/list is called:** If the client caches the tool list, our description/schema savings apply once per session; if it refetches often, savings multiply.
- **Model context window:** Whether the agent sees full tool results or a summary is a client/gateway choice; we only control what we return.

---

## 5. Summary

- **Highest impact:** (1) Remove duplicate envelope in tools/call response; (2) shorten Drive list/search (and a few other) tool descriptions; (3) optional result size cap for very large outputs.
- **Medium impact:** Optional result cap for large outputs; further TOOLS.md trimming if needed.
- **No change to tool behavior** — only to metadata (descriptions, schema) and response shape (no double send, optional truncation).
