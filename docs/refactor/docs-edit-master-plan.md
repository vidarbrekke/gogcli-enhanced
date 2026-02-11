# Google Docs Editing Master Plan

## Objective

Deliver safe, script-friendly editing of existing Google Docs in `gogcli-enhanced` with an MVP-first rollout, then expand to power-user workflows.

This plan combines:
- The fork survey and green-field conclusion in `docs/refactor/docs-edit.md`
- The detailed implementation breakdown from `/Users/vidarbrekke/Dev/ClawBackup/gogcli-enhanced`

## What We Know

- No reviewed fork has implemented docs editing yet (KrauseFx, alexknowshtml, shaneholloman, rohan-patnaik).
- This repo already has docs read/create/copy/export and Docs API auth plumbing.
- Google Docs editing should use `documents.batchUpdate` via `google.golang.org/api/docs/v1`.

References:
- https://github.com/steipete/gogcli/forks
- https://developers.google.com/workspace/docs/api/reference/rest/v1/documents/batchUpdate
- https://developers.google.com/workspace/docs/api/reference/rest/v1/documents/request

## Product Scope

### MVP (Phase A)

Implement the smallest useful and low-risk edit surface:

1. `gog docs edit replace <docId> <find> <replace> [--match-case]`
2. `gog docs edit append <docId> <text>`
3. `gog docs edit insert <docId> <text> [--index N]`
4. `gog docs edit delete <docId> <start> <end>`

Why this order:
- `replace` is simplest and most robust (no index math).
- `append` unlocks report-style automation.
- `insert` and `delete` complete core primitives.

### Post-MVP (Phase B)

5. `gog docs edit batch <docId> --requests-file <path|->`
6. Optional: `--required-revision-id` / `--target-revision-id` write-control flags.
7. Optional higher-level helpers (`prepend`, template helpers, first-match replace).

### Explicitly Deferred

- Rich formatting commands (bold/italic/color), tables, images, list styling.
- Interactive mode.
- Any feature requiring config/version/server changes without explicit approval.

## CLI Design (Final)

Use a grouped structure that matches existing `gog` patterns:

```text
gog docs
  edit
    replace
    append
    insert
    delete
    batch   (Phase B)
```

Rationale:
- Avoids polluting top-level `docs` commands.
- Leaves room for future edit operations.
- Keeps discoverability through `gog docs edit --help`.

## Technical Design

### API strategy

- Use `svc.Documents.BatchUpdate(docID, req).Context(ctx).Do()`.
- Use request types:
  - `ReplaceAllTextRequest`
  - `InsertTextRequest`
  - `DeleteContentRangeRequest`

### Shared helpers (in `internal/cmd/docs.go` first, split later if needed)

- `docsResolveEndIndex(ctx, svc, docID) (int64, error)`  
  - Fetch document with `Documents.Get`.
  - Return `last.EndIndex - 1` to avoid trailing newline issues.
- `docsValidateEditInputs(...) error` for command-specific checks.
- `docsBatchUpdate(...)` wrapper for consistent error mapping/output.
- `isDocsNotFound(err)` reuse, plus `isDocsBadRequest(err)` helper.

### Error handling standards

- Keep errors actionable and consistent with existing CLI style:
  - not found / wrong mime / permission
  - invalid index/range
  - empty text/find value
- Preserve parseable stdout behavior: user hints to stderr, data to stdout.

### Output standards

- `--json`: machine-friendly object with command-specific summary.
- default/plain: concise lines like:
  - `replaced\t<N>`
  - `inserted_chars\t<N>`
  - `deleted_chars\t<N>`

## TDD Execution Plan

### Phase A1: Replace (test-first)

1. Add failing tests for:
   - successful replace, case-sensitive and default case-insensitive
   - zero occurrences
   - empty `find`
   - docs not found
2. Implement `DocsReplaceCmd`.
3. Ensure JSON + text output tests pass.

### Phase A2: Append

1. Add failing tests for:
   - append to empty/non-empty doc
   - end-index resolution
   - bad doc ID/not found
2. Implement helper for end index + `DocsAppendCmd`.

### Phase A3: Insert + Delete

1. Add failing tests for index/range validation and success paths.
2. Implement commands and shared validation.
3. Verify behavior near boundaries (start, end-1, invalid end).

### Phase A4: Docs and cleanup

1. Update README command examples.
2. Add `docs/editing.md` focused on indexes and safety.
3. Update changelog entry.

## Testing Plan

### Unit tests (required)

- File: `internal/cmd/docs_edit_test.go` (or split by command as needed).
- Mock Docs API endpoints with `httptest`.
- Validate outbound batchUpdate payloads, not just outputs.
- Target: high coverage on new command logic and validations.

### Integration tests (optional but recommended)

- Add a build-tagged integration test or script:
  - create doc -> replace/append/insert/delete -> verify with `docs cat`.

### Verification gate per phase

- `make fmt`
- `make test`
- `make lint`

## Risk Register and Mitigations

1. **Index off-by-one bugs**  
   Mitigation: central end-index helper + boundary tests.

2. **Conflicts from concurrent human edits**  
   Mitigation: defer strict write-control to Phase B, document caveat.

3. **Overly large first release**  
   Mitigation: strict MVP ordering and command-by-command rollout.

## Delivery Milestones

### Milestone 1 (MVP core)
- `replace`, `append`, `insert`, `delete` implemented with tests.
- README and help text updated.

### Milestone 2 (advanced)
- `batch` command with JSON request file/stdin.
- Optional revision control flags.

### Milestone 3 (enhancements)
- Formatting/structure helpers based on real usage demand.

## Ownership and Workflow

- Implement one subcommand per PR when possible.
- Each PR includes:
  - tests first (or same commit sequence with clear red/green intent),
  - command help text,
  - docs updates for user-visible behavior.
- Do not introduce server/config/package/version changes without explicit approval.

## Immediate Next Action

Start Phase A1 now:

1. Add tests for `gog docs edit replace`.
2. Implement `DocsEditCmd` + `DocsReplaceCmd`.
3. Run `make test` and `make lint`.
4. Open a focused PR for first feedback.
