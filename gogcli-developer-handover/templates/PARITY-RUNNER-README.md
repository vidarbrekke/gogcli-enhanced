# Parity Runner (Minimal Skeleton)

Goal: run fixtures through normalization + schema validation + diff, and emit a JSON report.

## Non-goals (YAGNI)
- No live API calls.
- No automatic discovery pinning.
- No complex diff UI.

## Inputs
- Fixture root: `docs/merge/goldens/<case>/<provider>/{stdout.json,stderr.json,exit_code.txt}`
- Schemas: `docs/merge/schemas/*.json`
- Normalization rules docs: `docs/merge/normalization/*.md` (human reference)

## Output
- `report.json` with:
  - `breaking[]`, `drift[]` (each a json-pointer + summary)
  - `provider`, `provider_version`, optional `discovery_snapshot_hash`
  - `normalizations_applied[]`

## Suggested implementation modules (Go)
- `internal/parity/io` (load fixtures)
- `internal/parity/classify` (success vs error; stdout/stderr handling)
- `internal/parity/normalize` (provider-specific normalization, e.g., gws errors → canonical)
- `internal/parity/schema` (jsonschema validation)
- `internal/parity/diff` (canonicalized deep diff + allowlists)
- `cmd/gog-parity` (CLI entrypoint)

## Libraries
- JSON Schema: github.com/santhosh-tekuri/jsonschema/v5 (simple, widely used in Go)
- Diff: implement minimal recursive diff (YAGNI) or use github.com/google/go-cmp with custom transformers.
