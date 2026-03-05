# Native gog envelope outputs (for envelope.schema.json and parity runner)

Real stdout/stderr from current gog. Use these to draft **envelope.schema.json**, **diff rules for the parity runner**, and a **minimal golden capture checklist**.

---

## 1) Error envelope (stderr)

When a command fails in `--json` mode, gog writes a **single JSON object** to **stderr**. Success payload (if any) goes to stdout; on failure stdout is typically empty.

**Example trigger:** `gog --json --op-id op-test docs edit insert d1 "   "`  
(empty text → invalid_argument)

**Exact stderr (native gog):**

```json
{"error":{"error_code":"invalid_argument","message":"empty text","opId":"op-test","operation":"insert","resource_id":"d1","service":"docs"}}
```

**Field semantics:**

| Field | Location | Required | Description |
|-------|----------|----------|-------------|
| `error` | root | yes | Single object containing all error details. |
| `error.message` | error | yes | Human-readable message (errfmt.Format). |
| `error.error_code` | error | when available | Stable code (e.g. invalid_argument, not_found, permission_denied). From EditError or fallback (e.g. parse_error, command_not_enabled). |
| `error.service` | error | when available | Service name (e.g. docs, sheets, gmail). |
| `error.operation` | error | when available | Operation name (e.g. insert, batch). |
| `error.resource_id` | error | when available | Resource ID (doc_id, spreadsheet_id, etc.). |
| `error.opId` | error | when set | Set when `--op-id` or `GOG_OP_ID` is provided. |
| `error.http_status` | error | when available | HTTP status from Google API (e.g. 404). |
| `error.google_reason` | error | when available | API reason (e.g. notFound). |
| `error.request_index` | error | when available | Index into batch request (for batch failures). |

**Minimal error envelope** (e.g. generic/parse errors): root has `error` with at least `message`; `error_code` may be set via fallback. Other fields only when the error implements `JSONErrorFields()` (e.g. EditError).

---

## 2) Success envelope with metadata (stdout)

For **edit/validate/dry-run** flows, success stdout can include **opId** and **requestHash** (when `--op-id` is set). Read-only commands (e.g. `gmail labels list`) currently write **payload only** to stdout (no wrapper; no opId unless we add it later).

**Example trigger:** `gog --json --op-id op-test docs edit insert d1 "hello" --validate-only`

**Exact stdout (native gog):**

```json
{
  "documentId": "d1",
  "index": 1,
  "insertedChars": 5,
  "opId": "op-test",
  "requestHash": "72a01d689ace6c8f04cb83a5ad56cc2339e16dd34cf62d9534ec8c24b817d868",
  "valid": true,
  "validateOnly": true
}
```

**Field semantics (success with metadata):**

| Field | Required | Description |
|-------|----------|-------------|
| `opId` | when set | Echoed from `--op-id` / `GOG_OP_ID`. |
| `requestHash` | in validate/dry-run | SHA256 hex of normalized request (64 chars). |
| Command-specific | varies | e.g. documentId, validateOnly, valid, dryRun, service, resourceId, request. |

**Read-command success** (e.g. `gog --json gmail labels list`): stdout is the **payload only** (e.g. `{"labels":[...]}`), no top-level opId/requestHash unless we extend.

---

## 3) Stream / convention summary

- **stdout:** One JSON value per successful command (object or array). No leading/trailing newlines beyond the single value. Edit flows may add `opId` and `requestHash` at top level.
- **stderr:** On failure in JSON mode, exactly one JSON object: `{"error":{...}}`. No other stderr output in JSON mode for this envelope.
- **Exit code:** Non-zero on failure; 0 on success (or when command exits 0 despite internal error—see stableExitCode).

---

## 4) For the reviewer: what to draft

From these envelopes you can:

1. **envelope.schema.json** — One schema (or split success/error) that describes:
   - Root `error` object and all known fields (message, error_code, service, operation, resource_id, opId, http_status, google_reason, request_index).
   - Optional: success envelope shape when opId/requestHash are present (top-level optional fields).

2. **Exact diff rules for parity runner:**
   - **Error:** Compare stderr as single JSON; require same `error.error_code`, `error.service`, `error.operation`, `error.resource_id`; allow `message` to differ by backend; optional fields (http_status, google_reason) normalize or accept+detect per discovery-drift-policy.
   - **Success:** Compare stdout as single JSON; require same payload keys that are in the contract; opId/requestHash optional or normalize; ignore ordering of keys if not significant.

3. **Minimal golden capture checklist:**
   - One golden per command (or per outcome type): success payload **or** success envelope (if metadata used); one error envelope per command or per error_code.
   - Capture: stdout and stderr separately; exit code.
   - Don’t gold every flag combination—only representative cases that exercise envelope shape and critical fields.

These samples are the real native gog envelope output; use them as the source of truth for schema and diff rules.
