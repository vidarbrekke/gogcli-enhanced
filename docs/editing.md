# Google Docs Editing Guide

This guide covers inline editing commands under `gog docs edit`.

These commands use the Google Docs API `documents.batchUpdate` endpoint.

## Prerequisites

- You are authenticated with a user/account that can edit the target doc.
- The account has Docs scope (`https://www.googleapis.com/auth/documents`).
- You know the target `docId`.

Quick check:

```bash
gog docs info <docId>
```

## Commands

### Replace text everywhere

```bash
gog docs edit replace <docId> "old text" "new text"
gog docs edit replace <docId> "TODO" "DONE" --match-case
```

JSON output:

```bash
gog --json docs edit replace <docId> "old" "new"
```

Returns:
- `documentId`
- `occurrencesChanged`

### Append text

```bash
gog docs edit append <docId> $'\nChangelog:\n- item'
gog --json docs edit append <docId> $'\nChangelog:\n- item' --validate-only
```

Append inserts text right before the document's trailing newline.
For `--validate-only` and `--dry-run`, append uses a safe local index preview and does not require auth/API calls.

To persist the normalized request for chained or batch flows, use `--output-request-file` (same as other edit commands). You can combine append with batch: use `gog docs positions end <docId>` to get the append index, then use batch with `insertText` at that index, or run append then execute a batch from a saved request file.

### Insert text at index

```bash
gog docs edit insert <docId> "Prefix: " --index 1
gog docs edit insert <docId> "middle" --index 42
```

Indexes are 1-based.

### Delete text range

```bash
gog docs edit delete <docId> 10 40
```

- `start` is inclusive.
- `end` is exclusive.
- `end` must be greater than `start`.

### Batch operations from JSON

From file:

```bash
gog docs edit batch <docId> --requests-file ./docs/examples/docs-edit-batch.json
gog docs edit batch <docId> --requests-file ./docs/examples/docs-edit-batch.json --validate-only
gog docs edit batch <docId> --requests-file ./docs/examples/docs-edit-batch.json --validate-only --pretty
gog docs edit batch <docId> --requests-file ./docs/examples/docs-edit-batch.json --validate-only --output-request-file ./normalized.json
```

From stdin:

```bash
cat ops.json | gog docs edit batch <docId> --requests-file -
```

Use `--pretty` when you want deterministic, normalized request JSON in output (useful for agent pipelines and review steps).

Use `--output-request-file` to persist normalized request JSON for chained agent steps:

```bash
cat ops.json | gog docs edit batch <docId> --requests-file - --validate-only --output-request-file ./normalized.json
```

`--validate-only` output also includes `requestHash` (SHA256 over normalized request JSON), which is useful for idempotency checks and agent step correlation.

### Merge data (template -> many docs)

```bash
gog docs edit merge-data <templateId> --data-file ./records.json --validate-only
gog docs edit merge-data <templateId> --data-file ./records.json --dry-run
gog docs edit merge-data <templateId> --data-file ./records.json --filename-format "Offer - {{name}}" --output-folder-id <folderId>
```

`merge-data` returns per-record results with `status` and `stage` fields. Common failure stages include `copy` (template copy/create) and `batch-update` (placeholder replacement).

### Safety flags for agents

All edit commands support:

- `--dry-run`: build and print request payload without executing.
- `--require-revision <revisionId>`: optimistic concurrency guard.

Example:

```bash
gog docs edit replace <docId> "Draft" "Final" --dry-run --require-revision <revisionId>
```

For `delete`, non-JSON human mode requires `--force` (or `--dry-run`) to reduce accidental destructive edits.

Reference example:

- `docs/examples/docs-edit-batch.json`

Minimal inline example:

```json
{
  "requests": [
    {
      "insertText": {
        "location": { "index": 1 },
        "text": "Title\n"
      }
    },
    {
      "replaceAllText": {
        "containsText": { "text": "Draft", "matchCase": true },
        "replaceText": "Final"
      }
    }
  ]
}
```

## Index rules and pitfalls

- Google Docs API uses 1-based positions for content operations.
- Documents keep a trailing newline at the end.
- For destructive operations, test on a copy first:

```bash
gog docs copy <docId> "Safe Copy"
```

