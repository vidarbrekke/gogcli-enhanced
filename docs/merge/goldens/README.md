# Golden Fixtures for Provider Parity

This directory holds **canonical outputs** from native `gog` and optional provider (e.g. gws) commands. Use them to:

1. **Define contracts** — Schemas in `../schemas/` describe these shapes.
2. **Compare backends** — Run the same logical request through native and gws; diff against these goldens (or normalize first).
3. **Classify normalization rules** — Document which diffs are acceptable vs require normalization.

## Layout

Fixtures are organized by **case** and **provider**:

```
goldens/
  <case>/                    # e.g. gmail-labels-list, gmail-labels-401-unauthenticated
    <provider>/              # native | gws
      stdout.json            # JSON written to stdout (success or error envelope)
      stderr.json            # JSON written to stderr (often {} for success)
      exit_code.txt          # single line: integer exit code (0 = success)
```

The parity runner (`cmd/gog-parity`) discovers cases by scanning for immediate subdirectories that contain at least one provider subdirectory with `stdout.json`, `stderr.json`, and `exit_code.txt`.

**Note:** gws often emits error JSON on **stdout** (exit code 0). Classification must check both stdout and stderr for a top-level `error` key.

## Source of truth

- **Native goldens** are produced by current `gog` with the Go Google API clients. They match the exact stdout of `gog --json <command> ...` for the given fixture inputs.
- Fixtures are either from **unit-test mock responses** (deterministic, no live account) or from **one-time capture** against a real account (see "How to capture" below).

## Cases in this directory

| Case | Providers | Description |
|------|-----------|-------------|
| `gmail-labels-list` | native, gws | List labels success. |
| `gmail-labels-get` | native | Single label with counts (e.g. INBOX). |
| `gmail-labels-get-not-found` | gws | 404 not_found (authenticated, non-existent label ID). |
| `gmail-labels-401-unauthenticated` | gws | 401 authError (no credentials). |
| `gmail-labels-403-forbidden` | gws | 403 — placeholder until real capture (see `../CAPTURE-403-RUNBOOK.md`). |
| `drive-ls` | native | One page of files (test fixture). |

## How to capture native output (for reviewers)

With a built `gog` and a configured account:

```bash
# Create case/provider dir first, e.g. for a new case:
mkdir -p docs/merge/goldens/gmail-labels-list/native

# Gmail labels list
gog --json gmail labels list > docs/merge/goldens/gmail-labels-list/native/stdout.json
echo '{}' > docs/merge/goldens/gmail-labels-list/native/stderr.json
echo 0 > docs/merge/goldens/gmail-labels-list/native/exit_code.txt

# Gmail labels get (use a label ID or name that exists in your mailbox)
mkdir -p docs/merge/goldens/gmail-labels-get/native
gog --json gmail labels get INBOX > docs/merge/goldens/gmail-labels-get/native/stdout.json
echo '{}' > docs/merge/goldens/gmail-labels-get/native/stderr.json
echo 0 > docs/merge/goldens/gmail-labels-get/native/exit_code.txt

# Drive ls (root, one page)
mkdir -p docs/merge/goldens/drive-ls/native
gog --json drive ls --max 5 > docs/merge/goldens/drive-ls/native/stdout.json
echo '{}' > docs/merge/goldens/drive-ls/native/stderr.json
echo 0 > docs/merge/goldens/drive-ls/native/exit_code.txt
```

To get **deterministic** goldens without a live account, run the existing unit tests and capture the JSON they produce. The checked-in samples in this directory are currently **test-fixture based** so they are stable and safe for CI.

## Diff and normalization

1. Produce gws output for the same logical request (same account/inputs).
2. Save under `<case>/gws/stdout.json`, `stderr.json`, `exit_code.txt`.
3. Run the parity runner: `go run ./cmd/gog-parity --fixtures docs/merge/goldens --schemas docs/merge/schemas --provider gws`.
4. Classify each difference: **Normalize** (gws must match), **Accept + detect** (drift-only), or **Pin** (strict parity). See `../discovery-drift-policy.md`.
