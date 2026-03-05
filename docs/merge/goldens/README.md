# Golden Fixtures for Provider Parity

This directory holds **canonical JSON outputs** from native `gog` commands. Use them to:

1. **Define contracts** — Schemas in `../schemas/` describe these shapes.
2. **Compare backends** — Run the same logical request through native and gws; diff against these goldens (or normalize first).
3. **Classify normalization rules** — Document which diffs are acceptable vs require normalization.

## Source of truth

- **Native goldens** are produced by current `gog` with the Go Google API clients. They match the exact stdout of `gog --json <command> ...` for the given fixture inputs.
- Fixtures are either from **unit-test mock responses** (deterministic, no live account) or from **one-time capture** against a real account (see "How to capture" below).

## Naming

- `*-native.json` — output from native gog (current implementation).
- `*-gws.json` — (optional) output from gws for the same logical request; add once gws adapter exists for comparison.

## How to capture native output (for reviewers)

With a built `gog` and a configured account:

```bash
# Gmail labels list
gog --json gmail labels list > docs/merge/goldens/gmail-labels-list-native.json

# Gmail labels get (use a label ID or name that exists in your mailbox)
gog --json gmail labels get INBOX > docs/merge/goldens/gmail-labels-get-native.json

# Drive ls (root, one page)
gog --json drive ls --max 5 > docs/merge/goldens/drive-ls-native.json
```

To get **deterministic** goldens without a live account, run the existing unit tests and capture the JSON they produce (e.g. by temporarily writing stdout to a file in the test). The checked-in samples in this directory are currently **test-fixture based** so they are stable and safe for CI.

## Diff and normalization

1. Produce gws output for the same logical request (same account/inputs).
2. Diff `*-native.json` vs `*-gws.json` (e.g. `diff`, or a JSON-aware diff tool).
3. Classify each difference:
   - **Normalize**: gws output must be transformed to match native (document rule in `../schemas/` or in command dossier).
   - **Accept + detect**: difference is acceptable; record it so we can detect regressions (e.g. optional field presence).
   - **Pin**: native is source of truth; gws must not change this field (strict parity).

See `../discovery-drift-policy.md` for when to pin/capture vs accept+detect.

## Files in this directory

| File | Command | Description |
|------|---------|-------------|
| `gmail-labels-list-native.json` | `gog --json gmail labels list` | List response (test fixture). |
| `gmail-labels-get-native.json` | `gog --json gmail labels get INBOX` | Single label with counts (test fixture). |
| `drive-ls-native.json` | `gog --json drive ls` | One page of files (test fixture). |
| `gmail-labels-list-gws.json` | `gws gmail users labels list --params '{"userId":"me"}'` | **Placeholder** — replace with gws stdout when captured (authenticated). |
| `gmail-labels-get-not-found-gws.json` | `gws gmail users labels get --params '{"userId":"me","id":"Label_DoesNotExist_123"}'` | **Placeholder** — replace with gws 404 stdout when captured (authenticated). |