## Troubleshooting

- `doc not found or not a Google Doc`:
  - verify `docId`
  - confirm you have access
  - ensure the file is a Google Doc

- `insufficient permissions`:
  - re-auth with docs service scope and consent

```bash
gog auth add you@example.com --services docs --force-consent
```

---

# Google Sheets Editing Guide

This guide covers inline editing commands under `gog sheets edit`.

These commands use the Google Sheets API (`spreadsheets.values.*` and `spreadsheets.batchUpdate`) with agentic safety flags.

## Prerequisites

- You are authenticated with a user/account that can edit the target sheet.
- The account has Sheets scope (`https://www.googleapis.com/auth/spreadsheets`).
- You know the target `spreadsheetId`.

Quick check:

```bash
gog sheets metadata <spreadsheetId>
```

## Commands

### Update values in a range

```bash
gog sheets edit values <spreadsheetId> "Sheet1!A1:B2" "a|b, c|d"
gog sheets edit values <spreadsheetId> "Sheet1!A1:B2" --values-json '[[\"a\",\"b\"],[\"c\",\"d\"]]'
```

### Append values

```bash
gog sheets edit append <spreadsheetId> "Sheet1!A:C" "a|b"
gog sheets edit append <spreadsheetId> "Sheet1!A:C" "a|b" --insert INSERT_ROWS
```

### Clear values

```bash
gog sheets edit clear <spreadsheetId> "Sheet1!A1:B2" --force
```

### Batch operations from JSON

From file:

```bash
gog sheets edit batch <spreadsheetId> --requests-file ./ops.json --validate-only
gog sheets edit batch <spreadsheetId> --requests-file ./ops.json --validate-only --pretty
gog sheets edit batch <spreadsheetId> --requests-file ./ops.json --validate-only --output-request-file ./normalized.json
```

From stdin:

```bash
cat ops.json | gog sheets edit batch <spreadsheetId> --requests-file -
```

## Safety flags for agents

All sheets edit commands support:

- `--dry-run`: build and print request payload without executing.
- `--validate-only`: local validation only, no API call.
- `--pretty`: include normalized request JSON + hash in output.
- `--output-request-file`: persist normalized request JSON.
- `--execute-from-file`: run a previously generated request file (batch only).

For `clear`, non-JSON human mode requires `--force` (or `--dry-run`) to reduce accidental destructive edits.

---

# Google Slides Editing Guide

This guide covers editing commands under `gog slides edit`, including batch operations with the same agentic safety pattern as Docs and Sheets.

## Prerequisites

- You are authenticated with a user/account that can edit the target presentation.
- The account has Slides scope (`https://www.googleapis.com/auth/presentations`).
- You know the target `presentationId`.

Quick check:

```bash
gog slides info <presentationId>
```

## Commands

### Batch operations from JSON

Apply multiple Slides API requests in one call (same pattern as Docs/Sheets):

```bash
gog slides edit batch <presentationId> --requests-file ./ops.json
gog slides edit batch <presentationId> --requests-file ./ops.json --validate-only
gog slides edit batch <presentationId> --requests-file ./ops.json --validate-only --pretty
gog slides edit batch <presentationId> --requests-file ./ops.json --validate-only --output-request-file ./normalized.json
```

From stdin:

```bash
cat ops.json | gog slides edit batch <presentationId> --requests-file -
```

Execute a previously saved request file:

```bash
gog slides edit batch <presentationId> --execute-from-file ./normalized.json
```

### Other edit commands

- `replace-text`: find and replace text across slides.
- `create-slide`, `duplicate-slide`, `delete-slide`: slide lifecycle.
- `insert-table`, `replace-image`, `update-notes`, `refresh-charts`, `merge-data`: see `gog slides edit --help`.

## Safety flags for agents

Slides edit commands (including batch) support:

- `--dry-run`: build and print request payload without executing.
- `--validate-only`: validate request structure locally, no API call.
- `--pretty`: include normalized request JSON and `requestHash` in output.
- `--output-request-file`: write normalized request JSON to a file.
- `--execute-from-file`: run from a saved request file (batch only).

`--validate-only` output includes `requestHash` for idempotency and agent step correlation.
