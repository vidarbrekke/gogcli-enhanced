# GWS routing extension — strategy choice

**Scope:** Add live gws routing for Tier A Drive read commands beyond `drive ls`: **drive get** and **drive search**. Docs/Sheets remain native-only for now (no gws goldens; don’t fix what’s not broken).

---

## 1. Four strategies evaluated

**Strategy A — Copy-paste per command**  
Add `RunDriveGet` / `RunDriveSearch` in `internal/backend/gws` and, in `drive.go`, duplicate the existing pattern for each: `runGWSGet` / `runGWSSearch` with the same classify → normalize → `BackendError` / write block as in `runGWSLs`. **Complexity:** Low. **DRY:** Poor (same ~15-line block repeated). **YAGNI:** Good (only what’s needed). **Scalability:** Poor (every new command repeats the block).

**Strategy B — Generic gws runner + shared response handler**  
Introduce `gws.Run(ctx, service, method, params)` and a single `handleGWSResult(ctx, res, resultKey, tableWriter)` in cmd that does classify, normalize, `BackendError`, then `writeGWSJSON` or table writer. Each command calls the helper. **Complexity:** Medium. **DRY:** Good. **YAGNI:** Slight overkill for two new commands. **Scalability:** Good (new command = one `Run` + one helper call).

**Strategy C — Config-driven mapping**  
YAML/JSON maps gog command → gws service/method/param template; one interpreter runs gws and the shared normalize/write pipeline. **Complexity:** High. **DRY:** Best. **YAGNI:** Bad (config schema, loader, many commands not yet needed). **Scalability:** Best if we grow to many commands.

**Strategy D — Minimal extension + one shared helper**  
Add only **Drive get** and **Drive search** (no Docs/Sheets). Add `RunDriveGet` and `RunDriveSearch` in gws. Extract a **single** helper in cmd that runs “classify → normalize error → return `BackendError` or call success callback”; refactor `runGWSLs` to use it and add `runGWSGet` / `runGWSSearch` that use the same helper. No generic runner, no config. **Complexity:** Low. **DRY:** Improved (one helper, three call sites). **YAGNI:** Best (only Drive get/search). **Scalability:** Same as today (add more commands by adding `RunX` + one call to the helper).

---

## 2. Comparison and choice

| Criterion    | A (copy-paste) | B (generic runner) | C (config-driven) | D (minimal + helper) |
|-------------|----------------|---------------------|-------------------|-----------------------|
| Complexity  | Low            | Medium              | High              | Low                   |
| DRY         | Poor           | Good                | Best              | Good                  |
| YAGNI       | Good           | Slight overkill     | Bad               | Best                  |
| Scalability | Poor           | Good                | Best              | Adequate              |
| Don’t fix what’s not broken | Yes | Yes                 | No                | Yes                   |

**Chosen: Strategy D.** It extends routing only for the two Drive commands we need, improves DRY with one small helper (used by ls, get, search), avoids new abstractions or config, and leaves native path and Docs/Sheets untouched. Future commands (e.g. Docs info) can follow the same pattern: add `RunX` in gws and one `runGWSX` that uses the helper.

---

## 3. Implementation notes

- **backend/gws:** `RunDriveGet(ctx, fileId string)`, `RunDriveSearch(ctx, query, pageToken string, pageSize int64)` invoking `gws drive files get` and `gws drive files list` with `--params` JSON.
- **cmd/drive.go:** `handleGWSResult(ctx, service, operation, resourceID string, res gws.Result, onSuccess func([]byte) error) error`; refactor `runGWSLs` to use it; add `runGWSGet` / `runGWSSearch` and `writeGWSDriveGetTable`; in `DriveGetCmd.Run` / `DriveSearchCmd.Run`, when `Backend() == BackendGWS` and no `--page-count` / no `--all`, call gws path.
- **GWS get:** No `--page-count` when using gws (native path used if user passes `--page-count`).
- **GWS search:** No `--all` when using gws (single-page only; native path if `--all`).
