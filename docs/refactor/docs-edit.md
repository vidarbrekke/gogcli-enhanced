# Editing existing Google Docs (design)

## Goal

Enable editing of existing Google Docs from the CLI (e.g. append text, find/replace, or structured updates).

## Fork survey: doc editing

**None of the checked forks implement doc editing.**

Checked (source of `internal/cmd/docs.go`):

| Fork | Doc edit? | Notes |
|------|-----------|--------|
| [KrauseFx/gogcli](https://github.com/KrauseFx/gogcli) | No | Same as upstream: export, info, create, copy, cat |
| [alexknowshtml/gogcli](https://github.com/alexknowshtml/gogcli) | No | Cobra-based; docs cat via Drive export to text/plain, no Docs API batchUpdate |
| [shaneholloman/gogcli](https://github.com/shaneholloman/gogcli) | No | Same as upstream |
| [rohan-patnaik/gogcli-plus](https://github.com/rohan-patnaik/gogcli-plus) | No | README mirrors upstream; no docs edit in described commands |

So this is a green-field feature for this fork.

## Recommended approach: Google Docs API `batchUpdate`

- **API:** [documents.batchUpdate](https://developers.google.com/workspace/docs/api/reference/rest/v1/documents/batchUpdate) (Docs API v1).
- **Go client:** `google.golang.org/api/docs/v1` — `Documents.BatchUpdate(documentId, &docs.BatchUpdateDocumentRequest{ Requests: requests }).Do()`.
- **Auth:** Existing `docs` service already has `https://www.googleapis.com/auth/documents`; no new scope if we only use batchUpdate for content edits.

### Request types (examples)

- **insertText** — insert at index (e.g. end of body).
- **replaceAllText** — find/replace (no index math).
- **deleteContentRange** — delete by range.
- **updateTextStyle** — bold, italic, etc.
- **createParagraphBullets** / **deleteParagraphBullets**.
- **insertTable** / **insertTableRow** / **insertTableColumn**.
- **insertInlineImage**.

Best practice: apply edits in **descending index order** so earlier changes don’t shift indices for later ones. Use **WriteControl** (`requiredRevisionId` / `targetRevisionId`) when collaborating.

## Possible CLI surface

1. **Minimal (recommended first step)**  
   - `gog docs append <docId> --text "..."`  
     - Resolve end-of-body index (e.g. from `documents.get` → `body.content` end), then single `insertText` request.
   - `gog docs replace-all <docId> --find "..." --replace "..."`  
     - Single `replaceAllText` request; no index handling.

2. **Structured**  
   - `gog docs update <docId> --requests-file requests.json`  
     - JSON array of [Request](https://developers.google.com/workspace/docs/api/reference/rest/v1/documents/request) objects passed to batchUpdate (flexible, scriptable).

3. **Later**  
   - Optional `--required-revision-id` for WriteControl, or more helpers (e.g. “insert at start”, “replace first occurrence”) as needed.

## Implementation notes (this repo)

- **Where:** New subcommands under `DocsCmd` in `internal/cmd/docs.go` (e.g. `DocsAppendCmd`, `DocsReplaceAllCmd`, optionally `DocsUpdateCmd`).
- **Service:** Reuse `newDocsService`; add calls to `svc.Documents.BatchUpdate(...)`.
- **End index for append:** Get document with `Documents.Get(docId)`; compute end index from `Body.Content` (last element’s `endIndex`) or use a fixed strategy (e.g. insert at index 1 after the first newline if doc is “empty”).
- **Tests:** Unit tests with httptest mocking `Documents.Get` and `Documents.BatchUpdate`; optional integration test under `internal/integration` if we add one for docs.
- **Output:** Same pattern as existing docs commands: `--json` for machine output, otherwise human-friendly (e.g. “Appended N characters” / “Replaced M occurrences”).

## References

- [Docs API: documents.batchUpdate](https://developers.google.com/workspace/docs/api/reference/rest/v1/documents/batchUpdate)
- [Request types](https://developers.google.com/workspace/docs/api/reference/rest/v1/documents/request)
- [Best practices (edit order, WriteControl)](https://developers.google.com/workspace/docs/api/how-tos/best-practices)
- Go package: `google.golang.org/api/docs/v1`
