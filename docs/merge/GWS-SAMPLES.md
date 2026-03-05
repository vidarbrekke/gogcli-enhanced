# gws sample stdout/stderr for same fixtures as gog

Same logical fixtures as the native goldens: **gmail labels list** (success) and **labels get → not_found** (error). Right now only **one** real gws sample exists (401 auth error); the other two are capture instructions. Once those two are captured and uploaded, we can lock the Gmail parity suite end-to-end with concrete goldens (native vs gws), strict payload schema for the fields we promise (e.g. `labels[].id`, `labels[].name`, optional `labels[].type`), and exact allowlisted diffs. A parity-runner v2 that treats stdout-as-error JSON correctly will prevent the biggest class of false failures.

**If you capture (1) and (2) below and upload/paste those two gws JSON outputs**, the reviewer will produce ready-to-commit:
- `gmail-labels-list.result.schema.json`
- `gmail-labels-get.error.schema.json`
that minimize drift and maintenance.

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

## 4) Summary: gws vs gog envelope differences

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

## 5) Commands reference (copy-paste)

```bash
# 1) Auth error (no credentials) — already captured in section 1
gws gmail users labels list --params '{"userId":"me"}'

# 2) Labels list (success) — run with gws auth; save stdout to goldens
gws gmail users labels list --params '{"userId":"me"}' > docs/merge/goldens/gmail-labels-list-gws.json

# 3) Labels get not_found — run with gws auth; save stdout to goldens
gws gmail users labels get --params '{"userId":"me","id":"Label_DoesNotExist_123"}' > docs/merge/goldens/gmail-labels-get-not-found-gws.json
```

**After (2) and (3) are captured:** Replace the placeholder content in those two golden files with the real gws stdout, then commit or upload. The reviewer will then produce `gmail-labels-list.result.schema.json` and `gmail-labels-get.error.schema.json` and we can lock the Gmail parity suite with allowlisted diffs.
