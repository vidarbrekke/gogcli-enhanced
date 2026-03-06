# gws sample stdout/stderr for same fixtures as gog

Same logical fixtures as the native goldens: **gmail labels list** (success) and **labels get → not_found** (error). We now have **401** (unauthenticated) and **404** (not_found) goldens; **403** (permission denied) is documented below for capture. With 401 + 403 + 404 we can fully freeze the Gmail error taxonomy mapping with real goldens.

**Golden files:** `gmail-labels-list-gws.json` (success), `gmail-labels-get-not-found-gws.json` (404), `gmail-labels-401-unauthenticated-gws.json` (401), `gmail-labels-403-forbidden-gws.json` (403). **The developer has no app or environment access;** the 403 golden must be captured by the maintainer (see `CAPTURE-403-RUNBOOK.md`) and committed.

---

## 1) gws error envelope (real sample — no auth)

**Fixture:** Any Gmail call without credentials (equivalent “auth error” for parity comparison).

**Command:** `gws gmail users labels list --params '{"userId":"me"}'`

**Exit code:** 1

**stdout (exact):**

```json
{
  "error": {
    "code": 401,
    "message": "Access denied. No credentials provided. Run `gws auth login` or set GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE to an OAuth credentials JSON file.",
    "reason": "authError"
  }
}
```

**stderr:** (empty in this run; gws may use stderr for non-JSON hints, e.g. API not enabled.)

**Notes:** gws uses **stdout** for both success and error JSON. Root key is `error`; fields are `code` (numeric HTTP-style), `message`, `reason` (e.g. `authError`). No `error_code` (string), `service`, `operation`, `resource_id`, or `opId` like gog — normalization will need to map `code`/`reason` → stable `error_code` and add service/operation if required.

---

## 2) gws success: gmail labels list — capture required

**Fixture:** Same as `gog --json gmail labels list` (list all labels).

**Command:**

```bash
gws gmail users labels list --params '{"userId":"me"}'
```

**Capture:** Run with gws authenticated; save **full stdout** to `docs/merge/goldens/gmail-labels-list-gws.json` (replace the placeholder content there). That file is the golden for list success. Once uploaded, the reviewer will produce `gmail-labels-list.result.schema.json` (strict schema for the fields we promise: e.g. `labels[].id`, `labels[].name`, optional `labels[].type`).

**Note:** gws uses `GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE` for API calls. After `gws auth login`, run `gws auth export --unmasked > ~/.config/gws/credentials.json` and `export GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE=~/.config/gws/credentials.json` (or set that env var to the exported JSON path) so the labels command returns data instead of 401.

**Expected shape (Gmail API):** Root object with `labels` array; each label has `id`, `name`, `type`, and optionally `messageListVisibility`, `labelListVisibility`, `messagesTotal`, `messagesUnread`, `threadsTotal`, `threadsUnread`.

---

## 3) gws error: labels get not_found (404) — capture required

**Fixture:** Same as `gog --json gmail labels get <nonExistentId>` → not_found.

**Command (use a label ID that does not exist in your mailbox):**

```bash
gws gmail users labels get --params '{"userId":"me","id":"Label_DoesNotExist_123"}'
```

**Capture:** Run with gws authenticated; gws returns error JSON on **stdout** (exit code non-zero). Save **full stdout** to `docs/merge/goldens/gmail-labels-get-not-found-gws.json` (replace the placeholder content there). That file is the golden for get 404. Once uploaded, the reviewer will produce `gmail-labels-get.error.schema.json`.

**Expected:** Gmail API 404 typically gives `code: 404`, `reason: "notFound"` (or similar) in the `error` object.

---

## 4) gws error: 403 (permission denied) — capture for error taxonomy

**Fixture:** Gmail labels list with credentials that **do not have Gmail scope** (e.g. OAuth client with only Drive or other scopes). The API returns 403 Forbidden / Insufficient Permission.

**Who captures:** The **developer has no app or environment access.** A maintainer (someone with gws + Google Cloud Console) must capture the 403 once and commit the golden. See **`CAPTURE-403-RUNBOOK.md`** in this directory for step-by-step instructions.

**How to capture (maintainer):**

1. Create a separate OAuth client (or use an existing one) that has **no** Gmail scope — e.g. only `https://www.googleapis.com/auth/drive.readonly`.
2. Run `gws auth login` with that client (point `~/.config/gws/client_secret.json` at that client’s JSON, then login).
3. Export: `gws auth export --unmasked > ~/.config/gws/credentials-no-gmail.json`.
4. Run labels list with that credentials file:
   ```bash
   GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE=~/.config/gws/credentials-no-gmail.json gws gmail users labels list --params '{"userId":"me"}'
   ```
5. Save **full stdout** to `docs/merge/goldens/gmail-labels-403-forbidden-gws.json`. Delete any `_replace_with_*` placeholder key if present.

**Expected shape:** gws stdout is JSON with `error.code` 403 and a `reason` such as `forbidden` or `insufficientPermissions` (exact wording may vary). Use the real capture to freeze the Gmail error taxonomy.

---

## 5) Summary: gws vs gog envelope differences

| Aspect | gog | gws |
|--------|-----|-----|
| Error stream | stderr | stdout |
| Success stream | stdout | stdout |
| Error root | `error` object | `error` object |
| Error code | `error_code` (string, e.g. invalid_argument, not_found) | `code` (number, HTTP-style) + `reason` (string) |
| Context | `service`, `operation`, `resource_id`, optional `opId` | (none in sample; may vary by method) |
| Message | `message` | `message` |

For parity runner: normalize gws `error.code`/`error.reason` → gog `error_code`; optionally inject `service`/`operation`/`resource_id` from command context.

---

## 6) Commands reference (copy-paste)

```bash
# 401 (no credentials) — golden: gmail-labels-401-unauthenticated-gws.json
unset GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE
gws gmail users labels list --params '{"userId":"me"}' > docs/merge/goldens/gmail-labels-401-unauthenticated-gws.json

# Success — golden: gmail-labels-list-gws.json (set GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE first)
gws gmail users labels list --params '{"userId":"me"}' > docs/merge/goldens/gmail-labels-list-gws.json

# 404 — golden: gmail-labels-get-not-found-gws.json
gws gmail users labels get --params '{"userId":"me","id":"Label_DoesNotExist_123"}' > docs/merge/goldens/gmail-labels-get-not-found-gws.json

# 403 — golden: gmail-labels-403-forbidden-gws.json (use credentials without Gmail scope; see section 4)
GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE=~/.config/gws/credentials-no-gmail.json gws gmail users labels list --params '{"userId":"me"}' > docs/merge/goldens/gmail-labels-403-forbidden-gws.json
```

**After 403 is captured:** Replace the placeholder in `gmail-labels-403-forbidden-gws.json` with the real gws stdout (and remove any `_replace_with_*` key). Then 401 + 403 + 404 goldens fully define the Gmail error taxonomy for parity mapping.
