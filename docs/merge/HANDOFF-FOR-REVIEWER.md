# Hand-off for external reviewer: native gog JSON samples

This file gives you **real native gog JSON output** for the first commands we want to compare to gws. Use it to tighten schemas, add golden fixtures, and define normalization rules with minimal ambiguity.

## Source of samples

The samples below match **current gog** `--json` stdout. They are taken from unit-test mock responses (deterministic, no live account). Equivalent live capture:

```bash
gog --json gmail labels list
gog --json gmail labels get INBOX
gog --json drive ls --max 5
```

---

## 1) Gmail labels list

**Command:** `gog --json gmail labels list`

```json
{
  "labels": [
    {
      "id": "INBOX",
      "name": "INBOX",
      "type": "system"
    },
    {
      "id": "Label_1",
      "name": "Custom",
      "type": "user"
    }
  ]
}
```

**Notes:** Root key is `labels` (array). Each item has `id`, `name`, `type`. Optional fields from Gmail API (e.g. `messagesTotal`, `messageListVisibility`) may appear when present; list endpoint often omits counts.

---

## 2) Gmail labels get

**Command:** `gog --json gmail labels get INBOX`

```json
{
  "label": {
    "id": "INBOX",
    "name": "INBOX",
    "type": "system",
    "messagesTotal": 123,
    "messagesUnread": 7,
    "threadsTotal": 50,
    "threadsUnread": 3
  }
}
```

**Notes:** Root key is `label` (single object). Counts are integers. Same optional visibility fields can appear as in list.

---

## 3) Drive ls

**Command:** `gog --json drive ls` (one page, default max)

```json
{
  "files": [
    {
      "id": "f1",
      "name": "Doc",
      "mimeType": "application/pdf",
      "size": "1024",
      "modifiedTime": "2025-12-12T14:37:47Z"
    },
    {
      "id": "d1",
      "name": "Folder",
      "mimeType": "application/vnd.google-apps.folder",
      "size": "0",
      "modifiedTime": "2025-12-11T00:00:00Z"
    }
  ],
  "nextPageToken": "npt"
}
```

**Notes:** Root keys `files` (array) and `nextPageToken` (string; empty or absent on last page). Drive API returns `size` as string. `modifiedTime` is RFC3339.

---

## What we’ve already added

- **Goldens (deterministic):** `docs/merge/goldens/gmail-labels-list-native.json`, `gmail-labels-get-native.json`, `drive-ls-native.json` — same content as above.
- **Schemas:** `docs/merge/schemas/gmail-labels-list.json`, `gmail-labels-get.json`, `drive-ls.json` — draft JSON Schema for each response.
- **Discovery drift policy:** `docs/merge/discovery-drift-policy.md` — when to **pin/capture** vs **accept+detect** when diffing native vs gws.
- **Dossiers:** `docs/merge/commands/gmail-labels-read.md` and `docs/merge/commands/drive-read.md` — updated with capture instructions and diff/classification steps.

## What would help next (from you)

1. **Tighten schemas** — Use the samples above (and any live captures you run) to refine the JSON Schemas in `docs/merge/schemas/` (required vs optional, enums, formats).
2. **Produce gws goldens** — Run the same logical requests through gws, save as `*-gws.json`, and diff against the native goldens.
3. **Classify normalization rules** — For each diff, decide: normalize to match native (pin/capture), or allow and document (accept+detect). Record rules in the schemas or dossiers.
4. **Decide discovery drift policy** — Confirm or adjust pin/capture vs accept+detect for label order, optional fields, and drive `size`/metadata (see `discovery-drift-policy.md`).

After that we can implement the first adapter and conformance tests with clear contracts.
