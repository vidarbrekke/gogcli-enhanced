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
```

Append inserts text right before the document's trailing newline.

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
```

From stdin:

```bash
cat ops.json | gog docs edit batch <docId> --requests-file -
```

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
