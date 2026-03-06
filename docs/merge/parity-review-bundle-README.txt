# Parity review bundle — file map for external reviewer

**Requested vs actual paths**

| You asked for | We have |
|---------------|---------|
| `gmail-labels-list.result.schema.json` | `docs/merge/schemas/gmail-labels-list.json` (success response schema) |
| `gmail.error.schema.json` | No separate file. Error envelope is code-defined in `internal/parity/normalize/normalize.go` (CanonicalEnvelope + comment: contractual vs drift). |
| `gmail-labels-get-404-not-found/` | `docs/merge/goldens/gmail-labels-get-not-found/` |
| `gmail-labels-list-success/` | `docs/merge/goldens/gmail-labels-list/` (native + gws) |
| `gmail-labels-403-permission-denied/` | `docs/merge/goldens/gmail-labels-403-forbidden/` (placeholder) |

**Bundle contents**

1. **Runner + orchestration:** `cmd/gog-parity/main.go`, `main_test.go`
2. **Engines:** `internal/parity/io/*.go`, `classify/*.go`, `normalize/*.go`, `diff/*.go`, `schema/*.go`
3. **Tests:** all `*_test.go` under `internal/parity/` and `cmd/gog-parity/`
4. **Wiring:** `docs/merge/PARITY-REVIEWER-CHECKLIST.md`, `Makefile`, `.github/workflows/parity.yml`
5. **Fixtures + schemas:** `docs/merge/schemas/gmail-labels-list.json`, `gmail-labels-get.json`; `docs/merge/goldens/gmail-labels-401-unauthenticated/`, `gmail-labels-get-not-found/`, `gmail-labels-list/`, `gmail-labels-403-forbidden/`

Run parity locally: `make parity` or `go run ./cmd/gog-parity --fixtures docs/merge/goldens --schemas docs/merge/schemas --provider gws`.
