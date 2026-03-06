# Surgical review bundle — where the three issues live

**1. cmd/gog-parity/main.go**
- Case loop + silent skips: lines 83–165. ProvidersForCase error → CaseResult Outcome "ERROR" (no runnerFailure). LoadFixture error → same. Schema load/validate: only run when err == nil (116–127). Diff: only when Unmarshal succeeds (132–133). None of these record runnerFailures or fail CI.
- Report emission: 161–165.
- Exit logic: 166–173. Only checks report.Breaking; 401/404 never add to breaking (they're ERROR path, no diff/schema), so hard-gate never fires.

**2. internal/parity/diff/*.go**
- diff.go: diffNode iterates map keys with `for k := range allKeys` (line 79) — allKeys is a map, so iteration order is nondeterministic. Same in diffLabelsSetByID (idsSorted is sorted, but diffEntry order from diffNode for object keys is not). Resulting `all` slice order depends on map iteration before the final split into breaking/drift.

**3. internal/parity/schema/*.go**
- schema.go: LoadSchema returns error; Validate returns (violations, error). main.go calls both but only proceeds when err == nil. No runnerFailure or exit on schema not found / invalid schema / validation failure.

**4. internal/parity/io/*.go**
- io.go: LoadFixture returns error on missing/unreadable file or bad exit_code parse. main.go treats that as CaseResult Outcome "ERROR" and continues; no runnerFailure[], no CI fail.

**5. docs/merge/sample-parity-report.json**
- Current CI-style output from: go run ./cmd/gog-parity --fixtures docs/merge/goldens --schemas docs/merge/schemas --provider gws

**Fix plan:** docs/merge/PARITY-CONTROL-PLANE-FIXES.md (in repo and in full bundle).
